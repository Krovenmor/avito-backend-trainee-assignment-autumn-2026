package decisions

import (
	"context"
	"errors"
	"sync"
	"time"

	"avito-kitchen/services/kitchen/internal/orders/domain"

	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"go.uber.org/zap"
)

const (
	reconcileInterval  = 30 * time.Second
	reconcileBatchSize = 100
	orderLeaseTTL      = 90 * time.Second

	webhookBudget     = 5 * time.Minute
	initialRetryDelay = 2 * time.Second
	maxRetryDelay     = time.Minute
	applyTimeout      = 10 * time.Second

	notifyBuffer = 256
)

type Processor struct {
	repo     Repository
	trm      trm.Manager
	webhook  WebhookClient
	events   EventPublisher
	log      *zap.Logger
	instance domain.InstanceID

	notified chan domain.PendingOrder

	ctx     context.Context
	stop    context.CancelFunc
	running sync.WaitGroup

	mu     sync.Mutex
	active map[string]struct{}
}

func New(
	repo Repository,
	trManager trm.Manager,
	webhook WebhookClient,
	events EventPublisher,
	instance domain.InstanceID,
	log *zap.Logger,
) *Processor {
	ctx, stop := context.WithCancel(context.Background())

	return &Processor{
		repo:     repo,
		trm:      trManager,
		webhook:  webhook,
		events:   events,
		log:      log,
		instance: instance,
		notified: make(chan domain.PendingOrder, notifyBuffer),
		ctx:      ctx,
		stop:     stop,
		active:   make(map[string]struct{}),
	}
}

func (p *Processor) Notify(order domain.PendingOrder) {
	select {
	case p.notified <- order:
	default:
		p.log.Warn("decision notification dropped, reconciler will pick the order up",
			zap.String("order_id", order.ID))
	}
}

func (p *Processor) Start() {
	p.running.Add(1)
	go func() {
		defer p.running.Done()
		p.run()
	}()
}

func (p *Processor) run() {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	p.reconcile(p.ctx)

	for {
		select {
		case order := <-p.notified:
			p.startWorker(order)
		case <-ticker.C:
			p.reconcile(p.ctx)
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Processor) Stop(ctx context.Context) error {
	p.stop()

	drained := make(chan struct{})
	go func() {
		defer close(drained)
		p.running.Wait()
	}()

	select {
	case <-drained:
		p.log.Info("decision workers drained")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Processor) reconcile(ctx context.Context) {
	active := p.activeIDs()
	if err := p.repo.RenewOrderLeases(ctx, string(p.instance), active); err != nil {
		p.logReconcileError("renew order leases", err)
		return
	}
	if len(active) > 0 {
		p.log.Debug("renewed order leases", zap.Int("count", len(active)))
	}

	var orders []domain.PendingOrder
	err := p.trm.Do(ctx, func(ctx context.Context) error {
		var err error
		orders, err = p.repo.ClaimPendingOrders(ctx, domain.ClaimPendingOrdersParams{
			Owner:    string(p.instance),
			LeaseTTL: orderLeaseTTL,
			Limit:    reconcileBatchSize,
			Active:   active,
		})
		return err
	})
	if err != nil {
		p.logReconcileError("claim pending orders", err)
		return
	}

	if len(orders) > 0 {
		p.log.Info("claimed pending orders", zap.Int("count", len(orders)))
	}

	for _, order := range orders {
		if time.Since(order.CreatedAt) >= webhookBudget {
			p.applyDecision(ctx, order, domain.Decision{Status: domain.StatusExpired})
			continue
		}
		p.startWorker(order)
	}
}

func (p *Processor) logReconcileError(msg string, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	p.log.Error(msg, zap.Error(err))
}

func (p *Processor) startWorker(order domain.PendingOrder) {
	if !p.markActive(order.ID) {
		return
	}

	p.log.Debug("decision worker started",
		zap.String("order_id", order.ID),
		zap.String("webhook_url", order.WebhookURL),
	)

	p.running.Add(1)
	go func() {
		defer p.running.Done()
		defer p.markDone(order.ID)
		p.requestDecision(order)
	}()
}

func (p *Processor) markActive(orderID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.active[orderID]; exists {
		return false
	}
	p.active[orderID] = struct{}{}
	return true
}

func (p *Processor) markDone(orderID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.active, orderID)
}

func (p *Processor) activeIDs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	ids := make([]string, 0, len(p.active))
	for orderID := range p.active {
		ids = append(ids, orderID)
	}
	return ids
}
