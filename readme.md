**Clarification**

The "multi" timings measure the time to perform the op on N threads/routines and N is 2 to ensure sufficient cores are available so that these timings can be directly compared to the non concurrent timings.

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

I added an 'unshared' cache to the Java tests for an additional baseline. 

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
BenchmarkUnsharedCachePutGet-8    	10000000	       121 ns/op
BenchmarkUnsharedCachePut-8       	20000000	        66.4 ns/op
BenchmarkUnsharedCacheGet-8       	20000000	        62.5 ns/op
BenchmarkLockCachePutGet-8        	10000000	       196 ns/op
BenchmarkLockCachePut-8           	20000000	       112 ns/op
BenchmarkLockCacheGet-8           	20000000	        92.6 ns/op
BenchmarkSyncCachePutGet-8        	 3000000	       439 ns/op
BenchmarkSyncCachePut-8           	 5000000	       265 ns/op
BenchmarkSyncCacheGet-8           	10000000	       159 ns/op
BenchmarkChannelCachePutGet-8     	 1000000	      1389 ns/op
BenchmarkChannelCachePut-8        	 3000000	       541 ns/op
BenchmarkChannelCacheGet-8        	 2000000	       887 ns/op
BenchmarkLockCacheMultiPut-8      	 5000000	       266 ns/op
BenchmarkSyncCacheMultiPut-8      	 5000000	       279 ns/op
BenchmarkChannelCacheMultiPut-8   	 2000000	       809 ns/op
BenchmarkLockCacheMultiGet-8      	10000000	       140 ns/op
BenchmarkSyncCacheMultiGet-8      	10000000	       177 ns/op
BenchmarkChannelCacheMultiGet-8   	 1000000	      1429 ns/op
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
using 1 warm-up iteration, and 3 iterations of 1 sec

Benchmark                            (arg)  Mode  Cnt   Score    Error  Units
TestJavaCache.Test0PutGet         unshared  avgt    3  36.375 ±  7.426  ns/op
TestJavaCache.Test0PutGet       concurrent  avgt    3  48.144 ±  6.811  ns/op
TestJavaCache.Test1Put            unshared  avgt    3  27.740 ±  6.255  ns/op
TestJavaCache.Test1Put          concurrent  avgt    3  33.531 ±  7.990  ns/op
TestJavaCache.Test2Get            unshared  avgt    3  14.736 ±  7.691  ns/op
TestJavaCache.Test2Get          concurrent  avgt    3  15.844 ±  4.401  ns/op
TestJavaCache.Test3MultiPutGet    unshared  avgt    3  50.509 ± 24.125  ns/op
TestJavaCache.Test3MultiPutGet  concurrent  avgt    3  85.440 ± 28.356  ns/op
TestJavaCache.Test4MultiPut       unshared  avgt    3  41.901 ±  0.181  ns/op
TestJavaCache.Test4MultiPut     concurrent  avgt    3  51.607 ±  6.243  ns/op
TestJavaCache.Test5MultiGet       unshared  avgt    3  12.421 ±  2.952  ns/op
TestJavaCache.Test5MultiGet     concurrent  avgt    3  16.409 ±  6.469  ns/op

```
*** The Java multi-unshared are not valid, but no easy way to exclude with jmh. It doesn't crash because the maps are
pre-populated and don't resize.

**Java Analysis**

1. The warm-up phase matters little in tests like this, since so many iterations are required.
1. The Java test results are across the board far superior to the equivalent Go operations, almost 10x faster in the best case, and 2x in the worst.
2. The Java 'multi get' is in line with the 'get', since it should be a lock-free volatile read.
3. The results shows that Java's dynamic inlining can achieve amazing performance.
4. The Java boxing of primitives into objects (required by CHM), seems very efficient. 

**Overall Findings**

The Go performance numbers as compared to Java show that there is a lot of room for improvement in the available Go
structures for concurrent applications. The Go "use channels" is only suitable when the handling performs a significant amount of work, and/or
simplicity of code (as it is single-threaded, and the use of locks has other issues). The sync.Map should be nearly identical to the 'unshared map'
for reads (as it is for Java, especially the Get vs. MultiGet), but it is 2x slower - this is a indicator of a poor implementation, poor compiler optimizations,
or that the underlying native operations used (CAS, etc.) need to be examined. 

_My experience with Go is ongoing, and there is a chance I've made some errors in these tests, and I welcome the community input to improve them._
