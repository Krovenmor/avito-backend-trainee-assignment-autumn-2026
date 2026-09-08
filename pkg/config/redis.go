package config

import "github.com/caarlos0/env/v11"

type RedisConfig struct {
	Addr string `env:"REDIS_ADDR,required,notEmpty"`
}

func GetRedisConfig() (RedisConfig, error) {
	var conf RedisConfig
	return conf, env.Parse(&conf)
}
