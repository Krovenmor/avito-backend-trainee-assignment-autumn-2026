package repo

import (
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewTrManager(pool *pgxpool.Pool) (trm.Manager, error) {
	return manager.New(trmpgx.NewDefaultFactory(pool))
}

func NewCtxGetter() *trmpgx.CtxGetter {
	return trmpgx.DefaultCtxGetter
}
