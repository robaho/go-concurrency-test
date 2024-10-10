**Update 10/09/2024**

With Go generics, it is straightforward to use the techniques listed below to create a highly performant concurrent map implementation.

I have updated the timings using OpenJDK 21, Go 1.22.5, and clang version 14.0.0 (clang-1400.0.29.202) on an iMac Intel(R) Core(TM) i7-6700K CPU @ 4.00GHz

**Update 11/26/18**

After discussion in Go incident [28938](https://github.com/golang/go/issues/28938) I am prepared to make some final conclusions.

Concurrent map data structures in Go are most efficiently implemented using locks and the built-in map implementations. This is especially true
if the structure can be sharded to avoid single write mutex contention (see "shared shard" impl).
Locks are very efficient in Go due to the user-threading of Go routines.

The problem with Go's sync.Map is that it is "non-internal", unlike the built-in map implementation. So multiple layers of indirection are required
to accomplish some operations (use of interfaces, no atomic CAS at the table entry level).

In my tests I developed a fully concurrent "shared intmap" which shows excellent performance, but for a variety of reasons this is not a general solution,
 but it shows the performance potential of a redesigned sync.Map.

The Go incident above links to several issues #[21031](https://github.com/golang/go/issues/54720), #[21035](https://github.com/golang/go/issues/21035) that when resolved 
should bring sync.Map performance inline with Java's ConcurrentHashMap.

To be fair, most of performance issues with sync.Map only surface with a large map, with a large working set of active keys, as the indirection is
especially painful due to cpu cache misses. 

**Update 11/24/18**

I added 'intmap' versions - which are fixed sized maps (without resizing). The 'intmap2' is sized to
avoid any collisions, 'intmap' has 4x collisions.

I added 'C' versions of the 'intmap' for comparison. I also added a 'lock map' in Java similar to Go.

I increased the number of elements to 1000000 to avoid the cache fitting in the L2.

In all cases, the maps are pre-allocated to avoid any allocation overhead.

The major change across the board was to use random get/put indexes as the linear read/put was biased towards Java, since the Go uses
an alternate hash method so that sequential keys are not sequential in memory.

I re-ran the tests with latest versions, Go 1.11.2, and Java 1.8_191, and C was compiled with clang-1000.11.45.5

The tests now show Go to be a clear winner when using Locks, but sync.Map has significant performance issues.

Both Go and Java perform very close to the optimized C versions, with Java bettering Go in almost all cases (intmap tests) - this is probably
the reason the Go map implementation uses arrays of structs rather than a linked list of nodes.

*** A note about the multi timings... They are not divided by the number of Go routines/threads (which doubles the number of operations), but since there
are only 2 and ample cores available- meaning they should execute concurrently, it measures the overhead more directly since the timings
 can be directly compared to the single routine/thread case. 

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

The Go sync.Map implementation has significant room for improvement, performing far worse than Java ConcurrentHashMap, 
and locks should be used for low concurrent access, especially for high read weighted use cases.

The Go synchronization primitives and Go routine scheduling outperform Java with locks by a very wide margin. The user level context switching
is ideal.

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

The software versions are Go 1.11.2, and Java 1.8_191, Java OpenJDK11 with Shenandoah GC, and LLVM version 10.0.0 (clang-1000.11.45.5)

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
cpu: Intel(R) Core(TM) i7-6700K CPU @ 4.00GHz
BenchmarkRand-8         669243439                1.781 ns/op
populating maps...
BenchmarkMain/unshared.get-8            18877132                63.53 ns/op
BenchmarkMain/unshared.put-8            17971988                69.09 ns/op
BenchmarkMain/unshared.putget-8          9268077               131.4 ns/op
BenchmarkMain/unshared.multiget-8       17787741                66.86 ns/op
BenchmarkMain/lock.get-8                17399762                70.33 ns/op
BenchmarkMain/lock.put-8                10549318               114.1 ns/op
BenchmarkMain/lock.putget-8              6803950               178.8 ns/op
BenchmarkMain/lock.multiget-8           10305220               116.4 ns/op
BenchmarkMain/lock.multiput-8            4546767               255.9 ns/op
BenchmarkMain/lock.multiputget-8         2609588               448.0 ns/op
BenchmarkMain/sync.get-8                 5393236               216.7 ns/op
BenchmarkMain/sync.put-8                 3445075               353.9 ns/op
BenchmarkMain/sync.putget-8              1963018               651.0 ns/op
BenchmarkMain/sync.multiget-8            3567596               326.3 ns/op
BenchmarkMain/sync.multiput-8            3339694               377.1 ns/op
BenchmarkMain/sync.multiputget-8         1865768               676.4 ns/op
BenchmarkMain/channel.get-8              2028018               614.3 ns/op
BenchmarkMain/channel.put-8              3150184               384.3 ns/op
BenchmarkMain/channel.putget-8           1226596               984.5 ns/op
BenchmarkMain/channel.multiget-8         1000000              1169 ns/op
BenchmarkMain/channel.multiput-8         1468042               790.1 ns/op
BenchmarkMain/channel.multiputget-8       641374              1873 ns/op
BenchmarkMain/shard.get-8               16428013                71.47 ns/op
BenchmarkMain/shard.put-8               12421777                85.71 ns/op
BenchmarkMain/shard.putget-8             7720970               152.4 ns/op
BenchmarkMain/shard.multiget-8          15689773                75.81 ns/op
BenchmarkMain/shareshard.get-8          16479192                72.35 ns/op
BenchmarkMain/shareshard.put-8           9910390               110.3 ns/op
BenchmarkMain/shareshard.putget-8        6203272               183.1 ns/op
BenchmarkMain/shareshard.multiget-8     14384760                83.03 ns/op
BenchmarkMain/shareshard.multiput-8      8925080               133.5 ns/op
BenchmarkMain/shareshard.multiputget-8           4910743               248.7 ns/op
BenchmarkMain/intmap.get-8                      14669283                83.42 ns/op
BenchmarkMain/intmap.put-8                       6084549               191.5 ns/op
BenchmarkMain/intmap.putget-8                    4504140               250.0 ns/op
BenchmarkMain/intmap.multiget-8                 12845756                88.15 ns/op
BenchmarkMain/intmap.multiput-8                  5912864               190.2 ns/op
BenchmarkMain/intmap.multiputget-8               4621642               255.4 ns/op
BenchmarkMain/intmap2.get-8                     30600810                38.98 ns/op
BenchmarkMain/intmap2.put-8                      8988387               125.8 ns/op
BenchmarkMain/intmap2.putget-8                   5373553               210.6 ns/op
BenchmarkMain/intmap2.multiget-8                27520656                41.71 ns/op
BenchmarkMain/intmap2.multiput-8                 9286503               128.7 ns/op
BenchmarkMain/intmap2.multiputget-8              5525810               211.8 ns/op
BenchmarkMain/sharedint.get-8                   20388027                58.56 ns/op
BenchmarkMain/sharedint.put-8                    5578724               210.7 ns/op
BenchmarkMain/sharedint.putget-8                 3231453               365.0 ns/op
BenchmarkMain/sharedint.multiget-8              17616985                65.32 ns/op
BenchmarkMain/sharedint.multiput-8               5640673               207.1 ns/op
BenchmarkMain/sharedint.multiputget-8            3212062               364.5 ns/op
```

**Go Analysis**

There are several interesting, and disconcerting aspects.

1. The sync.Map performs far worse than the map using locks, even for Get - almost 3x slower.
2. Continuing on #1, the 'multi get' using sync performs worse than get. Again, this should only be a volatile load, with the
expectation of performance equal to the unshared get, or at least the sync.Map singular read.
3. The channel method is more than 5x slower than the others, and this is using a very simple key/value structure, although a more complex one would
probably necessitate using pointers.

**Java Test Results**

```
using 1 fork, 1 warm-up iteration, and 5 iterations of 3 sec

Benchmark                            (arg)  Mode  Cnt     Score     Error  Units
TestJavaCache.Test0Get            unshared  avgt    5    50.360 ±   5.020  ns/op
TestJavaCache.Test0Get          concurrent  avgt    5    48.672 ±   1.108  ns/op
TestJavaCache.Test0Get                lock  avgt    5    99.454 ±  13.884  ns/op
TestJavaCache.Test0Get              intmap  avgt    5    48.244 ±   8.969  ns/op
TestJavaCache.Test0Get             intmap2  avgt    5    18.599 ±   0.447  ns/op
TestJavaCache.Test2Put            unshared  avgt    5   109.816 ±  38.564  ns/op
TestJavaCache.Test2Put          concurrent  avgt    5   174.497 ±  47.735  ns/op
TestJavaCache.Test2Put                lock  avgt    5   174.281 ±  31.701  ns/op
TestJavaCache.Test2Put              intmap  avgt    5   105.264 ±   6.934  ns/op
TestJavaCache.Test2Put             intmap2  avgt    5    78.434 ±   1.202  ns/op
TestJavaCache.Test3PutGet         unshared  avgt    5   247.885 ±  55.511  ns/op
TestJavaCache.Test3PutGet       concurrent  avgt    5   282.537 ±  22.152  ns/op
TestJavaCache.Test3PutGet             lock  avgt    5   298.475 ±  29.188  ns/op
TestJavaCache.Test3PutGet           intmap  avgt    5   149.688 ±   3.338  ns/op
TestJavaCache.Test3PutGet          intmap2  avgt    5   119.706 ±   1.820  ns/op
TestJavaCache.Test4MultiGet       unshared  avgt    5    53.801 ±   0.663  ns/op
TestJavaCache.Test4MultiGet     concurrent  avgt    5    53.011 ±   1.252  ns/op
TestJavaCache.Test4MultiGet           lock  avgt    5   329.347 ±   2.941  ns/op
TestJavaCache.Test4MultiGet         intmap  avgt    5    51.659 ±   9.043  ns/op
TestJavaCache.Test4MultiGet        intmap2  avgt    5    22.869 ±   3.840  ns/op
TestJavaCache.Test5MultiPut       unshared  avgt    5   108.950 ±  28.201  ns/op
TestJavaCache.Test5MultiPut     concurrent  avgt    5   159.692 ±  23.058  ns/op
TestJavaCache.Test5MultiPut           lock  avgt    5   470.977 ±  84.851  ns/op
TestJavaCache.Test5MultiPut         intmap  avgt    5   104.542 ±   1.934  ns/op
TestJavaCache.Test5MultiPut        intmap2  avgt    5    78.117 ±   3.363  ns/op
TestJavaCache.Test6MultiPutGet    unshared  avgt    5   240.432 ±  59.746  ns/op
TestJavaCache.Test6MultiPutGet  concurrent  avgt    5   285.872 ±  38.001  ns/op
TestJavaCache.Test6MultiPutGet        lock  avgt    5  1678.923 ± 288.992  ns/op
TestJavaCache.Test6MultiPutGet      intmap  avgt    5   151.847 ±   5.489  ns/op
TestJavaCache.Test6MultiPutGet     intmap2  avgt    5   122.743 ±   3.129  ns/op

```
*** The Java multi-unshared and intmap are not valid, but no easy way to exclude with jmh. It doesn't crash because the maps are
pre-populated and don't resize.

**Java Analysis**

1. The warm-up phase matters little in tests like this, since so many iterations are required.
2. The Java concurrent 'multi get' is in line with the 'get', since it should be a lock-free volatile read.
3. The results shows that Java's dynamic inlining can achieve amazing performance.
4. The Java boxing of primitives into objects (required by CHM), seems very efficient. 
5. When using equal read & write, the Java "lock" methods shows very poor performance compared with the concurrent.

**C test Results**

```
With -O0
intmap put = 170.991800 ns/op
intmap get = 134.725800 ns/op
intmap2 put = 108.658200 ns/op
intmap2 get = 79.461800 ns/op

With -O3
intmap put = 116.890400 ns/op
intmap get = 56.260000 ns/op
intmap2 put = 94.516000 ns/op
intmap2 get = 25.283200 ns/op
```

**Overall Findings**

The Go performance numbers as compared to Java show that there is a lot of room for improvement in the available Go
structures for concurrent applications. The Go "use channels" is only suitable when the handling performs a significant amount of work, and/or
simplicity of code (as it is single-threaded, and the use of locks has other issues). The sync.Map should be nearly identical to the 'unshared map'
for reads (as it is for Java, especially the Get vs. MultiGet), but it is 2x slower - this is a indicator of a poor implementation, poor compiler optimizations,
or that the underlying native operations used (CAS, etc.) need to be examined. 

_My experience with Go is ongoing, and there is a chance I've made some errors in these tests, and I welcome the community input to improve them._
