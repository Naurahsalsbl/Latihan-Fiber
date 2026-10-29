// Alat uji beban C.3: membuktikan batas 20 proyek bertahan saat diserang serentak.
//
// Langkah:
//  1. PUT portofolio berisi -prefill proyek (bawaan 18). PUT mengganti seluruh
//     isi, sehingga alat ini dapat diulang tanpa skrip reset.
//  2. Kirim -n request POST proyek SERENTAK (bawaan 10).
//  3. Baca portofolio lewat GET dan hitung proyeknya.
//
// Yang benar: tepat (max - prefill) = 2 berhasil, sisanya 409 PROJECT_LIMIT,
// dan jumlah proyek akhir tepat 20.
//
// Alat ini memakai token akun admin pertama di tabel users, jadi admin
// harus memegang permission portfolio:manage (migration 008) dan server
// harus dijalankan ulang setelah migration.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/config"
	"latihan-fiber/modul4/database"
	"latihan-fiber/modul4/helper"
)

const baseURL = "http://localhost:3000/api/v1"

type result struct {
	Status int
	Body   string
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

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

	close(start) // buka gerbang
	wg.Wait()    // tunggu SEMUA goroutine selesai
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally
}

func request(method, path, token string, body any) result {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return result{Status: 0, Body: err.Error()}
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		return result{Status: 0, Body: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return result{Status: 0, Body: err.Error()}
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	return result{Status: resp.StatusCode, Body: string(data)}
}

func label(r result) string {
	if r.Status == 0 {
		return "ERROR koneksi/timeout"
	}
	var payload struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal([]byte(r.Body), &payload)
	if payload.Code != "" {
		return fmt.Sprintf("%d %s", r.Status, payload.Code)
	}
	if r.Status >= 200 && r.Status < 300 {
		return fmt.Sprintf("%d OK", r.Status)
	}
	return fmt.Sprintf("%d (tanpa kode)", r.Status)
}

func summarize(tally map[string]int) {
	fmt.Println("hasil request:")
	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-28s %d\n", k, tally[k])
	}
}

func successes(tally map[string]int) int {
	total := 0
	for k, v := range tally {
		if strings.HasPrefix(k, "2") {
			total += v
		}
	}
	return total
}

func createAdminToken(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	secret := config.GetEnv("JWT_SECRET", "")
	if len(secret) < 32 {
		return "", fmt.Errorf("JWT_SECRET kosong atau kurang dari 32 karakter")
	}
	issuer := config.GetEnv("JWT_ISSUER", "praktikum-backend")
	accessTTL := time.Duration(
		config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15),
	) * time.Minute
	jwtManager := helper.NewJWTManager(secret, issuer, accessTTL)

	var u model.User
	err := pool.QueryRow(ctx,
		"SELECT id, username, role FROM users WHERE role = 'admin' ORDER BY id LIMIT 1",
	).Scan(&u.ID, &u.Username, &u.Role)
	if err != nil {
		return "", fmt.Errorf("mengambil akun admin: %w", err)
	}
	u.IsActive = true

	token, err := jwtManager.GenerateAccess(u)
	if err != nil {
		return "", fmt.Errorf("membuat token admin: %w", err)
	}
	return token, nil
}

func firstStudentID(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var id int
	err := pool.QueryRow(ctx, "SELECT id FROM students ORDER BY id LIMIT 1").Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("mengambil mahasiswa pertama: %w", err)
	}
	return id, nil
}

func projectBody(title string) map[string]any {
	return map[string]any{
		"title":        title,
		"technologies": []string{"go", "mongodb"},
		"url":          "https://example.com/proyek",
		"year":         2025,
	}
}

func run() error {
	studentFlag := flag.Int("student", 0, "id mahasiswa (bawaan: mahasiswa pertama)")
	prefill := flag.Int("prefill", 18, "jumlah proyek yang diisi dulu")
	n := flag.Int("n", 10, "jumlah request serentak")
	maxProjects := flag.Int("max", 20, "batas proyek yang diharapkan")
	flag.Parse()

	config.LoadEnv()

	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		return fmt.Errorf("koneksi database loadtest: %w", err)
	}
	defer pool.Close()

	token, err := createAdminToken(ctx, pool)
	if err != nil {
		return err
	}

	studentID := *studentFlag
	if studentID == 0 {
		studentID, err = firstStudentID(ctx, pool)
		if err != nil {
			return err
		}
	}
	base := fmt.Sprintf("/students/%d/portfolio", studentID)

	// 1. Persiapan: PUT mengganti seluruh portofolio dengan -prefill proyek.
	projects := make([]map[string]any, 0, *prefill)
	for i := 1; i <= *prefill; i++ {
		projects = append(projects, projectBody(fmt.Sprintf("Proyek isi awal %02d", i)))
	}
	put := request(http.MethodPut, base, token, map[string]any{
		"headline":     "Portofolio uji beban",
		"bio":          "Dibuat oleh alat uji beban.",
		"projects":     projects,
		"certificates": []map[string]any{},
	})
	if put.Status != http.StatusOK {
		return fmt.Errorf("persiapan gagal (HTTP %d): %s", put.Status, put.Body)
	}
	fmt.Printf("persiapan: portofolio mahasiswa %d berisi %d proyek\n", studentID, *prefill)

	// 2. Serangan serentak.
	fmt.Printf("%d permintaan tambah proyek serentak (batas %d)...\n", *n, *maxProjects)
	begin := time.Now()
	tally := fire(*n, func(i int) string {
		body := projectBody(fmt.Sprintf("Proyek serentak %02d", i+1))
		return label(request(http.MethodPost, base+"/projects", token, body))
	})
	elapsed := time.Since(begin)

	summarize(tally)
	fmt.Printf("waktu total : %s\n", elapsed.Round(time.Millisecond))

	// 3. Bukti akhir dari sisi database (lewat GET).
	get := request(http.MethodGet, base, token, nil)
	if get.Status != http.StatusOK {
		return fmt.Errorf("GET portofolio gagal (HTTP %d): %s", get.Status, get.Body)
	}
	var payload struct {
		Data struct {
			Projects []json.RawMessage `json:"projects"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(get.Body), &payload); err != nil {
		return fmt.Errorf("membaca response GET: %w", err)
	}
	final := len(payload.Data.Projects)
	fmt.Printf("jumlah proyek akhir: %d\n", final)

	want := *maxProjects - *prefill
	got := successes(tally)
	if got == want && final == *maxProjects {
		fmt.Printf("HASIL: LULUS — tepat %d berhasil, proyek akhir %d\n", want, final)
	} else {
		fmt.Printf("HASIL: GAGAL — berhasil %d (seharusnya %d), proyek akhir %d (seharusnya %d)\n",
			got, want, final, *maxProjects)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Gagal:", err)
		os.Exit(1)
	}
}