package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/cloudfoundry/noaa/consumer"
)

var (
	apiAddr     string
	insecure    bool
	username    string
	password    string
	appGuid     string
	riemannAddr string
	environment string

	interval  int
	eventsTtl float64
	queueSize int
)

func init() {
	flag.StringVar(&apiAddr, "api", "https://api.bosh-lite.com", "Address of the Cloud Foundry API")
	flag.BoolVar(&insecure, "insecure", false, "Please, please, don't!")
	flag.StringVar(&username, "username", "admin", "Cloud Foundry user")
	flag.StringVar(&password, "password", "admin", "Cloud Foundry password")
	flag.StringVar(&appGuid, "app-guid", "", "Cloud Foundry application GUID")

	flag.StringVar(&riemannAddr, "riemann", "127.0.0.1:5555", "Address of the Riemann endpoint")
	flag.StringVar(&environment, "environment", "", "Environment, e.g. test, staging, prod")

	flag.IntVar(&interval, "interval", 5, "Interval (in seconds) between reports")
	flag.Float64Var(&eventsTtl, "events-ttl", 30.0, "TTL for emitted events (in seconds)")
	flag.IntVar(&queueSize, "events-queue-size", 256, "Queue size for outgoing events")
}

func main() {
	flag.Parse()
	cf, err := NewCloudFoundry(Target{
		API:                       apiAddr,
		Username:                  username,
		Password:                  password,
		InsecureSkipSSLValidation: insecure,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "mozzle: error creating cf client: %v\n", err)
		os.Exit(1)
	}

	app, err := cf.Summary(appGuid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mozzle: error retrieving app info: %v\n", err)
		os.Exit(1)
	}

	authToken, err := cf.client.GetToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mozzle: error retrieving token: %v\n", err)
		os.Exit(1)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
	}
	c := consumer.New(cf.DopplerEndpoint(), tlsConfig, http.ProxyFromEnvironment)
	c.RefreshTokenFrom(cf)
	msgChan, errorChan := c.Stream(appGuid, authToken)

	go func() {
		for err := range errorChan {
			fmt.Fprintf(os.Stderr, "mozzle: error received: %v\n", err.Error())
		}
	}()

	Initialize(riemannAddr, app.Name, environment, float32(interval), queueSize)
	Emit(msgChan)
}
