package database

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"myLibrary/internal/config"
	"myLibrary/package/logger"
)

func Init(config *config.Config) *sql.DB {
	logger.Log.Info(fmt.Sprintf("Connecting to host=%s port=%d user=%s dbname=%s",
		config.Storage.Host, config.Storage.Port, config.Storage.Username, config.Storage.Database))
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Storage.Host, config.Storage.Port, config.Storage.Username, config.Storage.Password, config.Storage.Database)

	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		logger.Log.Error(err)
		logger.Log.Fatal("Can not connect to database")
	}

	err = db.Ping()

	logger.Log.Info("Connected to database")
	return db
}
