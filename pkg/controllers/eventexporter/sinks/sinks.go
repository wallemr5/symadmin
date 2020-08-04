package sinks

import (
	"context"

	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/kube"
)

type Sink interface {
	Send(ctx context.Context, ev *kube.EnhancedEvent) error
	Close()
}
