package go_concurrency_test

import (
	"fmt"
	"github.com/robaho/go-concurrency-test"
	"sync"
	"testing"
)

const NGOS = 2 // number of concurrent go routines for read/load tests

var um = go_concurrency.NewUnsharedCache()
var lm = go_concurrency.NewLockCache()
var sm = go_concurrency.NewSyncCache()
var cm = go_concurrency.NewChannelCache()
var cc = go_concurrency.NewConcurrentCache()

func BenchmarkMain(m *testing.B) {
	fmt.Println("populating maps...")
	for i := 0; i < 1000000; i++ {
		um.Put(i, i)
		lm.Put(i, i)
		sm.Put(i, i)
		cm.Put(i, i)
		cc.Put(i, i)
	}
}

func BenchmarkUnsharedCachePutGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		um.Put(i, i)
		sum += um.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}

func BenchmarkUnsharedCachePut(b *testing.B) {
	for i := 0; i < b.N; i++ {
		um.Put(i, i)
	}
}

func BenchmarkUnsharedCacheGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		sum += um.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}

func BenchmarkLockCachePutGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		lm.Put(i, i)
		sum += lm.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}
func BenchmarkLockCachePut(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lm.Put(i, i)
	}
}
func BenchmarkLockCacheGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		sum += lm.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}

func BenchmarkSyncCachePutGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		sm.Put(i, i)
		sum += sm.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}
func BenchmarkSyncCachePut(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sm.Put(i, i)
	}
}
func BenchmarkSyncCacheGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		sum += sm.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}

func BenchmarkChannelCachePutGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		cm.Put(i, i)
		sum += cm.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}

func BenchmarkChannelCachePut(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cm.Put(i, i)
	}
}
func BenchmarkChannelCacheGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		sum += cm.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}

func BenchmarkConcurrentCachePutGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		cc.Put(i, i)
		sum += cc.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}

func BenchmarkConcurrentCachePut(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cc.Put(i, i)
	}
}
func BenchmarkConcurrentCacheGet(b *testing.B) {
	var sum int
	for i := 0; i < b.N; i++ {
		sum += cc.Get(i)
	}
	if sum < 0 {
		fmt.Println("wrong value")
	}
}

func BenchmarkLockCacheMultiPut(b *testing.B) {
	wg := sync.WaitGroup{}
	for g := 0; g < NGOS; g++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				lm.Put(i, i)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
func BenchmarkSyncCacheMultiPut(b *testing.B) {
	wg := sync.WaitGroup{}
	for g := 0; g < NGOS; g++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				sm.Put(i, i)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
func BenchmarkChannelCacheMultiPut(b *testing.B) {
	wg := sync.WaitGroup{}
	for g := 0; g < NGOS; g++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				cm.Put(i, i)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkLockCacheMultiGet(b *testing.B) {
	wg := sync.WaitGroup{}
	for g := 0; g < NGOS; g++ {
		wg.Add(1)
		go func() {
			var sum int
			for i := 0; i < b.N; i++ {
				sum += lm.Get(i)
			}
			if sum < 0 {
				fmt.Println("wrong value")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
func BenchmarkSyncCacheMultiGet(b *testing.B) {
	wg := sync.WaitGroup{}
	for g := 0; g < NGOS; g++ {
		wg.Add(1)
		go func() {
			var sum int
			for i := 0; i < b.N; i++ {
				sum += sm.Get(i)
			}
			if sum < 0 {
				fmt.Println("wrong value")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
func BenchmarkChannelCacheMultiGet(b *testing.B) {
	wg := sync.WaitGroup{}
	for g := 0; g < NGOS; g++ {
		wg.Add(1)
		go func() {
			var sum int
			for i := 0; i < b.N; i++ {
				sum += cm.Get(i)
			}
			if sum < 0 {
				fmt.Println("wrong value")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkConcurrentCacheMultiPut(b *testing.B) {
	wg := sync.WaitGroup{}
	for g := 0; g < NGOS; g++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				cc.Put(i, i)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkConcurrentCacheMultiGet(b *testing.B) {
	wg := sync.WaitGroup{}
	for g := 0; g < NGOS; g++ {
		wg.Add(1)
		go func() {
			var sum int
			for i := 0; i < b.N; i++ {
				sum += cc.Get(i)
			}
			if sum < 0 {
				fmt.Println("wrong value")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
