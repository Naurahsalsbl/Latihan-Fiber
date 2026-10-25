
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/config"
	"latihan-fiber/modul4/database"
	"latihan-fiber/modul4/helper"
	
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxCredits = 24

type TestUser struct {
	ID       int
	Username string
	Role     string
	Token    string
}

type Course struct {
	ID            int
	Code          string
	Credits		  int
	Quota         int
	EnrolledCount int
	Version		  int
}

func fire(n int, job func(i int) string) map[string]int {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int, out chan<- string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}

	close(start)
	wg.Wait()
	close(results)

	tally := make(map[string]int)
	for result := range results {
		tally[result]++
	}

	return tally
}

func main() {
	mode := flag.String("mode", "crowd", "Mode load test: crowd atau credits")
	flag.Parse()

	if *mode != "crowd" && *mode != "credits" && *mode != "swap" && *mode != "sixth" && *mode != "duplicate" && *mode != "drop" && *mode != "swapfull" && *mode != "quota" && *mode != "quota-forbidden" && *mode != "quota-below-enrolled" && *mode != "supervision-quota" {
		fmt.Println("Mode tidak dikenal. Gunakan -mode crowd, -mode credits, -mode swap, mode sixth, mode duplicate, mode drop, mode swapfull, mode quota, mode quota-forbidden, mode quota-below-enrolled, atau mode supervision-quota")
		os.Exit(1)
	}

	config.LoadEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := database.NewPool(ctx)
	if err != nil {
		fmt.Println("Gagal terhubung ke database:", err)
		os.Exit(1)
	}
	defer pool.Close()

	secret := config.GetEnv("JWT_SECRET", "")
	if len(secret) < 32 {
		fmt.Println("JWT_SECRET belum diisi atau panjangnya kurang dari 32 karakter.")
		os.Exit(1)
	}

	jwtManager := helper.NewJWTManager(
		secret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	users, err := loadUsers(ctx, pool, jwtManager)
	if err != nil {
		fmt.Println("Gagal menyiapkan akun uji:", err)
		os.Exit(1)
	}

	baseURL := strings.TrimRight(
		config.GetEnv(
			"LOADTEST_BASE_URL",
			"http://localhost:"+config.GetEnv("APP_PORT", "3000"),
		),
		"/",
	)

	client := &http.Client{Timeout: 30 * time.Second}

	switch *mode {
	case "crowd":
		if len(users) < 60 {
			fmt.Printf("Mode crowd membutuhkan 60 akun mhs%%; ditemukan %d.\n", len(users))
			os.Exit(1)
		}
		runCrowd(ctx, pool, users[:60], baseURL, client)

	case "credits":
		if len(users) < 10 {
			fmt.Printf("Mode credits membutuhkan minimal 10 akun mhs%%; ditemukan %d.\n", len(users))
			os.Exit(1)
		}
		runCredits(ctx, pool, users[:10], baseURL, client)

	case "swap":
		if len(users) < 40 {
			fmt.Printf("Mode swap membutuhkan 40 akun mhs%%; ditemukan %d.\n", len(users))
			os.Exit(1)
		}
		runSwap(ctx, pool, users[:40], baseURL, client)

	case "sixth":
		if len(users) < 6 {
			fmt.Printf("Mode sixth membutuhkan minimal 6 akun mhs%%; ditemukan %d.\n", len(users))
			os.Exit(1)
		}
		runSixth(ctx, pool, users[:6], baseURL, client)
	
	case "duplicate":
		if len(users) < 1 {
			fmt.Println("Mode duplicate membutuhkan minimal 1 akun mahasiswa.")
			os.Exit(1)
		}
		runDuplicate(ctx, pool, users[:1], baseURL, client)

	case "drop":
		if len(users) < 1 {
			fmt.Println("Mode drop membutuhkan minimal 1 akun mahasiswa.")
			os.Exit(1)
		}
		runDrop(ctx, pool, users, baseURL, client)

	case "swapfull":
		if len(users) < 1 {
			fmt.Println("Mode swapfull membutuhkan minimal 1 akun mahasiswa.")
			os.Exit(1)
		}
		runSwapFull(ctx, pool, users, baseURL, client)
	
	case "quota":
		runQuota(ctx, pool, jwtManager, baseURL, client)

	case "quota-forbidden":
		if len(users) < 1 {
			fmt.Println("Mode quota-forbidden membutuhkan minimal 1 akun mahasiswa.")
			os.Exit(1)
		}
		runQuotaForbidden(ctx, pool, users[0], baseURL, client)

	case "quota-below-enrolled":
    	runQuotaBelowEnrolled(ctx, pool, jwtManager, baseURL, client)

	case "supervision-quota":
    	runSupervisionQuota(ctx, pool, jwtManager, baseURL, client)
	}

}

func loadUsers(
	ctx context.Context,
	pool *pgxpool.Pool,
	jwtManager *helper.JWTManager,
) ([]TestUser, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, username, role
		FROM users
		WHERE username LIKE 'mhs%'
		ORDER BY username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []TestUser
	for rows.Next() {
		var user TestUser
		if err := rows.Scan(&user.ID, &user.Username, &user.Role); err != nil {
			return nil, err
		}

		token, err := jwtManager.GenerateAccess(model.User{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		})
		if err != nil {
			return nil, err
		}

		user.Token = token
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func loadCourseByCode(
	ctx context.Context,
	pool *pgxpool.Pool,
	code string,
) (Course, error) {
	var course Course
	err := pool.QueryRow(ctx, `
		SELECT id, code, credits, quota, enrolled_count, version
		FROM courses
		WHERE code = $1
	`, code).Scan(
		&course.ID,
		&course.Code,
		&course.Credits,
		&course.Quota,
		&course.EnrolledCount,
		&course.Version,
	)

	if err != nil {
        return Course{}, err
    }

	return course, err
}

func loadRegularCourses(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]Course, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, code, credits, quota, enrolled_count
		FROM courses
		WHERE code <> 'IF399'
		ORDER BY code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []Course
	for rows.Next() {
		var course Course
		if err := rows.Scan(
			&course.ID,
			&course.Code,
			&course.Credits,
			&course.Quota,
			&course.EnrolledCount,
		); err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(courses) != 12 {
		return nil, fmt.Errorf("dibutuhkan 12 kelas reguler, ditemukan %d", len(courses))
	}
	return courses, nil
}

func sendEnroll(
	client *http.Client,
	baseURL string,
	user TestUser,
	course Course,
) string {
	url := fmt.Sprintf("%s/api/v1/courses/%d/enroll", baseURL, course.ID)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader("{}"))
	if err != nil {
		return "ERROR membuat request"
	}

	req.Header.Set("Authorization", "Bearer "+user.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "ERROR mengirim request: " + err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyText := string(body)

	if resp.StatusCode == http.StatusCreated {
		return "HTTP 201"
	}
	if resp.StatusCode == http.StatusConflict {
		switch {
		case strings.Contains(bodyText, "CREDIT_LIMIT_EXCEEDED"):
			return "HTTP 409 CREDIT_LIMIT_EXCEEDED"
		case strings.Contains(bodyText, "COURSE_FULL"):
			return "HTTP 409 COURSE_FULL"
		case strings.Contains(bodyText, "DUPLICATE"),
			strings.Contains(bodyText, "ALREADY_ENROLLED"):
			return "HTTP 409 DUPLICATE"
		}
	}

	return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(bodyText))
}

func sendSwap(
    client *http.Client,
    baseURL string,
    user TestUser,
    fromCourseID int,
    toCourseID int,
) string {
    url := baseURL + "/api/v1/enrollments/swap"

    body := fmt.Sprintf(
        `{"from_course_id":%d,"to_course_id":%d}`,
        fromCourseID,
        toCourseID,
    )

    req, err := http.NewRequest(
        http.MethodPost,
        url,
        strings.NewReader(body),
    )
    if err != nil {
        return "ERROR membuat request"
    }

    req.Header.Set("Authorization", "Bearer "+user.Token)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return "ERROR mengirim request: " + err.Error()
    }
    defer resp.Body.Close()

    responseBody, _ := io.ReadAll(resp.Body)

    if resp.StatusCode == http.StatusOK {
        return "HTTP 200"
    }

    return fmt.Sprintf(
        "HTTP %d: %s",
        resp.StatusCode,
        strings.TrimSpace(string(responseBody)),
    )
}

func sendDrop(
	client *http.Client,
	baseURL string,
	user TestUser,
	course Course,
) string {
	url := fmt.Sprintf("%s/api/v1/courses/%d/enroll", baseURL, course.ID)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return "ERROR membuat request"
	}

	req.Header.Set("Authorization", "Bearer "+user.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "ERROR mengirim request: " + err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyText := string(body)

	if resp.StatusCode == http.StatusNoContent {
		return "HTTP 204"
	}

	if resp.StatusCode == http.StatusConflict &&
		strings.Contains(bodyText, "NOT_ENROLLED") {
		return "HTTP 409 NOT_ENROLLED"
	}

	return fmt.Sprintf(
		"HTTP %d: %s",
		resp.StatusCode,
		strings.TrimSpace(bodyText),
	)
}

func printResults(result map[string]int) {
	keys := make([]string, 0, len(result))
	for key := range result {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("%-40s : %d\n", key, result[key])
	}
}

func runCrowd(
	ctx context.Context,
	pool *pgxpool.Pool,
	users []TestUser,
	baseURL string,
	client *http.Client,
) {
	course, err := loadCourseByCode(ctx, pool, "IF399")
	if err != nil {
		fmt.Println("Gagal membaca kelas IF399:", err)
		os.Exit(1)
	}

	fmt.Println("=== LOAD TEST: CROWD ===")
	fmt.Println("Kelas:", course.Code, "| Kuota:", course.Quota)
	fmt.Println("Akun uji:", len(users))
	fmt.Println("URL:", baseURL)

	result := fire(len(users), func(i int) string {
		return sendEnroll(client, baseURL, users[i], course)
	})

	fmt.Println("\n=== HASIL HTTP ===")
	printResults(result)

	var enrolledCount, enrollmentRows int
	err = pool.QueryRow(ctx, `
		SELECT enrolled_count FROM courses WHERE id = $1
	`, course.ID).Scan(&enrolledCount)
	if err != nil {
		fmt.Println("Gagal membaca enrolled_count:", err)
		os.Exit(1)
	}
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM enrollments WHERE course_id = $1
	`, course.ID).Scan(&enrollmentRows)
	if err != nil {
		fmt.Println("Gagal menghitung enrollment:", err)
		os.Exit(1)
	}

	fmt.Println("\n=== VERIFIKASI DATABASE ===")
	fmt.Println("Kuota:", course.Quota)
	fmt.Println("enrolled_count:", enrolledCount)
	fmt.Println("Jumlah enrollment:", enrollmentRows)

	if result["HTTP 201"] == course.Quota &&
		result["HTTP 409 COURSE_FULL"] == len(users)-course.Quota &&
		enrolledCount == course.Quota &&
		enrollmentRows == course.Quota {
		fmt.Println("\nHASIL: PASS — kuota kelas terjaga.")
	} else {
		fmt.Println("\nHASIL: BELUM SESUAI TARGET.")
		fmt.Println("Pastikan IF399 kosong sebelum pengujian crowd.")
	}
}

func runCredits(
	ctx context.Context,
	pool *pgxpool.Pool,
	users []TestUser,
	baseURL string,
	client *http.Client,
) {
	courses, err := loadRegularCourses(ctx, pool)
	if err != nil {
		fmt.Println("Gagal membaca kelas reguler:", err)
		os.Exit(1)
	}

	fmt.Println("=== LOAD TEST: CREDITS ===")
	fmt.Println("Mahasiswa:", len(users))
	fmt.Println("Kelas reguler per mahasiswa:", len(courses))
	fmt.Println("Request serentak:", len(users)*len(courses))
	fmt.Println("Batas SKS:", maxCredits)
	fmt.Println("URL:", baseURL)
	fmt.Println("\nMengirim pendaftaran secara bersamaan...")

	type task struct {
		user   TestUser
		course Course
	}

	var tasks []task
	for _, user := range users {
		for _, course := range courses {
			tasks = append(tasks, task{user: user, course: course})
		}
	}

	result := fire(len(tasks), func(i int) string {
		t := tasks[i]
		status := sendEnroll(client, baseURL, t.user, t.course)
		return t.user.Username + " | " + status
	})

	counts := make(map[string]int)
	for label, count := range result {
		parts := strings.SplitN(label, " | ", 2)
		if len(parts) == 2 {
			counts[parts[1]] += count
		}
	}

	fmt.Println("\n=== HASIL HTTP ===")
	printResults(counts)

	fmt.Println("\n=== VERIFIKASI SKS PER MAHASISWA ===")
	allWithinLimit := true

	for _, user := range users {
		var totalCredits int
		err := pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(c.credits), 0)
			FROM enrollments e
			JOIN courses c ON c.id = e.course_id
			WHERE e.user_id = $1
		`, user.ID).Scan(&totalCredits)
		if err != nil {
			fmt.Printf("%s: gagal membaca total SKS: %v\n", user.Username, err)
			allWithinLimit = false
			continue
		}

		fmt.Printf("%-12s : %2d SKS\n", user.Username, totalCredits)
		if totalCredits > maxCredits {
			allWithinLimit = false
			fmt.Println("  !! MELEBIHI BATAS SKS !!")
		}
	}

	fmt.Println("\n=== KONSISTENSI JUMLAH ENROLLMENT ===")
	var inconsistent int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM courses c
		WHERE c.enrolled_count <> (
			SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id
		)
	`).Scan(&inconsistent)
	if err != nil {
		fmt.Println("Gagal memeriksa konsistensi:", err)
		os.Exit(1)
	}
	fmt.Println("Kelas dengan enrolled_count tidak konsisten:", inconsistent)

	if allWithinLimit && inconsistent == 0 &&
		counts["HTTP 201"]+counts["HTTP 409 CREDIT_LIMIT_EXCEEDED"]+
			counts["HTTP 409 COURSE_FULL"]+counts["HTTP 409 DUPLICATE"] == len(tasks) {
		fmt.Println("\nHASIL: PASS — tidak ada mahasiswa melebihi 24 SKS.")
	} else {
		fmt.Println("\nHASIL: BELUM PASS — periksa error HTTP, data enrollment, dan log server.")
	}
}

func runSwap(
    ctx context.Context,
    pool *pgxpool.Pool,
    users []TestUser,
    baseURL string,
    client *http.Client,
) {
    fromCourse, err := loadCourseByCode(ctx, pool, "IF101")
    if err != nil {
        fmt.Println("Gagal membaca IF101:", err)
        os.Exit(1)
    }

    toCourse, err := loadCourseByCode(ctx, pool, "IF102")
    if err != nil {
        fmt.Println("Gagal membaca IF102:", err)
        os.Exit(1)
    }

    var deadlocksBefore int64
    err = pool.QueryRow(ctx, `
        SELECT deadlocks
        FROM pg_stat_database
        WHERE datname = current_database()
    `).Scan(&deadlocksBefore)
    if err != nil {
        fmt.Println("Gagal membaca statistik deadlock:", err)
        os.Exit(1)
    }

    fmt.Println("=== LOAD TEST: SWAP ===")
    fmt.Println("Arah 1: mhs001-mhs020, IF101 -> IF102")
    fmt.Println("Arah 2: mhs021-mhs040, IF102 -> IF101")
    fmt.Println("Request serentak:", len(users))
    fmt.Println("URL:", baseURL)

    start := time.Now()

    result := fire(len(users), func(i int) string {
        fromID := fromCourse.ID
        toID := toCourse.ID

        if i >= 20 {
            fromID, toID = toCourse.ID, fromCourse.ID
        }

        status := sendSwap(
            client,
            baseURL,
            users[i],
            fromID,
            toID,
        )

        return status
    })

    elapsed := time.Since(start)

    fmt.Println("\n=== HASIL HTTP ===")
    printResults(result)
    fmt.Printf("\nWaktu total: %v\n", elapsed)

    var deadlocksAfter int64
    err = pool.QueryRow(ctx, `
        SELECT deadlocks
        FROM pg_stat_database
        WHERE datname = current_database()
    `).Scan(&deadlocksAfter)
    if err != nil {
        fmt.Println("Gagal membaca statistik deadlock:", err)
        os.Exit(1)
    }

    deadlocks := deadlocksAfter - deadlocksBefore
    fmt.Println("Deadlock terdeteksi PostgreSQL selama uji:", deadlocks)

    fmt.Println("\n=== VERIFIKASI DATABASE ===")

    consistent := true

    for _, code := range []string{"IF101", "IF102"} {
        course, err := loadCourseByCode(ctx, pool, code)
        if err != nil {
            fmt.Printf("%s: gagal membaca kelas: %v\n", code, err)
            consistent = false
            continue
        }

        var enrollmentRows int
        err = pool.QueryRow(ctx, `
            SELECT COUNT(*)
            FROM enrollments
            WHERE course_id = $1
        `, course.ID).Scan(&enrollmentRows)
        if err != nil {
            fmt.Printf("%s: gagal menghitung enrollment: %v\n", code, err)
            consistent = false
            continue
        }

        fmt.Printf(
            "%s: enrolled_count=%d, baris enrollments=%d\n",
            code,
            course.EnrolledCount,
            enrollmentRows,
        )

        if course.EnrolledCount != enrollmentRows {
            consistent = false
        }
        if enrollmentRows != 20 {
            consistent = false
        }
    }

    if result["HTTP 200"] == 40 &&
        deadlocks == 0 &&
        consistent &&
        elapsed < time.Second {
        fmt.Println("\nHASIL: PASS — swap berhasil, tanpa deadlock, data konsisten.")
    } else {
        fmt.Println("\nHASIL: BELUM PASS — periksa status HTTP, deadlock, waktu, dan database.")
    }
}

func runSixth(
    ctx context.Context,
    pool *pgxpool.Pool,
    users []TestUser,
    baseURL string,
    client *http.Client,
) {
    course, err := loadCourseByCode(ctx, pool, "IF399")
    if err != nil {
        fmt.Println("Gagal membaca kelas IF399:", err)
        os.Exit(1)
    }

    fmt.Println("=== TEST: MAHASISWA KE-6 ===")
    fmt.Println("Kelas:", course.Code)
    fmt.Println("Kuota:", course.Quota)
    fmt.Println("Enrolled saat ini:", course.EnrolledCount)

    // Pengujian ini mengharuskan lima kursi sudah terisi.
    if course.Quota != 5 || course.EnrolledCount != 0 {
        fmt.Println("HASIL: TIDAK BISA DIUJI — IF399 harus memiliki kuota 5 dan sudah terisi 5.")
        fmt.Println("Siapkan kondisi awal database terlebih dahulu.")
        return
    }

    // Pastikan keenam akun tidak sedang terdaftar di IF399.
    for _, user := range users {
        var exists bool
        err := pool.QueryRow(ctx, `
            SELECT EXISTS (
                SELECT 1 FROM enrollments
                WHERE user_id = $1 AND course_id = $2
            )
        `, user.ID, course.ID).Scan(&exists)

        if err != nil {
            fmt.Println("Gagal memeriksa enrollment:", err)
            os.Exit(1)
        }

        if exists {
            fmt.Printf(
                "HASIL: TIDAK BISA DIUJI — %s sudah terdaftar di IF399.\n",
                user.Username,
            )
            fmt.Println("Siapkan kondisi awal database terlebih dahulu.")
            return
        }
    }

    // Lima mahasiswa pertama mengisi lima kursi.
    setupResult := fire(5, func(i int) string {
        return sendEnroll(client, baseURL, users[i], course)
    })

    fmt.Println("\n=== MENYIAPKAN LIMA KURSI ===")
    printResults(setupResult)

    // Mahasiswa keenam mencoba mendaftar.
    result := sendEnroll(client, baseURL, users[5], course)

    fmt.Println("\n=== HASIL MAHASISWA KE-6 ===")
    fmt.Println(users[5].Username, ":", result)

    updatedCourse, err := loadCourseByCode(ctx, pool, "IF399")
    if err != nil {
        fmt.Println("Gagal membaca ulang IF399:", err)
        os.Exit(1)
    }

    var enrollmentRows int
    err = pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM enrollments
        WHERE course_id = $1
    `, course.ID).Scan(&enrollmentRows)
    if err != nil {
        fmt.Println("Gagal menghitung enrollment:", err)
        os.Exit(1)
    }

    fmt.Println("\n=== VERIFIKASI DATABASE ===")
    fmt.Println("enrolled_count:", updatedCourse.EnrolledCount)
    fmt.Println("Jumlah enrollment:", enrollmentRows)

    if setupResult["HTTP 201"] == 5 &&
        result == "HTTP 409 COURSE_FULL" &&
        updatedCourse.EnrolledCount == 5 &&
        enrollmentRows == 5 {
        fmt.Println("\nHASIL: PASS — mahasiswa ke-6 ditolak dengan HTTP 409 COURSE_FULL.")
    } else {
        fmt.Println("\nHASIL: BELUM PASS — periksa status HTTP dan kondisi database.")
    }
}

