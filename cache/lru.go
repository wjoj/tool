package cache

import "container/list"

type (
	Lru interface {
		Add(key string)
		Remove(key string)
	}

	noneLru struct{}

	keyLru struct {
		limit    int
		evicts   *list.List
		elements map[string]*list.Element
		onEvict  func(key string)
	}
)

var (
	_ Lru = &noneLru{}
	_ Lru = &keyLru{}
)

// NewNoneLru return an empty lru implement, do not manager keys.
// when cache have a limit of count, use this to make the flow correct
func NewNoneLru() Lru {
	return &noneLru{}
}

func (l *noneLru) Add(key string) {}

func (l *noneLru) Remove(key string) {}

// NewLru return a Lru entry with least-recently-use algorithm
func NewLru(limit int, onEvict func(key string)) Lru {
	return &keyLru{
		limit:    limit,
		evicts:   list.New(),
		elements: make(map[string]*list.Element),
		onEvict:  onEvict,
	}
}

func (l *keyLru) Remove(key string) {
	if elem, ok := l.elements[key]; ok {
		l.removeElem(elem)
	}
}

func (l *keyLru) Add(key string) {
	if v, ok := l.elements[key]; ok {
		// 元素存在, 移至队首
		l.evicts.MoveToFront(v)
		return
	}

	// 新增元素
	elem := l.evicts.PushFront(key)
	l.elements[key] = elem

	// 超出列表长度, 移除队尾元素
	if l.evicts.Len() > l.limit {
		l.removeOldest()
	}
}

func (l *keyLru) removeOldest() {
	elem := l.evicts.Back()
	l.removeElem(elem)
}

func (l *keyLru) removeElem(e *list.Element) {
	if e == nil {
		return
	}
	l.evicts.Remove(e)
	key := e.Value.(string)
	delete(l.elements, key)
	l.onEvict(key)
}
