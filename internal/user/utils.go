package user

import "database/sql"

func IsUsernameEmailTaken(user *User, db *sql.DB) (bool, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1 OR email = $2",
		user.Username, user.Email).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
