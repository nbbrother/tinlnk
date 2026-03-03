package bloom

import (
	"hash/fnv"
	"math"
	"sync"
)

type Filter struct {
	mu      sync.RWMutex
	bits    []uint64
	size    uint64 // 位数组大小
	hashNum uint64 // 哈希函数个数
}

func New(n uint64, p float64) *Filter {
	m := uint64(-float64(n) * math.Log(p) / (math.Ln2 * math.Ln2))
	k := uint64(float64(m) / float64(n) * math.Ln2)
	if k < 1 {
		k = 1
	}

	size := (m + 63) / 64
	return &Filter{
		bits:    make([]uint64, size),
		size:    m,
		hashNum: k,
	}
}

// Add 添加元素
func (f *Filter) Add(data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	h1, h2 := f.hash(data)
	for i := uint64(0); i < f.hashNum; i++ {
		// 使用双重哈希：h(i) = h1 + i*h2
		pos := (h1 + i*h2) % f.size
		f.setBit(pos)
	}
}

// Contains 检查元素是否存在
func (f *Filter) Contains(data []byte) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	h1, h2 := f.hash(data)
	for i := uint64(0); i < f.hashNum; i++ {
		pos := (h1 + i*h2) % f.size
		if !f.getBit(pos) {
			return false
		}
	}
	return true
}

// hash 双重哈希
func (f *Filter) hash(data []byte) (uint64, uint64) {
	h1 := fnv.New64()
	h1.Write(data)
	h2 := fnv.New64a()
	h2.Write(data)
	return h1.Sum64(), h2.Sum64()
}

func (f *Filter) setBit(pos uint64) {
	f.bits[pos/64] |= 1 << (pos % 64)
}

func (f *Filter) getBit(pos uint64) bool {
	return (f.bits[pos/64] & (1 << (pos % 64))) != 0
}

// EstimatedFalsePositiveRate 估算当前误判率
func (f *Filter) EstimatedFalsePositiveRate(n uint64) float64 {
	return math.Pow(1-math.Exp(-float64(f.hashNum*n)/float64(f.size)), float64(f.hashNum))
}
