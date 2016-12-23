package mozzle

import (
	"context"
	"crypto/tls"
	"log"
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

type appMonitor struct {
	cf       *cloudFoundry
	firehose *consumer.Consumer
	errLog   *log.Logger

	refreshInterval time.Duration

	mu        sync.Mutex // guards
	monitored map[application]struct{}
}

// Monitor monitors a target for events and sends them to Riemann.
func Monitor(ctx context.Context, t Target) error {
	cfClient, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        t.API,
		Username:          t.Username,
		Password:          t.Password,
		SkipSslValidation: t.Insecure,
	})
	if err != nil {
		return err
	}

	cf := &cloudFoundry{cfClient}
	tlsConfig := &tls.Config{InsecureSkipVerify: t.Insecure}
	firehose := consumer.New(cf.DopplerEndpoint(), tlsConfig, nil)
	firehose.RefreshTokenFrom(cf)

	mon := appMonitor{
		cf:              cf,
		firehose:        firehose,
		errLog:          log.New(os.Stderr, "mozzle: ", 0),
		refreshInterval: time.Second * 5,
		monitored:       make(map[application]struct{}),
	}

	return mon.Monitor(ctx, t.Org, t.Space)
}

func (m *appMonitor) Monitor(ctx context.Context, org, space string) error {
	ticker := time.NewTicker(m.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			apps, err := m.cf.Apps(org, space)
			if err != nil {
				m.errLog.Printf("error fetching apps: %v\n", err)
				continue
			}
			m.mu.Lock()
			for _, app := range apps {
				if _, ok := m.monitored[app]; ok {
					continue
				}
				m.monitored[app] = struct{}{}
				go m.monitorApp(ctx, app)
			}
			m.mu.Unlock()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *appMonitor) monitorApp(ctx context.Context, app application) {
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
			if err := m.emitAppSummary(app); isAppNotFound(err) {
				return
			}
			m.emitAppEvents(app, now.Add(-1*m.refreshInterval))
		case <-ctx.Done():
			return
		}
	}
}

func (m *appMonitor) monitorFirehose(ctx context.Context, app application) {
	authToken, err := m.cf.RefreshAuthToken()
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
func (m *appMonitor) emitAppSummary(app application) error {
	summary, err := m.cf.AppSummary(app.Guid)
	if err != nil {
		m.errLog.Printf("error fetching app summary: %v\n", err)
		return err
	}
	applicationMetrics{summary, app}.Emit()
	return nil
}

func (m *appMonitor) emitAppEvents(app application, since time.Time) {
	events, err := m.appEventsSince(app, since)
	if err != nil {
		m.errLog.Printf("error fetching app events: %v\n", err)
		return
	}
	for _, event := range events {
		applicationEvent{event, app}.Emit()
	}
}

func (m *appMonitor) appEventsSince(app application, t time.Time) ([]appEvent, error) {
	acteeQuery := query{
		Filter:   FilterActee,
		Operator: OperatorEqual,
		Value:    app.Guid,
	}
	timestampQuery := query{
		Filter:   FilterTimestamp,
		Operator: OperatorGreater,
		Value:    t.String(),
	}
	return m.cf.Events(acteeQuery, timestampQuery)
}

func isAppNotFound(err error) bool {
	_, ok := err.(appNotFoundError)
	return ok
}
