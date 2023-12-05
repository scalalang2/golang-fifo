<h1 align="center">golang-fifo</h1>
<p align="center">
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/license-MIT-_red.svg"></a>
</p>

This is a modern cache implementation, **inspired** by the following papers, for providing high efficiency and.

- **S3-FIFO** | [FIFO queues are all you need for cache eviction](https://dl.acm.org/doi/10.1145/3600006.3613147) (SOSP'23)
- **SIEVE** | [SIEVE is Simpler than LRU: an Efficient Turn-Key Eviction Algorithm for Web Caches](https://junchengyang.com/publication/nsdi24-SIEVE.pdf) (NSDI'24)

`golanf-fifo` provides several cache eviction algorithms including S3-FIFO and SIEVE.

This offers state-of-the-art efficiency and scalability compared to other LRU-based cache algorithms.

### Why LRU Cache is not good enough?

- LRU is often implemented with a doubly linked list and a hash table, requiring two pointers per cache entry,
  which becomes large overhead when the object is small.
- It promotes objects to the head of the queue upon cache hit, which performs at least six random memory accesses
  protected by lock, which limits the scalability.

### S3-FIFO & SIEVE
Various workloads typically follows Power law distribution (e.g. Zipfian distribution) as shown in the following figure.

![zipflaw_discovered_by_realworld](../golang-fifo/docs/zipf_law_discovered_by_realworld_traces.png)

The analysis reveals that most requests are "one-hit-wonders", accessed only once.
Consequently, a cache eviction strategy should quickly remove most objects after insertion.

**S3-FIFO** and **SIEVE** achieves this goal with simplicity, efficiency, and scalability using simple FIFO queue only.

![s3-fifo-is-powerful-algorithm](../golang-fifo/docs/graphs_shows_s3_fifo_is_powerful.png)

### Contribution
How to run unit test
```bash
$ go test -v ./...
```

How to run benchmark test
```bash
$ go test -bench=. -benchtime=10s ./...
```
