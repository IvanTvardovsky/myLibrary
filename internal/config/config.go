package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"myLibrary/package/logger"
	"sync"
)

type Config struct {
	IsDebug *bool         `yaml:"is_debug" env-required:"true"`
	Listen  Listener      `yaml:"listen"`
	Storage StorageConfig `yaml:"storage"`
	Key     JWTSecretKey  `yaml:"authorization"`
}

type Listener struct {
	Type   string `yaml:"type"`
	BindIp string `yaml:"bind_ip"`
	Port   string `yaml:"port"`
}

type StorageConfig struct {
	Host     string `yaml:"host"`
	Port     rune   `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type JWTSecretKey struct {
	SecretKey string `yaml:"key"`
}

var instance *Config
var once sync.Once

func GetConfig() *Config {
	once.Do(func() {
		logger.Log.Info("Reading app configuration")
		instance = &Config{}
		if err := cleanenv.ReadConfig("config.yml", instance); err != nil {
			help, _ := cleanenv.GetDescription(instance, nil)
			logger.Log.Error(help)
			logger.Log.Fatal(err)
		}
	})
	return instance
}
