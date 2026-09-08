package tests

import (
	"net/http"
	"testing"
)

func TestProtectedEndpoints_MissingOrInvalidAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place := registerTestPlace(t, nil)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"update place", http.MethodPatch, "/api/v1/b2b/places/" + place.PlaceID},
		{"add menu items", http.MethodPost, "/api/v1/b2b/places/" + place.PlaceID + "/menu"},
		{"update menu item", http.MethodPatch, "/api/v1/b2b/places/" + place.PlaceID + "/menu/sku-1"},
		{"delete menu item", http.MethodDelete, "/api/v1/b2b/places/" + place.PlaceID + "/menu/sku-1"},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/missing key", func(t *testing.T) {
			doJSON(t, httpClient, tc.method, baseURL+tc.path, "", nil, http.StatusUnauthorized, nil)
		})
		t.Run(tc.name+"/invalid key", func(t *testing.T) {
			doJSON(t, httpClient, tc.method, baseURL+tc.path, "not-a-real-key", nil, http.StatusUnauthorized, nil)
		})
	}
}

func TestProtectedEndpoints_WrongPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	placeA := registerTestPlace(t, nil)
	placeB := registerTestPlace(t, nil)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"update place", http.MethodPatch, "/api/v1/b2b/places/" + placeB.PlaceID},
		{"add menu items", http.MethodPost, "/api/v1/b2b/places/" + placeB.PlaceID + "/menu"},
		{"update menu item", http.MethodPatch, "/api/v1/b2b/places/" + placeB.PlaceID + "/menu/sku-1"},
		{"delete menu item", http.MethodDelete, "/api/v1/b2b/places/" + placeB.PlaceID + "/menu/sku-1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doJSON(t, httpClient, tc.method, baseURL+tc.path, placeA.APIKey, nil, http.StatusForbidden, nil)
		})
	}
}
