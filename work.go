package tool

import "sync"

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

func DoTraverseSyncWait(lng int, sF func(idx int) func(d any)) {
	var w sync.WaitGroup
	for i := 0; i < lng; i++ {
		w.Add(1)
		go func(idx int) {
			sF(idx)
			w.Done()
		}(i)
	}
	w.Wait()
}