func runSwapFull(
	ctx context.Context,
	pool *pgxpool.Pool,
	users []TestUser,
	baseURL string,
	client *http.Client,
) {
	target, err := loadCourseByCode(ctx, pool, "IF399")
	if err != nil {
		fmt.Println("Gagal membaca IF399:", err)
		os.Exit(1)
	}

	var targetRows int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM enrollments
		WHERE course_id = $1
	`, target.ID).Scan(&targetRows)
	if err != nil {
		fmt.Println("Gagal menghitung enrollment IF399:", err)
		os.Exit(1)
	}

	if target.Quota != 5 ||
		target.EnrolledCount != 5 ||
		targetRows != 5 {
		fmt.Println("HASIL: TIDAK BISA DIUJI — IF399 harus penuh dan konsisten.")
		fmt.Printf(
			"quota=%d, enrolled_count=%d, baris=%d\n",
			target.Quota,
			target.EnrolledCount,
			targetRows,
		)
		return
	}

	// Cari mahasiswa yang terdaftar di kelas asal,
	// tetapi belum terdaftar di IF399.
	var userID int
	var username string
	var fromCourse Course

	err = pool.QueryRow(ctx, `
		SELECT u.id, u.username,
		       c.id, c.code, c.credits, c.quota, c.enrolled_count
		FROM users u
		JOIN enrollments e ON e.user_id = u.id
		JOIN courses c ON c.id = e.course_id
		WHERE u.username LIKE 'mhs%'
		  AND c.id <> $1
		  AND NOT EXISTS (
		      SELECT 1
		      FROM enrollments e2
		      WHERE e2.user_id = u.id
		        AND e2.course_id = $1
		  )
		ORDER BY u.username, c.code
		LIMIT 1
	`, target.ID).Scan(
		&userID,
		&username,
		&fromCourse.ID,
		&fromCourse.Code,
		&fromCourse.Credits,
		&fromCourse.Quota,
		&fromCourse.EnrolledCount,
	)
	if err != nil {
		fmt.Println("Gagal mencari mahasiswa dengan kelas asal:", err)
		os.Exit(1)
	}

	var user TestUser
	found := false
	for _, candidate := range users {
		if candidate.ID == userID {
			user = candidate
			found = true
			break
		}
	}
	if !found {
		fmt.Println("Token mahasiswa tidak ditemukan:", username)
		os.Exit(1)
	}

	fmt.Println("=== TEST: SWAP KE KELAS PENUH ===")
	fmt.Println("Mahasiswa:", user.Username)
	fmt.Println("Kelas asal:", fromCourse.Code)
	fmt.Println("Kelas tujuan:", target.Code)
	fmt.Println("Kuota tujuan:", target.Quota)
	fmt.Println("Enrolled tujuan:", target.EnrolledCount)

	// Simpan kondisi kelas asal sebelum request.
	var sourceRowsBefore int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM enrollments
		WHERE course_id = $1
	`, fromCourse.ID).Scan(&sourceRowsBefore)
	if err != nil {
		fmt.Println("Gagal menghitung kelas asal:", err)
		os.Exit(1)
	}

	result := sendSwap(
		client,
		baseURL,
		user,
		fromCourse.ID,
		target.ID,
	)
	fmt.Println("\nHasil request swap:", result)

	// Verifikasi mahasiswa tetap di kelas asal.
	var stillInSource bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM enrollments
			WHERE user_id = $1 AND course_id = $2
		)
	`, user.ID, fromCourse.ID).Scan(&stillInSource)
	if err != nil {
		fmt.Println("Gagal memeriksa kelas asal:", err)
		os.Exit(1)
	}

	var enrolledInTarget bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM enrollments
			WHERE user_id = $1 AND course_id = $2
		)
	`, user.ID, target.ID).Scan(&enrolledInTarget)
	if err != nil {
		fmt.Println("Gagal memeriksa kelas tujuan:", err)
		os.Exit(1)
	}

	updatedSource, err := loadCourseByCode(ctx, pool, fromCourse.Code)
	if err != nil {
		fmt.Println("Gagal membaca ulang kelas asal:", err)
		os.Exit(1)
	}

	updatedTarget, err := loadCourseByCode(ctx, pool, "IF399")
	if err != nil {
		fmt.Println("Gagal membaca ulang IF399:", err)
		os.Exit(1)
	}

	var sourceRowsAfter, targetRowsAfter int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM enrollments WHERE course_id = $1
	`, fromCourse.ID).Scan(&sourceRowsAfter)
	if err != nil {
		fmt.Println("Gagal menghitung ulang kelas asal:", err)
		os.Exit(1)
	}

	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM enrollments WHERE course_id = $1
	`, target.ID).Scan(&targetRowsAfter)
	if err != nil {
		fmt.Println("Gagal menghitung ulang kelas tujuan:", err)
		os.Exit(1)
	}

	fmt.Println("\n=== VERIFIKASI DATABASE ===")
	fmt.Printf(
		"%s: enrolled_count=%d, baris=%d sebelum=%d\n",
		fromCourse.Code,
		updatedSource.EnrolledCount,
		sourceRowsAfter,
		sourceRowsBefore,
	)
	fmt.Printf(
		"IF399: enrolled_count=%d, baris=%d\n",
		updatedTarget.EnrolledCount,
		targetRowsAfter,
	)
	fmt.Println("Mahasiswa masih di kelas asal:", stillInSource)
	fmt.Println("Mahasiswa masuk kelas tujuan:", enrolledInTarget)

	if strings.Contains(result, "HTTP 409:") &&
		strings.Contains(result, "COURSE_FULL") &&
		stillInSource &&
		!enrolledInTarget &&
		sourceRowsAfter == sourceRowsBefore &&
		updatedSource.EnrolledCount == sourceRowsAfter &&
		updatedTarget.EnrolledCount == targetRowsAfter &&
		updatedTarget.EnrolledCount == 5 &&
		targetRowsAfter == 5 {
		fmt.Println("\nHASIL: PASS — swap ditolak, mahasiswa tetap di kelas asal.")
	} else {
		fmt.Println("\nHASIL: BELUM PASS — periksa respons HTTP dan kondisi database.")
	}
}

