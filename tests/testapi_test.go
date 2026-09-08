package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTestapi_RegistersItselfAndSeedsMenu(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	startTestapi(t, map[string]string{"OFFLINE_SALES_INTERVAL": "0"})
	place := waitForTestapiPlace(t)

	menu := placeMenu(t, place.ID)
	if len(menu) == 0 {
		t.Fatal("expected testapi to seed its menu on the platform")
	}

	margherita := menuItemByName(t, place.ID, "Пицца Маргарита")
	if margherita.StockQuantity != 20 {
		t.Fatalf("expected the seeded stock of 20, got %d", margherita.StockQuantity)
	}
	if margherita.Price != "590.00" {
		t.Fatalf("expected the seeded price 590.00, got %q", margherita.Price)
	}
}

func TestTestapi_AcceptsOrderAndMarksItDelivered(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	startTestapi(t, map[string]string{
		"OFFLINE_SALES_INTERVAL": "0",
		"DELIVERY_DELAY":         "500ms",
	})
	place := waitForTestapiPlace(t)
	margherita := menuItemByName(t, place.ID, "Пицца Маргарита")

	userID := uuid.NewString()
	order := createOrder(t, userID, place.ID, []map[string]any{orderItem(margherita.ProductID, 2)})

	waitOrderStatus(t, userID, order.OrderID, "accepted")
	waitOrderStatus(t, userID, order.OrderID, "completed")

	if stock := productStock(t, place.ID, margherita.ProductID); stock != 18 {
		t.Fatalf("expected the accepted order to keep the stock reserved (18), got %d", stock)
	}
}

func TestTestapi_RejectsWhenItsOwnStockRanOut(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	startTestapi(t, map[string]string{
		"OFFLINE_SALES_INTERVAL": "20ms",
		"DELIVERY_DELAY":         "500ms",
	})
	place := waitForTestapiPlace(t)

	time.Sleep(5 * time.Second)

	margherita := menuItemByName(t, place.ID, "Пицца Маргарита")
	if margherita.StockQuantity == 0 {
		t.Fatal("expected the platform to still believe the product is in stock")
	}

	userID := uuid.NewString()
	order := createOrder(t, userID, place.ID, []map[string]any{orderItem(margherita.ProductID, 1)})

	waitOrderStatus(t, userID, order.OrderID, "rejected")

	if stock := productStock(t, place.ID, margherita.ProductID); stock != 0 {
		t.Fatalf("expected the platform to zero the stock after an out_of_stock rejection, got %d", stock)
	}
}
