package main

import (
	"log"
	"strconv"
	"time"

	"github.com/bigdatadev/goryman"
	cfevent "github.com/cloudfoundry/sonde-go/events"
)

var client *goryman.GorymanClient
var eventPrefix string
var eventHost string
var eventTtl float32

var events chan *goryman.Event

func Initialize(riemannAddr, host, prefix string, ttl float32, queueSize int) {
	client = goryman.NewGorymanClient(riemannAddr)
	eventPrefix = prefix
	eventHost = host
	eventTtl = ttl
	events = make(chan *goryman.Event, queueSize)

	go emitLoop()
}

func Emit(events <-chan *cfevent.Envelope) {
	for event := range events {
		switch event.GetEventType() {
		case cfevent.Envelope_ContainerMetric:
			ContainerMetrics{event.GetContainerMetric()}.Emit()
		case cfevent.Envelope_HttpStartStop:
			HTTPMetrics{event.GetHttpStartStop()}.Emit()
		}
	}
}

type ContainerMetrics struct {
	*cfevent.ContainerMetric
}

func (c ContainerMetrics) Emit() {
	attributes := make(map[string]string)
	attributes["instance"] = strconv.Itoa(int(c.GetInstanceIndex()))
	attributes["application_id"] = c.GetApplicationId()

	emit(&goryman.Event{
		Service:    "memory used_bytes",
		Metric:     int(c.GetMemoryBytes()),
		State:      "ok",
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Service:    "memory total_bytes",
		Metric:     int(c.GetMemoryBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Service:    "memory used_ratio",
		Metric:     ratio(c.GetMemoryBytes(), c.GetMemoryBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	})

	emit(&goryman.Event{
		Service:    "disk used_bytes",
		Metric:     int(c.GetDiskBytes()),
		State:      "ok",
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Service:    "disk total_bytes",
		Metric:     int(c.GetDiskBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	})
	emit(&goryman.Event{
		Service:    "disk used_ratio",
		Metric:     ratio(c.GetDiskBytes(), c.GetDiskBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	})

	emit(&goryman.Event{
		Service:    "cpu_percent",
		Metric:     c.GetCpuPercentage(),
		State:      "ok",
		Attributes: attributes,
	})
}

type HTTPMetrics struct {
	*cfevent.HttpStartStop
}

func (r HTTPMetrics) Emit() {
	if r.GetPeerType() == cfevent.PeerType_Client {
		attributes := make(map[string]string)
		attributes["instance"] = strconv.Itoa(int(r.GetInstanceIndex()))
		attributes["application_id"] = r.GetApplicationId().String()
		attributes["method"] = r.GetMethod().String()
		attributes["request_id"] = r.GetRequestId().String()
		attributes["content_length"] = strconv.Itoa(int(r.GetContentLength()))
		attributes["status_code"] = strconv.Itoa(int(r.GetStatusCode()))

		durationMillis := (r.GetStopTimestamp() - r.GetStartTimestamp()) / 1000000
		emit(&goryman.Event{
			Service:    "http response time_ms",
			Metric:     int(durationMillis),
			State:      "ok",
			Attributes: attributes,
		})
	}
}

func emit(e *goryman.Event) {
	if e.Ttl == 0.0 {
		e.Ttl = eventTtl
	}
	e.Time = time.Now().Unix()
	e.Host = eventHost

	if eventPrefix != "" {
		e.Service = eventPrefix + " " + e.Service
	}

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
