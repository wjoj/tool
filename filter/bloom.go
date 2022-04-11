package filter

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"

	"github.com/wjoj/tool/base"
)

type Bloom struct {
	bitMap   *base.BitMap
	size     uint64
	hashN    uint64
	keys     [][]byte
	hashFunc func(message, key []byte) uint64
}

func NewBloom(size, hashN uint64) (*Bloom, error) {
	return &Bloom{
		bitMap:   base.NewBitMap(size),
		size:     size,
		hashN:    hashN,
		hashFunc: hmacHash,
	}, nil
}

func (b *Bloom) InitHashFunc(f func(message, key []byte) uint64) {
	b.hashFunc = f
}

func (b *Bloom) InitKeys(hashKeys [][]byte) {
	b.keys = hashKeys
}

func (b *Bloom) RandomKeys(keyLength int) {
	var hashKeys [][]byte
	for i := 0; i < int(b.hashN); i++ {
		randBytes := randBytes(10)
		hashKeys = append(hashKeys, randBytes)
	}
	b.keys = hashKeys
}

func (b *Bloom) Insert(msg []byte) bool {
	for _, v := range b.keys {
		val := b.hashFunc(msg, v)
		b.bitMap.Set(val % b.size)
	}
	return true
}

func (b *Bloom) Lookup(msg []byte) bool {
	for _, v := range b.keys {
		val := b.hashFunc(msg, v)
		if b.bitMap.IsSet(val % b.size) {
			return true
		}
	}
	return false
}

func hmacHash(msg, key []byte) uint64 {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	res := binary.BigEndian.Uint64(mac.Sum(nil))
	return res
}

func randBytes(length int) []byte {
	data := make([]byte, length)
	rand.Read(data)

	return data
}
