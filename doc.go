// Package mozzle implements Cloud Foundry application monitor that emits
// metric events.
//
// Events for the following metrics are emitted.
//
// Regarding memory usage of each instance.
//			memory used_bytes
//			memory total_bytes
//			memory used_ratio
// Regarding disk usage of each instance.
//			disk used_bytes
//			disk total_bytes
//			disk used_ratio
// Regarding CPU consumption.
//			cpu_percent
// Regarding each HTTP event.
//			http response time_ms
//			http response content_length_bytes
// Regarding availability.
//			instance running_count
//			instance configured_count
// Regarding application events.
//			app event
//
// Each of the events has attributes specifying the application's
// org, space, name, and the insntace index (when appropriate).
//
// Additionally, the HTTP events has attributes specifying the method,
// request_id, content length and the returned status code.
package mozzle
