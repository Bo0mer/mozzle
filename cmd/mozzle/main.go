package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/Bo0mer/mozzle"
)

var (
	apiAddr  string
	insecure bool
	username string
	password string
	org      string
	space    string

	riemannAddr string

	eventsTtl float64
	queueSize int
)

func init() {
	flag.StringVar(&apiAddr, "api", "https://api.bosh-lite.com", "Address of the Cloud Foundry API")
	flag.BoolVar(&insecure, "insecure", false, "Please, please, don't!")
	flag.StringVar(&username, "username", "admin", "Cloud Foundry user")
	flag.StringVar(&password, "password", "admin", "Cloud Foundry password")
	flag.StringVar(&org, "org", "NASA", "Cloud Foundry organization")
	flag.StringVar(&space, "space", "rocket", "Cloud Foundry space")

	flag.StringVar(&riemannAddr, "riemann", "127.0.0.1:5555", "Address of the Riemann endpoint")

	flag.Float64Var(&eventsTtl, "events-ttl", 30.0, "TTL for emitted events (in seconds)")
	flag.IntVar(&queueSize, "events-queue-size", 256, "Queue size for outgoing events")
}

func main() {
	flag.Parse()
	mozzle.Initialize(riemannAddr, float32(eventsTtl), queueSize)
	t := mozzle.Target{
		API:      apiAddr,
		Username: username,
		Password: password,
		Insecure: insecure,
		Org:      org,
		Space:    space,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig
		fmt.Println("exiting...")
		cancel()
	}()

	if err := mozzle.Monitor(ctx, t); err != nil {
		log.Fatal(err)
	}
}
