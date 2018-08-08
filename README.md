Run the subscribers:

```bash
(for i in $(seq 10); do nrsub -numsubs=5000 -debugaddr=:$(expr $i + 8000) & ; sleep 1; done) |& tee clients.log
```

Run the publisher:

```bash
npub -rate=8000 -numtopics=1000
```

Changing the cardinality of topics makes NATS to fall behind:


```bash
npub -rate=8000 -numtopics=10000

```


run prometheus:

```
docker run --rm -p 9090:9090 -v $(pwd)/prom.yml:/etc/prometheus/prometheus.yml prom/prometheus
```

```
histogram_quantile(0.99, sum(rate(nats_latency_bucket[1m])) by (le)) * 1000
```
