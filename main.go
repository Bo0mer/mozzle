package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	apiAddr     string
	insecure    bool
	username    string
	password    string
	appGuid     string
	riemannAddr string

	interval int
)

func init() {
	flag.StringVar(&apiAddr, "api", "", "Address of the Cloud Foundry API")
	flag.BoolVar(&insecure, "insecure", false, "Please, please, don't!")
	flag.StringVar(&username, "username", "admin", "Cloud Foundry user")
	flag.StringVar(&password, "password", "admin", "Cloud Foundry password")
	flag.StringVar(&appGuid, "app-guid", "", "Cloud Foundry application GUID")
	flag.StringVar(&riemannAddr, "riemann", "127.0.0.1:5555", "Address of the Riemann endpoint")
	flag.IntVar(&interval, "interval", 5, "Interval (in seconds) between reports")
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

	apper := NewCFApper(cf, appGuid)
	app, err := apper.App()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mozzle: error initializing app info: %v", err)
		os.Exit(1)
	}

	Initialize(riemannAddr, app.Name, "", 30.0, 256)

	d := time.Duration(interval) * time.Second
	for range time.Tick(d) {
		app, err := apper.App()
		if err != nil {
			fmt.Fprintf(os.Stderr, "mozzle: error fetching app info: %v\n", err)
			continue
		}

		summary := app.EnvironmentSummary
		Emit("overall cpu percent", summary.TotalCPU)

		Emit("overall memory total_bytes", int(summary.TotalMemoryConfigured))
		Emit("overall memory used_bytes", int(summary.TotalMemoryUsage))
		ratio := float64(summary.TotalMemoryUsage) / float64(summary.TotalMemoryConfigured)
		Emit("overall memory used_ratio", ratio)

		Emit("overall disk total_bytes", int(summary.TotalDiskConfigured))
		Emit("overall disk used_bytes", int(summary.TotalDiskUsage))
		ratio = float64(summary.TotalDiskUsage) / float64(summary.TotalDiskConfigured)
		Emit("overall disk used_ratio", ratio)

		Emit("overall instance configured_count", int(app.InstanceCount.Configured))
		Emit("overall instance running_count", int(app.InstanceCount.Running))
		ratio = float64(app.InstanceCount.Running) / float64(app.InstanceCount.Configured)
		Emit("overall instance availability_ratio", ratio)

		for _, instance := range app.Instances {
			instancePrefix := fmt.Sprintf("instance %d ", instance.Index)
			Emit(instancePrefix+"memory total_bytes", int(instance.MemoryAvailable))
			Emit(instancePrefix+"memory used_bytes", int(instance.MemoryUsage))
			ratio = float64(instance.MemoryUsage) / float64(instance.MemoryAvailable)
			Emit(instancePrefix+"memory used_ratio", ratio)

			Emit(instancePrefix+"disk total_bytes", int(instance.DiskAvailable))
			Emit(instancePrefix+"disk used_bytes", int(instance.DiskUsage))
			ratio = float64(instance.DiskUsage) / float64(instance.DiskAvailable)
			Emit(instancePrefix+"disk used_ratio", ratio)

			Emit(instancePrefix+"cpu_percent", instance.CPUUsage)
		}
	}
}
