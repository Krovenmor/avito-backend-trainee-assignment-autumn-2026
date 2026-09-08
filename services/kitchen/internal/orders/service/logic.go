package service

import (
	"context"
	"encoding/json"

	"avito-kitchen/services/kitchen/internal/orders/domain"

	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type idempotencyRecord struct {
	BodyHash string             `json:"body_hash"`
	OrderID  string             `json:"order_id,omitempty"`
	Status   domain.OrderStatus `json:"status,omitempty"`
}

type svc struct {
	repo     Repository
	users    UserRepository
	ratings  RatingRepository
	trm      trm.Manager
	events   EventStream
	idem     IdempotencyStore
	notifier Notifier
	instance domain.InstanceID
	log      *zap.Logger
}

func New(
	repo Repository,
	users UserRepository,
	ratings RatingRepository,
	trManager trm.Manager,
	events EventStream,
	idem IdempotencyStore,
	notifier Notifier,
	instance domain.InstanceID,
	log *zap.Logger,
) Service {
	return &svc{
		repo:     repo,
		users:    users,
		ratings:  ratings,
		trm:      trManager,
		events:   events,
		idem:     idem,
		notifier: notifier,
		instance: instance,
		log:      log,
	}
}

func (s *svc) CreateOrder(ctx context.Context, in domain.CreateOrderInput) (*domain.CreateOrderOutput, error) {
	record := idempotencyRecord{BodyHash: in.BodyHash}
	value, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	stored, acquired, err := s.idem.Acquire(ctx, in.IdempotencyKey, string(value))
	if err != nil {
		return nil, err
	}
	if !acquired {
		return replayIdempotent(stored, in.BodyHash)
	}

	out, err := s.createOrder(ctx, in)
	if err != nil {
		if releaseErr := s.idem.Release(ctx, in.IdempotencyKey); releaseErr != nil {
			s.log.Error("release idempotency key", zap.Error(releaseErr))
		}
		return nil, err
	}

	record.OrderID = out.OrderID
	record.Status = out.Status
	if value, err = json.Marshal(record); err == nil {
		if err := s.idem.Save(ctx, in.IdempotencyKey, string(value)); err != nil {
			s.log.Error("save idempotency response", zap.String("order_id", out.OrderID), zap.Error(err))
		}
	}

	return out, nil
}

func (s *svc) createOrder(ctx context.Context, in domain.CreateOrderInput) (*domain.CreateOrderOutput, error) {
	params := domain.CreateOrderParams{
		OrderID:  uuid.NewString(),
		UserID:   in.UserID,
		PlaceID:  in.PlaceID,
		LockedBy: string(s.instance),
		Items:    in.Items,
	}

	var (
		place *domain.PlaceForOrder
		total *domain.OrderTotal
	)

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		if err := s.users.UpsertUser(ctx, params.UserID); err != nil {
			return err
		}

		var err error
		place, err = s.repo.GetPlaceForOrder(ctx, params.PlaceID)
		if err != nil {
			return err
		}
		if !place.IsActive || !place.IsOpen {
			return domain.ErrPlaceClosed
		}

		if err := s.reserveItems(ctx, params); err != nil {
			return err
		}

		total, err = s.repo.InsertOrder(ctx, params)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.log.Debug("order created",
		zap.String("order_id", params.OrderID),
		zap.String("user_id", params.UserID),
		zap.String("place_id", params.PlaceID),
		zap.Int("items", len(params.Items)),
		zap.String("total_amount", total.TotalAmount),
	)

	event := domain.OrderEvent{OrderID: params.OrderID, UserID: params.UserID, Source: domain.SourcePlatform, Status: string(domain.StatusPending)}
	if err := s.events.Publish(ctx, event); err != nil {
		s.log.Error("publish order event", zap.String("order_id", params.OrderID), zap.Error(err))
	}

	s.notifier.Notify(domain.PendingOrder{
		ID:         params.OrderID,
		UserID:     params.UserID,
		PlaceID:    params.PlaceID,
		CreatedAt:  total.CreatedAt,
		WebhookURL: place.WebhookURL,
	})

	return &domain.CreateOrderOutput{OrderID: params.OrderID, Status: domain.StatusPending}, nil
}

func (s *svc) reserveItems(ctx context.Context, params domain.CreateOrderParams) error {
	productIDs := uniqueProductIDs(params.Items)

	found, err := s.repo.LockPlaceProducts(ctx, params.PlaceID, productIDs)
	if err != nil {
		return err
	}
	if len(found) != len(productIDs) {
		return domain.ErrUnknownProducts
	}

	unavailable, err := s.repo.ReserveStock(ctx, params.Items)
	if err != nil {
		return err
	}
	if len(unavailable) > 0 {
		return &domain.OutOfStockError{Items: unavailable}
	}

	return nil
}

func uniqueProductIDs(items []domain.CreateOrderItemInput) []string {
	ids := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, exists := seen[item.ProductID]; exists {
			continue
		}
		seen[item.ProductID] = struct{}{}
		ids = append(ids, item.ProductID)
	}
	return ids
}

func (s *svc) CancelOrder(ctx context.Context, orderID, userID string) error {
	err := s.trm.Do(ctx, func(ctx context.Context) error {
		cancelled, err := s.repo.CancelPendingOrder(ctx, orderID, userID)
		if err != nil {
			return err
		}
		if !cancelled {
			return s.cancelRejection(ctx, orderID, userID)
		}

		return s.repo.ReturnStock(ctx, orderID, nil)
	})
	if err != nil {
		return err
	}

	s.log.Debug("order cancelled by user", zap.String("order_id", orderID), zap.String("user_id", userID))

	event := domain.OrderEvent{OrderID: orderID, UserID: userID, Source: domain.SourcePlatform, Status: string(domain.StatusCancelled)}
	if err := s.events.Publish(ctx, event); err != nil {
		s.log.Error("publish order event", zap.String("order_id", orderID), zap.Error(err))
	}

	return nil
}

