package observability

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTP instruments the server with the global OpenTelemetry provider. Bootstrap
// leaves that provider as a no-op; a reviewed exporter can be configured later.
func HTTP(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "repforge-api")
}
