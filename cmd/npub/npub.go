package main

import (
	"flag"
	"log"

	"net/http"
	_ "net/http/pprof"

	nats "github.com/nats-io/go-nats"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/znly/natsworkout"
	"github.com/znly/natsworkout/nanotime"
)

var (
	numSubs = flag.Int("numsubs", 1, "number of subscriptions")

	debugAddr = flag.String("debugport", ":0", "debug addr")
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

	subs := make([]string, *numSubs)
	for i := 0; i < *numSubs; i++ {
		subs[i] = wc.Next()
	}

	// timer := time.NewTicker(10 * time.Millisecond)
	// defer timer.Stop()

	var msg natsworkout.Message
	wbuf := make([]byte, 1024*1024)
	buf := make([]byte, 4096*1024)
	var seq uint64
	for i := 0; ; i++ {
		// <-timer.C
		var n int
		for i := 0; i < 3; i++ {
			w := wc.Next()

			if n+len(w) >= len(wbuf) {
				break
			}

			copy(wbuf[n:], w)
			n += len(w)
		}

		msg.TimestampNano = uint64(nanotime.Offset())
		msg.Seq = seq
		seq++

		msg.Payload = wbuf[:n]

		n, err := msg.MarshalTo(buf)
		if err != nil {
			panic(err)
		}

		if err := nc.Publish(subs[i%len(subs)], buf[:n]); err != nil {
			panic(err)
		}

		nc.Flush()
	}

	log.Println("started.")

	<-make(chan struct{})
}
