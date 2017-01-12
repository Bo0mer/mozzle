package mozzle

import "github.com/Bo0mer/ccv2"

type applicationMetrics struct {
	ccv2.ApplicationSummary
	App application
}

func (m applicationMetrics) EmitTo(e Emitter) {
	attributes := attributes(m.App)
	state := "ok"
	if m.RunningInstances < m.Instances {
		state = "warn"
		if m.RunningInstances == 0 {
			state = "critical"
		}
	}

	e.Emit(forApp(m.App, Metric{
		Service:    "instance running_count",
		Metric:     m.RunningInstances,
		State:      state,
		Attributes: attributes,
	}))

	e.Emit(forApp(m.App, Metric{
		Service:    "instance configured_count",
		Metric:     m.Instances,
		State:      state,
		Attributes: attributes,
	}))
}
