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
	App appMetadata
}

func (c containerMetrics) Emit() {
	attributes := make(map[string]string)
	attributes["org"] = c.App.Org
	attributes["space"] = c.App.Space
	attributes["application"] = c.App.Name
	attributes["instance"] = strconv.Itoa(int(c.GetInstanceIndex()))
	attributes["application_id"] = c.GetApplicationId()

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
	App appMetadata
}

func (r httpMetrics) Emit() {
	if r.GetPeerType() == cfevent.PeerType_Client {
		attributes := make(map[string]string)
		attributes["org"] = r.App.Org
		attributes["space"] = r.App.Space
		attributes["application"] = r.App.Name
		attributes["application_id"] = r.GetApplicationId().String()
		attributes["instance"] = strconv.Itoa(int(r.GetInstanceIndex()))

		attributes["method"] = r.GetMethod().String()
		attributes["request_id"] = r.GetRequestId().String()
		attributes["content_length"] = strconv.Itoa(int(r.GetContentLength()))
		attributes["status_code"] = strconv.Itoa(int(r.GetStatusCode()))

		durationMillis := (r.GetStopTimestamp() - r.GetStartTimestamp()) / 1000000
		emit(&goryman.Event{
			Host:       r.App.Name,
			Service:    "http response time_ms",
			Metric:     int(durationMillis),
			State:      "ok",
			Attributes: attributes,
		})
	}
}

type applicationMetrics struct {
	appSummary
	App appMetadata
}

func (m applicationMetrics) Emit() {
	attributes := make(map[string]string)
	attributes["org"] = m.App.Org
	attributes["space"] = m.App.Space
	attributes["application"] = m.App.Name
	attributes["application_id"] = m.Id

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
