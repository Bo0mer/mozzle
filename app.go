package mozzle

import "time"

type appMetadata struct {
	Org   string
	Space string
	Guid  string
	Name  string
}

type appSummary struct {
	Id               string `json:"guid"`
	Name             string `json:"name"`
	Space            string `json:"space_guid"`
	Diego            bool   `json:"diego"`
	Memory           int32  `json:"memory"`
	Instances        int    `json:"instances"`
	RunningInstances int    `json:"running_instances"`
	DiskQuota        int32  `json:"disk_quota"`
	State            string `json:"state"`
}

type appEvent struct {
	Type      string    `json:"type"`
	ActorName string    `json:"actor_name"`
	ActorType string    `json:"actor_type"`
	ActeeName string    `json:"actee_name"`
	ActeeType string    `json:"actee_type"`
	Timestamp time.Time `json:"timestamp"`
}
