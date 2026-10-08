package model

const (
	RoleAdmin     = "admin"
	RoleMahasiswa = "mahasiswa"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	User        User   `json:"user"`
}

type MeResponse struct {
	ID      int      `json:"id"`
	Email   string   `json:"email"`
	Role    string   `json:"role"`
	Student *Student `json:"student,omitempty"`
}