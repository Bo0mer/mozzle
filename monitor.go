package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	cfevent "github.com/cloudfoundry/sonde-go/events"
)

type appNotFoundError string

func (e appNotFoundError) Error() string {
	return fmt.Sprintf("application %s not found", string(e))
}

type AppMonitor struct {
	cloudFoundryClient *cfclient.Client
	firehose           *consumer.Consumer
	errLog             *log.Logger

	refreshInterval time.Duration

	mu        sync.Mutex // guards
	monitored map[AppMetadata]struct{}
}

func NewAppMonitor(cf *cfclient.Client, firehose *consumer.Consumer, errLog *log.Logger) *AppMonitor {
	return &AppMonitor{
		cloudFoundryClient: cf,
		firehose:           firehose,
		errLog:             errLog,

		refreshInterval: time.Second * 5,

		monitored: make(map[AppMetadata]struct{}),
	}
}

func (m *AppMonitor) Monitor(org, space string) error {
	targetSpace, err := m.space(org, space)
	if err != nil {
		return err
	}

	for range time.Tick(m.refreshInterval) {
		apps, err := m.spaceApps(targetSpace.Guid)
		if err != nil {
			m.errLog.Printf("error fetching apps: %v\n", err)
			continue
		}
		m.mu.Lock()
		for _, app := range apps {
			appMetadata := AppMetadata{
				Org:   org,
				Space: space,
				Guid:  app.Guid,
				Name:  app.Name,
			}
			if _, ok := m.monitored[appMetadata]; ok {
				continue
			}
			m.monitored[appMetadata] = struct{}{}
			go m.monitorApp(appMetadata)
		}
		m.mu.Unlock()
	}

	return nil
}

func (m *AppMonitor) monitorApp(app AppMetadata) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		m.mu.Lock()
		delete(m.monitored, app)
		m.mu.Unlock()
		cancel()
	}()

	go m.monitorFirehose(ctx, app)
	for range time.Tick(time.Second * 5) {
		summary, err := m.appSummary(app.Guid)
		if err != nil {
			if _, ok := err.(appNotFoundError); ok {
				return
			}
			m.errLog.Printf("error fetching app summary: %v\n", err)
			continue
		}
		ApplicationMetrics{summary, app}.Emit()
	}
}

func (m *AppMonitor) monitorFirehose(ctx context.Context, app AppMetadata) {
	authToken, err := m.cloudFoundryClient.GetToken()
	if err != nil {
		return
	}

	msgChan, errorChan := m.firehose.StreamWithoutReconnect(app.Guid, authToken)
	for {
		select {
		case event := <-msgChan:
			switch event.GetEventType() {
			case cfevent.Envelope_ContainerMetric:
				ContainerMetrics{event.GetContainerMetric(), app}.Emit()
			case cfevent.Envelope_HttpStartStop:
				HTTPMetrics{event.GetHttpStartStop(), app}.Emit()
			}
		case <-ctx.Done():
			m.errLog.Printf("stopping firehose monitor for app %s due to: %v",
				app.Guid, ctx.Err())
			return
		case err := <-errorChan:
			m.errLog.Printf("error streaming from firehose: %v\n", err)
		}
	}
}

func (m *AppMonitor) appSummary(appGuid string) (AppSummary, error) {
	path := fmt.Sprintf("/v2/apps/%s/summary", appGuid)
	req := m.cloudFoundryClient.NewRequest("GET", path)
	resp, err := m.cloudFoundryClient.DoRequest(req)
	if err != nil {
		return AppSummary{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return AppSummary{}, appNotFoundError(appGuid)
	}

	var s AppSummary
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return AppSummary{}, err
	}
	return s, nil
}

func (m *AppMonitor) spaceApps(guid string) ([]cfclient.App, error) {
	spaceApps := make([]cfclient.App, 0)
	apps, err := m.cloudFoundryClient.ListApps()
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		if app.SpaceData.Meta.Guid == guid {
			spaceApps = append(spaceApps, app)
		}
	}
	return spaceApps, nil
}

func (m *AppMonitor) space(orgName, spaceName string) (cfclient.Space, error) {
	var targetOrg cfclient.Org
	orgs, err := m.cloudFoundryClient.ListOrgs()
	if err != nil {
		return cfclient.Space{}, err
	}
	for _, org := range orgs {
		if org.Name == orgName {
			targetOrg = org
			break
		}
	}

	spaces, err := m.cloudFoundryClient.OrgSpaces(targetOrg.Guid)
	if err != nil {
		return cfclient.Space{}, err
	}

	for _, space := range spaces {
		if space.Name == spaceName {
			return space, nil
		}
	}
	return cfclient.Space{}, fmt.Errorf("space %s not found", spaceName)
}
