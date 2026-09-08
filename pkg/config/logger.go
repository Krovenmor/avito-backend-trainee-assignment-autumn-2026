package config

import "github.com/caarlos0/env/v11"

type LoggerConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

func GetLoggerConfig() (LoggerConfig, error) {
	var conf LoggerConfig
	return conf, env.Parse(&conf)
}