func runDuplicate(
    ctx context.Context,
    pool *pgxpool.Pool,
    users []TestUser,
    baseURL string,
    client *http.Client,
) {
    user := users[0]

    // IF101 digunakan karena kuotanya lebih besar daripada IF399.
    course, err := loadCourseByCode(ctx, pool, "IF101")
    if err != nil {
        fmt.Println("Gagal membaca kelas IF101:", err)
        os.Exit(1)
    }

    fmt.Println("=== TEST: DUPLICATE ENROLLMENT ===")
    fmt.Println("Mahasiswa:", user.Username)
    fmt.Println("Kelas:", course.Code)
    fmt.Println("Kuota:", course.Quota)
    fmt.Println("Enrolled saat ini:", course.EnrolledCount)

    // Pastikan mahasiswa belum terdaftar di kelas.
    var alreadyEnrolled bool
    err = pool.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM enrollments
            WHERE user_id = $1 AND course_id = $2
        )
    `, user.ID, course.ID).Scan(&alreadyEnrolled)

    if err != nil {
        fmt.Println("Gagal memeriksa enrollment:", err)
        os.Exit(1)
    }

    if alreadyEnrolled {
        fmt.Println("HASIL: TIDAK BISA DIUJI — mahasiswa sudah terdaftar di IF101.")
        fmt.Println("Hapus enrollment mahasiswa tersebut terlebih dahulu.")
        return
    }

    if course.EnrolledCount >= course.Quota {
        fmt.Println("HASIL: TIDAK BISA DIUJI — kelas IF101 sudah penuh.")
        return
    }

    // Request pertama: mahasiswa mendaftar.
    firstResult := sendEnroll(client, baseURL, user, course)
    fmt.Println("\nPendaftaran pertama:", firstResult)

    // Request kedua: mahasiswa mencoba mendaftar lagi.
    secondResult := sendEnroll(client, baseURL, user, course)
    fmt.Println("Pendaftaran kedua:", secondResult)

    // Periksa kondisi database setelah kedua request.
    updatedCourse, err := loadCourseByCode(ctx, pool, "IF101")
    if err != nil {
        fmt.Println("Gagal membaca ulang IF101:", err)
        os.Exit(1)
    }

    var enrollmentRows int
    err = pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM enrollments
        WHERE course_id = $1
    `, course.ID).Scan(&enrollmentRows)

    if err != nil {
        fmt.Println("Gagal menghitung enrollment:", err)
        os.Exit(1)
    }

    var userEnrollmentRows int
    err = pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM enrollments
        WHERE user_id = $1 AND course_id = $2
    `, user.ID, course.ID).Scan(&userEnrollmentRows)

    if err != nil {
        fmt.Println("Gagal menghitung enrollment mahasiswa:", err)
        os.Exit(1)
    }

    fmt.Println("\n=== VERIFIKASI DATABASE ===")
    fmt.Println("enrolled_count:", updatedCourse.EnrolledCount)
    fmt.Println("Jumlah baris enrollment kelas:", enrollmentRows)
    fmt.Println("Enrollment mahasiswa di IF101:", userEnrollmentRows)

    if firstResult == "HTTP 201" &&
        secondResult == "HTTP 409 DUPLICATE" &&
        userEnrollmentRows == 1 &&
        updatedCourse.EnrolledCount == enrollmentRows {
        fmt.Println("\nHASIL: PASS — pendaftaran duplikat ditolak dan data tetap konsisten.")
    } else {
        fmt.Println("\nHASIL: BELUM PASS — periksa respons HTTP dan kondisi database.")
    }
}

func runDrop(
	ctx context.Context,
	pool *pgxpool.Pool,
	users []TestUser,
	baseURL string,
	client *http.Client,
) {
	var userID int
	var username string
	var course Course

	err := pool.QueryRow(ctx, `
		SELECT u.id, u.username,
		       c.id, c.code, c.credits, c.quota, c.enrolled_count
		FROM users u
		CROSS JOIN courses c
		WHERE u.username LIKE 'mhs%'
		  AND c.code <> 'IF399'
		  AND NOT EXISTS (
		      SELECT 1
		      FROM enrollments e
		      WHERE e.user_id = u.id
		        AND e.course_id = c.id
		  )
		ORDER BY u.username, c.code
		LIMIT 1
	`).Scan(
		&userID,
		&username,
		&course.ID,
		&course.Code,
		&course.Credits,
		&course.Quota,
		&course.EnrolledCount,
	)

	if err != nil {
		fmt.Println("Gagal mencari pasangan mahasiswa dan kelas:", err)
		fmt.Println("Pastikan ada mahasiswa mhs% yang belum terdaftar di kelas reguler.")
		os.Exit(1)
	}

	var user TestUser
	found := false
	for _, candidate := range users {
		if candidate.ID == userID {
			user = candidate
			found = true
			break
		}
	}

	if !found {
		fmt.Println("Mahasiswa ditemukan di database, tetapi tokennya tidak tersedia.")
		os.Exit(1)
	}

	var rowsBefore int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM enrollments
		WHERE course_id = $1
	`, course.ID).Scan(&rowsBefore)
	if err != nil {
		fmt.Println("Gagal menghitung enrollment awal:", err)
		os.Exit(1)
	}

	fmt.Println("=== TEST: DROP KELAS YANG TIDAK DIIKUTI ===")
	fmt.Println("Mahasiswa:", username)
	fmt.Println("Kelas:", course.Code)
	fmt.Println("enrolled_count awal:", course.EnrolledCount)
	fmt.Println("Jumlah enrollment awal:", rowsBefore)

	result := sendDrop(client, baseURL, user, course)
	fmt.Println("\nHasil request drop:", result)

	updatedCourse, err := loadCourseByCode(ctx, pool, course.Code)
	if err != nil {
		fmt.Println("Gagal membaca ulang kelas:", err)
		os.Exit(1)
	}

	var rowsAfter int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM enrollments
		WHERE course_id = $1
	`, course.ID).Scan(&rowsAfter)
	if err != nil {
		fmt.Println("Gagal menghitung enrollment akhir:", err)
		os.Exit(1)
	}

	var userEnrollmentRows int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM enrollments
		WHERE user_id = $1 AND course_id = $2
	`, user.ID, course.ID).Scan(&userEnrollmentRows)
	if err != nil {
		fmt.Println("Gagal memeriksa enrollment mahasiswa:", err)
		os.Exit(1)
	}

	fmt.Println("\n=== VERIFIKASI DATABASE ===")
	fmt.Println("enrolled_count akhir:", updatedCourse.EnrolledCount)
	fmt.Println("Jumlah enrollment akhir:", rowsAfter)
	fmt.Println("Enrollment mahasiswa di kelas:", userEnrollmentRows)

	if result == "HTTP 409 NOT_ENROLLED" &&
		updatedCourse.EnrolledCount == course.EnrolledCount &&
		rowsAfter == rowsBefore &&
		userEnrollmentRows == 0 &&
		updatedCourse.EnrolledCount == rowsAfter {
		fmt.Println("\nHASIL: PASS — drop ditolak dan data tidak berubah.")
	} else {
		fmt.Println("\nHASIL: BELUM PASS — periksa respons HTTP dan kondisi database.")
	}
}

