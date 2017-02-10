// Package mozzle implements Cloud Foundry application monitor that emits
// metric events.
//
// Metrics for the following services are emitted.
//
// Regarding memory usage of each application instance.
//			memory used_bytes
//			memory total_bytes
//			memory used_ratio
// Regarding disk usage of each application instance.
//			disk used_bytes
//			disk total_bytes
//			disk used_ratio
// Regarding CPU consumption.
//			cpu_percent
// Regarding each HTTP event (request-response).
//			http response time_ms
//			http response content_length_bytes
// Regarding application availability.
//			instance running_count
//			instance configured_count
// Regarding application events.
//			app event
//
// Each of the events has attributes specifying the application's
// org, space, name, id, and the insntace index (when appropriate).
//
// Additionally, the HTTP events have attributes specifying the method,
// request_id, content length the returned status code and the peer type.
// There are two peer types - client and server. Client means that measurements
// are recorded via the Cloud Foundry router's HTTP client that requested the
// application container and server means that the measurements are recorded
// for responding to the end user via the router server.
//
// The application event metrics have attributes that describe the event's
// actor and actee, as well as their ids.
package mozzle
