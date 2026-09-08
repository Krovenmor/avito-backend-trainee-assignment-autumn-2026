package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

type ratedPlace struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Rating      *string `json:"rating"`
	RatingCount int32   `json:"rating_count"`
}

func placeByID(t *testing.T, placeID string) ratedPlace {
	t.Helper()

	var places []ratedPlace
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places?limit=100", "", nil, http.StatusOK, &places)
	for _, place := range places {
		if place.ID == placeID {
			return place
		}
	}

	t.Fatalf("place %s is not in the listing", placeID)
	return ratedPlace{}
}

func completedOrder(t *testing.T, place placeRegistration, productID, userID string) createOrderResult {
	t.Helper()

	order := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(productID, 1)})
	waitOrderStatus(t, userID, order.OrderID, "accepted")

	doJSON(t, httpClient, http.MethodPost,
		baseURL+"/api/v1/b2b/orders/"+order.OrderID+"/delivered", place.APIKey, nil, http.StatusOK, nil)
	waitOrderStatus(t, userID, order.OrderID, "completed")

	return order
}

func rateOrder(t *testing.T, orderID, userID string, score int, wantStatus int) {
	t.Helper()

	doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/orders/"+orderID+"/rate", "",
		map[string]any{"user_id": userID, "score": score}, wantStatus, nil)
}

func ratedPlaceWithMenu(t *testing.T) (placeRegistration, string) {
	t.Helper()

	stub := acceptingWebhook(t)
	place := registerPlaceWithWebhook(t, stub)
	addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
		{"external_id": "sku-1", "name": "Margherita", "price": "10.00", "stock_quantity": 20},
	})

	return place, placeMenu(t, place.PlaceID)[0].ProductID
}

func TestRateOrder_AveragesIntoPlaceRating(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place, productID := ratedPlaceWithMenu(t)

	if before := placeByID(t, place.PlaceID); before.Rating != nil || before.RatingCount != 0 {
		t.Fatalf("expected a fresh place to have no rating, got %+v", before)
	}

	for _, score := range []int{5, 4} {
		userID := uuid.NewString()
		order := completedOrder(t, place, productID, userID)
		rateOrder(t, order.OrderID, userID, score, http.StatusOK)
	}

	after := placeByID(t, place.PlaceID)
	if after.Rating == nil || *after.Rating != "4.50" {
		t.Fatalf("expected the average of 5 and 4 to be 4.50, got %+v", after.Rating)
	}
	if after.RatingCount != 2 {
		t.Fatalf("expected 2 ratings, got %d", after.RatingCount)
	}
}

func TestRateOrder_OnlyOwnCompletedOrderAndOnlyOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place, productID := ratedPlaceWithMenu(t)

	userID := uuid.NewString()
	pending := createOrder(t, userID, place.PlaceID, []map[string]any{orderItem(productID, 1)})
	waitOrderStatus(t, userID, pending.OrderID, "accepted")
	rateOrder(t, pending.OrderID, userID, 5, http.StatusConflict)

	order := completedOrder(t, place, productID, userID)

	rateOrder(t, order.OrderID, uuid.NewString(), 5, http.StatusNotFound)
	rateOrder(t, order.OrderID, userID, 5, http.StatusOK)
	rateOrder(t, order.OrderID, userID, 1, http.StatusConflict)

	if got := placeByID(t, place.PlaceID); got.RatingCount != 1 {
		t.Fatalf("expected the rejected attempts to leave a single rating, got %d", got.RatingCount)
	}
}

func TestRateOrder_ValidatesScore(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place, productID := ratedPlaceWithMenu(t)
	userID := uuid.NewString()
	order := completedOrder(t, place, productID, userID)

	for _, score := range []int{0, 6, -1} {
		rateOrder(t, order.OrderID, userID, score, http.StatusBadRequest)
	}
}

func TestListPlaces_FiltersByMinRating(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	place, productID := ratedPlaceWithMenu(t)

	userID := uuid.NewString()
	order := completedOrder(t, place, productID, userID)
	rateOrder(t, order.OrderID, userID, 3, http.StatusOK)

	var above []ratedPlace
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places?min_rating=2.5&limit=100", "", nil, http.StatusOK, &above)
	if !containsPlace(above, place.PlaceID) {
		t.Fatal("expected a place rated 3 to pass min_rating=2.5")
	}

	var below []ratedPlace
	doJSON(t, httpClient, http.MethodGet, baseURL+"/api/v1/places?min_rating=4&limit=100", "", nil, http.StatusOK, &below)
	if containsPlace(below, place.PlaceID) {
		t.Fatal("expected a place rated 3 to be filtered out by min_rating=4")
	}

	unrated := registerTestPlace(t, nil)
	if containsPlace(below, unrated.PlaceID) {
		t.Fatal("expected an unrated place to be excluded by min_rating")
	}
}

func containsPlace(places []ratedPlace, placeID string) bool {
	for _, place := range places {
		if place.ID == placeID {
			return true
		}
	}
	return false
}
