package operation

import (
	"net/http"

	"github.com/davidsbond/autopgo/internal/api"
)

type (
	// The HTTPController type is used to serve the health and readiness endpoints.
	HTTPController struct {
		healthChecks []Checker
	}
)

// NewHTTPController returns a new instance of the HTTPController type that will serve the provided Checker
// implementations as health checks.
func NewHTTPController(healthChecks []Checker) *HTTPController {
	return &HTTPController{
		healthChecks: healthChecks,
	}
}

// Register endpoints onto the http.ServeMux.
func (h *HTTPController) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", h.GetHealth)
	mux.HandleFunc("GET /api/ready", h.GetReadiness)
}

type (
	// The GetHealthResponse type contains fields describing the health of the application.
	GetHealthResponse struct {
		// The overall health status across all components.
		Status HealthStatus `json:"status"`
		// The status of individual dependencies.
		Dependencies []Dependency `json:"dependencies"`
	}
)

// GetHealth handles an inbound HTTP request that returns the current health status of the application and its
// dependencies. The top-level status in the response will be HealthStatusUnhealthy if one or more of the dependencies
// report the same status. When bad health is detected, the response code is 503.
func (h *HTTPController) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := GetHealthResponse{
		Status: HealthStatusHealthy,
	}

	for _, checker := range h.healthChecks {
		component := Dependency{
			Name:   checker.Name(),
			Status: HealthStatusHealthy,
		}

		if err := checker.Check(ctx); err != nil {
			component.Status = HealthyStatusUnhealthy
			component.Message = err.Error()
		}

		resp.Dependencies = append(resp.Dependencies, component)
	}

	for _, component := range resp.Dependencies {
		if component.Status != HealthStatusHealthy {
			resp.Status = component.Status
			break
		}
	}

	switch resp.Status {
	case HealthStatusHealthy:
		api.Respond(ctx, w, http.StatusOK, resp)
		return
	case HealthyStatusUnhealthy:
		api.Respond(ctx, w, http.StatusServiceUnavailable, resp)
		return
	}
}

type (
	// The GetReadinessResponse type is the response given when calling the readiness endpoint.
	GetReadinessResponse struct{}
)

// GetReadiness handles an inbound HTTP request to determine if the application is ready to handle HTTP traffic. This
// endpoint currently just returns an empty response with a 200 code, as if this request can be served then
// readiness is expected.
func (h *HTTPController) GetReadiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	api.Respond(ctx, w, http.StatusOK, GetReadinessResponse{})
}
