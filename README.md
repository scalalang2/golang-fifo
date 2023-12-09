<h1 align="center">golang-fifo</h1>
<p align="center">
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/license-MIT-_red.svg"></a>
</p>

This is a modern cache implementation, **inspired** by the following papers, provides high efficiency.

- **SIEVE** | [SIEVE is Simpler than LRU: an Efficient Turn-Key Eviction Algorithm for Web Caches](https://junchengyang.com/publication/nsdi24-SIEVE.pdf) (NSDI'24)
- **S3-FIFO** | [FIFO queues are all you need for cache eviction](https://dl.acm.org/doi/10.1145/3600006.3613147) (SOSP'23)

This offers state-of-the-art efficiency and scalability compared to other LRU-based cache algorithms.

## Benchmark Result
The benchmark result were obtained using [go-cache-benchmark](https://github.com/scalalang2/go-cache-benchmkark)

```
itemSize=500000, workloads=7500000, cacheSize=0.10%, zipf's alpha=0.99, concurrency=16

      CACHE      | HITRATE | MEMORY  |   QPS   |  HITS   | MISSES   
-----------------+---------+---------+---------+---------+----------
  sieve          | 47.60%  | 0.12MiB | 2853881 | 3570146 | 3929854  
  tinylfu        | 47.39%  | 0.11MiB | 1983602 | 3554428 | 3945572  
  slru           | 46.48%  | 0.11MiB | 1948558 | 3486176 | 4013824  
  s4lru          | 46.15%  | 0.12MiB | 2417016 | 3461316 | 4038684  
  two-queue      | 45.49%  | 0.17MiB | 1863817 | 3411840 | 4088160  
  clock          | 37.33%  | 0.10MiB | 1927525 | 2800086 | 4699914  
  lru-groupcache | 36.57%  | 0.11MiB | 1898254 | 2742607 | 4757393  
  lru-hashicorp  | 36.56%  | 0.08MiB | 2072396 | 2741646 | 4758354 
```

**SIEVE** not only provides a high hit ratio, but also the highest QPS (Queries Per Second). 
This means that SIEVE is able to process more requests per second than any other cache. 
Additionally, SIEVE is about 10% more efficient than a simple LRU cache. 

While LRU promotes accessed objects to the head of the queue, 
requiring a potentially slow lock acquisition, 
SIEVE only needs to update a single bit upon a cache hit. 
This update can be done with a significantly faster reader lock, leading to increased performance.

## Usage
```go
import "github.com/scalalang2/golang-fifo/v2"

size := 1e5
cache := fifo.NewSieve[string, string](size)

cache.Set("hello", "world")
cache.Get("hello") // => "world"
```

## Apendix

<details>
<summary>Why LRU Cache is not good enough?</summary>

- LRU is often implemented with a doubly linked list and a hash table, requiring two pointers per cache entry,
  which becomes large overhead when the object is small.
- It promotes objects to the head of the queue upon cache hit, which performs at least six random memory accesses
  protected by lock, which limits the scalability.
</details>

<details>
<summary>Brief overview of SIEVE & S3-FIFO</summary>

Various workloads typically follows **Power law distribution (e.g. Zipf's law)** as shown in the following figure.

![zipflaw_discovered_by_realworld](./docs/zipf_law_discovered_by_realworld_traces.png)

The analysis reveals that most requests are "one-hit-wonders", which means it's accessed only once.
Consequently, a cache eviction strategy should quickly remove most objects after insertion.

**S3-FIFO** and **SIEVE** achieves this goal with simplicity, efficiency, and scalability using simple FIFO queue only.

![s3-fifo-is-powerful-algorithm](./docs/graphs_shows_s3_fifo_is_powerful.png)
</details>

## Contribution
How to run unit test
```bash
$ go test -v ./...
```

How to run benchmark test
```bash
$ go test -bench=. -benchtime=10s
```
