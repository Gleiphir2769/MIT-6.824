package timewheel

import (
	"6.824/raft/lib/sync/atomic"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDelay(t *testing.T) {
	ch := make(chan time.Time)
	beginTime := time.Now()
	Delay(time.Second, "", func() {
		ch <- time.Now()
	})
	execAt := <-ch
	delayDuration := execAt.Sub(beginTime)
	// usually 1.0~2.0 s
	if delayDuration < time.Second || delayDuration > 2*time.Second {
		t.Error("wrong execute time")
	}
	fmt.Println("real execute time: ", delayDuration)
}

func TestRefresh(t *testing.T) {
	refreshed := atomic.Boolean(0)
	mu := sync.Mutex{}
	now := time.Now()
	at := now.Add(time.Millisecond * 400)
	At(at, "test", func() {
		mu.Lock()
		defer mu.Unlock()
		if refreshed.Get() {
			fmt.Println("first time mission has been interrupt")
			return
		}
		fmt.Println("expire:", time.Since(now))
	})
	time.Sleep(time.Millisecond * 400)
	now = time.Now()
	at = now.Add(time.Millisecond * 400)
	RefreshAt(at, "test", func() {
		mu.Lock()
		defer mu.Unlock()
		refreshed.Set(true)
		fmt.Println("refresh expire:", time.Since(now))
	})

	time.Sleep(time.Second)
}
