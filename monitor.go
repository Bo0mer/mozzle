package mozzle

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/Bo0mer/ccv2"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

// DefaultRPCTimeout is the default timeout when making remote calls.
const DefaultRPCTimeout = 5 * time.Second

// DefaultRefreshInterval is the default interval for refreshing application
// states.
const DefaultRefreshInterval = 5 * time.Second

// Firehose should implement a streaming firehose client that streams all
// log and event messages.
type Firehose interface {
	// Stream should listen indefinitely for all log and event messages.
	//
	// The clients should not made any assumption about the order of the
	// received events.
	//
	// Whenever an error is encountered, the error should be sent down the error
	// channel and Stream should attempt to reconnect indefinitely.
	Stream(appGUID string, authToken string) (outputChan <-chan *events.Envelope, errorChan <-chan error)
}

// Target describes a monitoring target.
type Target struct {
	API      string
	Username string
	Password string
	Insecure bool
	Org      string
	Space    string
}

// AppMonitor implements a Cloud Foundry application monitor that collects
// various application metrics and emits them using a provided emitter.
type AppMonitor struct {
	// Emitter is the emitter used for sending metrics.
	Emitter Emitter
	// CloudController client for the API.
	CloudController *ccv2.Client
	// Firehose streaming client used for receiving logs and events.
	Firehose Firehose
	// UAA should provide valid OAuth2 tokens for the specific Cloud Foundry system.
	UAA oauth2.TokenSource
	// ErrLog is used for logging erros that occur when monitoring applications.
	ErrLog *log.Logger

	// RPCTimeout configures the timeouts when making RPCs.
	RPCTimeout time.Duration
	// RefreshInterval configures the polling interval for application
	// state changes.
	RefreshInterval time.Duration

	initOnce  sync.Once
	mu        sync.Mutex // guards
	monitored map[string]struct{}
}

// Monitor monitors a target for events and emits them using the provided.
// Emitter.
// It is wrapper for creating new AppMonitor and starting it for the specified
// organization and space.
// It uses default implementations of Firehose, UAA and ccv2.Client.
func Monitor(ctx context.Context, t Target, e Emitter) (err error) {
	u, err := url.Parse(t.API)
	if err != nil {
		return err
	}
	cf := &ccv2.Client{
		API:        u,
		HTTPClient: http.DefaultClient,
	}

	infoCtx, cancel := context.WithTimeout(ctx, DefaultRPCTimeout)
	defer cancel()
	info, err := cf.Info(infoCtx)
	if err != nil {
		return err
	}

	oauthConfig := &oauth2.Config{
		ClientID: "cf",
		Endpoint: oauth2.Endpoint{
			AuthURL:  info.TokenEndpoint + "/oauth/auth",
			TokenURL: info.TokenEndpoint + "/oauth/token",
		},
	}

	token, err := oauthConfig.PasswordCredentialsToken(context.Background(), t.Username, t.Password)
	if err != nil {
		return err
	}
	cf = &ccv2.Client{
		API:        u,
		HTTPClient: oauthConfig.Client(context.Background(), token),
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: t.Insecure}
	firehose := consumer.New(info.DopplerEndpoint, tlsConfig, nil)
	defer func() {
		if cerr := firehose.Close(); cerr != nil && err == nil {
			cerr = err
		}
	}()

	uaa := oauthConfig.TokenSource(context.Background(), token)
	tr := tokenRefresher{uaa}
	firehose.RefreshTokenFrom(&tr)

	mon := AppMonitor{
		ErrLog:          log.New(os.Stderr, "mozzle: ", 0),
		RefreshInterval: DefaultRefreshInterval,
		RPCTimeout:      DefaultRPCTimeout,

		CloudController: cf,
		Firehose:        firehose,
		Emitter:         e,
		UAA:             uaa,
	}

	return mon.Monitor(ctx, t.Org, t.Space)
}

