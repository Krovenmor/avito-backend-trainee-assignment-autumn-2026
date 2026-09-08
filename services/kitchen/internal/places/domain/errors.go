package domain

import "errors"

var (
	ErrNotFound            = errors.New("places: not found")
	ErrDuplicateExternalID = errors.New("places: external_id already exists for this place")
	ErrStockConflict       = errors.New("places: stock_quantity + stock_delta would go negative")
	ErrInvalidPrice        = errors.New("places: price must be non-negative")
)
