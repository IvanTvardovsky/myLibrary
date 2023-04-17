package user

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"myLibrary/internal/handlers"
	"myLibrary/package/logger"
	"net/http"
)

const (
	usersURL = "/users"
	userURL  = "/user/:uuid"
)

type handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) handlers.Handler {
	return &handler{db}
}

func (h *handler) Register(router *httprouter.Router) {
	router.POST(userURL, h.CreateUser)
	router.GET(userURL, h.GetUserByUUID)
	router.PUT(userURL, h.FullyUpdateUser)
	router.PATCH(userURL, h.UpdateUser)
	router.DELETE(userURL, h.DeleteUser)
	//router.GET(usersURL, h.GetList)
	// функция авторизации?
}

func (h *handler) CreateUser(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	var requestUser User
	err := json.NewDecoder(r.Body).Decode(&requestUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad request: ") + err.Error())
		return
	}

	if len(requestUser.Username) > 32 || len(requestUser.Password) > 128 || len(requestUser.Email) > 64 {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		http.Error(w, "User created, but while sending JSON for respond: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("User created, but while sending JSON for respond: ") + err.Error())
		return
	}
}

func (h *handler) GetUserByUUID(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var respondUser User
	respondUser.ID = params.ByName("uuid")

	err := h.db.QueryRow("SELECT username, email FROM users WHERE user_id = $1",
		respondUser.ID).Scan(&respondUser.Username, &respondUser.Email)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}

	respondUserJSON, err := json.Marshal(respondUser)
	if err != nil {
		http.Error(w, "User created, but while making JSON for respond: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("User created, but while making JSON for respond: ") + err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(respondUserJSON)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		http.Error(w, "User created, but while sending JSON for respond: "+err.Error(), http.StatusInternalServerError)
		logger.Log.Info(fmt.Sprintf("User created, but while sending JSON for respond: ") + err.Error())
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

	var IdExists int
	err = h.db.QueryRow("SELECT COUNT(*) FROM users WHERE user_id = $1", requestUser.ID).Scan(&IdExists)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database unavailable: ")+err.Error(), http.StatusServiceUnavailable)
		logger.Log.Info(fmt.Sprintf("Database unavailable: ") + err.Error())
		return
	}
	if IdExists == 0 {
		http.Error(w, "Bad request: User not found", http.StatusNotFound)
		logger.Log.Info("Bad request: User not found")
		return
	}

	if len(requestUser.Username) > 32 || len(requestUser.Password) > 128 || len(requestUser.Email) > 64 {
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
	w.Write([]byte("user updated"))
}

func (h *handler) DeleteUser(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	w.Write([]byte("user deleted"))
}

/*func (h *handler) GetList(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	w.Write([]byte("this is list of users"))
}*/
