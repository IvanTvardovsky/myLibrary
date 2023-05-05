package user

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	"myLibrary/internal/config"
	"myLibrary/internal/handlers"
	"myLibrary/package/logger"
	"net/http"
	"strconv"
	"time"
)

const (
	RegisterUrl      = "/register"
	LoginUrl         = "/login"
	UserUuidUrl      = "/user/:uuid"
	BooksUrl         = "/books"
	FinishedBooksUrl = "/finished"
	WishlistBooksUrl = "/wishlist"
)

type handler struct {
	db  *sql.DB
	cfg *config.Config
}

func NewHandler(db *sql.DB, cfg *config.Config) handlers.Handler {
	return &handler{db, cfg}
}

func (h *handler) Register(router *httprouter.Router) {
	router.POST(RegisterUrl, h.RegisterUser)
	router.POST(LoginUrl, h.LoginUser)
	router.GET(UserUuidUrl, h.GetUserByUUID)
	router.PUT(UserUuidUrl, h.FullyUpdateUser)
	router.PATCH(UserUuidUrl, h.UpdateUser)
	router.DELETE(UserUuidUrl, h.DeleteUser)
	router.POST(UserUuidUrl+BooksUrl+FinishedBooksUrl, h.AddFinishedBook)
	router.POST(UserUuidUrl+BooksUrl+WishlistBooksUrl, h.AddWishlistBook)
	router.GET(UserUuidUrl+BooksUrl+FinishedBooksUrl, h.GetFinishedBooks)
	router.GET(UserUuidUrl+BooksUrl+WishlistBooksUrl, h.GetWishlistBooks)
	router.PUT(UserUuidUrl+BooksUrl+FinishedBooksUrl, h.FromWishlistToFinished)
}

func (h *handler) GetFinishedBooks(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userID := params.ByName("uuid")
	rows, err := h.db.Query("SELECT id, title, author, cover_image_url, date_added, rating, comment FROM books WHERE user_id = $1 AND is_read = $2", userID, true)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
			logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
			return
		}
	}(rows)

	var finishedBooks []FinishedBook
	for rows.Next() {
		var book FinishedBook
		err := rows.Scan(&book.ID, &book.Title, &book.Author, &book.CoverImage, &book.DateWhenAdded, &book.Rating, &book.Comment)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
			logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
			return
		}
		finishedBooks = append(finishedBooks, book)
	}

	w.Header().Set("Content-Type", "application/json")
	if len(finishedBooks) == 0 {
		w.Write([]byte("[]"))
		return
	}
	err = json.NewEncoder(w).Encode(finishedBooks)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error while sending JSON: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Error while sending JSON: ") + err.Error())
		return
	}
}

func (h *handler) GetWishlistBooks(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userID := params.ByName("uuid")
	rows, err := h.db.Query("SELECT id, title, author, cover_image_url, date_added FROM books WHERE user_id = $1 AND is_read = $2", userID, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
			logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
			return
		}
	}(rows)

	var wishlistBooks []WishlistBook
	for rows.Next() {
		var book WishlistBook
		err := rows.Scan(&book.ID, &book.Title, &book.Author, &book.CoverImage, &book.DateWhenAdded)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
			logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
			return
		}
		wishlistBooks = append(wishlistBooks, book)
	}

	w.Header().Set("Content-Type", "application/json")
	if len(wishlistBooks) == 0 {
		w.Write([]byte("[]"))
		return
	}
	err = json.NewEncoder(w).Encode(wishlistBooks)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error while sending JSON: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Error while sending JSON: ") + err.Error())
		return
	}
}

