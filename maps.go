package go_concurrency

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

func nextPowerOf2(v int) int {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}

type node struct {
	key, value int
	next       *node
}

type IntMap struct {
	table []*node
	mask  int
}

func NewIntMap(size int) *IntMap {
	size = nextPowerOf2(size)
	m := IntMap{}
	m.table = make([]*node, size)
	m.mask = size - 1
	return &m
}

func (m *IntMap) Get(key int) int {
	node := m.table[key&m.mask]
	if node == nil {
		return 0
	}
	for ; node != nil; node = node.next {
		if node.key == key {
			return node.value
		}
	}
	return 0
}
func (m *IntMap) Put(key int, value int) {
	head := m.table[key&m.mask]
	for node := head; node != nil; node = node.next {
		if node.key == key {
			node.value = value
			return
		}
	}
	n := &node{key: key, value: value, next: head}
	m.table[key&m.mask] = n
}

type SharedIntMap struct {
	table []*node
	mask  int
}

func NewSharedIntMap(size int) *SharedIntMap {
	size = nextPowerOf2(size)
	m := SharedIntMap{}
	m.table = make([]*node, size)
	m.mask = size - 1
	return &m
}

func (m *SharedIntMap) Get(key int) int {
	p := (*unsafe.Pointer)(unsafe.Pointer(&m.table[key&m.mask]))
	node := (*node)(atomic.LoadPointer(p))

	for ; node != nil; node = node.next {
		if node.key == key {
			return node.value
		}
	}
	return 0
}
func (m *SharedIntMap) Put(key int, value int) {

	p := (*unsafe.Pointer)(unsafe.Pointer(&m.table[key&m.mask]))

	for {
		head := (*node)(atomic.LoadPointer(p))
		for node := head; node != nil; node = node.next {
			if node.key == key {
				node.value = value
				//if !atomic.CompareAndSwapPointer(p,head,head) {
				//	continue
				//}
				return
			}
		}
		n := &node{key: key, value: value, next: head}
		if atomic.CompareAndSwapPointer(p, unsafe.Pointer(head), unsafe.Pointer(n)) {
			continue
		}
	}
}

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
	val, _ := m.m[key]
	m.RUnlock() // non-idiomatic go, but avoid defer performance hit
	return val
}
func (m *LockCache) Put(key int, value int) {
	m.Lock()
	m.m[key] = value
	m.Unlock() // non-idiomatic go, but avoid defer performance hit
}

type ShardCache struct {
	maps [10]map[int]int
}

func NewShardCache() *ShardCache {
	m := ShardCache{}
	for i := 0; i < 10; i++ {
		m.maps[i] = make(map[int]int)
	}
	return &m
}

func (m *ShardCache) Get(key int) int {
	val, _ := m.maps[key%10][key]
	return val
}
func (m *ShardCache) Put(key int, value int) {
	m.maps[key%10][key] = value
}

const SharedShardMask = 128 - 1

type imap map[int]int

type shard struct {
	imap
	sync.RWMutex
}

type SharedShardCache struct {
	shards [128]*shard
}

func NewSharedShardCache() *SharedShardCache {
	m := SharedShardCache{}
	for i := 0; i < 128; i++ {
		s := shard{imap: make(imap)}
		m.shards[i] = &s
	}
	return &m
}

func (m *SharedShardCache) Get(key int) int {
	s := m.shards[key&SharedShardMask]

	s.RLock()
	val, ok := s.imap[key]
	s.RUnlock()

	if !ok {
		return 0
	}
	return val
}

func (m *SharedShardCache) Put(key int, value int) {
	s := m.shards[key&SharedShardMask]
	s.Lock()
	s.imap[key] = value
	s.Unlock()
}

type UnsharedCache map[int]int

func NewUnsharedCache() *UnsharedCache {
	m := UnsharedCache{}
	return &m
}

func (m *UnsharedCache) Get(key int) int {
	val := (*m)[key]
	return val
}
func (m *UnsharedCache) Put(key int, value int) {
	(*m)[key] = value
}

type SyncCache struct {
	m sync.Map
}

func NewSyncCache() *SyncCache {
	m := SyncCache{}
	return &m
}

func (m *SyncCache) Get(key int) int {
	val, _ := m.m.Load(key)
	if val == nil {
		return 0
	}
	return val.(int)
}
func (m *SyncCache) Put(key int, value int) {
	m.m.Store(key, value)
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
				val, ok := c.m[request.(GetRequest).key]
				if !ok {
					val = 0
				}
				c.response <- val
			case PutRequest:
				c.m[request.(PutRequest).key] = request.(PutRequest).value
			}
		}
	}()
	return c
}
