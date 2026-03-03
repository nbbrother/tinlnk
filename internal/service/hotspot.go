package service

import (
	"container/heap"
	"sync"
	"time"
)

// HotSpotDetector 热点探测器
// 使用滑动窗口 + 最小堆实现实时热点检测
type HotSpotDetector struct {
	mu        sync.RWMutex
	counters  map[string]*counter
	window    time.Duration
	threshold int
}

type counter struct {
	key       string
	count     int64
	timestamp time.Time
}

func NewHotSpotDetector(threshold int, window time.Duration) *HotSpotDetector {
	hd := &HotSpotDetector{
		counters:  make(map[string]*counter),
		window:    window,
		threshold: threshold,
	}
	go hd.cleanup()
	return hd
}

// Record 记录访问
func (h *HotSpotDetector) Record(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if c, ok := h.counters[key]; ok {
		c.count++
		c.timestamp = time.Now()
	} else {
		h.counters[key] = &counter{
			key:       key,
			count:     1,
			timestamp: time.Now(),
		}
	}
}

// IsHotSpot 判断是否为热点
func (h *HotSpotDetector) IsHotSpot(key string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if c, ok := h.counters[key]; ok {
		return c.count >= int64(h.threshold)
	}
	return false
}

// GetTopK 获取TopK热点
func (h *HotSpotDetector) GetTopK(k int) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 使用最小堆找TopK
	pq := make(PriorityQueue, 0, k)
	heap.Init(&pq)

	for _, c := range h.counters {
		if pq.Len() < k {
			heap.Push(&pq, c)
		} else if c.count > pq[0].count {
			heap.Pop(&pq)
			heap.Push(&pq, c)
		}
	}

	result := make([]string, pq.Len())
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = heap.Pop(&pq).(*counter).key
	}
	return result
}

// cleanup 定期清理过期数据
func (h *HotSpotDetector) cleanup() {
	ticker := time.NewTicker(h.window)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		now := time.Now()
		for key, c := range h.counters {
			if now.Sub(c.timestamp) > h.window {
				delete(h.counters, key)
			}
		}
		h.mu.Unlock()
	}
}

// PriorityQueue 最小堆实现
type PriorityQueue []*counter

func (pq PriorityQueue) Len() int            { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool  { return pq[i].count < pq[j].count }
func (pq PriorityQueue) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i] }
func (pq *PriorityQueue) Push(x interface{}) { *pq = append(*pq, x.(*counter)) }
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}
