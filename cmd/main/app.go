package main

import (
	"database/sql"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"myLibrary/internal/config"
	"myLibrary/internal/user"
	"myLibrary/package/client/database"
	"myLibrary/package/logger"
	"net"
	"net/http"
	"time"
)

func main() {
	cfg := config.GetConfig()

	logger.Log.Info("Starting database")
	db := database.Init(cfg)

	router := httprouter.New()

	handler := user.NewHandler(db, cfg)
	handler.Register(router)

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			logger.Log.Error("Can not close database")
		}
	}(db)

	logger.Log.Info("Starting app")
	start(router, cfg)
}

func start(router *httprouter.Router, cfg *config.Config) {
	logger.Log.Info("Starting router")
	logger.Log.Info("Listening TCP")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.Listen.BindIp, cfg.Listen.Port))
	logger.Log.Info("Listening ", fmt.Sprintf("%s:%s", cfg.Listen.BindIp, cfg.Listen.Port))

	if err != nil {
		logger.Log.Fatal("Listener was not created")
		panic(err)
	}
	server := &http.Server{
		Handler:      router,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	err = server.Serve(listener)
	if err != nil {
		logger.Log.Fatal("Server was not created")
		panic(err)
	}
}
