package main

import (
	"flag"
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
	log.Printf("latency=%s", delta)

	if l := *simulatedLatency; l > 0 {
		time.Sleep(l)
	}

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

	options := nats.Options{}
	options.Url = "nats://localhost:4222"

	options.ReconnectedCB = func(nc *nats.Conn) {
		log.Println("NATS: Got reconnected")
	}

	options.AsyncErrorCB = func(nc *nats.Conn, s *nats.Subscription, err error) {
		log.Printf("NATS: async error callback subject=%s queue=%s err=%v", s.Subject, s.Queue, err)
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
	handlers := make([]*handler, *numSubs)

	for i := 0; i < len(handlers); i++ {
		subject := wc.Next()
		handler, err := newHandler(nc, subject)
		if err != nil {
			log.Fatalf("could not subscribe: %v", err)
		}

		handlers[i] = handler
	}

	log.Println("started.")

	<-make(chan struct{})
}
