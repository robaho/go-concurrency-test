package go_concurrency

import "sync"

type Cache interface {
	Get(key int) int
	Put(key int, value int)
}

type LockCache struct {
	sync.RWMutex
	m map[int]int
}

func NewLockCache() *LockCache{
	m := LockCache{m:make(map[int]int)}
	return &m
}

func (m *LockCache) Get(key int) int {
	m.RLock()
	defer m.RUnlock()
	val,ok := m.m[key%1000000]
	if !ok {
		return 0
	}
	return val
}
func (m *LockCache) Put(key int, value int) {
	m.Lock()
	defer m.Unlock()
	m.m[key%1000000] = value
}

type UnsharedCache struct {
	m map[int]int
}

func NewUnsharedCache() *UnsharedCache{
	m := UnsharedCache{m:make(map[int]int)}
	return &m
}

func (m *UnsharedCache) Get(key int) int {
	val,ok := m.m[key%1000000]
	if !ok {
		return 0
	}
	return val
}
func (m *UnsharedCache) Put(key int, value int) {
	m.m[key%1000000] = value
}

type SyncCache struct {
	m sync.Map
}

func NewSyncCache() *SyncCache {
	m := SyncCache{}
	return &m
}

func (m *SyncCache) Get(key int) int {
	val, _ := m.m.Load(key%1000000)
	if val==nil {
		return 0
	}
	return val.(int)
}
func (m *SyncCache) Put(key int, value int) {
	m.m.Store(key%1000000, value)
}

type KeyValue struct {
	key, value int
}
type ChannelCache struct {
	m      map[int]int
	read   chan int
	write  chan KeyValue
	result chan int
}

func (m *ChannelCache) Get(key int) int {
	m.read <- key
	return <- m.result
}
func (m *ChannelCache) Put(key int, value int) {
	m.write <- KeyValue{key,value}
}

func NewChannelCache() *ChannelCache {
	c := &ChannelCache{m:make(map[int]int),read:make(chan int),write:make(chan KeyValue),result:make(chan int)}
	go func() {
		for {
			select {
			case key := <- c.read:
				val,ok := c.m[key%1000000]
				if !ok {
					val=0
				}
				c.result <- val
			case request:= <- c.write:
				c.m[request.key]=request.value
			}
		}
	}()
	return c
}
