package utils

import (
	"fmt"
	"testing"
	"time"
)

type R struct {
	t time.Time
	i int
}

func (r *R) Merge(req Request) Request {
	return req
}

func TestNew(t *testing.T) {
	d := NewDebounce(time.Second*3, time.Second*10, func(req Request) {
		fmt.Println(time.Now())
		r := req.(*R)
		fmt.Println(r.t, r.i)
	})

	for i := 0; i < 15; i++ {
		d.Put(&R{
			t: time.Now(),
			i: i,
		})
		time.Sleep(time.Millisecond * 900)
	}

	time.Sleep(20 * time.Second)
}