// Monitor starts monitoring all applications under the specified organization
// and space.
// Monitor blocks until the context is canceled.
func (m *AppMonitor) Monitor(ctx context.Context, org, space string) error {
	m.initOnce.Do(func() {
		m.monitored = make(map[string]struct{})
		if m.ErrLog == nil {
			m.ErrLog = log.New(ioutil.Discard, "", 0)
		}
	})

	spaceCtx, cancel := context.WithTimeout(ctx, m.RPCTimeout)
	defer cancel()
	spaceEntity, err := getSpace(spaceCtx, m.CloudController, org, space)
	if err != nil {
		return err
	}
	applications := func() ([]application, error) {
		appQuery := ccv2.Query{
			Filter: ccv2.FilterSpaceGUID,
			Op:     ccv2.OperatorEqual,
			Value:  spaceEntity.GUID,
		}

		appCtx, cancel := context.WithTimeout(ctx, m.RPCTimeout)
		defer cancel()
		apps, err := m.CloudController.Applications(appCtx, appQuery)
		if err != nil {
			return nil, err
		}
		var res []application
		for _, app := range apps {
			res = append(res, application{app, org, space})
		}
		return res, nil
	}

	ticker := time.NewTicker(m.RefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			apps, err := applications()
			if err != nil {
				m.ErrLog.Printf("error fetching apps: %v\n", err)
				continue
			}
			m.mu.Lock()
			for _, app := range apps {
				if _, ok := m.monitored[app.GUID]; ok {
					continue
				}
				m.monitored[app.GUID] = struct{}{}
				go m.monitorApp(ctx, app)
			}
			m.mu.Unlock()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// monitorApp monitors particular application.
func (m *AppMonitor) monitorApp(ctx context.Context, app application) {
	monitorCtx, cancel := context.WithCancel(ctx)
	defer func() {
		m.mu.Lock()
		delete(m.monitored, app.GUID)
		m.mu.Unlock()
		cancel()
	}()

	go m.monitorFirehose(monitorCtx, app)

	ticker := time.NewTicker(m.RefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			if err := m.emitAppSummary(ctx, app); isAppNotFound(err) {
				return
			}
			m.emitAppEvents(ctx, app, now.Add(-1*m.RefreshInterval))
		case <-ctx.Done():
			return
		}
	}
}

// monitorFirehose streams events from the firehose endpoint and creates
// metrics based on the received events.
func (m *AppMonitor) monitorFirehose(ctx context.Context, app application) {
	token, err := m.UAA.Token()
	if err != nil {
		return
	}

	tokenStr := token.TokenType + " " + token.AccessToken
	msgChan, errorChan := m.Firehose.Stream(app.GUID, tokenStr)
	for {
		select {
		case event := <-msgChan:
			switch event.GetEventType() {
			case events.Envelope_ContainerMetric:
				containerMetrics{event.GetContainerMetric(), app}.EmitTo(m.Emitter)
			case events.Envelope_HttpStartStop:
				httpMetrics{event.GetHttpStartStop(), app}.EmitTo(m.Emitter)
			}
		case <-ctx.Done():
			m.ErrLog.Printf("stopping firehose monitor for app %s due to: %v",
				app.GUID, ctx.Err())
			return
		case err, ok := <-errorChan:
			if !ok {
				m.ErrLog.Printf("firehose error chan closed, exiting\n")
				return
			}
			m.ErrLog.Printf("error streaming from firehose: %v\n", err)
		}
	}
}

func (m *AppMonitor) emitAppSummary(ctx context.Context, app application) error {
	summaryCtx, cancel := context.WithTimeout(ctx, m.RPCTimeout)
	defer cancel()
	summary, err := m.CloudController.ApplicationSummary(summaryCtx, app.Application)
	if err != nil {
		m.ErrLog.Printf("error fetching app summary: %v\n", err)
		return err
	}
	applicationMetrics{summary, app}.EmitTo(m.Emitter)
	return nil
}

func (m *AppMonitor) emitAppEvents(ctx context.Context, app application, since time.Time) {
	events, err := m.appEventsSince(ctx, app, since)
	if err != nil {
		m.ErrLog.Printf("error fetching app events: %v\n", err)
		return
	}
	for _, event := range events {
		applicationEvent{event, app}.EmitTo(m.Emitter)
	}
}

func (m *AppMonitor) appEventsSince(ctx context.Context, app application, t time.Time) ([]ccv2.Event, error) {
	acteeQuery := ccv2.Query{
		Filter: ccv2.FilterActee,
		Op:     ccv2.OperatorEqual,
		Value:  app.GUID,
	}
	timestampQuery := ccv2.Query{
		Filter: ccv2.FilterTimestamp,
		Op:     ccv2.OperatorGreater,
		Value:  t.String(),
	}
	eventsCtx, cancel := context.WithTimeout(ctx, m.RPCTimeout)
	defer cancel()
	return m.CloudController.Events(eventsCtx, acteeQuery, timestampQuery)
}

// getSpace returns the Space entity described by the org, space pair.
func getSpace(ctx context.Context, cc *ccv2.Client, org, space string) (ccv2.Space, error) {
	orgNameQuery := ccv2.Query{
		Filter: ccv2.FilterName,
		Op:     ccv2.OperatorEqual,
		Value:  org,
	}
	orgs, err := cc.Organizations(ctx, orgNameQuery)
	if err != nil {
		return ccv2.Space{}, err
	}
	if len(orgs) != 1 {
		return ccv2.Space{}, fmt.Errorf("%q does not describe a single organization", org)
	}

	spaceQuery := ccv2.Query{
		Filter: ccv2.FilterOrganizationGUID,
		Op:     ccv2.OperatorEqual,
		Value:  orgs[0].GUID,
	}
	spaces, err := cc.Spaces(ctx, spaceQuery)
	if err != nil {
		return ccv2.Space{}, err
	}
	if len(spaces) != 1 {
		return ccv2.Space{}, fmt.Errorf("%q does not describe a single space", space)
	}

	return spaces[0], nil
}

func isAppNotFound(err error) bool {
	rerr, ok := err.(*ccv2.UnexpectedResponseError)
	if !ok {
		return false
	}
	// If the HTTP status code is Not Found (404), the application is missing.
	return rerr.StatusCode == 404
}

// application wraps ccv2.Application and adds the name of the org and space
// in which the application resides.
type application struct {
	ccv2.Application
	Org   string
	Space string
}

type tokenRefresher struct {
	oauth2.TokenSource
}

func (tr *tokenRefresher) RefreshAuthToken() (string, error) {
	t, err := tr.Token()
	if err != nil {
		return "", err
	}
	return t.TokenType + " " + t.AccessToken, nil
}
