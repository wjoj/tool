package util

import "sync/atomic"

type AtomicBool uint32

func NewAtomicBool() *AtomicBool {
	return new(AtomicBool)
}

func ForAtomicBool(val bool) *AtomicBool {
	ab := NewAtomicBool()
	ab.Set(val)
	return ab
}

func (a *AtomicBool) CompareAndSwap(old, new bool) bool {
	var oldV, newV uint32
	if old {
		oldV = 1
	}
	if new {
		newV = 1
	}

	return atomic.CompareAndSwapUint32((*uint32)(a), oldV, newV)
}

func (a *AtomicBool) Set(new bool) {
	if new {
		atomic.StoreUint32((*uint32)(a), 1)
	} else {
		atomic.StoreUint32((*uint32)(a), 0)
	}
}

func (a *AtomicBool) True() bool {
	return atomic.LoadUint32((*uint32)(a)) == 1
}
