package model

import "time"

// User adalah entitas untuk autentikasi, terpisah dari Student.
// Field Password diberi tag json:"-" agar TIDAK PERNAH ikut keluar
// sebagai JSON pada respons apa pun, termasuk saat register.
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}