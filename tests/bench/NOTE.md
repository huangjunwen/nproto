### grpc vs nproto

#### Concurrency 100/Payload 1
```
~/gowork/src/google.golang.org/grpc/benchmark/benchmain$ ./benchmain -benchtime=10s -workloads=unary -compression=off -maxConcurrentCalls=100 -trace=off -reqSizeBytes=1 -respSizeBytes=1 -networkMode=Local
Unary-traceMode_false-latency_0s-kbps_0-MTU_0-maxConcurrentCalls_100-reqSize_1B-respSize_1B-Compressor_false:
50_Latency: 2232.1450 µs 	90_Latency: 3285.1720 µs 	99_Latency: 4135.8700 µs 	Avg latency: 2290.1080 µs 	Count: 436406 	9118 Bytes/op	167 Allocs/op
Histogram (unit: µs)
Count: 436406  Min: 301.6  Max: 15185.1  Avg: 2290.11
------------------------------------------------------------
[     301.641000,      301.642000)       1    0.0%    0.0%
[     301.642000,      301.647266)       0    0.0%    0.0%
[     301.647266,      301.680259)       0    0.0%    0.0%
[     301.680259,      301.886981)       0    0.0%    0.0%
[     301.886981,      303.182232)       0    0.0%    0.0%
[     303.182232,      311.297841)       0    0.0%    0.0%
[     311.297841,      362.147506)       2    0.0%    0.0%
[     362.147506,      680.754361)     400    0.1%    0.1%
[     680.754361,     2677.037454)  314699   72.1%   72.2%  #######
[    2677.037454,    15185.074000)  121303   27.8%  100.0%  ###
[   15185.074000,             inf)       1    0.0%  100.0%
```

```
~/gowork/src/github.com/huangjunwen/nproto/tests/bench/client$ ./client -c 1 -l 1 -p 100 -n 100000 -r 100000
2018/12/16 16:14:57 Nats URL: "nats://localhost:4222"
2018/12/16 16:14:57 Payload length (-l): 1
2018/12/16 16:14:57 Total RPC number (-n): 100000
2018/12/16 16:14:57 Client number (-c): 1
2018/12/16 16:14:57 Parallel go routines (-p): 100
2018/12/16 16:14:57 Target call rate per second (-r): 100000
2018/12/16 16:14:57 RPC timeout in seconds (-t): 3
2018/12/16 16:14:57 === Wating ===
2018/12/16 16:15:00 Elapse=2.830009859s
2018/12/16 16:15:00 Actual call rate=35335.566 RPC/sec
2018/12/16 16:15:00 Median latency=2.487469ms
2018/12/16 16:15:00 Actual concurency=87.896
2018/12/16 16:15:00 Latency HDR Percentiles:
2018/12/16 16:15:00 10:       1.395087ms
2018/12/16 16:15:00 50:       2.487439ms
2018/12/16 16:15:00 75:       3.248703ms
2018/12/16 16:15:00 80:       3.499791ms
2018/12/16 16:15:00 90:       4.441471ms
2018/12/16 16:15:00 95:       5.893279ms
2018/12/16 16:15:00 99:       8.657599ms
2018/12/16 16:15:00 99.99:    15.610495ms
2018/12/16 16:15:00 99.999:   17.811327ms
2018/12/16 16:15:00 100:      20.435455ms
```

#### Concurrency 1000/Payload 1

