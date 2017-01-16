// mozzle is a command-line utility which subscribes collects Cloud Foundry
// application events and emits them to Riemann.
package main

import (
	"context"
	"flag"
	"fmt"
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

	eventsTTL     float64
	queueSize     int
	reportVersion bool
)

// populated using -ldflags.
var (
	version    string
	build      string
	buildstamp string
)

func init() {
	flag.StringVar(&apiAddr, "api", "https://api.bosh-lite.com", "Address of the Cloud Foundry API")
	flag.BoolVar(&insecure, "insecure", false, "Please, please, don't!")
	flag.StringVar(&username, "username", "admin", "Cloud Foundry user")
	flag.StringVar(&password, "password", "admin", "Cloud Foundry password")
	flag.StringVar(&org, "org", "NASA", "Cloud Foundry organization")
	flag.StringVar(&space, "space", "rocket", "Cloud Foundry space")

	flag.StringVar(&riemannAddr, "riemann", "127.0.0.1:5555", "Address of the Riemann endpoint")

	flag.Float64Var(&eventsTTL, "events-ttl", 30.0, "TTL for emitted events (in seconds)")
	flag.IntVar(&queueSize, "events-queue-size", 256, "Queue size for outgoing events")
	flag.BoolVar(&reportVersion, "v", false, "Report mozzle version")
	flag.BoolVar(&reportVersion, "version", false, "Report mozzle version")
}

func main() {
	flag.Parse()
	if reportVersion {
		printVersion()
		os.Exit(0)
	}
	riemann := &mozzle.RiemannEmitter{}
	riemann.Initialize(riemannAddr, float32(eventsTTL), queueSize)
	defer func() {
		if err := riemann.Close(); err != nil {
			fmt.Printf("mozzle: error closing riemann emitter: %v\n", err)
		}
	}()

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

	if err := mozzle.Monitor(ctx, t, riemann); err != nil {
		fmt.Printf("mozzle: error occured during Monitor: %v\n", err)
	}
}

func printVersion() {
	fmt.Printf("mozzle version %s build %s at %s\n", version, build, buildstamp)
}
