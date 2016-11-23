package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type CloudFoundry struct {
	target Target
	client *cfclient.Client
}

type Target struct {
	API                       string
	Username                  string
	Password                  string
	InsecureSkipSSLValidation bool
}

func NewCloudFoundry(target Target) (*CloudFoundry, error) {
	client, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        target.API,
		Username:          target.Username,
		Password:          target.Password,
		HttpClient:        http.DefaultClient,
		SkipSslValidation: target.InsecureSkipSSLValidation,
	})

	if err != nil {
		return nil, err
	}

	return &CloudFoundry{
		target: target,
		client: client,
	}, nil
}

// Summary returns a summary for application.
func (cf *CloudFoundry) Summary(guid string) (AppSummaryResponse, error) {
	path := fmt.Sprintf("/v2/apps/%s/summary", guid)
	req := cf.client.NewRequest("GET", path)
	resp, err := cf.client.DoRequest(req)
	if err != nil {
		return AppSummaryResponse{}, err
	}
	defer resp.Body.Close()

	var s AppSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return AppSummaryResponse{}, err
	}
	return s, nil
}

func (cf *CloudFoundry) Stats(guid string) (AppStatsResponse, error) {
	path := fmt.Sprintf("/v2/apps/%s/stats", guid)
	req := cf.client.NewRequest("GET", path)
	resp, err := cf.client.DoRequest(req)
	if err != nil {
		return AppStatsResponse{}, err
	}
	defer resp.Body.Close()

	var s AppStatsResponse = make(map[string]InstanceResponse)
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return AppStatsResponse{}, err
	}
	return s, nil
}

type AppSummaryResponse struct {
	Name             string `json:"name"`
	Space            string `json:"space_guid"`
	Diego            bool   `json:"diego"`
	Memory           int32  `json:"memory"`
	Instances        int    `json:"instances"`
	RunningInstances int    `json:"running_instances"`
	DiskQuota        int32  `json:"disk_quota"`
	State            string `json:"state"`
}

type AppStatsResponse map[string]InstanceResponse

type InstanceResponse struct {
	State string        `json:"state"`
	Stats InstanceStats `json:"stats"`
}

type InstanceStats struct {
	DiskQuota uint64     `json:"disk_quota"`
	FdsQuota  uint64     `json:"fds_quota"`
	Host      string     `json:"host"`
	MemQuota  uint64     `json:"mem_quota"`
	Uptime    int32      `json:"uptime"`
	Usage     UsageStats `json:"usage"`
}

type UsageStats struct {
	CPU    float64 `json:"cpu"`
	Disk   uint64  `json:"disk"`
	Memory uint64  `json:"mem"`
}
