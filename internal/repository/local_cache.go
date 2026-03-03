package repository

import (
	"sync"
	"time"
)

// LocalCache 本地缓存（L1缓存）
// 使用LRU + TTL策略，减少Redis访问
type LocalCache struct {
	mu      sync.RWMutex
	data    map[string]*cacheItem
	maxSize int
	ttl     time.Duration
	lruList *lruList
}

type cacheItem struct {
	value    string
	expireAt time.Time
	lruNode  *lruNode
}

type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

type lruList struct {
	head *lruNode
	tail *lruNode
	size int
}

func NewLocalCache(maxSize int, ttl time.Duration) *LocalCache {
	lc := &LocalCache{
		data:    make(map[string]*cacheItem),
		maxSize: maxSize,
		ttl:     ttl,
		lruList: &lruList{},
	}

	// 初始化哨兵节点
	lc.lruList.head = &lruNode{}
	lc.lruList.tail = &lruNode{}
	lc.lruList.head.next = lc.lruList.tail
	lc.lruList.tail.prev = lc.lruList.head

	// 启动过期清理
	go lc.cleanup()

	return lc
}

func (c *LocalCache) Get(key string) (string, bool) {
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		return "", false
	}

	// 检查过期
	if time.Now().After(item.expireAt) {
		c.Delete(key)
		return "", false
	}

	// 移动到头部（最近访问）
	c.mu.Lock()
	c.moveToHead(item.lruNode)
	c.mu.Unlock()

	return item.value, true
}

func (c *LocalCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 已存在则更新
	if item, ok := c.data[key]; ok {
		item.value = value
		item.expireAt = time.Now().Add(c.ttl)
		c.moveToHead(item.lruNode)
		return
	}

	// 淘汰策略：超过容量删除最久未访问
	if c.lruList.size >= c.maxSize {
		tail := c.lruList.tail.prev
		if tail != c.lruList.head {
			c.removeNode(tail)
			delete(c.data, tail.key)
		}
	}

	// 添加新节点
	node := &lruNode{key: key}
	c.addToHead(node)
	c.data[key] = &cacheItem{
		value:    value,
		expireAt: time.Now().Add(c.ttl),
		lruNode:  node,
	}
}

func (c *LocalCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.data[key]; ok {
		c.removeNode(item.lruNode)
		delete(c.data, key)
	}
}

func (c *LocalCache) addToHead(node *lruNode) {
	node.prev = c.lruList.head
	node.next = c.lruList.head.next
	c.lruList.head.next.prev = node
	c.lruList.head.next = node
	c.lruList.size++
}

func (c *LocalCache) removeNode(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
	c.lruList.size--
}

func (c *LocalCache) moveToHead(node *lruNode) {
	c.removeNode(node)
	c.addToHead(node)
}

func (c *LocalCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.data {
			if now.After(item.expireAt) {
				c.removeNode(item.lruNode)
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}
