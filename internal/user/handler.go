package user

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"myLibrary/internal/handlers"
	"myLibrary/package/logger"
	"net/http"
	"strconv"
)

const (
	RegisterUrl = "/register"
	LoginUrl    = "/login"
	userUuidUrl = "/user/:uuid"
)

type handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) handlers.Handler {
	return &handler{db}
}

func (h *handler) Register(router *httprouter.Router) {
	router.POST(RegisterUrl, h.RegisterUser) // можно пытаться создать пользователя с заданным uuid
	router.POST(LoginUrl, h.LoginUser)
	router.GET(userUuidUrl, h.GetUserByUUID)
	router.PUT(userUuidUrl, h.FullyUpdateUser)
	router.PATCH(userUuidUrl, h.UpdateUser)
	router.DELETE(userUuidUrl, h.DeleteUser)
	//router.GET("", h.GetList)
	// функция авторизации
	// функция получения всех прочитанных книг пользователя
	// функция получения всех книг из вишлиста пользователя
}

func (h *handler) LoginUser(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var loginRequest LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Log.Info(fmt.Sprintf("Bad login request: ") + err.Error())
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

	if !SuitableForRestrictions(len(requestUser.Username), len(requestUser.Password), len(requestUser.Email)) {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	if !SuitableForRestrictions(len(requestUser.Username), len(requestUser.Password), len(requestUser.Email)) {
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

	if !SuitableForRestrictions(len(requestUser.Username), len(requestUser.Password), len(requestUser.Email)) {
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

/*func (h *handler) GetList(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	w.Write([]byte("this is list of users"))
}*/
