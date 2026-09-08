package tests

import (
	"net/http"
	"testing"
)

func TestRegisterPlace_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	base := map[string]any{
		"name":        "Test Place",
		"city":        "Moscow",
		"address":     "Lenina 1",
		"timezone":    "Europe/Moscow",
		"opens_at":    "09:00:00",
		"closes_at":   "22:00:00",
		"type":        "restaurant",
		"webhook_url": "https://example.com/webhook",
	}

	withOverride := func(field string, value any) map[string]any {
		body := make(map[string]any, len(base)+1)
		for k, v := range base {
			body[k] = v
		}
		body[field] = value
		return body
	}

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing name", withOverride("name", "")},
		{"invalid timezone", withOverride("timezone", "Not/A_Timezone")},
		{"invalid opens_at format", withOverride("opens_at", "09:00")},
		{"invalid type enum", withOverride("type", "bakery")},
		{"invalid webhook_url", withOverride("webhook_url", "not-a-url")},
		{"unknown field rejected", withOverride("surprise_field", "value")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/places", "", tc.body, http.StatusBadRequest, nil)
		})
	}
}

func TestUpdatePlace_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, nil)

	doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID, place.APIKey, map[string]any{
		"timezone": "Not/A_Timezone",
	}, http.StatusBadRequest, nil)

	doJSON(t, httpClient, http.MethodPatch, baseURL+"/api/v1/b2b/places/"+place.PlaceID, place.APIKey, map[string]any{
		"name": "Renamed",
	}, http.StatusOK, nil)
}

func TestListPlaces_InvalidIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/not-a-uuid/menu", "", nil, http.StatusBadRequest, nil)

	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places/11111111-1111-4111-8111-111111111111/menu", "", nil, http.StatusNotFound, nil)
}

func TestListPlaces_OutOfRangePagination(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	for _, query := range []string{"?from=3000000000", "?limit=3000000000", "?from=-1", "?limit=abc"} {
		doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places"+query, "", nil, http.StatusOK, nil)
	}
}
