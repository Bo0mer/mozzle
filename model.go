package main

type InstanceCount struct {
	Configured int `json:"configured"`
	Running    int `json:"running"`
}

type Instance struct {
	Index  int64  `json:"index"`
	CellIP string `json:"cell_ip"`

	CPUUsage        float64 `json:"cpu_usage"`  //CPUPercentage
	DiskUsage       uint64  `json:"disk_usage"` //DiskBytes
	DiskAvailable   uint64  `json:"disk_available"`
	MemoryUsage     uint64  `json:"memory_usage"` //MemBytes
	MemoryAvailable uint64  `json:"memory_available"`

	Uptime int32  `json:"uptime"`
	State  string `json:"state"`
}

type EnvironmentSummary struct {
	TotalCPU float64 `json:"total_cpu"`

	TotalDiskConfigured uint64 `json:"total_disk_configured"`
	TotalDiskUsage      uint64 `json:"total_disk_usage"`

	TotalMemoryConfigured uint64 `json:"total_memory_configured"`
	TotalMemoryUsage      uint64 `json:"total_memory_usage"`
}

type App struct {
	GUID         string `json:"guid"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Space        string `json:"space"`

	Buildpack string `json:"buildpack"`
	Diego     bool   `json:"diego"`

	State              string             `json:"state"`
	EnvironmentSummary EnvironmentSummary `json:"environment_summary"`
	InstanceCount      InstanceCount      `json:"instance_count"`
	Instances          []Instance         `json:"instances"`
}
