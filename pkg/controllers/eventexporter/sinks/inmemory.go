package sinks

import (
	"context"

	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/kube"
)

type InMemoryConfig struct {
	Ref *InMemory
}

type InMemory struct {
	Events []*kube.EnhancedEvent
	Config *InMemoryConfig
}

func (i *InMemory) Send(ctx context.Context, ev *kube.EnhancedEvent) error {
	i.Events = append(i.Events, ev)
	return nil
}

func (i *InMemory) Close() {
	// No-op
}
