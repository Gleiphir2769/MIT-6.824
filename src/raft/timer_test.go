package raft

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vt, voteChan := MakeSimpleVoteTimer()
	go vt.Start(ctx)
	now := time.Now()
	go func() {
		time.Sleep(time.Millisecond * 200)
		vt.Refresh()
	}()
	<-voteChan
	dur := time.Since(now)
	fmt.Println("timer time is", dur)
}

func TestRand(t *testing.T) {
	fmt.Println(0 % 5)
}
