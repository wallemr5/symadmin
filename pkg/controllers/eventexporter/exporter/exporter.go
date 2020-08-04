package exporter

import (
	"context"

	"sync"

	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/kube"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/sinks"
	"k8s.io/klog"
)

// ReceiverRegistry registers a receiver with the appropriate sink
type ReceiverRegistry interface {
	SendEvent(string, *kube.EnhancedEvent)
	Register(string, sinks.Sink)
	Close()
}

type ChannelBasedReceiverRegistry struct {
	ch     map[string]chan kube.EnhancedEvent
	exitCh map[string]chan interface{}
	wg     *sync.WaitGroup
}

func (r *ChannelBasedReceiverRegistry) SendEvent(name string, event *kube.EnhancedEvent) {
	ch := r.ch[name]
	if ch == nil {
		klog.Errorf("There is no channel name: %s ", name)
		return
	}

	go func() {
		ch <- *event
	}()
}

func (r *ChannelBasedReceiverRegistry) Register(name string, receiver sinks.Sink) {
	if r.ch == nil {
		r.ch = make(map[string]chan kube.EnhancedEvent)
		r.exitCh = make(map[string]chan interface{})
	}

	ch := make(chan kube.EnhancedEvent)
	exitCh := make(chan interface{})

	r.ch[name] = ch
	r.exitCh[name] = exitCh

	if r.wg == nil {
		r.wg = &sync.WaitGroup{}
	}
	r.wg.Add(1)

	go func() {
	Loop:
		for {
			select {
			case ev := <-ch:
				klog.Infof("sending event to sink: %s event: %s", name, ev.Message)
				err := receiver.Send(context.Background(), &ev)
				if err != nil {
					klog.Errorf("Cannot send event sink: %s event: %s err: %+v", name, ev.Message, err)
				}
			case <-exitCh:
				klog.Infof("Closing the sink: %s ", name)
				break Loop
			}
		}
		receiver.Close()
		klog.Infof("Closed the sink: %s ", name)
		r.wg.Done()
	}()
}

// Close signals closing to all sinks and waits for them to complete.
// The wait could block indefinitely depending on the sink implementations.
func (r *ChannelBasedReceiverRegistry) Close() {
	// Send exit command and wait for exit of all sinks
	for _, ec := range r.exitCh {
		ec <- 1
	}
	r.wg.Wait()
}
