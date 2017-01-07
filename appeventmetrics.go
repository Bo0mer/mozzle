package mozzle

import "github.com/Bo0mer/ccv2"

type applicationEvent struct {
	ccv2.Event
	App application
}

func (e applicationEvent) EmitTo(emitter Emitter) {
	attributes := attributes(e.App)
	attributes["event"] = e.Entity.Type
	attributes["actee"] = e.Entity.ActeeName
	attributes["actee_type"] = e.Entity.ActeeType
	attributes["actor"] = e.Entity.ActorName
	attributes["actor_type"] = e.Entity.ActorType

	emitter.Emit(forApp(e.App, Metric{
		Time:       e.Entity.Timestamp.Unix(),
		Service:    "app event",
		Metric:     1,
		State:      "ok",
		Attributes: attributes,
	}))
}
