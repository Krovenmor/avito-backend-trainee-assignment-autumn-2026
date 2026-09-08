package main

import (
	"avito-kitchen/services/kitchen/internal/di"

	_ "avito-kitchen/docs"

	"go.uber.org/fx"
)

// @title Avito Kitchen API
// @version 0.1.0
// @description MVP платформы «Авито.Кухня».
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {
	fx.New(di.Module).Run()
}
