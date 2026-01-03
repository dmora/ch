// Package backend provides sync backend implementations.
package backend

// Stats holds backend statistics.
type Stats struct {
	SpansSent   int
	SpansFailed int
	BytesSent   int64
}
