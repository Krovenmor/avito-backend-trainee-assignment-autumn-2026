package http

type orderItemPayload struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Quantity   int32  `json:"quantity"`
	Price      string `json:"price"`
}

type orderPayload struct {
	OrderID     string             `json:"order_id"`
	PlaceID     string             `json:"place_id"`
	TotalAmount string             `json:"total_amount"`
	Items       []orderItemPayload `json:"items"`
}

type decisionResponse struct {
	Decision               string   `json:"decision"`
	Reason                 string   `json:"reason,omitempty"`
	UnavailableExternalIDs []string `json:"unavailable_external_ids,omitempty"`
}

func toRequestedLines(items []orderItemPayload) map[string]int32 {
	lines := make(map[string]int32, len(items))
	for _, item := range items {
		lines[item.ExternalID] += item.Quantity
	}
	return lines
}