func (h *handler) FromWishlistToFinished(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var additionalInfo FinishedBook // we can add comment + rating here
	err := json.NewDecoder(r.Body).Decode(&additionalInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: ") + err.Error())
		return
	}

	bookID, err := strconv.Atoi(additionalInfo.ID)

	if err != nil {
		http.Error(w, "Bad request: Invalid book ID", http.StatusBadRequest)
		logger.Log.Info("Bad request: Invalid book ID")
		return
	}

	counter, err := BookExists(bookID, h.db)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}
	if counter == 0 {
		http.Error(w, "Bad request: Book not found", http.StatusNotFound)
		logger.Log.Info("Bad request: Book not found")
		return
	}

	_, err = h.db.Exec("UPDATE books SET is_read = $1, comment = $2, rating = $3 WHERE id = $4", true, additionalInfo.Comment, additionalInfo.Rating, additionalInfo.ID)
	if err != nil {
		http.Error(w, "Error updating book: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) LoginUser(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var loginRequest LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad login request: ") + err.Error())
		return
	}

	var userForToken User
	err = h.db.QueryRow("SELECT user_id, username, password FROM users WHERE email = $1 AND EXISTS(SELECT 1 FROM users WHERE email = $1)",
		loginRequest.Email).Scan(&userForToken.ID, &userForToken.Username, &userForToken.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			logger.Log.Info("User not found")
			return
		}
		http.Error(w, "Database error: "+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info("Database error: " + err.Error())
		return
	}

	if loginRequest.Password != userForToken.Password {
		http.Error(w, "Error when trying to login: incorrect password", http.StatusUnauthorized)
		logger.Log.Info(fmt.Sprintf("Error when trying to login: incorrect password"))
		return
	}

	payload := jwt.MapClaims{
		"sub": loginRequest.Email,
		"exp": time.Now().Add(time.Hour * 14 * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	if token == nil {
		http.Error(w, "Error when creating token: "+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Error when creating token: " + err.Error()))
		return
	}

	t, err := token.SignedString([]byte(h.cfg.Key.SecretKey))
	if err != nil {
		http.Error(w, "Error when signing the string: "+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Error when signing the string: " + err.Error()))
		return
	}

	var responseToken LoginResponse
	responseToken.AccessToken = t

	jsonResponse, err := json.Marshal(responseToken)
	if err != nil {
		http.Error(w, "Error when marshaling response: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("Error when marshaling response: " + err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(jsonResponse)
	if err != nil {
		http.Error(w, "User created, but while sending JSON for respond: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("User created, but while sending JSON for respond: ") + err.Error())
		return
	}
}

func (h *handler) RegisterUser(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	var requestUser User
	err := json.NewDecoder(r.Body).Decode(&requestUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: ") + err.Error())
		return
	}

	if !UserSuitableForRestrictions(len(requestUser.Username), len(requestUser.Password), len(requestUser.Email)) {
		http.Error(w, "Too big length of username/password/email", http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: Too big length of username/password/email"))
		return
	}

	used, err := IsUsernameEmailTaken(&requestUser, h.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: ")+err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: ") + err.Error())
		return
	}

	if used {
		http.Error(w, "Bad request: Username or Email already taken", http.StatusBadRequest)
		logger.Log.Info("Bad request: Username or Email already taken")
		return
	}

	_, err = h.db.Exec("INSERT INTO users (username, password, email) VALUES ($1, $2, $3)",
		requestUser.Username, requestUser.Password, requestUser.Email)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	var respondUser User
	err = h.db.QueryRow("SELECT user_id, username, email FROM users WHERE username = $1",
		requestUser.Username).Scan(&respondUser.ID, &respondUser.Username, &respondUser.Email)
	if err != nil {
		http.Error(w, "User created, but while making JSON for respond: "+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("User created, but while making JSON for respond: ") + err.Error())
		return
	}

	respondUserJSON, err := json.Marshal(respondUser)
	if err != nil {
		http.Error(w, "User created, but while making JSON for respond: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("User created, but while making JSON for respond: ") + err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respondUserJSON)

	if err != nil {
		http.Error(w, "User created, but while sending JSON for respond: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("User created, but while sending JSON for respond: ") + err.Error())
		return
	}
}

func (h *handler) GetUserByUUID(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var respondUser User
	respondUser.ID = params.ByName("uuid")

	intID, err := strconv.Atoi(respondUser.ID)
	counter, err := IdExists(intID, h.db)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}
	if counter == 0 {
		http.Error(w, "Bad request: User not found", http.StatusNotFound)
		logger.Log.Info("Bad request: User not found")
		return
	}

	err = h.db.QueryRow("SELECT username, email FROM users WHERE user_id = $1",
		respondUser.ID).Scan(&respondUser.Username, &respondUser.Email)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	respondUserJSON, err := json.Marshal(respondUser)
	if err != nil {
		http.Error(w, "User was gotten, but while making JSON for respond: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("User was gotten, but while making JSON for respond: ") + err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respondUserJSON)

	if err != nil {
		http.Error(w, "User was gotten, but while sending JSON for respond: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("User was gotten, but while sending JSON for respond: ") + err.Error())
		return
	}
}

func (h *handler) FullyUpdateUser(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var requestUser User
	err := json.NewDecoder(r.Body).Decode(&requestUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: ") + err.Error())
		return
	}

	uuid := params.ByName("uuid")

	if requestUser.ID != "" && requestUser.ID != uuid {
		http.Error(w, "Bad request: UUID and ID in request are different", http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: UUID and ID in request are different"))
		return
	}

	if requestUser.ID == "" {
		requestUser.ID = uuid
	}

	intID, err := strconv.Atoi(requestUser.ID)
	counter, err := IdExists(intID, h.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}
	if counter == 0 {
		http.Error(w, "Bad request: User not found", http.StatusNotFound)
		logger.Log.Info("Bad request: User not found")
		return
	}

	if !UserSuitableForRestrictions(len(requestUser.Username), len(requestUser.Password), len(requestUser.Email)) {
		http.Error(w, "Too big length of username/password/email", http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: Too big length of username/password/email"))
		return
	}

	used, err := IsUsernameEmailTaken(&requestUser, h.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: ")+err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: ") + err.Error())
		return
	}

	if used {
		http.Error(w, "Bad request: Username or Email already taken", http.StatusBadRequest)
		logger.Log.Info("Bad request: Username or Email already taken")
		return
	}

	_, err = h.db.Exec("UPDATE users SET username = $1, password = $2, email = $3 WHERE user_id = $4;",
		requestUser.Username, requestUser.Password, requestUser.Email, requestUser.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) UpdateUser(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var requestUser User
	err := json.NewDecoder(r.Body).Decode(&requestUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: ") + err.Error())
		return
	}

	uuid := params.ByName("uuid")

	if requestUser.ID != "" && requestUser.ID != uuid {
		http.Error(w, "Bad request: UUID and ID in request are different", http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: UUID and ID in request are different"))
		return
	}

	if requestUser.ID == "" {
		requestUser.ID = uuid
	}

	intID, err := strconv.Atoi(requestUser.ID)
	counter, err := IdExists(intID, h.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}
	if counter == 0 {
		http.Error(w, "Bad request: User not found", http.StatusNotFound)
		logger.Log.Info("Bad request: User not found")
		return
	}

	if !UserSuitableForRestrictions(len(requestUser.Username), len(requestUser.Password), len(requestUser.Email)) {
		http.Error(w, "Too big length of username/password/email", http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: Too big length of username/password/email"))
		return
	}

	used, err := IsUsernameEmailTaken(&requestUser, h.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: ")+err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: ") + err.Error())
		return
	}

	if used {
		http.Error(w, "Bad request: Username or Email already taken", http.StatusBadRequest)
		logger.Log.Info("Bad request: Username or Email already taken")
		return
	}

	var realUsername, realEmail string
	err = h.db.QueryRow("SELECT username, email FROM users WHERE user_id = $1",
		requestUser.ID).Scan(&realUsername, &realEmail)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	if requestUser.Username == "" {
		requestUser.Username = realUsername
	}
	if requestUser.Email == "" {
		requestUser.Email = realEmail
	}
	_, err = h.db.Exec("UPDATE users SET username = $1, email = $2 WHERE user_id = $3;",
		requestUser.Username, requestUser.Email, requestUser.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	if requestUser.Password != "" {
		_, err = h.db.Exec("UPDATE users SET password = $1 WHERE user_id = $2;",
			requestUser.Password, requestUser.ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
			logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) DeleteUser(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	uuid := params.ByName("uuid")

	intID, err := strconv.Atoi(uuid)
	counter, err := IdExists(intID, h.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}
	if counter == 0 {
		http.Error(w, "Bad request: User not found", http.StatusNotFound)
		logger.Log.Info("Bad request: User not found")
		return
	}

	_, err = h.db.Exec("DELETE FROM users WHERE user_id = $1", uuid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) AddFinishedBook(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	AddBook(w, r, params, true, h.db)
}

func (h *handler) AddWishlistBook(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	AddBook(w, r, params, false, h.db)
}

func AddBook(w http.ResponseWriter, r *http.Request, params httprouter.Params, isFinished bool, db *sql.DB) {

	var book interface{}
	var err error

	if isFinished {
		finishedBook := FinishedBook{}
		err = json.NewDecoder(r.Body).Decode(&finishedBook)
		book = finishedBook
	} else {
		wishlistBook := WishlistBook{}
		err = json.NewDecoder(r.Body).Decode(&wishlistBook)
		book = wishlistBook
	}

	if err != nil {
		http.Error(w, "Bad request body: "+err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request body: ") + err.Error())
		return
	}

	userID := params.ByName("uuid")

	now := time.Now()
	year, month, day := now.Date()
	date := fmt.Sprintf("%d-%d-%d", year, month, day)

	switch s := book.(type) {
	case WishlistBook:
		_, err = db.Exec(`
		INSERT INTO books (title, author, date_added, user_id, is_read, cover_image_url)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, ''))
		`, s.Title, s.Author, date, userID, 0, s.CoverImage)
	case FinishedBook:
		_, err = db.Exec(`
		INSERT INTO books (title, author, date_added, user_id, is_read, rating, comment, cover_image_url)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, ''), COALESCE($8, ''))
		`, s.Title, s.Author, date, userID, 1, s.Rating, s.Comment, s.CoverImage)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	return
}