func (s *svc) cancelRejection(ctx context.Context, orderID, userID string) error {
	owner, err := s.repo.GetOrderOwner(ctx, orderID)
	if err != nil {
		return err
	}
	if owner.UserID != userID {
		return domain.ErrNotFound
	}

	return domain.ErrStatusConflict
}

func (s *svc) MarkDelivered(ctx context.Context, orderID, placeID string) error {
	userID, delivered, err := s.repo.MarkAcceptedOrderDelivered(ctx, orderID, placeID)
	if err != nil {
		return err
	}

	if !delivered {
		status, err := s.repo.GetOrderStatusForPlace(ctx, orderID, placeID)
		if err != nil {
			return err
		}
		if status != domain.StatusCompleted {
			return domain.ErrStatusConflict
		}
		return nil
	}

	s.log.Debug("order delivered", zap.String("order_id", orderID), zap.String("place_id", placeID))

	event := domain.OrderEvent{OrderID: orderID, UserID: userID, Source: domain.SourcePlatform, Status: string(domain.StatusCompleted)}
	if err := s.events.Publish(ctx, event); err != nil {
		s.log.Error("publish order event", zap.String("order_id", orderID), zap.Error(err))
	}

	return nil
}

func (s *svc) AddTracking(ctx context.Context, in domain.AddTrackingInput) error {
	params := domain.AddTrackingParams{
		TrackingID:   uuid.NewString(),
		OrderID:      in.OrderID,
		PlaceID:      in.PlaceID,
		Source:       domain.SourcePlace,
		ExternalCode: in.ExternalCode,
		Status:       in.Status,
		OccurredAt:   in.OccurredAt,
	}

	userID, added, err := s.repo.InsertOrderTracking(ctx, params)
	if err != nil {
		return err
	}
	if !added {
		return domain.ErrNotFound
	}

	s.log.Debug("order tracking recorded",
		zap.String("order_id", in.OrderID),
		zap.String("place_id", in.PlaceID),
		zap.String("status", string(in.Status)),
		zap.String("external_code", in.ExternalCode),
	)

	event := domain.OrderEvent{
		OrderID: in.OrderID,
		UserID:  userID,
		Source:  domain.SourcePlace,
		Status:  string(in.Status),
	}
	if err := s.events.Publish(ctx, event); err != nil {
		s.log.Error("publish order event", zap.String("order_id", in.OrderID), zap.Error(err))
	}

	return nil
}

func (s *svc) GetOrder(ctx context.Context, orderID, userID string) (*domain.Order, error) {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, domain.ErrNotFound
	}

	return order, nil
}

func (s *svc) ListOrderTracking(ctx context.Context, orderID, userID string) ([]domain.TrackingEvent, error) {
	owner, err := s.repo.GetOrderOwner(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if owner.UserID != userID {
		return nil, domain.ErrNotFound
	}

	return s.repo.ListOrderTracking(ctx, orderID)
}

func (s *svc) ListUserOrders(ctx context.Context, p domain.ListUserOrdersParams) ([]domain.OrderSummary, error) {
	return s.repo.ListUserOrders(ctx, p)
}

func (s *svc) SubscribeOrderEvents(ctx context.Context, orderID, userID string) (<-chan domain.OrderEvent, error) {
	events, err := s.events.Subscribe(ctx, orderID)
	if err != nil {
		return nil, err
	}

	first, ok := <-events
	if !ok {
		return nil, domain.ErrNotFound
	}
	if first.UserID != userID {
		return nil, domain.ErrNotFound
	}

	out := make(chan domain.OrderEvent)
	go func() {
		defer close(out)

		select {
		case out <- first:
		case <-ctx.Done():
			return
		}

		for event := range events {
			select {
			case out <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

func replayIdempotent(stored, bodyHash string) (*domain.CreateOrderOutput, error) {
	var record idempotencyRecord
	if err := json.Unmarshal([]byte(stored), &record); err != nil {
		return nil, domain.ErrIdempotencyInProgress
	}
	if record.BodyHash != bodyHash {
		return nil, domain.ErrIdempotencyMismatch
	}
	if record.OrderID == "" {
		return nil, domain.ErrIdempotencyInProgress
	}

	return &domain.CreateOrderOutput{OrderID: record.OrderID, Status: record.Status}, nil
}

func (s *svc) RateOrder(ctx context.Context, in domain.RateOrderInput) error {
	params := domain.RateOrderParams{
		RatingID: uuid.NewString(),
		OrderID:  in.OrderID,
		UserID:   in.UserID,
		Score:    in.Score,
	}

	err := s.trm.Do(ctx, func(ctx context.Context) error {
		placeID, rated, err := s.ratings.InsertOrderRating(ctx, params)
		if err != nil {
			return err
		}
		if !rated {
			return s.rateRejection(ctx, in)
		}

		return s.ratings.AddPlaceRating(ctx, placeID, in.Score)
	})
	if err != nil {
		return err
	}

	s.log.Debug("order rated",
		zap.String("order_id", in.OrderID),
		zap.String("user_id", in.UserID),
		zap.Int16("score", in.Score),
	)

	return nil
}

func (s *svc) rateRejection(ctx context.Context, in domain.RateOrderInput) error {
	owner, err := s.repo.GetOrderOwner(ctx, in.OrderID)
	if err != nil {
		return err
	}
	if owner.UserID != in.UserID {
		return domain.ErrNotFound
	}
	if owner.Status != domain.StatusCompleted {
		return domain.ErrStatusConflict
	}

	return domain.ErrAlreadyRated
}
