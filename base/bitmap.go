package base

type BitMap struct {
	data []uint8
	size uint64
}

func NewBitMap(n uint64) *BitMap {
	return &BitMap{
		data: make([]uint8, n/8+1),
		size: n,
	}
}

func (b *BitMap) Set(pos uint64) bool {
	if pos > b.size {
		return false
	}
	b.data[pos/8] |= 1 << (pos % 8)
	return !false
}

func (b *BitMap) UnSet(pos uint64) bool {
	if pos > b.size {
		return false
	}
	b.data[pos/8] &= ^(1 << (pos % 8))
	return !false
}

func (b *BitMap) IsSet(pos uint64) bool {
	if pos > b.size {
		return false
	}
	if p := b.data[pos/8] & (1 << (pos % 8)); p != 0 {
		return true
	}
	return false
}

func (b *BitMap) Sort() (ret []uint64) {
	ret = make([]uint64, 0)
	for i := uint64(0); i < b.size; i++ {
		if b.IsSet(i) {
			ret = append(ret, i)
		}
	}
	return
}
