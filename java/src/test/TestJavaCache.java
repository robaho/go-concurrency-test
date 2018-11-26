package test;

import org.openjdk.jmh.annotations.*;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.*;
import java.util.concurrent.locks.ReadWriteLock;
import java.util.concurrent.locks.ReentrantReadWriteLock;

interface AnyCache {
    int get(int key);
    void put(int key,int value);
}

class MyConcurrentCache implements AnyCache {

    final ConcurrentHashMap<Integer,Integer> m = new ConcurrentHashMap();
    @Override
    public int get(int key) {
        return m.get(key);
    }

    @Override
    public void put(int key,int value) {
        m.put(key,value);
    }
}

class MyLockCache implements AnyCache {

    final ReadWriteLock rw = new ReentrantReadWriteLock(false);
    final HashMap<Integer,Integer> m = new HashMap();

    @Override
    public int get(int key) {
        rw.readLock().lock();
        try {
            return m.get(key);
        } finally {
            rw.readLock().unlock();
        }
    }

    @Override
    public void put(int key,int value) {
        rw.writeLock().lock();
        try {
            m.put(key, value);
        } finally {
            rw.writeLock().unlock();
        }
    }
}

/*
note, this would crash in a real "multi" environment, but only works here since
the map is pre-populated so it is never resized. There is no easy way in jmh to restrict
certain benchmarks to certain parameters
 */
class MyUnsharedCache implements AnyCache {
    final Map<Integer,Integer> m = new HashMap();

    @Override
    public int get(int key) {
        return m.get(key);
    }

    @Override
    public void put(int key,int value) {
        m.put(key,value);
    }
}

class IntMap implements AnyCache {
    static class node {
        int key,value;
        node next;
    }

    private final node[] table;
    private final int mask;
    private static int nextPowerOf2(int v) {
        v--;
        v |= v >> 1;
        v |= v >> 2;
        v |= v >> 4;
        v |= v >> 8;
        v |= v >> 16;
        v++;
        return v;
    }
    public IntMap(int size) {
        size = nextPowerOf2(size);
        table = new node[size];
        mask = size-1;
    }
    @Override
    public int get(int key) {
        node n = table[key&mask];
        if (n==null) {
            return 0;
        }
        for(;n!=null;n=n.next) {
            if(n.key==key){
                return n.value;
            }
        }
        return 0;
    }

    @Override
    public void put(int key, int value) {
        node head = table[key&mask];
        for(node n=head;n!=null;n=n.next) {
            if(n.key==key) {
                n.value=value;
                return;
            }
        }
        node n = new node();
        n.key=key;
        n.value=value;
        n.next=head;
        table[key&mask]=n;
    }
}

@State(Scope.Benchmark)
@Fork(1)
@Warmup(iterations = 1)
@Measurement(iterations = 5, time = 3)
@BenchmarkMode(Mode.AverageTime)
@OutputTimeUnit(TimeUnit.NANOSECONDS)

public class TestJavaCache {
    final int Mask = (1024*1024)-1;
    final int NTHREADS = 2;

    static int rand(int r) {
        /* Algorithm "xor" from p. 4 of Marsaglia, "Xorshift RNGs" */
        r ^= r << 13;
        r ^= r >> 17;
        r ^= r << 5;
        return r & 0x7fffffff;
    }

    @Param({"unshared", "concurrent", "lock","intmap","intmap2"})
    public String arg;

    static AnyCache m;

    static ExecutorService e;

    public int Sink;

    @Setup
    public void setup() {
        switch(arg){
            case "unshared":
                m = new MyUnsharedCache(); break;
            case "concurrent":
                m = new MyConcurrentCache(); break;
            case "lock":
                m = new MyLockCache(); break;
            case "intmap":
                m = new IntMap(256000); break;
            case "intmap2":
                m = new IntMap(1000000); break;
        }

        e = Executors.newFixedThreadPool(NTHREADS);
        for(int i=0;i<=Mask;i++){
            m.put(i,i);
        }
    }
    @TearDown
    public void tearDown() {
        e.shutdown();
        for(int i=0;i<=Mask;i++){
            if ((m.get(i)&Mask) != (i&Mask)) {
                throw new IllegalStateException("index "+i+" = "+m.get(i));
            }
        }
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test0Get() {
        int sum=0;
        int r = (int)System.nanoTime();
        for(int i=0;i<1000000;i++) {
            r = rand(r);
            sum+=m.get(r&Mask);
        }
        Sink = sum;
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test2Put() {
        int r = (int)System.nanoTime();
        for(int i=0;i<1000000;i++) {
            r = rand(r);
            m.put(r&Mask,r);
        }
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test3PutGet() {
        int r = (int)System.nanoTime();
        int sum=0;
        for(int i=0;i<1000000;i++) {
            r = rand(r);
            m.put(r&Mask,r);
            r = rand(r);
            sum+=m.get(r&Mask);
        }
        Sink = sum;
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test4MultiGet() throws InterruptedException {
        CountDownLatch latch = new CountDownLatch(NTHREADS);

        Runnable run = () -> {
            Test0Get();
            latch.countDown();
        };
        for(int i=0;i<NTHREADS;i++){
            e.execute(run);
        }
        latch.await();
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test5MultiPut() throws InterruptedException {
        CountDownLatch latch = new CountDownLatch(NTHREADS);

        Runnable run = () -> {
            Test2Put();
            latch.countDown();
        };
        for(int i=0;i<NTHREADS;i++){
            e.execute(run);
        }
        latch.await();
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test6MultiPutGet() throws InterruptedException {
        CountDownLatch latch = new CountDownLatch(NTHREADS);

        Runnable run = () -> {
            Test3PutGet();
            latch.countDown();
        };
        for(int i=0;i<NTHREADS;i++){
            e.execute(run);
        }
        latch.await();
    }


}
