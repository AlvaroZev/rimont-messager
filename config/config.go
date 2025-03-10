package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	DatabaseURL          string `envconfig:"database_url" default:""`
	WhatsAppDatabaseName string `envconfig:"whatsapp_database_name" default:"examplestore.db"`
	RestartDBonInit      bool   `envconfig:"restart_db_on_init" default:"false"`
	JwtSecret            string `envconfig:"jwt_secret"`
	JwtUser              string `envconfig:"jwt_user"`
	JwtPassword          string `envconfig:"jwt_password"`
	MainNumber           string `envconfig:"main_number"`
}

func NewLoadedConfig() (*Config, error) {
	godotenv.Load()

	var c Config
	err := envconfig.Process("rimont", &c)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &c, nil
}
