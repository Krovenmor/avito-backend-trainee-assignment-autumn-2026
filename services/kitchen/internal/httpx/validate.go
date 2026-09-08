package httpx

import (
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	NonnegativeTag = "nonnegative"
	TimezoneTag    = "tz"
)

func NewValidator() (*validator.Validate, error) {
	v := validator.New()

	if err := v.RegisterValidation(NonnegativeTag, validateNonnegative); err != nil {
		return nil, err
	}
	if err := v.RegisterValidation(TimezoneTag, validateTimezone); err != nil {
		return nil, err
	}

	return v, nil
}

func validateNonnegative(fl validator.FieldLevel) bool {
	value, err := strconv.ParseFloat(fl.Field().String(), 64)
	if err != nil {
		return false
	}
	return value >= 0
}

func validateTimezone(fl validator.FieldLevel) bool {
	_, err := time.LoadLocation(fl.Field().String())
	return err == nil
}
