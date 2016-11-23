package main

import "strconv"

type CFApper struct {
	cf      *CloudFoundry
	appGuid string
}

func NewCFApper(cf *CloudFoundry, appGuid string) *CFApper {
	return &CFApper{
		cf:      cf,
		appGuid: appGuid,
	}
}

func (a *CFApper) App() (App, error) {
	summary, err := a.cf.Summary(a.appGuid)
	if err != nil {
		return App{}, err
	}

	stats, err := a.cf.Stats(a.appGuid)
	if err != nil {
		return App{}, err
	}

	instances, envSummary := instances(stats)
	app := App{
		GUID:  a.appGuid,
		Name:  summary.Name,
		Space: summary.Space,

		State: summary.State,
		InstanceCount: InstanceCount{
			Configured: summary.Instances,
			Running:    summary.RunningInstances,
		},
		Instances:          instances,
		EnvironmentSummary: envSummary,
	}

	return app, nil
}

func instances(s AppStatsResponse) ([]Instance, EnvironmentSummary) {
	instances := make([]Instance, len(s))
	var summary EnvironmentSummary
	i := 0
	for idx, instance := range s {
		intIdx, _ := strconv.Atoi(idx)
		instances[i] = Instance{
			Index:  int64(intIdx),
			CellIP: instance.Stats.Host,

			CPUUsage:        instance.Stats.Usage.CPU,
			DiskUsage:       instance.Stats.Usage.Disk,
			DiskAvailable:   instance.Stats.DiskQuota,
			MemoryUsage:     instance.Stats.Usage.Memory,
			MemoryAvailable: instance.Stats.MemQuota,

			Uptime: instance.Stats.Uptime,
			State:  instance.State,
		}

		summary.TotalCPU += instance.Stats.Usage.CPU

		summary.TotalDiskConfigured += instance.Stats.DiskQuota
		summary.TotalDiskUsage += instance.Stats.Usage.Disk

		summary.TotalMemoryConfigured += instance.Stats.MemQuota
		summary.TotalMemoryUsage += instance.Stats.Usage.Memory

		i++
	}
	return instances, summary
}
