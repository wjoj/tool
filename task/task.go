package task

import "sync"

func TaskLimitItemsCoroutine[T any](limit int, itemsFunc func(*bool, func([]T)), goFunc func(T)) func() {
	quque := make(chan struct{}, limit)
	isClose := false
	itemsCh := make(chan []T)
	var wg sync.WaitGroup
	closeF := func() {
		isClose = true
		close(itemsCh)
		wg.Wait()
		close(quque)
	}

	var items []T
	var is bool
	for {
		if isClose {
			break
		}

		go func() {
			itemsFunc(&isClose, func(t []T) {
				if isClose {
					return
				}
				itemsCh <- t
			})
		}()
		items, is = <-itemsCh
		if !is {
			break
		}
		for i := range items {
			if isClose {
				break
			}
			wg.Add(1)
			quque <- struct{}{}
			go func(item T) {
				goFunc(item)
				<-quque
				wg.Done()
			}(items[i])
		}
	}
	return closeF
}
