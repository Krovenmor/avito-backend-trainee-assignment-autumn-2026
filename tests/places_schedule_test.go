package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// zoneWithLocalHour возвращает Etc/GMT-зону, в которой прямо сейчас указанный час.
// Etc/GMT-5 означает UTC+5: знак в этих именах инвертирован.
func zoneWithLocalHour(want int) string {
	offset := ((want-time.Now().UTC().Hour())%24 + 24) % 24
	if offset > 12 {
		offset -= 24
	}
	switch {
	case offset == 0:
		return "Etc/GMT"
	case offset > 0:
		return fmt.Sprintf("Etc/GMT-%d", offset)
	default:
		return fmt.Sprintf("Etc/GMT+%d", -offset)
	}
}

func TestPlaceSchedule_CrossingMidnight(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	cases := []struct {
		name     string
		opensAt  string
		closesAt string
		localNow int
		wantOpen bool
	}{
		{"через полночь, локально 23:00 — открыто", "22:00:00", "02:00:00", 23, true},
		{"через полночь, локально 01:00 — открыто", "22:00:00", "02:00:00", 1, true},
		{"через полночь, локально 12:00 — закрыто", "22:00:00", "02:00:00", 12, false},
		{"обычное окно, локально 12:00 — открыто", "09:00:00", "18:00:00", 12, true},
		{"обычное окно, локально 23:00 — закрыто", "09:00:00", "18:00:00", 23, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			zone := zoneWithLocalHour(tc.localNow)
			place := registerTestPlace(t, map[string]any{
				"name": "Расписание", "city": "Sched" + uuid.NewString()[:8],
				"timezone": zone, "opens_at": tc.opensAt, "closes_at": tc.closesAt,
			})

			var detail struct {
				IsOpen bool `json:"is_open"`
			}
			doRequest(t, http.MethodGet, baseURL+"/api/v1/places/"+place.PlaceID, nil, nil, http.StatusOK, &detail)
			if detail.IsOpen != tc.wantOpen {
				t.Fatalf("%s–%s в зоне %s: is_open=%v, ожидали %v", tc.opensAt, tc.closesAt, zone, detail.IsOpen, tc.wantOpen)
			}

			addTestMenuItems(t, place.PlaceID, place.APIKey, []map[string]any{
				{"external_id": "sku-1", "name": "Блюдо", "price": "10.00", "stock_quantity": 5},
			})
			menu := placeMenu(t, place.PlaceID)

			wantStatus := http.StatusConflict
			if tc.wantOpen {
				wantStatus = http.StatusCreated
			}
			createOrderRaw(t, uuid.NewString(), map[string]any{
				"user_id":  uuid.NewString(),
				"place_id": place.PlaceID,
				"items":    []map[string]any{orderItem(menu[0].ProductID, 1)},
			}, wantStatus, nil)
		})
	}
}
