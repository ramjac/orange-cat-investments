package httputil

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// InstrumentHandler adds OpenTelemetry tracing to the provided HTTP handler.
func InstrumentHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}
