package utils

import (
	"time"
)

type Request interface {
	Merge(Request) Request
}

type Debounce struct {
	ch          chan Request
	waitTime    time.Duration
	maxWaitTime time.Duration
	pushFn      func(req Request)
}

func NewDebounce(waitTime, maxWaitTime time.Duration, pushFn func(req Request)) *Debounce {
	d := &Debounce{
		ch:          make(chan Request, 0),
		waitTime:    waitTime,
		maxWaitTime: maxWaitTime,
		pushFn:      pushFn,
	}
	go d.start()
	return d
}

func (d *Debounce) start() {
	var timeChan <-chan time.Time
	var startTime time.Time
	var lastUpdateTime time.Time

	debouncedEvents := 0

	var req Request

	free := true
	freeCh := make(chan struct{}, 1)

	push := func(req Request) {
		d.pushFn(req)
		freeCh <- struct{}{}
	}

	pushWorker := func() {
		eventDelay := time.Since(startTime)
		quitTime := time.Since(lastUpdateTime)
		if quitTime >= d.waitTime || eventDelay >= d.maxWaitTime {
			if req != nil {
				free = false
				go push(req)
				req = nil
				debouncedEvents = 0
			}
		} else {
			timeChan = time.After(d.waitTime - quitTime)
		}
	}

	for {
		select {
		case <-freeCh:
			free = true
			pushWorker()

		case r, ok := <-d.ch:
			if !ok {
				return
			}

			lastUpdateTime = time.Now()
			if debouncedEvents == 0 {
				timeChan = time.After(d.waitTime)
				startTime = lastUpdateTime
			}
			debouncedEvents++

			if req == nil {
				req = r
				continue
			}
			req = req.Merge(r)

		case <-timeChan:
			if free {
				pushWorker()
			}
		}
	}
}

func (d *Debounce) Put(req Request) {
	d.ch <- req
}

func (d *Debounce) Close() {
	close(d.ch)
}
