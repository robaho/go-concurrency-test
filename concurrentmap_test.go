package go_concurrency_test

import (
	"github.com/robaho/go-concurrency-test"
	"testing"
)

type Int int

func (i Int) Hash() int { // implements Hashable
	return int(i)
}

func TestConcurrentMap(t *testing.T) {

	cm := go_concurrency.NewConcurrentMap()

	for i := 0; i < 1000000; i++ {
		cm.Put(Int(i), Int(i))
	}

	for i := 0; i < 1000000; i++ {
		v, _ := cm.Get(Int(i))

		if v.(int) != i {
			t.Fatal("incorrect value at ", i, " value is ", v)
		}
	}
}
