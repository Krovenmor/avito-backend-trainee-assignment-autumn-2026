package queries

import (
	"avito-kitchen/pkg/repo"
	"embed"
)

//go:embed *.sql
var EmbedQueries embed.FS

type Queries struct {
	UpsertUser          string
	GetPlaceForOrder    string
	LockOrderProducts   string
	ReserveStock        string
	InsertOrder         string
	GetOrder            string
	GetOrderItems       string
	ListUserOrders      string
	CancelOrder         string
	UpdateOrderStatus   string
	MarkDelivered       string
	GetOrderStatus      string
	GetOrderOwnerStatus string
	ReturnStock         string
	ZeroStock           string
	ListPendingOrders   string
	RenewOrderLeases    string
	InsertOrderRating   string
	AddPlaceRating      string
	InsertOrderTracking string
	ListOrderTracking   string
}

func GetQueries() (*Queries, error) {
	r := repo.NewQueriesReader(EmbedQueries)

	queries := Queries{
		UpsertUser:          r.GetQuery("upsert_user"),
		GetPlaceForOrder:    r.GetQuery("get_place_for_order"),
		LockOrderProducts:   r.GetQuery("lock_order_products"),
		ReserveStock:        r.GetQuery("reserve_stock"),
		InsertOrder:         r.GetQuery("insert_order"),
		GetOrder:            r.GetQuery("get_order"),
		GetOrderItems:       r.GetQuery("get_order_items"),
		ListUserOrders:      r.GetQuery("list_user_orders"),
		CancelOrder:         r.GetQuery("cancel_order"),
		UpdateOrderStatus:   r.GetQuery("update_order_status"),
		MarkDelivered:       r.GetQuery("mark_delivered"),
		GetOrderStatus:      r.GetQuery("get_order_status"),
		GetOrderOwnerStatus: r.GetQuery("get_order_owner_status"),
		ReturnStock:         r.GetQuery("return_stock"),
		ZeroStock:           r.GetQuery("zero_stock"),
		ListPendingOrders:   r.GetQuery("list_pending_orders"),
		RenewOrderLeases:    r.GetQuery("renew_order_leases"),
		InsertOrderRating:   r.GetQuery("insert_order_rating"),
		AddPlaceRating:      r.GetQuery("add_place_rating"),
		InsertOrderTracking: r.GetQuery("insert_order_tracking"),
		ListOrderTracking:   r.GetQuery("list_order_tracking"),
	}

	return &queries, r.Err()
}
