package metric

import "fmt"

// Iterator iterates the buckets within the window.
type Iterator struct {
	count         int // total count
	iteratedCount int // have been iterated
	cur           *Bucket
}

// Next returns true util all of the buckets has been iterated.
func (i *Iterator) Next() bool {
	return i.count != i.iteratedCount
}

func (i *Iterator) Bucket() *Bucket {
	if !(i.Next()) {
		panic(fmt.Errorf("stat/metric: iteration out of range iteratedCount: %d count: %d", i.iteratedCount, i.count))
	}

	bucket := i.cur
	i.iteratedCount++
	i.cur = i.cur.Next()
	return bucket
}
