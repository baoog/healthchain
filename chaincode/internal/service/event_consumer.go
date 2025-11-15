package service

import (
	"context"
	"log"

	"consentcc/internal/infra/client/fabric"
)

type EventConsumer struct {
	fc      fabric.FabricClient
	service *HealthChainService
}

type IEventConsumer interface {
	Start(ctx context.Context)
}

func NewEventConsumer(fc fabric.FabricClient, s *HealthChainService) *EventConsumer {
	return &EventConsumer{fc: fc, service: s}
}

func (c *EventConsumer) Start(ctx context.Context) {
	evCh, cancel, err := c.fc.ContractEvents(ctx)
	if err != nil {
		log.Fatalf("contract events: %v", err)
	}
	go func() {
		defer cancel()
		for {
			select {
			case ev, ok := <-evCh:
				if !ok || ctx.Err() != nil {
					return
				}
				// ev.EventName is your chaincode event; payload is []byte
				if err := c.service.Handle(ctx, ev.Payload); err != nil {
					log.Printf("handle event: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
