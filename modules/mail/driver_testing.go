package mail

import (
	"context"
	"sync"
)

type TestDriver struct {
	mu   sync.Mutex
	sent []*Message
}

func NewTestDriver() *TestDriver {
	return &TestDriver{
		sent: make([]*Message, 0),
	}
}

func (d *TestDriver) Send(ctx context.Context, msg *Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sent = append(d.sent, msg)
	return nil
}

func (d *TestDriver) Sent() []*Message {
	d.mu.Lock()
	defer d.mu.Unlock()
	res := make([]*Message, len(d.sent))
	copy(res, d.sent)
	return res
}

func (d *TestDriver) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sent = make([]*Message, 0)
}
