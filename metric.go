package main

import (
	"log"
	"time"

	"github.com/bigdatadev/goryman"
)

var client *goryman.GorymanClient
var prefix string
var eventHost string
var eventTtl float32

var events chan *goryman.Event

func Initialize(riemannAddr, host, prefix string, ttl float32, queueSize int) {
	client = goryman.NewGorymanClient(riemannAddr)
	eventHost = host
	eventTtl = ttl
	events = make(chan *goryman.Event, queueSize)

	go emitLoop()
}

func Emit(service string, value interface{}) {
	emit(&goryman.Event{
		State:   "ok",
		Service: service,
		Metric:  value,
	})

}

func emit(e *goryman.Event) {
	if e.Ttl == 0.0 {
		e.Ttl = eventTtl
	}
	e.Time = time.Now().Unix()
	e.Host = eventHost

	if prefix != "" {
		e.Service = prefix + e.Service
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
				continue
			}
			connected = true
		}

		if err := client.SendEvent(e); err != nil {
			log.Printf("metric: error sending event: %v\n", err)
		}
	}
}
