package main

import (
	"flag"

	nats "github.com/nats-io/go-nats"
)

var (
	numSubs = flag.Int("numsubs", 1, "number of subscriptions")
)

func main() {

	nats.Connect()

}
