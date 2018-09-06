**Update**

Based on feedback from Bryan Mills, I've updated the implementation using channels, and re-tested. I do not believe it has
made a significant difference. It does highlight that the 'Get' is worse than the 'Put' because 2 channels are involved.

He has written [Rethinking Classical Concurrency in Go](https://golang.org/wiki/Go-Community-Slides#rethinking-classical-concurrency-patterns)

He also provided that sync.Map has performance issues, see
[this search](https://github.com/golang/go/issues?utf8=✓&q=is%3Aissue+is%3Aopen+%22sync%3A%22+Map+in%3Atitle+label%3APerformance)

He also provided that the RWMutex has scalability issues (but I am not sure that applies here since only 2 routines are tested),
but here is the [incident](https://golang.org/issue/17973)

I removed the use of defer in the lock implementation as it is a known? performance issue.

I reduced the number of elements cached to be 500k which should allow the cache to fit entirely in the L3 on the testing machine.
It made some improvement.

I updated the testing methodology to make certain constraints on the test more clear.

I determined that the PutGet not matching the Put + Get times was because of cache locality, so PutGet was changed to read from
opposite sides, which corrected the problem, and so I removed the comment regarding potential 'go bench' accuracy issues.

**Summary**

The Go language has significant room for improvement in terms of concurrent data structure performance, with the current implementations being far
slower than the equivalent Java ones. This is no way to claim that Java is superior to Go, as there are other features of Go
(low GC pauses, lower memory footprint, no warm-up time, etc.) that make it a superior choice for many applications.

**Background**

This is a project designed to test the concurrency mechanisms available in Go. Go states,

[Do not communicate by sharing memory; instead, share memory by communicating.](https://blog.golang.org/share-memory-by-communicating)
and the basic mechanism to achieve this is channels.

In this test, I compare the 4 readily available ways to share information in Go:
 1. unshared/basic map 
 2. using locks 
 3. using sync.Map
 4. using channels
 
The above methods are used to build a "memory cache", which is a very common structure in high-performance financial applications, which is my current
area of work. This may not be the standard use case the Go authors envisioned as the amount of work per operation (store/load) is very small.

The 'unshared 'is not usable as a shared cache - possibly as a write-at-init, read only cache in a specialized application with cooperative immutability. 

Additionally, I wrote the same tests using Java, but only for the sync.Map case, which is fairly equivalent to Java's ConcurrentHashMap, and this
data structure and its methodologies use the same features as most lock-free structures (although ConcurrentHashMap does uses locks on store)

The tests run under [go bench]([https://golang.org/pkg/testing/) for Go, and [jmh](http://openjdk.java.net/projects/code-tools/jmh/) for Java.

In both cases, identical hardware was used, which is a 3.4 ghz Core i7, with 4 cores (8 threads), under OSX 10.13.6.

The software versions are Go 1.11, and Java 1.8_181.

**Testing Methodology**

The cache uses ints for keys and values, and the cache is limited to 1 million entries to avoid possible 'map' degradation.
The map is also pre-populated with the 1 million entries, and used for all of the tests, to ensure the tests do not measure map resizing costs, and
other start-up penalties. The expected cache size in memory is roughly 1 million * sizeof(int), but larger when pointers are used by the implementation,
especially in the case of Java. No tests were performed to measure the actual memory usage.

There are 3 operations tested Get, Put, and PutGet. They are tested in a uncontested scenario using 1 go routine, they are also tested
in a contested scenario (multi) using 2 go routines. The contested was limited to 2, since the machine only has 4 true cores, and there is 
housekeeper work performed by the benchmark harness, OS, etc. so this seemed fair.

The tests for both Go and Java were executed via Intellij, as it has support for both 'go bench' and 'jmh'

The testing harnesses do not operate identically, but I attempted to make the comparisons as fair as possible, mainly to use Java as a baseline
for what would be expected of the Go solution, since in my experience in many cases the Go code seems to outperform the equivalent Java code.

The caches are essentially 'static' structures. I did this for fairness. In testing there was no appreciable difference when the the cache structure
was allocated within the method under test. 

Any GC related activity is included in the tests. No attempt was made to reduce, or tune the GC activity on any platform.

**Go Test Results**

```
BenchmarkUnsharedCachePutGet-8    	10000000	       143 ns/op
BenchmarkUnsharedCachePut-8       	20000000	        78.8 ns/op
BenchmarkUnsharedCacheGet-8       	20000000	        73.6 ns/op
BenchmarkLockCachePutGet-8        	10000000	       231 ns/op
BenchmarkLockCachePut-8           	10000000	       128 ns/op
BenchmarkLockCacheGet-8           	20000000	       108 ns/op
BenchmarkSyncCachePutGet-8        	 3000000	       453 ns/op
BenchmarkSyncCachePut-8           	 5000000	       260 ns/op
BenchmarkSyncCacheGet-8           	10000000	       157 ns/op
BenchmarkChannelCachePutGet-8     	 1000000	      1427 ns/op
BenchmarkChannelCachePut-8        	 3000000	       554 ns/op
BenchmarkChannelCacheGet-8        	 2000000	       907 ns/op
BenchmarkLockCacheMultiPut-8      	 5000000	       357 ns/op
BenchmarkSyncCacheMultiPut-8      	 5000000	       298 ns/op
BenchmarkChannelCacheMultiPut-8   	 2000000	       984 ns/op
BenchmarkLockCacheMultiGet-8      	10000000	       153 ns/op
BenchmarkSyncCacheMultiGet-8      	10000000	       157 ns/op
BenchmarkChannelCacheMultiGet-8   	 1000000	      1508 ns/op
```

**Go Analysis**

There are several interesting, and disconcerting aspects.

1. The relative low performance of the unshared cache is disappointing, 
as this should be no more that a single indirection and a load or store. 
2. The sync.Map performs no better than the map using locks, even for Get. Ordinarliy, this would be a sign that sync.Map was implemented using
locks, but this is not the case. There is a lock-free component, but it doesn't appear to perform as it should.
3. Continuing on #3, the 'multi' using sync performs only marginally better than the lock version. Again, this should only be a volatile load, with the
expectation of performance equal to the unshared get.
4. The channel method is more than 5x slower than the others, and this is using a very simple key/value structure, although a more complex one would
probably necessitate using pointers.

**Java Test Results**

```
Benchmark                       Mode  Cnt   Score    Error  Units
TestJavaCache.Test0PutGet       avgt    3  62.720 ± 37.515  ns/op
TestJavaCache.Test1Put          avgt    3  43.883 ± 24.987  ns/op
TestJavaCache.Test2Get          avgt    3  14.854 ±  3.655  ns/op
TestJavaCache.Test3MultiPutGet  avgt    3  78.970 ± 14.730  ns/op
TestJavaCache.Test4MultiPut     avgt    3  52.705 ± 21.294  ns/op
TestJavaCache.Test5MultiGet     avgt    3  16.038 ±  6.033  ns/op


without warm-up

Benchmark                       Mode  Cnt   Score     Error  Units
TestJavaCache.Test0PutGet       avgt    3  87.939 ± 210.826  ns/op
TestJavaCache.Test1Put          avgt    3  50.545 ± 230.259  ns/op
TestJavaCache.Test2Get          avgt    3  16.661 ±  57.408  ns/op
TestJavaCache.Test3MultiPutGet  avgt    3  79.629 ±  34.733  ns/op
TestJavaCache.Test4MultiPut     avgt    3  54.272 ±  16.588  ns/op
TestJavaCache.Test5MultiGet     avgt    3  15.392 ±   2.863  ns/op

```

**Java Analysis**

1. There warm-up phase matters little in tests like this, since so many iterations are required.
1. The Java test results are across the board far superior to the equivalent Go operations, almost 10x faster in the best case, and 2x in the worst.
2. The Java 'multi get' is in line with the 'get', since it should be a lock-free volatile read.
3. The results shows that Java's dynamic inlining can achieve amazing performance.
4. The Java boxing of primitives into objects (required by CHM), seems very efficient. 

**Overall Findings**

Barring bugs in the Go benchmark harness, the Java performance numbers show that there is a lot of room for improvement in the available Go
structures for concurrent applications. The Go "use channels" is only suitable when the handling performs a significant amount of work, and/or
simplicity of code (as it is single-threaded, and the use of locks has otehr issues). The sync.Map should be nearly identically to the 'unshared map'
for reads, but it is 2x slower - this is a indicator of a poor implementation, or that the underlying native operations used (CAS, etc.) need to be
examined. 

_My experience with Go is ongoing, and there is a chance I've made some errors in these tests, and I welcome the community input to improve them._