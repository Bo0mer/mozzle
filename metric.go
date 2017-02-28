package mozzle

// Metric is a metric regarding an application.
type Metric struct {
	// Application is the name of the application.
	Application string
	// ApplicationID is the GUID of the application.
	ApplicationID string
	// Organization is the name of the application's organization.
	Organization string
	// Space is the name of the application's space.
	Space string

	// Time is the time when the event occurred.
	Time int64
	// Service for which the metric is relevant.
	Service string
	// Metric value. Could be int64, float32 or float64.
	Metric interface{}
	// State is a text description of the service's state - e.g. 'ok', 'warn'.
	State string
	// Attributes are key-value pairs describing the metric.
	Attributes map[string]string
}

// Emitter should emit application metrics.
type Emitter interface {
	// Emit emits the specified application metric.
	// It should not block and should be safe for concurrent use.
	Emit(m Metric)
}

func forApp(app application, m Metric) Metric {
	m.Application = app.Entity.Name
	m.ApplicationID = app.GUID
	m.Organization = app.Org
	m.Space = app.Space
	return m
}

func ratio(part, whole uint64) float64 {
	if whole == 0 {
		return 0.0
	}
	return float64(part) / float64(whole)
}

func attributes(app application) map[string]string {
	return map[string]string{
		"org":            app.Org,
		"space":          app.Space,
		"application":    app.Entity.Name,
		"application_id": app.GUID,
	}
}
