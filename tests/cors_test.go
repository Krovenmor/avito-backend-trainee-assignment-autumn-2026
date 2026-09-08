package tests

import (
	"net/http"
	"strings"
	"testing"
)

func TestCORS_PreflightAndActualRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	req, err := http.NewRequest(http.MethodOptions, baseURL+"/api/v1/orders", nil)
	if err != nil {
		t.Fatalf("new preflight request: %v", err)
	}
	req.Header.Set("Origin", "https://kitchen.avito.ru")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Idempotency-Key")

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", resp.StatusCode)
	}
	if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin == "" {
		t.Fatal("expected Access-Control-Allow-Origin on the preflight response")
	}
	for _, header := range []string{"Content-Type", "X-API-Key", "Idempotency-Key"} {
		if !strings.Contains(resp.Header.Get("Access-Control-Allow-Headers"), header) {
			t.Fatalf("expected %s to be allowed, got %q", header, resp.Header.Get("Access-Control-Allow-Headers"))
		}
	}

	listReq, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/places", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	listReq.Header.Set("Origin", "https://kitchen.avito.ru")

	listResp, err := httpClient.Do(listReq)
	if err != nil {
		t.Fatalf("list places: %v", err)
	}
	defer listResp.Body.Close()

	if origin := listResp.Header.Get("Access-Control-Allow-Origin"); origin == "" {
		t.Fatal("expected Access-Control-Allow-Origin on a normal response too")
	}
}
