//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"latihan-fiber/modul4/app/model"
)

// call mengirim request dengan body bebas atas nama student dan mengembalikan
// status code beserta field "code" pada response (kosong bila tanpa body).
func (a api) call(t *testing.T, student model.User, method, path, body string) (int, string) {
	t.Helper()
	token, err := a.jwt.GenerateAccess(student)
	if err != nil {
		t.Fatalf("membuat token: %v", err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var parsed struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	return resp.StatusCode, parsed.Code
}

func enrollmentRows(t *testing.T, userID, courseID int) int {
	t.Helper()
	return countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, userID, courseID)
}

func enrolledCount(t *testing.T, courseID int) int {
	t.Helper()
	return countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, courseID)
}

func outboxEvents(t *testing.T, userID int, eventType string) int {
	t.Helper()
	return countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		userID, eventType)
}

// Pindah ke kelas yang penuh harus gagal TANPA mengubah apa pun:
// mahasiswa tetap di kelas asal, jumlah peserta kedua kelas tetap, dan
// tidak ada event enrollment.swapped di outbox.
func TestSwapEndpoint_TargetFull_ChangesNothing(t *testing.T) {
	a := newAPI(t)
	from := seedCourse(t, 3, 40)
	to := seedCourse(t, 3, 1) // kuota 1
	student := seedStudent(t)
	other := seedStudent(t)

	if status, _ := a.enroll(t, student, from.ID); status != http.StatusCreated {
		t.Fatalf("siapkan kelas asal: status = %d, seharusnya 201", status)
	}
	if status, _ := a.enroll(t, other, to.ID); status != http.StatusCreated {
		t.Fatalf("isi kelas tujuan sampai penuh: status = %d, seharusnya 201", status)
	}
	fromBefore, toBefore := enrolledCount(t, from.ID), enrolledCount(t, to.ID)

	// ASUMSI 1: nama field body. Sesuaikan dengan model.SwapRequest.
	body := fmt.Sprintf(`{"from_course_id": %d, "to_course_id": %d}`, from.ID, to.ID)
	status, code := a.call(t, student, http.MethodPost, "/api/v1/enrollments/swap", body)
	if status != http.StatusConflict || code != "COURSE_FULL" {
		t.Fatalf("dapat %d %s, seharusnya 409 COURSE_FULL", status, code)
	}

	if n := enrollmentRows(t, student.ID, from.ID); n != 1 {
		t.Errorf("baris di kelas asal = %d, seharusnya 1 (mahasiswa harus tetap di kelas asal)", n)
	}
	if n := enrollmentRows(t, student.ID, to.ID); n != 0 {
		t.Errorf("baris di kelas tujuan = %d, seharusnya 0", n)
	}
	if got := enrolledCount(t, from.ID); got != fromBefore {
		t.Errorf("enrolled_count kelas asal = %d, seharusnya %d", got, fromBefore)
	}
	if got := enrolledCount(t, to.ID); got != toBefore {
		t.Errorf("enrolled_count kelas tujuan = %d, seharusnya %d", got, toBefore)
	}
	if n := outboxEvents(t, student.ID, model.EventEnrollmentSwapped); n != 0 {
		t.Errorf("event enrollment.swapped = %d, seharusnya 0", n)
	}
}

// Keluar dari kelas: 204, baris enrollment hilang, enrolled_count berkurang
// satu, dan outbox berisi tepat satu event enrollment.dropped.
func TestDropEndpoint_RemovesEnrollmentAndWritesOutbox(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("siapkan pendaftaran: status = %d, seharusnya 201", status)
	}
	if got := enrolledCount(t, course.ID); got != 1 {
		t.Fatalf("enrolled_count sebelum keluar = %d, seharusnya 1", got)
	}

	path := fmt.Sprintf("/api/v1/courses/%d/enroll", course.ID)
	if status, code := a.call(t, student, http.MethodDelete, path, ""); status != http.StatusNoContent {
		t.Fatalf("keluar: dapat %d %s, seharusnya 204", status, code)
	}

	if n := enrollmentRows(t, student.ID, course.ID); n != 0 {
		t.Errorf("baris enrollment = %d, seharusnya 0", n)
	}
	if got := enrolledCount(t, course.ID); got != 0 {
		t.Errorf("enrolled_count = %d, seharusnya 0", got)
	}
	if n := outboxEvents(t, student.ID, model.EventEnrollmentDropped); n != 1 {
		t.Errorf("event enrollment.dropped = %d, seharusnya 1", n)
	}
}