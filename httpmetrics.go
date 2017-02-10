package mozzle

import (
	"strconv"

	cfevent "github.com/cloudfoundry/sonde-go/events"
)

type httpMetrics struct {
	*cfevent.HttpStartStop
	App application
}

func (r httpMetrics) EmitTo(e Emitter) {
	attributes := attributes(r.App)
	attributes["instance"] = strconv.Itoa(int(r.GetInstanceIndex()))
	attributes["method"] = r.GetMethod().String()
	attributes["request_id"] = r.GetRequestId().String()
	attributes["status_code"] = strconv.Itoa(int(r.GetStatusCode()))

	switch r.GetPeerType() {
	case cfevent.PeerType_Client:
		attributes["peer"] = "client"
	case cfevent.PeerType_Server:
		attributes["peer"] = "server"
	default:
		attributes["peer"] = "unknown"
	}

	durationMillis := (r.GetStopTimestamp() - r.GetStartTimestamp()) / 1000000
	e.Emit(forApp(r.App, Metric{
		Service:    "http response time_ms",
		Metric:     int(durationMillis),
		State:      "ok",
		Attributes: attributes,
	}))
	e.Emit(forApp(r.App, Metric{
		Service:    "http response content_length_bytes",
		Metric:     int(r.GetContentLength()),
		State:      "ok",
		Attributes: attributes,
	}))

}
