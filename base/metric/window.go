package metric

// Bucket contains multiple float64 points.
type Bucket struct {
	Points []float64 // all of the points
	Count  int64     // this bucket point length
	next   *Bucket
}

// Append append a point to bucket points
func (b *Bucket) Append(point float64) {
	b.Points = append(b.Points, point)
	b.Count++
}

// Add adds the given value to the point.
func (b *Bucket) Add(offset int, delta float64) {
	b.Points[offset] += delta
	b.Count++
}

// Reset empties the bucket.
func (b *Bucket) Reset() {
	b.Points = b.Points[:0]
	b.Count = 0
}

// Next returns the next bucket.
func (b *Bucket) Next() *Bucket {
	return b.next
}

type (
	// Window contains multiple buckets.
	Window struct {
		buckets []Bucket
		size    int
	}

	// WindowOpts contains the arguments for creating Window.
	WindowOpts struct {
		size int
	}

	WithWindowOpt func(*WindowOpts)
)

// WithWindowSize return a WithWindowOpt with window size
func WithWindowSize(size int) WithWindowOpt {
	return func(opts *WindowOpts) {
		opts.size = size
	}
}

func NewWindow(fns ...WithWindowOpt) *Window {

	windowOpt := &WindowOpts{}

	for _, fn := range fns {
		fn(windowOpt)
	}

	buckets := make([]Bucket, windowOpt.size)

	// create a window with ring buckets
	for offset := range buckets {
		buckets[offset] = Bucket{
			Points: make([]float64, 0),
			Count:  0,
		}
		nextOffset := offset + 1
		if nextOffset == windowOpt.size {
			nextOffset = 0
		}
		buckets[offset].next = &buckets[nextOffset]
	}
	return &Window{
		buckets: buckets,
		size:    windowOpt.size,
	}

}

// Reset reset all of bucket in window
func (w *Window) Reset() {
	for i := range w.buckets {
		w.buckets[i].Reset()
	}
}

// ResetBucket reset one bucket from given offset
func (w *Window) ResetBucket(offset int) {
	w.buckets[offset].Reset()
}

// ResetBuckets reset one bucket from given offsets
func (w *Window) ResetBuckets(offsets []int) {
	for _, offset := range offsets {
		w.buckets[offset].Reset()
	}
}

// AppendBucketPoint appends the given value to
// the bucket points where index equals the given offset.
func (w *Window) AppendBucketPoint(offset int, point float64) {
	w.buckets[offset].Append(point)
}

// AddBucketPoint adds the given value to the latest point within bucket where index equals the given offset.
func (w *Window) AddBucketPoint(offset int, val float64) {
	if w.buckets[offset].Count == 0 {
		w.buckets[offset].Append(val)
		return
	}
	w.buckets[offset].Add(0, val)
}

// Bucket returns the bucket where index equals the given offset.
func (w *Window) Bucket(offset int) Bucket {
	return w.buckets[offset]
}

// Size returns the size of the window.
func (w *Window) Size() int {
	return w.size
}

func (w *Window) Iterator(offset int, count int) Iterator {
	return Iterator{
		count: count,
		cur:   &w.buckets[offset],
	}
}