```
~/gowork/src/google.golang.org/grpc/benchmark/benchmain$ ./benchmain -benchtime=10s -workloads=unary -compression=off -maxConcurrentCalls=1000 -trace=off -reqSizeBytes=1 -respSizeBytes=1 -networkMode=Local
Unary-traceMode_false-latency_0s-kbps_0-MTU_0-maxConcurrentCalls_1000-reqSize_1B-respSize_1B-Compressor_false:
50_Latency: 25.9032 ms 	90_Latency: 34.2567 ms 	99_Latency: 52.3147 ms 	Avg latency: 26.4082 ms 	Count: 378790 9064 Bytes/op	163 Allocs/op
Histogram (unit: ms)
Count: 378790  Min:   5.6  Max: 118.3  Avg: 26.41
------------------------------------------------------------
[        5.559022,         5.559023)       1    0.0%    0.0%
[        5.559023,         5.559030)       0    0.0%    0.0%
[        5.559030,         5.559084)       0    0.0%    0.0%
[        5.559084,         5.559505)       0    0.0%    0.0%
[        5.559505,         5.562813)       0    0.0%    0.0%
[        5.562813,         5.588770)       0    0.0%    0.0%
[        5.588770,         5.792449)       0    0.0%    0.0%
[        5.792449,         7.390672)     116    0.0%    0.0%
[        7.390672,        19.931587)   66296   17.5%   17.5%  ##
[       19.931587,       118.337462)  312376   82.5%  100.0%  ########
[      118.337462,              inf)       1    0.0%  100.0%
```

```
~/gowork/src/github.com/huangjunwen/nproto/tests/bench/client$ ./client -c 1 -l 1 -p 1000 -n 100000 -r 100000
2018/12/16 16:16:28 Nats URL: "nats://localhost:4222"
2018/12/16 16:16:28 Payload length (-l): 1
2018/12/16 16:16:28 Total RPC number (-n): 100000
2018/12/16 16:16:28 Client number (-c): 1
2018/12/16 16:16:28 Parallel go routines (-p): 1000
2018/12/16 16:16:28 Target call rate per second (-r): 100000
2018/12/16 16:16:28 RPC timeout in seconds (-t): 3
2018/12/16 16:16:28 === Wating ===
2018/12/16 16:16:30 Elapse=1.821589127s
2018/12/16 16:16:30 Actual call rate=54897.122 RPC/sec
2018/12/16 16:16:30 Median latency=16.102448ms
2018/12/16 16:16:30 Actual concurency=883.978
2018/12/16 16:16:30 Latency HDR Percentiles:
2018/12/16 16:16:30 10:       8.107135ms
2018/12/16 16:16:30 50:       16.102463ms
2018/12/16 16:16:30 75:       22.287359ms
2018/12/16 16:16:30 80:       23.875199ms
2018/12/16 16:16:30 90:       29.045631ms
2018/12/16 16:16:30 95:       34.195199ms
2018/12/16 16:16:30 99:       45.890303ms
2018/12/16 16:16:30 99.99:    70.871551ms
2018/12/16 16:16:30 99.999:   79.201279ms
2018/12/16 16:16:30 100:      83.497471ms
```

#### Concurrency 100/Payload 1024

```
~/gowork/src/google.golang.org/grpc/benchmark/benchmain$ ./benchmain -benchtime=10s -workloads=unary -compression=off -maxConcurrentCalls=100 -trace=off -reqSizeBytes=1024 -respSizeBytes=1024 -networkMode=LocalUnary-traceMode_false-latency_0s-kbps_0-MTU_0-maxConcurrentCalls_100-reqSize_1024B-respSize_1024B-Compressor_false:
50_Latency: 3169.2630 µs 	90_Latency: 4215.7540 µs 	99_Latency: 5392.0770 µs 	Avg latency: 3112.7950 µs 	Count: 321034 	22414 Bytes/op	169 Allocs/op
Histogram (unit: µs)
Count: 321034  Min: 203.5  Max: 13513.0  Avg: 3112.80
------------------------------------------------------------
[     203.451000,      203.452000)       1    0.0%    0.0%
[     203.452000,      203.457188)       0    0.0%    0.0%
[     203.457188,      203.489295)       0    0.0%    0.0%
[     203.489295,      203.687985)       0    0.0%    0.0%
[     203.687985,      204.917543)       0    0.0%    0.0%
[     204.917543,      212.526455)       1    0.0%    0.0%
[     212.526455,      259.612939)       4    0.0%    0.0%
[     259.612939,      550.999788)      49    0.0%    0.0%
[     550.999788,     2354.198670)   69530   21.7%   21.7%  ##
[    2354.198670,    13512.994000)  251448   78.3%  100.0%  ########
[   13512.994000,             inf)       1    0.0%  100.0%
```

