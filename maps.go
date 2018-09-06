package go_concurrency

import "sync"

const MaxMapSize = 500000

type Cache interface {
	Get(key int) int
	Put(key int, value int)
}

type LockCache struct {
	sync.RWMutex
	m map[int]int
}

func NewLockCache() *LockCache {
	m := LockCache{m: make(map[int]int)}
	return &m
}

func (m *LockCache) Get(key int) int {
	m.RLock()
	val, ok := m.m[key%MaxMapSize]
	m.RUnlock() // non-idiomatic go, but avoid defer performance hit
	if !ok {
		return 0
	}
	return val
}
func (m *LockCache) Put(key int, value int) {
	m.Lock()
	m.m[key%MaxMapSize] = value
	m.Unlock() // non-idiomatic go, but avoid defer performance hit
}

type UnsharedCache struct {
	m map[int]int
}

func NewUnsharedCache() *UnsharedCache {
	m := UnsharedCache{m: make(map[int]int)}
	return &m
}

func (m *UnsharedCache) Get(key int) int {
	val, ok := m.m[key%MaxMapSize]
	if !ok {
		return 0
	}
	return val
}
func (m *UnsharedCache) Put(key int, value int) {
	m.m[key%MaxMapSize] = value
}

type SyncCache struct {
	m sync.Map
}

func NewSyncCache() *SyncCache {
	m := SyncCache{}
	return &m
}

func (m *SyncCache) Get(key int) int {
	val, _ := m.m.Load(key % MaxMapSize)
	if val == nil {
		return 0
	}
	return val.(int)
}
func (m *SyncCache) Put(key int, value int) {
	m.m.Store(key%MaxMapSize, value)
}

type PutRequest struct {
	key, value int
}
type GetRequest struct {
	key int
}

type ChannelCache struct {
	m        map[int]int
	request  chan interface{}
	response chan int
}

func (m *ChannelCache) Get(key int) int {
	m.request <- GetRequest{key}
	return <-m.response
}

func (m *ChannelCache) Put(key int, value int) {
	m.request <- PutRequest{key, value}
}

func NewChannelCache() *ChannelCache {
	c := &ChannelCache{m: make(map[int]int), request: make(chan interface{}), response: make(chan int)}
	go func() {
		for {
			request := <-c.request
			switch request.(type) {
			case GetRequest:
				val, ok := c.m[request.(GetRequest).key%MaxMapSize]
				if !ok {
					val = 0
				}
				c.response <- val
			case PutRequest:
				c.m[request.(PutRequest).key%MaxMapSize] = request.(PutRequest).value
			}
		}
	}()
	return c
}

type ConcurrentMapCache struct {
	m *ConcurrentMap
}

func NewConcurrentCache() *ConcurrentMapCache {
	m := ConcurrentMapCache{m: NewConcurrentMap()}
	return &m
}

type Int int

func (i Int) Hash() int {
	return int(i)
}

func (m *ConcurrentMapCache) Get(key int) int {
	val, ok := m.m.Get(Int(key))
	if !ok {
		return 0
	}
	return val.(int)
}
func (m *ConcurrentMapCache) Put(key int, value int) {
	m.m.Put(Int(key), value)
}
