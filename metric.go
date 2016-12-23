package mozzle

import (
	"log"
	"strconv"
	"time"

	"github.com/bigdatadev/goryman"
	cfevent "github.com/cloudfoundry/sonde-go/events"
)

var client *goryman.GorymanClient
var eventTtl float32

var events chan *goryman.Event

// Initialize prepares for emitting to Riemann.
// It should be called only once, before any calls to the Monitor functionality.
// The queueSize argument specifies how many events will be kept in-memory
// if there is problem with emission.
func Initialize(riemannAddr string, ttl float32, queueSize int) {
	client = goryman.NewGorymanClient(riemannAddr)
	eventTtl = ttl
	events = make(chan *goryman.Event, queueSize)

	go emitLoop()
}

type containerMetrics struct {
	*cfevent.ContainerMetric
	App application
}

func (c containerMetrics) Emit() {
	attributes := attributes(c.App)
	attributes["instance"] = strconv.Itoa(int(c.GetInstanceIndex()))

	emit(&goryman.Event{
		Host:       c.App.Name,
		Service:    "memory used_bytes",
		Metric:     int(c.GetMemoryBytes()),
		State:      "ok",
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Host:       c.App.Name,
		Service:    "memory total_bytes",
		Metric:     int(c.GetMemoryBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Host:       c.App.Name,
		Service:    "memory used_ratio",
		Metric:     ratio(c.GetMemoryBytes(), c.GetMemoryBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	})

	emit(&goryman.Event{
		Host:       c.App.Name,
		Service:    "disk used_bytes",
		Metric:     int(c.GetDiskBytes()),
		State:      "ok",
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Host:       c.App.Name,
		Service:    "disk total_bytes",
		Metric:     int(c.GetDiskBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Host:       c.App.Name,
		Service:    "disk used_ratio",
		Metric:     ratio(c.GetDiskBytes(), c.GetDiskBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	})

	emit(&goryman.Event{
		Host:       c.App.Name,
		Service:    "cpu_percent",
		Metric:     c.GetCpuPercentage(),
		State:      "ok",
		Attributes: attributes,
	})
}

type httpMetrics struct {
	*cfevent.HttpStartStop
	App application
}

func (r httpMetrics) Emit() {
	attributes := attributes(r.App)
	attributes["instance"] = strconv.Itoa(int(r.GetInstanceIndex()))

	attributes["method"] = r.GetMethod().String()
	attributes["request_id"] = r.GetRequestId().String()
	attributes["status_code"] = strconv.Itoa(int(r.GetStatusCode()))

	switch r.GetPeerType() {
	case cfevent.PeerType_Client:
		durationMillis := (r.GetStopTimestamp() - r.GetStartTimestamp()) / 1000000
		emit(&goryman.Event{
			Host:       r.App.Name,
			Service:    "http response time_ms",
			Metric:     int(durationMillis),
			State:      "ok",
			Attributes: attributes,
		})
	case cfevent.PeerType_Server:
		emit(&goryman.Event{
			Host:       r.App.Name,
			Service:    "http response content_length_bytes",
			Metric:     int(r.GetContentLength()),
			State:      "ok",
			Attributes: attributes,
		})
	}
}

type applicationMetrics struct {
	appSummary
	App application
}

func (m applicationMetrics) Emit() {
	attributes := attributes(m.App)

	state := "ok"
	if m.RunningInstances < m.Instances {
		state = "warn"
		if m.RunningInstances == 0 {
			state = "critical"
		}
	}
	emit(&goryman.Event{
		Host:       m.App.Name,
		Service:    "instance running_count",
		Metric:     m.RunningInstances,
		State:      state,
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Host:       m.App.Name,
		Service:    "instance configured_count",
		Metric:     m.Instances,
		State:      "ok",
		Attributes: attributes,
	})
}

type applicationEvent struct {
	appEvent
	App application
}

func (e applicationEvent) Emit() {
	attributes := attributes(e.App)
	attributes["event"] = e.Type
	attributes["actee"] = e.ActeeName
	attributes["actee_type"] = e.ActeeType
	attributes["actor"] = e.ActorName
	attributes["actor_type"] = e.ActorType

	emit(&goryman.Event{
		Time:       e.Timestamp.Unix(),
		Host:       e.App.Name,
		Service:    "app event",
		Metric:     1,
		State:      "ok",
		Attributes: attributes,
	})
}

func emit(e *goryman.Event) {
	if e.Ttl == 0.0 {
		e.Ttl = eventTtl
	}
	e.Time = time.Now().Unix()

	select {
	case events <- e:
	default:
		log.Printf("queue full, dropping events\n")
	}
}

func emitLoop() {
	connected := false
	for e := range events {
		if !connected {
			if err := client.Connect(); err != nil {
				log.Printf("metric: error connecting to riemann: %v\n", err)
				continue
			}
			connected = true
		}

		if err := client.SendEvent(e); err != nil {
			log.Printf("metric: error sending event: %v\n", err)
		}
	}
}

func ratio(part, whole uint64) float64 {
	return float64(part) / float64(whole)
}

func attributes(app application) map[string]string {
	return map[string]string{
		"org":            app.Org,
		"space":          app.Space,
		"application":    app.Name,
		"application_id": app.Guid,
	}
}
