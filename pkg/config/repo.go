package config

import "github.com/caarlos0/env/v11"

type RepoConfig struct {
	ConnString string `env:"DB_CONN_STRING,required,notEmpty"`
}

func GetRepoConfig() (RepoConfig, error) {
	var conf RepoConfig
	return conf, env.Parse(&conf)
}
