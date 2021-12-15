package atomic

import "sync/atomic"

type Int32 int32

// Get reads the value atomically
func (i32 *Int32) Get() int32 {
	return atomic.LoadInt32((*int32)(i32))
}

// Set writes the value atomically
func (i32 *Int32) Set(v int32) {
	atomic.StoreInt32((*int32)(i32), v)
}

func (i32 *Int32) Incr() int32 {
	return atomic.AddInt32((*int32)(i32), 1)
}

func (i32 *Int32) Add(num int32) int32 {
	return atomic.AddInt32((*int32)(i32), num)
}

func (i32 *Int32) Dec(num int32) int32 {
	return atomic.AddInt32((*int32)(i32), -num)
}
