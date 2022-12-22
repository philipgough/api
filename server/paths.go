package server

import (
	"encoding/json"
	"github.com/go-chi/chi"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/observatorium/api/authentication"
	"github.com/observatorium/api/httperr"
	"github.com/observatorium/api/proxy"
	"net/http"
	"strings"
)

// PathsHandlerFunc lists all paths available from the provided routes.
func PathsHandlerFunc(logger log.Logger, routes []chi.Route) http.HandlerFunc {
	paths := make([]string, 0, len(routes))
	for _, r := range routes {
		paths = append(paths, r.Pattern)
	}

	pathsStruct := struct {
		Paths []string `json:"paths"`
	}{
		Paths: paths,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		externalPathJSON, err := json.MarshalIndent(pathsStruct, "", "  ")
		if err != nil {
			level.Error(logger).Log("msg", "failed to marshal paths input to JSON", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		if _, err := w.Write(externalPathJSON); err != nil {
			level.Error(logger).Log("msg", "could not write external paths", "err", err.Error())
		}
	}
}

// StripTenantPrefix strips the given prefix plus the tenant from the request's URL path.
// This information is NOT lost, but put in the request's context.
func StripTenantPrefix(prefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, ok := authentication.GetTenant(r.Context())
			if !ok {
				httperr.PrometheusAPIError(w, "tenant not found", http.StatusInternalServerError)
				return
			}

			tenantPrefix := strings.Replace(prefix, "{tenant}", tenant, 1)
			http.StripPrefix(tenantPrefix, proxy.WithPrefix(tenantPrefix, next)).ServeHTTP(w, r)
		})
	}
}
