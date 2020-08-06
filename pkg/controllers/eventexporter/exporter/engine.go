package exporter

import (
	"reflect"

	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/sinks"
	"k8s.io/klog"
)

// Config allows configuration
type Config struct {
	Namespace string                 `json:"namespace,omitempty" yaml:"namespace"`
	Route     Route                  `json:"route,omitempty" yaml:"route"`
	Receivers []sinks.ReceiverConfig `json:"receivers,omitempty" yaml:"receivers"`
}

// Engine is responsible for initializing the receivers from sinks
type Engine struct {
	Route    Route
	Registry ReceiverRegistry
}

func NewEngine(config *Config) *Engine {
	registry := &ChannelBasedReceiverRegistry{}
	for _, v := range config.Receivers {
		sink, err := v.GetSink()
		if err != nil {
			klog.Fatalf("Cannot initialize sink name: %s", v.Name)
		}

		klog.Infof("name: %s type: %s Registering sink", v.Name, reflect.TypeOf(sink).String())
		registry.Register(v.Name, sink)
	}

	return &Engine{
		Route:    config.Route,
		Registry: registry,
	}
}
