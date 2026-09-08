package di

import (
	"avito-kitchen/services/testapi/internal/client"
	"avito-kitchen/services/testapi/internal/stock"
)

var menu = []stock.Item{
	{ExternalID: "pizza-margherita", Name: "Пицца Маргарита", Description: "Томаты, моцарелла, базилик", Price: "590.00", Quantity: 20},
	{ExternalID: "pizza-pepperoni", Name: "Пицца Пепперони", Description: "Пепперони, моцарелла", Price: "690.00", Quantity: 15},
	{ExternalID: "pasta-carbonara", Name: "Паста Карбонара", Description: "Гуанчиале, пекорино, яйцо", Price: "540.00", Quantity: 12},
	{ExternalID: "salad-caesar", Name: "Салат Цезарь", Description: "Курица, пармезан, сухарики", Price: "430.00", Quantity: 10},
	{ExternalID: "drink-lemonade", Name: "Домашний лимонад", Description: "Лимон, мята, лёд", Price: "190.00", Quantity: 30},
}

func placeRegistration(webhookURL string) client.RegisterPlaceRequest {
	return client.RegisterPlaceRequest{
		Name:        "Тестовая Пиццерия",
		Description: "Демонстрационное заведение, поднимается вместе с платформой",
		City:        "Moscow",
		Address:     "Ленина, 1",
		Timezone:    "Europe/Moscow",
		OpensAt:     "00:00:00",
		ClosesAt:    "23:59:59",
		Type:        "restaurant",
		WebhookURL:  webhookURL,
	}
}
