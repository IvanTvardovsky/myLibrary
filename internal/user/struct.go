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

type WishlistBook struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	CoverImage    string `json:"cover_image"`
	DateWhenAdded string `json:"date_added"`
}

type FinishedBook struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	CoverImage    string `json:"cover_image"`
	DateWhenAdded string `json:"date_added"`
	Rating        int    `json:"rating"`
	Comment       string `json:"comment"`
}
