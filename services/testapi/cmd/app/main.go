package main

import (
	"avito-kitchen/services/testapi/internal/di"

	"go.uber.org/fx"
)

func main() {
	fx.New(di.Module).Run()
}
