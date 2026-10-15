package decisions

import (
	"context"
	"errors"
	"time"

	"avito-kitchen/services/kitchen/internal/orders/domain"

	"go.uber.org/zap"
)

func (p *Processor) requestDecision(order domain.PendingOrder) {
	ctx, cancel := context.WithDeadline(p.ctx, order.CreatedAt.Add(webhookBudget))
	defer cancel()

	full, err := p.repo.GetOrder(ctx, order.ID)
	if err != nil {
		p.log.Error("load order for webhook", zap.String("order_id", order.ID), zap.Error(err))
		return
	}

	delay := initialRetryDelay
	for {
		if decision, ok := p.webhook.CallOnce(ctx, order.WebhookURL, *full); ok {
			p.applyDecision(ctx, order, decision)
			return
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				p.applyDecision(ctx, order, domain.Decision{Status: domain.StatusExpired})
			}
			return
		}

		if !p.stillPending(ctx, order.ID) {
			return
		}

		if delay < maxRetryDelay {
			delay *= 2
		}
	}
}

func (p *Processor) stillPending(ctx context.Context, orderID string) bool {
	owner, err := p.repo.GetOrderOwner(ctx, orderID)
	if err != nil {
		if ctx.Err() == nil {
			p.log.Warn("check order status before calling the place again",
				zap.String("order_id", orderID), zap.Error(err))
		}
		return true
	}
	if owner.Status == domain.StatusPending {
		return true
	}

	p.log.Debug("order is no longer pending, stopped calling the place",
		zap.String("order_id", orderID), zap.String("status", string(owner.Status)))
	return false
}

func (p *Processor) applyDecision(ctx context.Context, order domain.PendingOrder, decision domain.Decision) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), applyTimeout)
	defer cancel()

	var changed bool
	err := p.trm.Do(ctx, func(ctx context.Context) error {
		var err error
		changed, err = p.repo.UpdatePendingOrderStatus(ctx, order.ID, decision.Status)
		if err != nil || !changed {
			return err
		}

		if decision.Status == domain.StatusAccepted {
			return nil
		}

		return p.releaseStock(ctx, order.ID, order.PlaceID, decision)
	})
	if err != nil {
		p.log.Error("apply decision", zap.String("order_id", order.ID), zap.Error(err))
		return
	}
	if !changed {
		p.log.Info("decision ignored, order is already final",
			zap.String("order_id", order.ID), zap.String("decision", string(decision.Status)))
		return
	}

	p.log.Info("order decision applied",
		zap.String("order_id", order.ID),
		zap.String("place_id", order.PlaceID),
		zap.String("status", string(decision.Status)),
		zap.String("reason", decision.Reason),
	)

	event := domain.OrderEvent{OrderID: order.ID, UserID: order.UserID, Source: domain.SourcePlatform, Status: string(decision.Status)}
	if err := p.events.Publish(ctx, event); err != nil {
		p.log.Error("publish order event", zap.String("order_id", order.ID), zap.Error(err))
	}
}

func (p *Processor) releaseStock(ctx context.Context, orderID, placeID string, decision domain.Decision) error {
	var unavailable []string
	if decision.Status == domain.StatusRejected && decision.Reason == domain.ReasonOutOfStock {
		unavailable = decision.UnavailableExternalIDs
	}

	if err := p.repo.ReturnStock(ctx, orderID, unavailable); err != nil {
		return err
	}

	zeroed, err := p.repo.ZeroStock(ctx, orderID, unavailable)
	if err != nil {
		return err
	}

	if foreign := notIn(unavailable, zeroed); len(foreign) > 0 {
		p.log.Warn("place reported unavailable external ids outside of the order",
			zap.String("order_id", orderID),
			zap.String("place_id", placeID),
			zap.Strings("external_ids", foreign),
		)
	}

	return nil
}

func notIn(values, known []string) []string {
	index := make(map[string]struct{}, len(known))
	for _, value := range known {
		index[value] = struct{}{}
	}

	rest := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := index[value]; !exists {
			rest = append(rest, value)
		}
	}
	return rest
}
