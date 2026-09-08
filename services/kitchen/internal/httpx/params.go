package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100

	MaxBodySize = 1 << 20

	QueryParamLimit  = "limit"
	QueryParamFrom   = "from"
	QueryParamSearch = "search"

	UUIDTag = "required,uuid4_rfc4122"

	decimalBase = 10
	int32Bits   = 32
)

func NormalizeUUID(value string) string {
	return strings.ToLower(value)
}

func ParsePagination(r *http.Request) (limit, offset int32) {
	limit = DefaultLimit
	if v := r.URL.Query().Get(QueryParamLimit); v != "" {
		if n, err := strconv.ParseInt(v, decimalBase, int32Bits); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	if v := r.URL.Query().Get(QueryParamFrom); v != "" {
		if n, err := strconv.ParseInt(v, decimalBase, int32Bits); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	return limit, offset
}

func OptionalQueryParam(r *http.Request, key string) *string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}
	return &v
}

func UUIDPathParam(v *validator.Validate, w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	value := r.PathValue(key)
	if err := v.Var(value, UUIDTag); err != nil {
		WriteError(w, http.StatusBadRequest, key+" must be a valid UUID")
		return "", false
	}
	return NormalizeUUID(value), true
}

func StringPathParam(v *validator.Validate, w http.ResponseWriter, r *http.Request, key, tag string) (string, bool) {
	value := r.PathValue(key)
	if err := v.Var(value, tag); err != nil {
		WriteError(w, http.StatusBadRequest, key+" is required")
		return "", false
	}
	return value, true
}

func UUIDQueryParam(v *validator.Validate, w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	value := r.URL.Query().Get(key)
	if err := v.Var(value, UUIDTag); err != nil {
		WriteError(w, http.StatusBadRequest, key+" must be a valid UUID")
		return "", false
	}
	return NormalizeUUID(value), true
}

func DecodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func DecodeAndValidate[T any](v *validator.Validate, r *http.Request) (T, error) {
	var req T
	if err := DecodeJSON(r, &req); err != nil {
		return req, err
	}
	return req, v.Struct(req)
}

func ReadBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r.Body, MaxBodySize))
}

func UnmarshalAndValidate[T any](v *validator.Validate, body []byte) (T, error) {
	var req T
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, err
	}
	return req, v.Struct(req)
}

func DecodeAndValidateSlice[T any](v *validator.Validate, r *http.Request) ([]T, error) {
	var req []T
	if err := DecodeJSON(r, &req); err != nil {
		return req, err
	}
	return req, v.Var(req, "min=1,dive")
}
