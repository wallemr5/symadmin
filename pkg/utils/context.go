package utils

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/gofrs/uuid"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type key string

const (
	loggerKey key = "logger"
)

// SetCtxLogger get context with logger
func SetCtxLogger(ctx context.Context, logger logr.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetCtxLogger get logger with context, if not exist create
func GetCtxLogger(ctx context.Context) logr.Logger {
	logger, ok := ctx.Value(loggerKey).(logr.Logger)
	if !ok {
		logger = log.KBLog.WithName("appset-controller").WithName("controller").WithValues("id", uuid.Must(uuid.NewV4()).String())
		ctx = context.WithValue(ctx, loggerKey, logger)
	}
	return logger
}
