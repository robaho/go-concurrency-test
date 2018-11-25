package go_concurrency_test

import (
	"fmt"
	"github.com/robaho/go-concurrency-test"
	"sync"
	"testing"
	"time"
)

const NGOS = 2 // number of concurrent go routines for read/load tests

var um = go_concurrency.NewUnsharedCache()
var lm = go_concurrency.NewLockCache()
var sm = go_concurrency.NewSyncCache()
var cm = go_concurrency.NewChannelCache()
var sc = go_concurrency.NewShardCache()
var im = go_concurrency.NewIntMap(256000)   // so there are 4x collisions
var im2 = go_concurrency.NewIntMap(1000000) // so there are no collisions

var Sink int

func rand(r int) int {
	/* Algorithm "xor" from p. 4 of Marsaglia, "Xorshift RNGs" */
	r ^= r << 13
	r ^= r >> 17
	r ^= r << 5
	return r & 0x7fffffff
}

func BenchmarkRand(m *testing.B) {
	r := time.Now().Nanosecond()
	for i := 0; i < m.N; i++ {
		r = rand(r)
	}
	Sink = r
}
func BenchmarkMain(m *testing.B) {
	fmt.Println("populating maps...")
	for i := 0; i < 1000000; i++ {
		um.Put(i, i)
		lm.Put(i, i)
		sm.Put(i, i)
		cm.Put(i, i)
		sc.Put(i, i)
		im.Put(i, i)
		im2.Put(i, i)
	}
	m.ResetTimer()

	impls := []go_concurrency.Cache{um, lm, sm, cm, sc, im, im2}
	names := []string{"unshared", "lock", "sync", "channel", "shard", "intmap", "intmap2"}
	multi := []bool{false, true, true, true, false, true, true}

	for i := 0; i < len(impls); i++ {
		impl := impls[i]
		m.Run(names[i]+".get", func(b *testing.B) {
			r := time.Now().Nanosecond()

			var sum int
			for i := 0; i < b.N; i++ {
				r = rand(r)
				sum += impl.Get(r)
			}
			Sink = sum
		})
		m.Run(names[i]+".put", func(b *testing.B) {
			r := time.Now().Nanosecond()
			for i := 0; i < b.N; i++ {
				r = rand(r)
				impl.Put(r, r)
			}
		})
		m.Run(names[i]+".putget", func(b *testing.B) {
			r := time.Now().Nanosecond()
			var sum int
			for i := 0; i < b.N; i++ {
				r = rand(r)
				impl.Put(r, r)
				r = rand(r)
				sum += impl.Get(r)
			}
			Sink = sum
		})
		m.Run(names[i]+".multiget", func(b *testing.B) {
			wg := sync.WaitGroup{}
			for g := 0; g < NGOS; g++ {
				wg.Add(1)
				go func() {
					r := time.Now().Nanosecond()
					var sum int
					for i := 0; i < b.N; i++ {
						r = rand(r)
						sum += impl.Get(r)
					}
					Sink = sum
					wg.Done()
				}()
			}
			wg.Wait()
		})
		if !multi[i] { // some impl do not support concurrent write
			continue
		}
		m.Run(names[i]+".multiput", func(b *testing.B) {
			wg := sync.WaitGroup{}
			for g := 0; g < NGOS; g++ {
				wg.Add(1)
				go func() {
					r := time.Now().Nanosecond()

					for i := 0; i < b.N; i++ {
						r = rand(r)
						impl.Put(r, r)
					}
					wg.Done()
				}()
			}
			wg.Wait()
		})
		m.Run(names[i]+".multiputget", func(b *testing.B) {
			wg := sync.WaitGroup{}
			for g := 0; g < NGOS; g++ {
				wg.Add(1)
				go func() {
					r := time.Now().Nanosecond()

					var sum int
					for i := 0; i < b.N; i++ {
						r = rand(r)
						sum += impl.Get(r)
						r = rand(r)
						impl.Put(r, r)
					}
					Sink = sum
					wg.Done()
				}()
			}
			wg.Wait()
		})
	}
}
