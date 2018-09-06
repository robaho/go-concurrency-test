package go_concurrency

import (
	"sync"
)

type Hashable interface {
	Hash() int
}

type hashTable map[Hashable]interface{}

type ConcurrentMap struct {
	tables [16]*hashTable
	locks  [16]sync.Mutex
}

func NewConcurrentMap() *ConcurrentMap {
	m := ConcurrentMap{}

	for i := 0; i < 16; i++ {
		mymap := make(hashTable)
		m.tables[i] = &mymap
	}
	return &m
}

func (m *ConcurrentMap) Get(key Hashable) (interface{}, bool) {
	table := key.Hash() % 16
	hashtable := m.tables[table]
	val, ok := (*hashtable)[key]
	return val, ok
}

func (m *ConcurrentMap) Put(key Hashable, value interface{}) {
	table := key.Hash() % 16
	lock := m.locks[table]

	lock.Lock()

	newmap := make(hashTable)
	hashtable := m.tables[table]

	for k, v := range *hashtable {
		newmap[k] = v
	}
	newmap[key] = value

	m.tables[table] = &newmap
	lock.Unlock()
}
