package user

type User struct {
	ID       string `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
