package main

import (
	"flag"
	"log"
	"time"

	nats "github.com/nats-io/go-nats"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/znly/natsworkout"
)

var (
	numSubs = flag.Int("numsubs", 1, "number of subscriptions")

	latency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "nats_latency",
		Help: "NATS latency",
		Buckets: []float64{
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
		},
	})
)

type handler struct {
	*nats.Subscription
	msg natsworkout.Message
}

func newHandler(nc *nats.Conn, subj string) (*handler, error) {
	sub, err := nc.Subscribe(subj, cb)
	if err != nil {
		return nil, err
	}
	return &handler{Subscription: sub}, nil
}

func (h *handler) handle(msg *nats.Msg) {
	if err := h.msg.UnmarshalFrom(msg); err != nil {
		panic(err)
	}

	delta := time.Since(h.msg.Timestamp())
	latency.Observe(delta.Seconds())
}

func main() {

	options := nats.Options{}
	options.Url = "localhost:4222"

	options.ReconnectedCB = func(nc *nats.Conn) {
		log.Println("NATS: Got reconnected")
	}

	options.AsyncErrorCB = func(nc *nats.Conn, s *nats.Subscription, err error) {
		log.Printf("NATS: async error callback subject=%s queue=%s err=%v", subject, queue, err)
	}

	options.ClosedCB = func(nc *nats.Conn) {
		log.Println("NATS: closed callback")
	}

	nc, err := options.Connect()
	if err != nil {
		log.Fatalf("connection error: %v", err)
	}

	words, err := natsworkout.OpenWords()
	if err != nil {
		log.Fatalf("failed to open words: %v", err)
	}

	wc := words.Cursor()
	handlers := make([]*handler, numSubs)

	for i := 0; i < numSubs; i++ {
		subject := wc.Next()
		handler, err := newHandler(nc, subject)
		if err != nil {
			log.Fatalf("could not subscribe: %v", err)
		}

		handlers = append(handlers, handler)
	}

	log.Println("started.")

	<-make(chan struct{})
}
