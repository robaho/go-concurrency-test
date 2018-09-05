package test;

import org.openjdk.jmh.annotations.*;

import java.util.concurrent.*;

interface AnyCache {
    int get(int key);
    void put(int key,int value);
}

class MyConcurrentCache implements AnyCache {
    static final int MaxMapSize = 500000;

    ConcurrentHashMap<Integer,Integer> m = new ConcurrentHashMap();
    @Override
    public int get(int key) {
        return m.get(key%MaxMapSize);
    }

    @Override
    public void put(int key,int value) {
        m.put(key%MaxMapSize,value%MaxMapSize);
    }
}

@State(Scope.Benchmark)
@Fork(0)
@Warmup(iterations = 1)
@Measurement(iterations = 3, time = 1)
@BenchmarkMode(Mode.AverageTime)
@OutputTimeUnit(TimeUnit.NANOSECONDS)

public class TestJavaCache {

    static AnyCache m = new MyConcurrentCache();

    static ExecutorService e;

    @Setup
    public void setup() {
        e = Executors.newFixedThreadPool(2);
        for(int i=0;i<MyConcurrentCache.MaxMapSize;i++){
            m.put(i,i);
        }
    }
    @TearDown
    public void tearDown() {
        e.shutdown();
        for(int i=0;i<MyConcurrentCache.MaxMapSize;i++){
            if (m.get(i)!=i) {
                throw new IllegalStateException("index "+i+" = "+m.get(i));
            }
        }
    }


    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test0PutGet() {
        int sum=0;
        for(int i=0;i<1000000;i++) {
            m.put(i,i);
            sum+=m.get(i);
        }
        if(sum<0){
            System.out.println("error");
        }
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test1Put() {
        for(int i=0;i<1000000;i++) {
            m.put(i,i);
        }
    }
    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test2Get() {
        int sum=0;
        for(int i=0;i<1000000;i++) {
            sum+=m.get(i);
        }
        if(sum<0){
            System.out.println("error");
        }
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test3MultiPutGet() throws InterruptedException {
        CountDownLatch latch = new CountDownLatch(2);

        Runnable r = () -> {
            for(int i=0;i<1000000;i++) {
                m.put(i,i);
                if(m.get(i)<0){
                    System.out.println("error");
                }
            }
            latch.countDown();
        };
        e.execute(r);
        e.execute(r);
        latch.await();
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test4MultiPut() throws InterruptedException {
        CountDownLatch latch = new CountDownLatch(2);

        Runnable r = () -> {
            for(int i=0;i<1000000;i++) {
                m.put(i,i);
            }
            latch.countDown();
        };
        e.execute(r);
        e.execute(r);
        latch.await();
    }

    @Benchmark
    @OperationsPerInvocation(1000000)
    public void Test5MultiGet() throws InterruptedException {
        CountDownLatch latch = new CountDownLatch(2);

        Runnable r = () -> {
            int sum=0;
            for(int i=0;i<1000000;i++) {
                sum+=m.get(i);
            }
            if(sum<0) {
                System.out.println("error");
            }
            latch.countDown();
        };
        e.execute(r);
        e.execute(r);
        latch.await();
    }


}
