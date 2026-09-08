package queries

import (
	"avito-kitchen/pkg/repo"
	"embed"
)

//go:embed *.sql
var EmbedQueries embed.FS

type Queries struct {
	InsertPlace          string
	GetPlaceByAPIKeyHash string
	UpdatePlace          string
	ListPlaces           string
	GetPlace             string
	PlaceExists          string
	InsertProducts       string
	UpdateProductMenu    string
	SoftDeleteProduct    string
	ListProducts         string
	SearchProducts       string
}

func GetQueries() (*Queries, error) {
	r := repo.NewQueriesReader(EmbedQueries)

	queries := Queries{
		InsertPlace:          r.GetQuery("insert_place"),
		GetPlaceByAPIKeyHash: r.GetQuery("get_place_by_api_key_hash"),
		UpdatePlace:          r.GetQuery("update_place"),
		ListPlaces:           r.GetQuery("list_places"),
		GetPlace:             r.GetQuery("get_place"),
		PlaceExists:          r.GetQuery("place_exists"),
		InsertProducts:       r.GetQuery("insert_products"),
		UpdateProductMenu:    r.GetQuery("update_product_menu"),
		SoftDeleteProduct:    r.GetQuery("soft_delete_product"),
		ListProducts:         r.GetQuery("list_products"),
		SearchProducts:       r.GetQuery("search_products"),
	}

	return &queries, r.Err()
}