func loadAdmin(
	ctx context.Context,
	pool *pgxpool.Pool,
	jwtManager *helper.JWTManager,
) (TestUser, error) {
	var user TestUser

	err := pool.QueryRow(ctx, `
		SELECT id, username, role
		FROM users
		WHERE role = 'admin'
		ORDER BY id
		LIMIT 1
	`).Scan(&user.ID, &user.Username, &user.Role)
	if err != nil {
		return user, err
	}

	token, err := jwtManager.GenerateAccess(model.User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
	if err != nil {
		return user, err
	}

	user.Token = token
	return user, nil
}

func runQuotaForbidden(
	ctx context.Context,
	pool *pgxpool.Pool,
	student TestUser,
	baseURL string,
	client *http.Client,
) {
	course, err := loadCourseByCode(ctx, pool, "IF101")
	if err != nil {
		fmt.Println("Gagal membaca IF101:", err)
		return
	}

	originalQuota := course.Quota
	originalVersion := course.Version

	fmt.Println("=== TEST: MAHASISWA DITOLAK MENGUBAH KUOTA ===")
	fmt.Println("Mahasiswa:", student.Username)
	fmt.Println("Kelas:", course.Code)
	fmt.Println("Kuota awal:", originalQuota)
	fmt.Println("Version awal:", originalVersion)

	result := sendUpdateQuota(
		client,
		baseURL,
		student,
		course,
		originalQuota+1,
		originalVersion,
	)
	fmt.Println("\nUpdate kuota oleh mahasiswa:", result)

	finalCourse, err := loadCourseByCode(ctx, pool, "IF101")
	if err != nil {
		fmt.Println("Gagal membaca IF101 setelah pengujian:", err)
		return
	}

	fmt.Println("\n=== VERIFIKASI AKHIR ===")
	fmt.Println("Kuota akhir:", finalCourse.Quota)
	fmt.Println("Version akhir:", finalCourse.Version)

	if strings.HasPrefix(result, "HTTP 403") &&
		finalCourse.Quota == originalQuota &&
		finalCourse.Version == originalVersion {
		fmt.Println("\nHASIL: PASS — mahasiswa ditolak dan data kelas tidak berubah.")
	} else {
		fmt.Println("\nHASIL: BELUM PASS — periksa respons API dan kondisi database.")
	}
}

func runQuotaBelowEnrolled(
    ctx context.Context,
    pool *pgxpool.Pool,
    jwtManager *helper.JWTManager,
    baseURL string,
    client *http.Client,
) {
    admin, err := loadAdmin(ctx, pool, jwtManager)
    if err != nil {
        fmt.Println("Gagal membuat token admin:", err)
        return
    }

    course, err := loadCourseByCode(ctx, pool, "IF101")
    if err != nil {
        fmt.Println("Gagal membaca IF101:", err)
        return
    }

    if course.EnrolledCount <= 0 {
        fmt.Println("HASIL: SKIP — IF101 belum memiliki peserta terdaftar.")
        return
    }

    originalQuota := course.Quota
    originalVersion := course.Version
    invalidQuota := course.EnrolledCount - 1

    fmt.Println("=== TEST: KUOTA DI BAWAH JUMLAH PESERTA ===")
    fmt.Println("Admin:", admin.Username)
    fmt.Println("Kelas:", course.Code)
    fmt.Println("Peserta terdaftar:", course.EnrolledCount)
    fmt.Println("Kuota awal:", originalQuota)
    fmt.Println("Version awal:", originalVersion)
    fmt.Println("Kuota yang diuji:", invalidQuota)

    result := sendUpdateQuota(
        client,
        baseURL,
        admin,
        course,
        invalidQuota,
        originalVersion,
    )
    fmt.Println("\nUpdate kuota:", result)

    finalCourse, err := loadCourseByCode(ctx, pool, "IF101")
    if err != nil {
        fmt.Println("Gagal membaca IF101 setelah pengujian:", err)
        return
    }

    fmt.Println("\n=== VERIFIKASI AKHIR ===")
    fmt.Println("Kuota akhir:", finalCourse.Quota)
    fmt.Println("Version akhir:", finalCourse.Version)

    if result == "HTTP 409 QUOTA_BELOW_ENROLLED" &&
        finalCourse.Quota == originalQuota &&
        finalCourse.Version == originalVersion {
        fmt.Println("\nHASIL: PASS — kuota di bawah jumlah peserta ditolak dan data tidak berubah.")
    } else {
        fmt.Println("\nHASIL: BELUM PASS — periksa respons API dan kondisi database.")
    }
}

func sendUpdateQuota(
	client *http.Client,
	baseURL string,
	admin TestUser,
	course Course,
	quota int,
	version int,
) string {
	url := fmt.Sprintf("%s/api/v1/courses/%d", baseURL, course.ID)
	body := fmt.Sprintf(
		`{"quota":%d,"version":%d}`,
		quota,
		version,
	)

	req, err := http.NewRequest(
		http.MethodPatch,
		url,
		strings.NewReader(body),
	)
	if err != nil {
		return "ERROR membuat request: " + err.Error()
	}

	req.Header.Set("Authorization", "Bearer "+admin.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "ERROR mengirim request: " + err.Error()
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	text := string(responseBody)

	if resp.StatusCode == http.StatusOK {
		return "HTTP 200"
	}

	if resp.StatusCode == http.StatusConflict &&
		strings.Contains(text, "VERSION_CONFLICT") {
		return "HTTP 409 VERSION_CONFLICT"
	}

	if resp.StatusCode == http.StatusConflict &&
		strings.Contains(text, "QUOTA_BELOW_ENROLLED") {
		return "HTTP 409 QUOTA_BELOW_ENROLLED"
	}

	return fmt.Sprintf(
		"HTTP %d: %s",
		resp.StatusCode,
		strings.TrimSpace(text),
	)
}

func runQuota(
	ctx context.Context,
	pool *pgxpool.Pool,
	jwtManager *helper.JWTManager,
	baseURL string,
	client *http.Client,
) {
	admin, err := loadAdmin(ctx, pool, jwtManager)
	if err != nil {
		fmt.Println("Gagal membuat token admin:", err)
		return
	}

	course, err := loadCourseByCode(ctx, pool, "IF101")
	if err != nil {
		fmt.Println("Gagal membaca IF101:", err)
		return
	}

	if course.Version < 1 || course.Quota >= 500 {
		fmt.Println("HASIL: TIDAK BISA DIUJI — versi harus positif dan kuota IF101 harus kurang dari 500.")
		return
	}

	originalQuota := course.Quota
	oldVersion := course.Version
	newQuota := originalQuota + 1

	fmt.Println("=== TEST: OPTIMISTIC LOCKING KUOTA ===")
	fmt.Println("Admin:", admin.Username)
	fmt.Println("Kelas:", course.Code)
	fmt.Println("Kuota awal:", originalQuota)
	fmt.Println("Version awal:", oldVersion)

	// Request pertama: ubah kuota menggunakan version terkini.
	firstResult := sendUpdateQuota(
		client, baseURL, admin, course, newQuota, oldVersion,
	)
	fmt.Println("\nUpdate dengan version terkini:", firstResult)

	updated, err := loadCourseByCode(ctx, pool, "IF101")
	if err != nil {
		fmt.Println("Gagal membaca IF101 setelah update:", err)
		return
	}

	if firstResult != "HTTP 200" ||
		updated.Quota != newQuota ||
		updated.Version != oldVersion+1 {
		fmt.Println("HASIL: BELUM PASS — update pertama atau kenaikan version tidak sesuai.")
		fmt.Printf(
			"Kuota sekarang=%d, version sekarang=%d\n",
			updated.Quota, updated.Version,
		)
		return
	}

	// Request kedua: kirim version lama. Seharusnya ditolak.
	staleResult := sendUpdateQuota(
		client, baseURL, admin, updated, newQuota+1, oldVersion,
	)
	fmt.Println("Update menggunakan version lama:", staleResult)

	afterStale, err := loadCourseByCode(ctx, pool, "IF101")
	if err != nil {
		fmt.Println("Gagal membaca IF101 setelah tes version lama:", err)
		return
	}

	staleUnchanged := afterStale.Quota == newQuota &&
		afterStale.Version == oldVersion+1

	// Pulihkan kuota awal menggunakan version terbaru.
	restoreResult := sendUpdateQuota(
		client, baseURL, admin, afterStale,
		originalQuota, afterStale.Version,
	)
	fmt.Println("Pemulihan kuota awal:", restoreResult)

	finalCourse, err := loadCourseByCode(ctx, pool, "IF101")
	if err != nil {
		fmt.Println("Gagal membaca IF101 setelah pemulihan:", err)
		return
	}

	fmt.Println("\n=== VERIFIKASI AKHIR ===")
	fmt.Println("Kuota awal:", originalQuota)
	fmt.Println("Kuota akhir:", finalCourse.Quota)
	fmt.Println("Version awal:", oldVersion)
	fmt.Println("Version akhir:", finalCourse.Version)

	if firstResult == "HTTP 200" &&
		staleResult == "HTTP 409 VERSION_CONFLICT" &&
		staleUnchanged &&
		restoreResult == "HTTP 200" &&
		finalCourse.Quota == originalQuota &&
		finalCourse.Version == oldVersion+2 {
		fmt.Println("\nHASIL: PASS — version lama ditolak dan kuota awal berhasil dipulihkan.")
	} else {
		fmt.Println("\nHASIL: BELUM PASS — periksa respons API dan kondisi database.")
	}
}

func runSupervisionQuota(
    ctx context.Context,
    pool *pgxpool.Pool,
    jwtManager *helper.JWTManager,
    baseURL string,
    client *http.Client,
) {
    const testNIMPrefix = "D3TEST"
    const testSupervisorName = "D3 Load Test"
    const requestCount = 30

    // Ambil token admin.
    admin, err := loadAdmin(ctx, pool, jwtManager)
    if err != nil {
        fmt.Println("Gagal mengambil akun admin:", err)
        return
    }

    // Ambil ID dosen testing.
    var supervisorID int
    err = pool.QueryRow(ctx, `
        SELECT id
        FROM supervisors
        WHERE name = $1
    `, testSupervisorName).Scan(&supervisorID)
    if err != nil {
        fmt.Println("Dosen testing tidak ditemukan:", err)
        return
    }

    // Ambil 30 ID mahasiswa testing.
    rows, err := pool.Query(ctx, `
        SELECT id
        FROM students
        WHERE nim LIKE $1
        ORDER BY nim
        LIMIT $2
    `, testNIMPrefix+"%", requestCount)
    if err != nil {
        fmt.Println("Gagal mengambil mahasiswa testing:", err)
        return
    }

    studentIDs := make([]int, 0, requestCount)

    for rows.Next() {
        var studentID int
        if err := rows.Scan(&studentID); err != nil {
            rows.Close()
            fmt.Println("Gagal membaca ID mahasiswa:", err)
            return
        }
        studentIDs = append(studentIDs, studentID)
    }

    rowsErr := rows.Err()
    rows.Close()

    if rowsErr != nil {
        fmt.Println("Gagal membaca data mahasiswa:", rowsErr)
        return
    }

    if len(studentIDs) != requestCount {
        fmt.Printf(
            "Dibutuhkan %d mahasiswa testing, ditemukan %d.\n",
            requestCount,
            len(studentIDs),
        )
        return
    }

    // Bersihkan penugasan sebelumnya untuk mahasiswa testing.
    // Bagian ini hanya menyentuh mahasiswa dengan NIM D3TEST.
    tx, err := pool.Begin(ctx)
    if err != nil {
        fmt.Println("Gagal memulai transaksi reset:", err)
        return
    }

    _, err = tx.Exec(ctx, `
        DELETE FROM supervisions
        WHERE student_id IN (
            SELECT id
            FROM students
            WHERE nim LIKE $1
        )
    `, testNIMPrefix+"%")
    if err != nil {
        _ = tx.Rollback(ctx)
        fmt.Println("Gagal membersihkan penugasan testing:", err)
        return
    }

    // Hitung ulang jumlah bimbingan dosen testing berdasarkan data aktual.
    _, err = tx.Exec(ctx, `
        UPDATE supervisors
        SET supervised_count = (
            SELECT COUNT(*)
            FROM supervisions
            WHERE supervisor_id = $1
        )
        WHERE id = $1
    `, supervisorID)
    if err != nil {
        _ = tx.Rollback(ctx)
        fmt.Println("Gagal mereset jumlah bimbingan:", err)
        return
    }

    if err := tx.Commit(ctx); err != nil {
        fmt.Println("Gagal menyimpan reset testing:", err)
        return
    }

    // Jalankan 30 request secara bersamaan.
    endpoint := fmt.Sprintf(
        "%s/api/v1/supervisors/%d/assign",
        baseURL,
        supervisorID,
    )

    fmt.Printf("Dosen testing ID : %d\n", supervisorID)
    fmt.Printf("Jumlah mahasiswa : %d\n", len(studentIDs))
    fmt.Println("Mengirim 30 request bersamaan...")

    tally := fire(requestCount, func(i int) string {
        body := fmt.Sprintf(
            `{"student_id":%d}`,
            studentIDs[i],
        )

        req, err := http.NewRequestWithContext(
            ctx,
            http.MethodPost,
            endpoint,
            strings.NewReader(body),
        )
        if err != nil {
            return "REQUEST_ERROR"
        }

        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer "+admin.Token)

        resp, err := client.Do(req)
        if err != nil {
            return "REQUEST_ERROR"
        }
        defer resp.Body.Close()

        // Status HTTP dipakai untuk menghitung hasil.
        return fmt.Sprintf("%d", resp.StatusCode)
    })

    fmt.Println("\n=== HASIL LOAD TEST ===")
    for _, status := range []string{"201", "400", "401", "403", "404", "409", "422", "500", "REQUEST_ERROR"} {
        if count := tally[status]; count > 0 {
            fmt.Printf("HTTP %-14s %d\n", status, count)
        }
    }

    // Verifikasi kondisi akhir langsung dari database.
    var quota, supervisedCount, actualCount int

    err = pool.QueryRow(ctx, `
        SELECT
            s.quota,
            s.supervised_count,
            (
                SELECT COUNT(*)
                FROM supervisions sp
                WHERE sp.supervisor_id = s.id
            )
        FROM supervisors s
        WHERE s.id = $1
    `, supervisorID).Scan(
        &quota,
        &supervisedCount,
        &actualCount,
    )
    if err != nil {
        fmt.Println("Gagal memverifikasi database:", err)
        return
    }

    fmt.Println("\n=== VERIFIKASI DATABASE ===")
    fmt.Printf("Kuota dosen             : %d\n", quota)
    fmt.Printf("supervised_count        : %d\n", supervisedCount)
    fmt.Printf("Jumlah relasi aktual    : %d\n", actualCount)

    fmt.Println("\n=== KESIMPULAN ===")
    if tally["201"] == 3 &&
        supervisedCount == 3 &&
        actualCount == 3 &&
        supervisedCount == actualCount &&
        quota == 3 {
        fmt.Println("LULUS: tepat 3 mahasiswa berhasil ditetapkan.")
    } else {
        fmt.Println("BELUM LULUS: hasil tidak sesuai target kuota 3.")
    }
}