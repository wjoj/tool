package filter

import (
	"bytes"
	"hash"
	"hash/fnv"
	"math/rand"
)

type Cuckoo struct {
	hashfn  hash.Hash
	buckets []bucket
	count   uint

	bSize  uint8
	fpSize uint8
	size   uint
	kicks  uint
}

func NewCuckoo(opts ...option) *Cuckoo {
	ck := new(Cuckoo)
	for _, opt := range opts {
		opt(ck)
	}
	configure(ck)
	ck.buckets = make([]bucket, ck.size)
	for i := range ck.buckets {
		ck.buckets[i] = make([]fingerprint, ck.bSize)
	}
	return ck
}

func (ck *Cuckoo) Insert(item []byte) bool {
	f := fprint(item, ck.fpSize, ck.hashfn)
	j := hashfp(item) % ck.size
	k := (j ^ hashfp(f)) % ck.size

	if ck.buckets[j].insert(f) || ck.buckets[k].insert(f) {
		ck.count++
		return true
	}

	i := [2]uint{j, k}[rand.Intn(2)]
	for n := uint(0); n < ck.kicks; n++ {
		f = ck.buckets[i].swap(f)
		i = (i ^ hashfp(f)) % ck.size

		if ck.buckets[i].insert(f) {
			ck.count++
			return true
		}
	}

	return false
}

func (ck *Cuckoo) Lookup(item []byte) bool {
	f := fprint(item, ck.fpSize, ck.hashfn)
	j := hashfp(item) % ck.size
	k := (j ^ hashfp(f)) % ck.size

	return ck.buckets[j].lookup(f) || ck.buckets[k].lookup(f)
}

func (ck *Cuckoo) Delete(item []byte) bool {
	f := fprint(item, ck.fpSize, ck.hashfn)
	j := hashfp(item) % ck.size
	k := (j ^ hashfp(f)) % ck.size

	if ck.buckets[j].remove(f) || ck.buckets[k].remove(f) {
		ck.count--
		return true
	}

	return false
}

func (ck *Cuckoo) Count() uint {
	return ck.count
}

type fingerprint []byte

func fprint(item []byte, fpSize uint8, hashfn hash.Hash) fingerprint {
	hashfn.Reset()
	hashfn.Write(item)
	h := hashfn.Sum(nil)

	fp := make(fingerprint, fpSize)
	copy(fp, h)

	return fp
}

func hashfp(f fingerprint) uint {
	var h uint = 5381
	for i := range f {
		h = ((h << 5) + h) + uint(f[i])
	}

	return h
}

func match(a, b fingerprint) bool {
	return bytes.Equal(a, b)
}

type bucket []fingerprint

func (b bucket) insert(f fingerprint) bool {

	for i, fp := range b {
		if fp == nil {
			b[i] = f
			return true
		}
	}

	return false
}

func (b bucket) lookup(f fingerprint) bool {
	for _, fp := range b {
		if match(fp, f) {
			return true
		}
	}

	return false
}

func (b bucket) remove(f fingerprint) bool {
	for i, fp := range b {
		if match(fp, f) {
			b[i] = nil
			return true
		}
	}

	return false
}

func (b bucket) swap(f fingerprint) fingerprint {
	i := rand.Intn(len(b))
	b[i], f = f, b[i]

	return f
}

type option func(*Cuckoo)

func Size(s uint) option {
	return func(cf *Cuckoo) {
		cf.size = s
	}
}

func BucketSize(s uint8) option {
	return func(cf *Cuckoo) {
		cf.bSize = s
	}
}

func FingerprintSize(s uint8) option {
	return func(cf *Cuckoo) {
		cf.fpSize = s
	}
}

func MaximumKicks(k uint) option {
	return func(cf *Cuckoo) {
		cf.kicks = k
	}
}

func HashFn(hashfn hash.Hash) option {
	return func(cf *Cuckoo) {
		cf.hashfn = hashfn
	}
}

func configure(cf *Cuckoo) {
	if cf.hashfn == nil {
		cf.hashfn = fnv.New64()
	}
	if cf.bSize == 0 {
		cf.bSize = 4
	}
	if cf.fpSize == 0 {
		cf.fpSize = 3
	}
	if cf.kicks == 0 {
		cf.kicks = 500
	}
	if cf.size == 0 {
		cf.size = (1 << 18) / uint(cf.bSize)
	}
}
