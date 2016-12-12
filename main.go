package main

import (
	"crypto/tls"
	"flag"
	"log"
	"os"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
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
	Initialize(riemannAddr, float32(eventsTtl), queueSize)
	cf, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        apiAddr,
		Username:          username,
		Password:          password,
		SkipSslValidation: insecure,
	})

	if err != nil {
		log.Fatal(err)
	}

	firehose := consumer.New(cf.Endpoint.DopplerEndpoint, &tls.Config{InsecureSkipVerify: insecure}, nil)
	m := NewAppMonitor(cf, firehose, log.New(os.Stdout, "mozzle: ", 0))

	if err := m.Monitor(org, space); err != nil {
		log.Fatal(err)
	}
}
