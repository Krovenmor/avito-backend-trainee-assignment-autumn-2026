package config

import (
	"net"

	"github.com/caarlos0/env/v11"
)

type ServConfig struct {
	Host string `env:"LISTEN_ADDRESS,required,notEmpty"`
	Port string `env:"LISTEN_PORT,required,notEmpty"`
}

func (c ServConfig) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

func GetServConfig() (ServConfig, error) {
	var conf ServConfig
	return conf, env.Parse(&conf)
}
