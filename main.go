package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/quipo/statsd"
)

var (
	apiAddr    string
	username   string
	password   string
	appGuid    string
	statsdAddr string

	interval int
)

func init() {
	flag.StringVar(&apiAddr, "api", "", "Address of the Cloud Foundry API")
	flag.StringVar(&username, "username", "admin", "Cloud Foundry user")
	flag.StringVar(&password, "password", "admin", "Cloud Foundry password")
	flag.StringVar(&appGuid, "app-guid", "", "Cloud Foundry application GUID")
	flag.StringVar(&statsdAddr, "statsd", "127.0.0.1:8125", "Address of the statsd endpoint")
	flag.IntVar(&interval, "interval", 5, "Interval (in seconds) between reports")
}

func main() {
	flag.Parse()
	cf, err := NewCloudFoundry(Target{
		API:      apiAddr,
		Username: username,
		Password: password,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "mozzle: error creating cf client: %v\n", err)
		os.Exit(1)
	}

	apper := NewCFApper(cf, appGuid)
	app, err := apper.App()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mozzle: error initializing app info: %v", err)
		os.Exit(1)
	}

	c := errStatsdClient{statsd.NewStatsdClient(statsdAddr, app.Name), nil}
	if err := c.CreateSocket(); err != nil {
		// TODO(ivan): retry if err is temporary
		fmt.Fprintf(os.Stderr, "mozzle: error connecting: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	d := time.Duration(interval) * time.Second
	for range time.Tick(d) {
		app, err := apper.App()
		if err != nil {
			fmt.Fprintf(os.Stderr, "mozzle: error fetching app info: %v\n", err)
			continue
		}

		summary := app.EnvironmentSummary
		c.FGauge("overall cpu_percent", summary.TotalCPU)

		c.Gauge("overall memory total_bytes", int64(summary.TotalMemoryConfigured))
		c.Gauge("overall memory used_bytes", int64(summary.TotalMemoryUsage))
		ratio := float64(summary.TotalMemoryUsage) / float64(summary.TotalMemoryConfigured)
		c.FGauge("overall memory used_ratio", ratio)

		c.Gauge("overall disk total_bytes", int64(summary.TotalDiskConfigured))
		c.Gauge("overall disk used_bytes", int64(summary.TotalDiskUsage))
		ratio = float64(summary.TotalDiskUsage) / float64(summary.TotalDiskConfigured)
		c.FGauge("oerall disk used_ratio", ratio)

		c.Gauge("overall instance configured_count", int64(app.InstanceCount.Configured))
		c.Gauge("overall instance running_count", int64(app.InstanceCount.Running))
		ratio = float64(app.InstanceCount.Running) / float64(app.InstanceCount.Configured)
		c.FGauge("overall instance availability_ratio", ratio)

		for _, instance := range app.Instances {
			instancePrefix := fmt.Sprintf("instance %d ", instance.Index)
			c.Gauge(instancePrefix+"memory total_bytes", int64(instance.MemoryAvailable))
			c.Gauge(instancePrefix+"memory used_bytes", int64(instance.MemoryUsage))
			ratio = float64(instance.MemoryUsage) / float64(instance.MemoryAvailable)
			c.FGauge(instancePrefix+"memory used_ratio", ratio)

			c.Gauge(instancePrefix+"disk total_bytes", int64(instance.DiskAvailable))
			c.Gauge(instancePrefix+"disk used_bytes", int64(instance.DiskUsage))
			ratio = float64(instance.DiskUsage) / float64(instance.DiskAvailable)
			c.FGauge(instancePrefix+"disk used_ratio", ratio)

			c.FGauge(instancePrefix+"cpu_percent", instance.CPUUsage)
		}

		if err := c.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "mozzle: error sending metric: %v\n", err)
		}
	}
}

// errStatsdClient records the last non-nil error returned by a StatsdClient.
type errStatsdClient struct {
	*statsd.StatsdClient

	err error
}

// Gauge calls the underlying Gauge method and records any non-nil errors.
func (c *errStatsdClient) Gauge(stat string, value int64) error {
	err := c.StatsdClient.Gauge(stat, value)
	if err != nil {
		c.err = err
	}
	return err
}

// FGauge calls the underlying Gauge method and records any non-nil errors.
func (c *errStatsdClient) FGauge(stat string, value float64) error {
	err := c.StatsdClient.FGauge(stat, value)
	if err != nil {
		c.err = err
	}
	return err
}

// Err reads and resets the last recorded non-nil error.
func (c *errStatsdClient) Err() error {
	err := c.err
	c.err = nil
	return err
}
