package raft

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"
)

const (
	defaultVoteDur            = time.Millisecond * 600
	defaultVotePrecision      = 6
	defaultHeartbeatDur       = time.Millisecond * 100
	defaultHeartbeatPrecision = 100
)

type Timer struct {
	mu sync.Mutex
	t  time.Duration
	r  bool
	// the precision of timer, advising value is 100 when t = 100ms
	precision int
	alarm     chan struct{}
}

func MakeSimpleVoteTimer() (*Timer, chan struct{}) {
	return MakeTimer(defaultVoteDur, defaultVotePrecision)
}

func MakeSimpleHeartbeatTimer() (*Timer, chan struct{}) {
	return MakeTimer(defaultHeartbeatDur, defaultHeartbeatPrecision)
}

func MakeTimer(t time.Duration, precision int) (*Timer, chan struct{}) {
	alarm := make(chan struct{})
	return &Timer{t: t, precision: precision, alarm: alarm}, alarm
}

func (vt *Timer) Start(ctx context.Context) {
	timeSlice := int(float32(vt.t) / float32(vt.precision))
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, _ := rand.Int(rand.Reader, big.NewInt(2))
			for i := int(n.Int64()); i < vt.precision; i++ {
				vt.mu.Lock()
				if vt.r {
					vt.r = false
					n, _ := rand.Int(rand.Reader, big.NewInt(2))
					i = int(n.Int64())
				}
				vt.mu.Unlock()
				time.Sleep(time.Duration(timeSlice))
			}
			vt.alarm <- struct{}{}
		}
	}
}

func (vt *Timer) Refresh() {
	vt.mu.Lock()
	vt.r = true
	vt.mu.Unlock()
}
