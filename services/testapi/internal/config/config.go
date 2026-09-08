package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	KitchenBaseURL       string        `env:"KITCHEN_BASE_URL,required,notEmpty"`
	PublicURL            string        `env:"PUBLIC_URL,required,notEmpty"`
	DeliveryDelay        time.Duration `env:"DELIVERY_DELAY" envDefault:"30s"`
	OfflineSalesInterval time.Duration `env:"OFFLINE_SALES_INTERVAL" envDefault:"20s"`
	RestockInterval      time.Duration `env:"RESTOCK_INTERVAL" envDefault:"60s"`
}

func GetConfig() (Config, error) {
	var conf Config
	return conf, env.Parse(&conf)
}
