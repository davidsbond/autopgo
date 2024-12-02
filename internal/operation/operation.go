// Package operation provides types used to expose health, rediness & other general operator focused functionality.
package operation

import "context"

type (
	// The HealthStatus string denotes the current health of an application or its components.
	HealthStatus string

	// The Dependency type contains health data for an individual dependency within an application.
	Dependency struct {
		// The name of the component.
		Name string `json:"name"`
		// The status of the component.
		Status HealthStatus `json:"status"`
		// Any error message returned when checking the component's health.
		Message string `json:"message,omitempty"`
	}

	// The Checker interface describes types whose health can be checked.
	Checker interface {
		// The Name of the check. This ends up used as Dependency.Name.
		Name() string
		// Check should return an error if the component is deemed unhealthy.
		Check(ctx context.Context) error
	}
)

// Constants for health statuses.
const (
	HealthStatusUnknown    HealthStatus = "unknown"
	HealthStatusHealthy    HealthStatus = "healthy"
	HealthyStatusUnhealthy HealthStatus = "unhealthy"
)
