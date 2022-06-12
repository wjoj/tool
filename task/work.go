package task

import (
	"sync"
)

func DoTraverse(lng int, sF func(idx int)) {
	for i := 0; i < lng; i++ {
		sF(i)
	}
}

func DoContrary(lng int, sF func(idx int)) {
	for i := lng; i != 0; i-- {
		sF(i)
	}
}

func DoTraverseAsyncWait(lng int, sF func(idx int)) {
	var w sync.WaitGroup
	w.Add(lng)
	for i := 0; i < lng; i++ {
		go func(idx int) {
			sF(idx)
			w.Done()
		}(i)
	}
	w.Wait()
}

func DoTraverseAsyncQueueWait(lng int, qNumber int, sF func(idx int) any, outF func(idx int, out any)) {
	ls := make(map[int]any, lng)
	var lock sync.RWMutex
	lsAdd := func(idx int, in any) {
		lock.Lock()
		defer lock.Unlock()
		ls[idx] = in
	}
	lsGet := func(idx int) (any, bool) {
		lock.RLock()
		defer lock.RUnlock()
		d, is := ls[idx]
		return d, is
	}
	revCount := 0
	send := func() {
		for {
			val, is := lsGet(revCount)
			if !is {
				break
			}
			outF(revCount, val)
			revCount++
		}
	}
	queue := make(chan struct{})
	go func() {
		i := 0
		sum := qNumber
		var que sync.WaitGroup
		for {
			if sum > lng {
				sum = lng
			}
			if i >= lng {
				break
			}
			que.Add(sum - i)
			for ; i < sum; i++ {
				go func(idx int) {
					lsAdd(idx, sF(idx))
					queue <- struct{}{}
					que.Done()
				}(i)
			}
			que.Wait()
			sum += qNumber
		}
	}()
	for i := 0; i < lng; i++ {
		<-queue
		send()
	}
}

func pcall(fn func()) {
	defer func() {
		if err := recover(); err != nil {

		}
	}()
	fn()
}

func AsyncDone(fn func()) {
	go pcall(fn)
}
