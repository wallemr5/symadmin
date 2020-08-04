package sinks

import (
	"errors"
)

// Receiver allows receiving
type ReceiverConfig struct {
	Name         string              `json:"name,omitempty",yaml:"name"`
	InMemory     *InMemoryConfig     `json:"inMemory,omitempty",yaml:"inMemory"`
	File         *FileConfig         `json:"file,omitempty",yaml:"file"`
	Webhook      *WebhookConfig      `json:"webhook,omitempty",yaml:"webhook"`
	AlertManager *AlertManagerConfig `json:"alertManager,omitempty",yaml:"alertManager"`
}

func (r *ReceiverConfig) Validate() error {
	return nil
}

func (r *ReceiverConfig) GetSink() (Sink, error) {
	if r.InMemory != nil {
		// This reference is used for test purposes to count the events in the sink.
		// It should not be used in production since it will only cause memory leak and (b)OOM
		sink := &InMemory{Config: r.InMemory}
		r.InMemory.Ref = sink
		return sink, nil
	}

	// Sorry for this code, but its Go
	if r.Webhook != nil {
		return NewWebhook(r.Webhook)
	}

	if r.File != nil {
		return NewFileSink(r.File)
	}

	if r.AlertManager != nil {
		return NewAlertManager(r.AlertManager)
	}

	return nil, errors.New("unknown sink")
}
