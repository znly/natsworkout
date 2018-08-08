package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"net/http"
	_ "net/http/pprof"

	nats "github.com/nats-io/go-nats"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/znly/natsworkout"
	"github.com/znly/natsworkout/nanotime"
)

var (
	numSubs = flag.Int("numsubs", 1, "number of subscriptions")

	latency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "nats_latency",
		Help: "NATS latency",
		Buckets: []float64{
			0.000001,
			0.000005,
			0.00001,
			0.00005,
			0.0001,
			0.0005,
			0.001,
			0.002,
			0.003,
			0.005,
			0.010,
			0.020,
			0.030,
			0.050,
			0.100,
			0.200,
			0.300,
			0.500,
			0.750,
			1.000,
			2.000,
			3.000,
			4.000,
			5.000,
		},
	})
)

func init() {
	prometheus.MustRegister(latency)
}

type handler struct {
	*nats.Subscription
	msg natsworkout.Message
}

func newHandler(nc *nats.Conn, subj string) (*handler, error) {
	var (
		err     error
		handler handler
	)

	handler.Subscription, err = nc.Subscribe(subj, handler.handle)

	if err == nil {
		handler.Subscription.SetPendingLimits(-1, -1)
	}

	go func() {
		var dropped int
		for {
			<-time.After(3 * time.Second)
			n, err := handler.Subscription.Dropped()
			if err != nil {
				return
			}
			if n != dropped {
				log.Printf("dropped=%d", n)
			}
			dropped = n
		}
	}()

	if err != nil {
		return nil, err
	}
	return &handler, nil
}

func (h *handler) handle(msg *nats.Msg) {
	if err := h.msg.UnmarshalFrom(msg.Data); err != nil {
		panic(err)
	}

	delta := (nanotime.Offset() - time.Duration(h.msg.TimestampNano)) * time.Nanosecond
	latency.Observe(float64(delta) / float64(time.Second))
	// log.Printf("latency=%s", delta)

	// if l := *simulatedLatency; l > 0 {
	// 	time.Sleep(l)
	// }
}

var (
	debugAddr        = flag.String("debugaddr", ":8080", "debug addr")
	simulatedLatency = flag.Duration("latency", 0, "simulated processing latency")
)

func main() {

	go func() {
		log.Printf("debugaddr=%s", *debugAddr)
		http.DefaultServeMux.Handle("/metrics", prometheus.UninstrumentedHandler())
		err := http.ListenAndServe(*debugAddr, nil)
		if err != nil {
			panic(err)
		}
	}()

	if !flag.Parsed() {
		flag.Parse()
	}

	options := nats.DefaultOptions
	options.Url = "nats://localhost:4222"

	options.ReconnectedCB = func(nc *nats.Conn) {
		log.Printf("NATS: Got reconnected: %v", nc.LastError())
	}

	options.AsyncErrorCB = func(nc *nats.Conn, s *nats.Subscription, err error) {
		log.Printf("NATS: async error callback subject=%s queue=%s err=%v", s.Subject, s.Queue, err)
	}

	options.ClosedCB = func(nc *nats.Conn) {
		log.Printf("NATS: closed callback: %v", nc.LastError())
	}

	nc, err := options.Connect()
	if err != nil {
		log.Fatalf("connection error: %v", err)
	}

	words, err := natsworkout.OpenWords()
	if err != nil {
		log.Fatalf("failed to open words: %v", err)
	}

	go func() {
		for i := uint64(0); ; i++ {
			<-time.After(2 * time.Second)
			msg := &natsworkout.Message{
				TimestampNano: uint64(nanotime.Offset()),
				Seq:           i,
			}
			start := time.Now()
			data, _ := msg.MarshalBinary()
			if err := nc.Publish("controlz", data); err != nil {
				log.Printf("control error: %v", err)
			} else {
				log.Printf("controlz latency=%s", time.Since(start))
			}
		}
	}()

	wc := words.Cursor()
	subjects := make([]string, *numSubs)
	for i := 0; i < len(subjects); i++ {
		subjects[i] = wc.Next()
	}

	handlers := make([]*handler, *numSubs)

	log.Printf("start subscriptions=%d handlers=%d", len(subjects), len(handlers))

	go func() {
		for j := 0; ; j++ {
			for i := 0; i < len(handlers); i++ {
				subj := subjects[i%len(subjects)]
				log.Printf("subscribe to=%s", subj)
				handler, err := newHandler(nc, subj)
				if err != nil {
					log.Fatalf("could not subscribe: %v", err)
				}

				handlers[i] = handler
			}

			<-time.After(3 * time.Second)

			for i := 0; i < len(handlers); i++ {
				log.Printf("unsubscribe from=%s", handlers[i].Subject)
				t := timeit(fmt.Sprintf("unsubscribe from=%s duration=%%s", handlers[i].Subject))
				if err := handlers[i].Unsubscribe(); err != nil {
					log.Printf("unsubscribe error=%v", err)
				}
				t()
			}
		}

	}()

	log.Println("started.")

	<-make(chan struct{})
}

func timeit(format string) func() {
	start := nanotime.Offset()
	return func() {
		log.Printf(format, nanotime.Offset()-start)
	}
}
