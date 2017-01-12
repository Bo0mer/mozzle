package mozzle

// Metric is a metric regarding an application.
type Metric struct {
	Application   string
	ApplicationID string
	Organization  string
	Space         string

	Time       int64
	Service    string
	Metric     interface{}
	State      string
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
