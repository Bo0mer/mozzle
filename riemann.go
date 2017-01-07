package mozzle

import (
	"log"
	"time"

	"github.com/bigdatadev/goryman"
)

// RiemannEmitter implements Emitter that interpretes metrics as Riemann events
// and emits them to a Riemann instance.
type RiemannEmitter struct {
	client    *goryman.GorymanClient
	eventTTL  float32
	events    chan *goryman.Event
	done      chan struct{}
	connected bool
}

// Initialize prepares for emitting to Riemann.
// It should be called only once, before the first call of Monitor.
// The queueSize argument specifies how many events will be kept in-memory
// if there is problem with emission.
func (r *RiemannEmitter) Initialize(riemannAddr string, ttl float32, queueSize int) {
	r.client = goryman.NewGorymanClient(riemannAddr)
	r.eventTTL = ttl
	r.events = make(chan *goryman.Event, queueSize)
	r.done = make(chan struct{})

	go r.emitLoop()
}

// Close renders the emitter unusable and frees all allocated resources.
// The emitter should not be used after it has been closed.
// There is no guarantee that any queued events will be sent before closing.
// This particular close never fails.
func (r *RiemannEmitter) Close() error {
	close(r.done)
	return nil
}

// Emit constructs a riemann event from the specified metric and emits it to
// Riemann. It is non-blocking and  safe for concurrent use by multiple goroutines.
//
// Emit must be used only after calling Initialize, and not after calling
// Shutdown.
func (r *RiemannEmitter) Emit(m Metric) {
	e := &goryman.Event{}
	e.Ttl = r.eventTTL

	e.Time = time.Now().Unix()
	if m.Time != 0 {
		e.Time = m.Time
	}
	e.Host = m.Application
	e.Service = m.Service
	e.Metric = m.Metric
	e.State = m.State
	e.Attributes = m.Attributes
	if e.Attributes == nil {
		e.Attributes = make(map[string]string)
	}
	e.Attributes["application"] = m.Application
	e.Attributes["application_id"] = m.ApplicationID
	e.Attributes["org"] = m.Organization
	e.Attributes["space"] = m.Space

	select {
	case r.events <- e:
	default:
		log.Printf("riemann: queue full, dropping events\n")
	}
}

func (r *RiemannEmitter) emitLoop() {
	r.connected = false
	for {
		select {
		case e := <-r.events:
			if !r.connected {
				if err := r.client.Connect(); err != nil {
					log.Printf("riemann: error connecting: %v\n", err)
					continue
				}
				r.connected = true
			}

			if err := r.client.SendEvent(e); err != nil {
				log.Printf("riemann: error sending event: %v\n", err)
				if cerr := r.client.Close(); cerr != nil {
					log.Printf("riemann: error closing conn: %v\n", cerr)
				}
				r.connected = false
			}
		case <-r.done:
			return
		}
	}
}
