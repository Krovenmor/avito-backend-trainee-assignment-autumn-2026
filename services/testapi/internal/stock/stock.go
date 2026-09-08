package stock

import (
	"math/rand/v2"
	"sort"
	"sync"
)

type Item struct {
	ExternalID  string
	Name        string
	Description string
	Price       string
	Quantity    int32
}

type Stock struct {
	mu      sync.Mutex
	items   map[string]*Item
	initial map[string]int32
	order   []string
}

func New(items []Item) *Stock {
	s := &Stock{
		items:   make(map[string]*Item, len(items)),
		initial: make(map[string]int32, len(items)),
		order:   make([]string, 0, len(items)),
	}
	for _, item := range items {
		stored := item
		s.items[item.ExternalID] = &stored
		s.initial[item.ExternalID] = item.Quantity
		s.order = append(s.order, item.ExternalID)
	}
	return s
}

func (s *Stock) Items() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Item, 0, len(s.order))
	for _, externalID := range s.order {
		items = append(items, *s.items[externalID])
	}
	return items
}

func (s *Stock) Reserve(lines map[string]int32) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var unavailable []string
	for externalID, quantity := range lines {
		item, known := s.items[externalID]
		if !known || item.Quantity < quantity {
			unavailable = append(unavailable, externalID)
		}
	}
	if len(unavailable) > 0 {
		sort.Strings(unavailable)
		return unavailable
	}

	for externalID, quantity := range lines {
		s.items[externalID].Quantity -= quantity
	}
	return nil
}

func (s *Stock) SellOffline(quantity int32) (string, int32, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	available := make([]string, 0, len(s.order))
	for _, externalID := range s.order {
		if s.items[externalID].Quantity > 0 {
			available = append(available, externalID)
		}
	}
	if len(available) == 0 {
		return "", 0, false
	}

	item := s.items[available[rand.IntN(len(available))]]
	if quantity > item.Quantity {
		quantity = item.Quantity
	}
	item.Quantity -= quantity

	return item.ExternalID, quantity, true
}

func (s *Stock) Restock() (string, int32, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		target *Item
		gap    int32
	)
	for _, externalID := range s.order {
		item := s.items[externalID]
		if missing := s.initial[externalID] - item.Quantity; missing > gap {
			target, gap = item, missing
		}
	}
	if target == nil {
		return "", 0, false
	}

	target.Quantity += gap
	return target.ExternalID, gap, true
}