```
~/gowork/src/github.com/huangjunwen/nproto/tests/bench/client$ ./client -c 1 -l 1024 -p 100 -n 100000 -r 100000
2018/12/16 16:20:37 Nats URL: "nats://localhost:4222"
2018/12/16 16:20:37 Payload length (-l): 1024
2018/12/16 16:20:37 Total RPC number (-n): 100000
2018/12/16 16:20:37 Client number (-c): 1
2018/12/16 16:20:37 Parallel go routines (-p): 100
2018/12/16 16:20:37 Target call rate per second (-r): 100000
2018/12/16 16:20:37 RPC timeout in seconds (-t): 3
2018/12/16 16:20:37 === Wating ===
2018/12/16 16:20:41 Elapse=3.927018713s
2018/12/16 16:20:41 Actual call rate=25464.610 RPC/sec
2018/12/16 16:20:41 Median latency=3.23512ms
2018/12/16 16:20:41 Actual concurency=82.381
2018/12/16 16:20:41 Latency HDR Percentiles:
2018/12/16 16:20:41 10:       1.750791ms
2018/12/16 16:20:41 50:       3.235103ms
2018/12/16 16:20:41 75:       4.545855ms
2018/12/16 16:20:41 80:       5.042303ms
2018/12/16 16:20:41 90:       6.849855ms
2018/12/16 16:20:41 95:       8.695295ms
2018/12/16 16:20:41 99:       13.262271ms
2018/12/16 16:20:41 99.99:    22.145919ms
2018/12/16 16:20:41 99.999:   23.211263ms
2018/12/16 16:20:41 100:      23.241087ms
```

#### Concurrency 1000/Payload 1024

