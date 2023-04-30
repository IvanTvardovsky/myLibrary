package user

import (
	"database/sql"
)

func IsUsernameEmailTaken(user *User, db *sql.DB) (bool, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1 OR email = $2",
		user.Username, user.Email).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func IdExists(ID int, db *sql.DB) (int, error) {
	var IdExists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE user_id = $1", ID).Scan(&IdExists)
	if err != nil {
		return 0, err
	}
	if IdExists == 0 {
		return 0, nil
	}
	return 1, nil
}

func UserSuitableForRestrictions(lenUsername, lenPassword, lenEmail int) bool {
	if lenUsername > 32 || lenPassword > 128 || lenEmail > 64 {
		return false
	}
	return true
}
