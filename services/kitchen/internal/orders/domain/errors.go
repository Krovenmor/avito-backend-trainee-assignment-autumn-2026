package domain

import "errors"

var (
	ErrNotFound              = errors.New("orders: not found")
	ErrPlaceClosed           = errors.New("orders: place is closed")
	ErrOutOfStock            = errors.New("orders: not enough stock")
	ErrUnknownProducts       = errors.New("orders: some products do not belong to this place")
	ErrStatusConflict        = errors.New("orders: order is not in a status that allows this transition")
	ErrAlreadyRated          = errors.New("orders: order is already rated")
	ErrIdempotencyInProgress = errors.New("orders: another request with this idempotency key is in progress")
	ErrIdempotencyMismatch   = errors.New("orders: idempotency key reused with a different body")
)

type OutOfStockError struct {
	Items []UnavailableItem
}

func (e *OutOfStockError) Error() string {
	return ErrOutOfStock.Error()
}

func (e *OutOfStockError) Is(target error) bool {
	return target == ErrOutOfStock
}