```
~/gowork/src/google.golang.org/grpc/benchmark/benchmain$ ./benchmain -benchtime=10s -workloads=unary -compression=off -maxConcurrentCalls=1000 -trace=off -reqSizeBytes=1024 -respSizeBytes=1024 -networkMode=Local
Unary-traceMode_false-latency_0s-kbps_0-MTU_0-maxConcurrentCalls_1000-reqSize_1024B-respSize_1024B-Compressor_false:
50_Latency: 37.5143 ms 	90_Latency: 44.0150 ms 	99_Latency: 53.0920 ms 	Avg latency: 37.7562 ms 	Count: 264977 22302 Bytes/op	165 Allocs/op
Histogram (unit: ms)
Count: 264977  Min:  16.5  Max: 151.5  Avg: 37.76
------------------------------------------------------------
[       16.539501,        16.539502)       1    0.0%    0.0%
[       16.539502,        16.539509)       0    0.0%    0.0%
[       16.539509,        16.539565)       0    0.0%    0.0%
[       16.539565,        16.540014)       0    0.0%    0.0%
[       16.540014,        16.543606)       0    0.0%    0.0%
[       16.543606,        16.572363)       0    0.0%    0.0%
[       16.572363,        16.802552)       0    0.0%    0.0%
[       16.802552,        18.645120)      54    0.0%    0.0%
[       18.645120,        33.394148)   42429   16.0%   16.0%  ##
[       33.394148,       151.454313)  222492   84.0%  100.0%  ########
[      151.454313,              inf)       1    0.0%  100.0%
```
~/gowork/src/github.com/huangjunwen/nproto/tests/bench/client$ ./client -c 1 -l 1024 -p 100 -n 100000 -r 100000
2018/12/16 16:20:37 Nats URL: "nats://localhost:4222"
2018/12/16 16:20:37 Payload length (-l): 1024
2018/12/16 16:20:37 Total RPC number (-n): 100000
2018/12/16 16:20:37 Client number (-c): 1
2018/12/16 16:20:37 Parallel go routines (-p): 100
2018/12/16 16:20:37 Target call rate per second (-r): 100000
2018/12/16 16:20:37 RPC timeout in seconds (-t): 3
2018/12/16 16:20:37 === Wating ===
2018/12/16 16:20:41 Elapse=3.927018713s
2018/12/16 16:20:41 Actual call rate=25464.610 RPC/sec
2018/12/16 16:20:41 Median latency=3.23512ms
2018/12/16 16:20:41 Actual concurency=82.381
2018/12/16 16:20:41 Latency HDR Percentiles:
2018/12/16 16:20:41 10:       1.750791ms
2018/12/16 16:20:41 50:       3.235103ms
2018/12/16 16:20:41 75:       4.545855ms
2018/12/16 16:20:41 80:       5.042303ms
2018/12/16 16:20:41 90:       6.849855ms
2018/12/16 16:20:41 95:       8.695295ms
2018/12/16 16:20:41 99:       13.262271ms
2018/12/16 16:20:41 99.99:    22.145919ms
2018/12/16 16:20:41 99.999:   23.211263ms
2018/12/16 16:20:41 100:      23.241087ms~/gowork/src/github.com/huangjunwen/nproto/tests/bench/client$ ./client -c 1 -l 1024 -p 100 -n 100000 -r 100000
2018/12/16 16:20:37 Nats URL: "nats://localhost:4222"
2018/12/16 16:20:37 Payload length (-l): 1024
2018/12/16 16:20:37 Total RPC number (-n): 100000
2018/12/16 16:20:37 Client number (-c): 1
2018/12/16 16:20:37 Parallel go routines (-p): 100
2018/12/16 16:20:37 Target call rate per second (-r): 100000
2018/12/16 16:20:37 RPC timeout in seconds (-t): 3
2018/12/16 16:20:37 === Wating ===
2018/12/16 16:20:41 Elapse=3.927018713s
2018/12/16 16:20:41 Actual call rate=25464.610 RPC/sec
2018/12/16 16:20:41 Median latency=3.23512ms
2018/12/16 16:20:41 Actual concurency=82.381
2018/12/16 16:20:41 Latency HDR Percentiles:
2018/12/16 16:20:41 10:       1.750791ms
2018/12/16 16:20:41 50:       3.235103ms
2018/12/16 16:20:41 75:       4.545855ms
2018/12/16 16:20:41 80:       5.042303ms
2018/12/16 16:20:41 90:       6.849855ms
2018/12/16 16:20:41 95:       8.695295ms
2018/12/16 16:20:41 99:       13.262271ms
2018/12/16 16:20:41 99.99:    22.145919ms
2018/12/16 16:20:41 99.999:   23.211263ms
2018/12/16 16:20:41 100:      23.241087ms
```
~/gowork/src/github.com/huangjunwen/nproto/tests/bench/client$ ./client -c 1 -l 1024 -p 1000 -n 100000 -r 100000
2018/12/16 16:19:22 Nats URL: "nats://localhost:4222"
2018/12/16 16:19:22 Payload length (-l): 1024
2018/12/16 16:19:22 Total RPC number (-n): 100000
2018/12/16 16:19:22 Client number (-c): 1
2018/12/16 16:19:22 Parallel go routines (-p): 1000
2018/12/16 16:19:22 Target call rate per second (-r): 100000
2018/12/16 16:19:22 RPC timeout in seconds (-t): 3
2018/12/16 16:19:22 === Wating ===
2018/12/16 16:19:25 Elapse=2.937079022s
2018/12/16 16:19:25 Actual call rate=34047.433 RPC/sec
2018/12/16 16:19:25 Median latency=27.66552ms
2018/12/16 16:19:25 Actual concurency=941.940
2018/12/16 16:19:25 Latency HDR Percentiles:
2018/12/16 16:19:25 10:       12.777599ms
2018/12/16 16:19:25 50:       27.665407ms
2018/12/16 16:19:25 75:       36.229887ms
2018/12/16 16:19:25 80:       38.609151ms
2018/12/16 16:19:25 90:       46.372095ms
2018/12/16 16:19:25 95:       51.672831ms
2018/12/16 16:19:25 99:       64.834559ms
2018/12/16 16:19:25 99.99:    82.864639ms
2018/12/16 16:19:25 99.999:   84.844031ms
2018/12/16 16:19:25 100:      86.377983ms
```
