package mozzle

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	cfevent "github.com/cloudfoundry/sonde-go/events"
)

// Target specifies a monitoring target.
type Target struct {
	API      string
	Username string
	Password string
	Insecure bool
	Org      string
	Space    string
}

type appNotFoundError string

func (e appNotFoundError) Error() string {
	return fmt.Sprintf("application %s not found", string(e))
}

type appMonitor struct {
	cloudFoundryClient *cfclient.Client
	firehose           *consumer.Consumer
	errLog             *log.Logger

	refreshInterval time.Duration

	mu        sync.Mutex // guards
	monitored map[appMetadata]struct{}
}

// Monitor monitors a target for events and sends them to Riemann.
func Monitor(ctx context.Context, t Target) error {
	cf, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        t.API,
		Username:          t.Username,
		Password:          t.Password,
		SkipSslValidation: t.Insecure,
	})
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: t.Insecure}
	firehose := consumer.New(cf.Endpoint.DopplerEndpoint, tlsConfig, nil)
	firehose.RefreshTokenFrom(tokenRefresher{cf})
	mon := appMonitor{
		cloudFoundryClient: cf,
		firehose:           firehose,
		errLog:             log.New(os.Stderr, "mozzle: ", 0),
		refreshInterval:    time.Second * 5,
		monitored:          make(map[appMetadata]struct{}),
	}

	return mon.Monitor(ctx, t.Org, t.Space)
}

func (m *appMonitor) Monitor(ctx context.Context, org, space string) error {
	targetSpace, err := m.space(org, space)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(m.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			apps, err := m.spaceApps(targetSpace.Guid)
			if err != nil {
				m.errLog.Printf("error fetching apps: %v\n", err)
				continue
			}
			m.mu.Lock()
			for _, app := range apps {
				appMetadata := appMetadata{
					Org:   org,
					Space: space,
					Guid:  app.Guid,
					Name:  app.Name,
				}
				if _, ok := m.monitored[appMetadata]; ok {
					continue
				}
				m.monitored[appMetadata] = struct{}{}
				go m.monitorApp(ctx, appMetadata)
			}
			m.mu.Unlock()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *appMonitor) monitorApp(ctx context.Context, app appMetadata) {
	monitorCtx, cancel := context.WithCancel(ctx)
	defer func() {
		m.mu.Lock()
		delete(m.monitored, app)
		m.mu.Unlock()
		cancel()
	}()

	go m.monitorFirehose(monitorCtx, app)

	ticker := time.NewTicker(m.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			m.emitAppSummary(app)
			m.emitAppEvents(app, now.Add(-1*m.refreshInterval))
		case <-ctx.Done():
			return
		}
	}
}

func (m *appMonitor) emitAppSummary(app appMetadata) {
	summary, err := m.appSummary(app.Guid)
	if err != nil {
		if _, ok := err.(appNotFoundError); ok {
			return
		}
		m.errLog.Printf("error fetching app summary: %v\n", err)
	}
	applicationMetrics{summary, app}.Emit()
}

func (m *appMonitor) emitAppEvents(app appMetadata, since time.Time) {
	events, err := m.appEventsSince(app, since)
	if err != nil {
		m.errLog.Printf("error fetching app events: %v\n", err)
		return
	}
	for _, event := range events {
		applicationEvent{event, app}.Emit()
	}
}

func (m *appMonitor) monitorFirehose(ctx context.Context, app appMetadata) {
	authToken, err := m.cloudFoundryClient.GetToken()
	if err != nil {
		return
	}

	msgChan, errorChan := m.firehose.Stream(app.Guid, authToken)
	for {
		select {
		case event := <-msgChan:
			switch event.GetEventType() {
			case cfevent.Envelope_ContainerMetric:
				containerMetrics{event.GetContainerMetric(), app}.Emit()
			case cfevent.Envelope_HttpStartStop:
				httpMetrics{event.GetHttpStartStop(), app}.Emit()
			}
		case <-ctx.Done():
			m.errLog.Printf("stopping firehose monitor for app %s due to: %v",
				app.Guid, ctx.Err())
			return
		case err, ok := <-errorChan:
			if !ok {
				m.errLog.Printf("firehose error chan closed, exiting\n")
				return
			}
			m.errLog.Printf("error streaming from firehose: %v\n", err)
		}
	}
}

func (m *appMonitor) appSummary(appGuid string) (appSummary, error) {
	path := fmt.Sprintf("/v2/apps/%s/summary", appGuid)
	req := m.cloudFoundryClient.NewRequest("GET", path)
	resp, err := m.cloudFoundryClient.DoRequest(req)
	if err != nil {
		return appSummary{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return appSummary{}, appNotFoundError(appGuid)
	}

	var s appSummary
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return appSummary{}, err
	}
	return s, nil
}

func (m *appMonitor) appEventsSince(app appMetadata, t time.Time) ([]appEvent, error) {
	var events []appEvent
	cf := m.cloudFoundryClient
	since := url.QueryEscape(t.String())
	path := fmt.Sprintf("/v2/events?q=actee:%s&q=timestamp:%s", app.Guid, since)
	req := cf.NewRequest("GET", path)
	resp, err := cf.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&events)
	return events, err
}

func (m *appMonitor) spaceApps(guid string) ([]cfclient.App, error) {
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

func (m *appMonitor) space(orgName, spaceName string) (cfclient.Space, error) {
	org, err := m.org(orgName)
	if err != nil {
		return cfclient.Space{}, err
	}

	spaces, err := m.cloudFoundryClient.OrgSpaces(org.Guid)
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

func (m *appMonitor) org(name string) (cfclient.Org, error) {
	cf := m.cloudFoundryClient

	path := fmt.Sprintf("/v2/organizations?q=name:%s", name)
	resp, err := cf.DoRequest(cf.NewRequest("GET", path))
	if err != nil {
		return cfclient.Org{}, fmt.Errorf("error requesting organizations: %v", err)
	}
	defer resp.Body.Close()

	var orgResp cfclient.OrgResponse
	d := json.NewDecoder(resp.Body)
	if err := d.Decode(&orgResp); err != nil {
		return cfclient.Org{}, fmt.Errorf("error decoding response: %v", err)
	}

	if orgResp.Count == 0 {
		return cfclient.Org{}, fmt.Errorf("org %q not found", name)
	}
	if orgResp.Count > 1 {
		return cfclient.Org{}, fmt.Errorf("name %q does not refer to a single org", name)
	}

	return cfclient.Org{
		Guid: orgResp.Resources[0].Meta.Guid,
		Name: orgResp.Resources[0].Entity.Name,
	}, nil

}

type tokenRefresher struct {
	*cfclient.Client
}

func (tr tokenRefresher) RefreshAuthToken() (string, error) {
	return tr.GetToken()
}
