package logger

import (
	"github.com/sirupsen/logrus"
	"os"
)

func Init() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)
	return logger
}

var Log = Init()
