package cache

import (
	"container/list"
)

// Value 使用Len来计算它需要多少字节
type Value interface {
	Len() int
}

type entry struct {
	key   string
	value Value
}

type LRU struct {
	maxElement int64
	nElement   int64
	ls         *list.List
	data       map[string]*list.Element
	OnEvicted  func(key string, value Value)
}

func NewLRU(maxElement int64, onEvicted func(string, Value)) *LRU {
	return &LRU{
		maxElement: maxElement,
		ls:         list.New(),
		data:       make(map[string]*list.Element),
		OnEvicted:  onEvicted,
	}
}

//Add
func (c *LRU) Add(key string, value Value) {
	if ele, ok := c.data[key]; ok {
		c.ls.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nElement += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ls.PushFront(&entry{key, value})
		c.data[key] = ele
		c.nElement += int64(len(key)) + int64(value.Len())
	}
	for c.maxElement != 0 && c.maxElement < c.nElement {
		c.RemoveOldest()
	}
}

// Get
func (c *LRU) Get(key string) (value Value, ok bool) {
	if ele, ok := c.data[key]; ok {
		c.ls.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

func (c *LRU) RemoveOldest() {
	ele := c.ls.Back()
	if ele != nil {
		c.ls.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.data, kv.key)
		c.nElement -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len
func (c *LRU) Len() int {
	return c.ls.Len()
}
