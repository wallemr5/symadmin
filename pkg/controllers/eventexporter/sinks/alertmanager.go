package sinks

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"

	"strconv"

	json "github.com/json-iterator/go"
	"github.com/prometheus/common/model"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/kube"
	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"k8s.io/klog"
)

type AlertManagerConfig struct {
	Endpoint string            `json:"endpoint,omitempty",yaml:"endpoint"`
	Headers  map[string]string `json:"headers,omitempty",yaml:"headers"`
}

func NewAlertManager(cfg *AlertManagerConfig) (Sink, error) {
	return &AlertManager{cfg: cfg}, nil
}

type AlertManager struct {
	cfg *AlertManagerConfig
}

func (w *AlertManager) Close() {
	// No-op
}

func (w *AlertManager) Send(ctx context.Context, ev *kube.EnhancedEvent) error {
	data := map[string]string{}
	data["job"] = "event-alert"
	data["service"] = "event-exporter"
	data["severity"] = "warning"
	data["cluster"] = ev.ClusterName
	data["kind"] = ev.Event.InvolvedObject.Kind
	data["reason"] = ev.Event.Reason
	data["name"] = ev.Event.InvolvedObject.Name
	data["namespace"] = ev.Event.InvolvedObject.Namespace
	data["count"] = strconv.Itoa(int(ev.Event.Count))
	data["message"] = ev.Event.Message
	data["component"] = ev.Event.Source.Component
	data["host"] = ev.Event.Source.Host

	if app, ok := ev.InvolvedObject.Labels[pkgLabels.ObserveMustLabelAppName]; ok {
		data["app"] = app
	}

	if group, ok := ev.InvolvedObject.Labels[pkgLabels.ObserveMustLabelGroupName]; ok {
		data["group"] = group
	}

	alertList := model.Alerts{}
	a := &model.Alert{}
	a.Labels = map[model.LabelName]model.LabelValue{}
	for k, v := range data {
		a.Labels[model.LabelName(k)] = model.LabelValue(v)
	}
	a.Annotations = map[model.LabelName]model.LabelValue{}
	a.Annotations[model.LabelName("description")] = model.LabelValue("k8s event alert")
	a.Annotations[model.LabelName("summary")] = model.LabelValue("Prometheus is failing rule evaluations.")

	alertList = append(alertList, a)
	reqBody, err := json.Marshal(alertList)
	if err != nil {
		return err
	}

	url := w.cfg.Endpoint + "/api/v1/alerts"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	for k, v := range w.cfg.Headers {
		req.Header.Add(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		klog.Errorf("not 200/201 body: %s", string(body))
		return errors.New("not 200/201: " + string(body))
	}

	return nil
}
