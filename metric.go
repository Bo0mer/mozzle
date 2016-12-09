package main

import (
	"fmt"
	"log"
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
	pfx := fmt.Sprintf("instance %d ", c.GetInstanceIndex())

	emit(&goryman.Event{
		Service: pfx + "memory used_bytes",
		Metric:  int(c.GetMemoryBytes()),
		State:   "ok",
	})
	emit(&goryman.Event{
		Service: pfx + "memory total_bytes",
		Metric:  int(c.GetMemoryBytesQuota()),
		State:   "ok",
	})
	emit(&goryman.Event{
		Service: pfx + "memory used_ratio",
		Metric:  ratio(c.GetMemoryBytes(), c.GetMemoryBytesQuota()),
		State:   "ok",
	})

	emit(&goryman.Event{
		Service: pfx + "disk used_bytes",
		Metric:  int(c.GetDiskBytes()),
		State:   "ok",
	})
	emit(&goryman.Event{
		Service: pfx + "disk total_bytes",
		Metric:  int(c.GetDiskBytesQuota()),
		State:   "ok",
	})
	emit(&goryman.Event{
		Service: pfx + "disk used_ratio",
		Metric:  ratio(c.GetDiskBytes(), c.GetDiskBytesQuota()),
		State:   "ok",
	})

	emit(&goryman.Event{
		Service: pfx + "cpu_percent",
		Metric:  c.GetCpuPercentage(),
		State:   "ok",
	})
}

type HTTPMetrics struct {
	*cfevent.HttpStartStop
}

func (r HTTPMetrics) Emit() {
	var durationMillis = (r.GetStopTimestamp() - r.GetStartTimestamp()) / 1000000
	emit(&goryman.Event{
		Service: "http response time_ms",
		Metric:  durationMillis,
		State:   "ok",
	})

	if r.GetPeerType() == cfevent.PeerType_Client {
		emit(&goryman.Event{
			Service: "http response code",
			Metric:  r.GetStatusCode(),
			State:   "ok",
		})
		emit(&goryman.Event{
			Service: "http response bytes_count",
			Metric:  int(r.GetContentLength()),
			State:   "ok",
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
