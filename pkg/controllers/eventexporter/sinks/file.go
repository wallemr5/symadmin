package sinks

import (
	"context"
	"encoding/json"
	"io"

	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/kube"
	"gopkg.in/natefinch/lumberjack.v2"
)

type FileConfig struct {
	Path       string                 `json:"path,omitempty",yaml:"path"`
	Layout     map[string]interface{} `json:"layout,omitempty",yaml:"layout"`
	MaxSize    int                    `json:"maxsize,omitempty",yaml:"maxsize"`
	MaxAge     int                    `json:"maxage,omitempty",yaml:"maxage"`
	MaxBackups int                    `json:"maxbackups,omitempty",yaml:"maxbackups"`
}

func (f *FileConfig) Validate() error {
	return nil
}

type File struct {
	writer  io.WriteCloser
	encoder *json.Encoder
	layout  map[string]interface{}
}

func NewFileSink(config *FileConfig) (*File, error) {
	writer := &lumberjack.Logger{
		Filename:   config.Path,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
	}

	return &File{
		writer:  writer,
		encoder: json.NewEncoder(writer),
		layout:  config.Layout,
	}, nil
}

func (f *File) Close() {
	_ = f.writer.Close()
}

func (f *File) Send(ctx context.Context, ev *kube.EnhancedEvent) error {
	if f.layout == nil {
		return f.encoder.Encode(ev)
	}

	res, err := convertLayoutTemplate(f.layout, ev)
	if err != nil {
		return err
	}

	return f.encoder.Encode(res)
}
