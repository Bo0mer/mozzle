package mozzle

import (
	"strconv"

	cfevent "github.com/cloudfoundry/sonde-go/events"
)

type containerMetrics struct {
	*cfevent.ContainerMetric
	App application
}

func (m containerMetrics) EmitTo(e Emitter) {
	attributes := attributes(m.App)
	attributes["instance"] = strconv.Itoa(int(m.GetInstanceIndex()))

	e.Emit(forApp(m.App, Metric{
		Service:    "memory used_bytes",
		Metric:     int(m.GetMemoryBytes()),
		State:      "ok",
		Attributes: attributes,
	}))
	e.Emit(forApp(m.App, Metric{
		Service:    "memory total_bytes",
		Metric:     int(m.GetMemoryBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	}))
	e.Emit(forApp(m.App, Metric{
		Service:    "memory used_ratio",
		Metric:     ratio(m.GetMemoryBytes(), m.GetMemoryBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	}))

	e.Emit(forApp(m.App, Metric{
		Service:    "disk used_bytes",
		Metric:     int(m.GetDiskBytes()),
		State:      "ok",
		Attributes: attributes,
	}))
	e.Emit(forApp(m.App, Metric{
		Service:    "disk total_bytes",
		Metric:     int(m.GetDiskBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	}))
	e.Emit(forApp(m.App, Metric{
		Service:    "disk used_ratio",
		Metric:     ratio(m.GetDiskBytes(), m.GetDiskBytesQuota()),
		State:      "ok",
		Attributes: attributes,
	}))

	e.Emit(forApp(m.App, Metric{
		Service:    "cpu_percent",
		Metric:     m.GetCpuPercentage(),
		State:      "ok",
		Attributes: attributes,
	}))
}
