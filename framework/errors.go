package framework

import (
	"errors"
	"fmt"
)

// Sentinel errors for framework operations
var (
	// ErrNamespaceRequired indicates that a namespace was not provided
	ErrNamespaceRequired = errors.New("namespace is required")

	// ErrCRDeletionTimeout indicates that CR deletion timed out
	ErrCRDeletionTimeout = errors.New("CR deletion timed out")

	// ErrFinalizerRemoval indicates failure to remove finalizers
	ErrFinalizerRemoval = errors.New("failed to remove finalizers")

	// ErrOperatorNotInstalled indicates that a required operator is not installed
	ErrOperatorNotInstalled = errors.New("operator not installed")

	// ErrCRDNotEstablished indicates that a CRD is not in established condition
	ErrCRDNotEstablished = errors.New("CRD not established")

	// ErrPodNotReady indicates that pods failed to become ready
	ErrPodNotReady = errors.New("pod not ready")

	// ErrJobFailed indicates that a k6 job failed
	ErrJobFailed = errors.New("k6 job failed")

	// ErrJobTimeout indicates that a k6 job timed out
	ErrJobTimeout = errors.New("k6 job timed out")

	// ErrMetricsCollection indicates failure to collect metrics
	ErrMetricsCollection = errors.New("metrics collection failed")

	// ErrPrometheusQuery indicates a Prometheus query error
	ErrPrometheusQuery = errors.New("prometheus query failed")

	// ErrResourceNotFound indicates that a resource was not found
	ErrResourceNotFound = errors.New("resource not found")

	// ErrClusterConnection indicates failure to connect to the cluster
	ErrClusterConnection = errors.New("failed to connect to cluster")

	// ErrContextCancelled indicates the operation was cancelled
	ErrContextCancelled = errors.New("operation cancelled")
)

// ResourceError represents an error related to a specific resource
type ResourceError struct {
	Kind      string
	Namespace string
	Name      string
	Err       error
}

func (e *ResourceError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("%s %s/%s: %v", e.Kind, e.Namespace, e.Name, e.Err)
	}
	return fmt.Sprintf("%s %s: %v", e.Kind, e.Name, e.Err)
}

func (e *ResourceError) Unwrap() error {
	return e.Err
}

// NewResourceError creates a new ResourceError
func NewResourceError(kind, namespace, name string, err error) *ResourceError {
	return &ResourceError{
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
		Err:       err,
	}
}

// PrerequisiteError represents an error when checking prerequisites
type PrerequisiteError struct {
	Component string
	Err       error
}

func (e *PrerequisiteError) Error() string {
	return fmt.Sprintf("prerequisite check failed for %s: %v", e.Component, e.Err)
}

func (e *PrerequisiteError) Unwrap() error {
	return e.Err
}

// NewPrerequisiteError creates a new PrerequisiteError
func NewPrerequisiteError(component string, err error) *PrerequisiteError {
	return &PrerequisiteError{
		Component: component,
		Err:       err,
	}
}

// CleanupError represents errors during cleanup operations
type CleanupError struct {
	Phase string
	Errs  []error
}

func (e *CleanupError) Error() string {
	return fmt.Sprintf("cleanup failed during %s phase: %v", e.Phase, errors.Join(e.Errs...))
}

func (e *CleanupError) Unwrap() error {
	return errors.Join(e.Errs...)
}

// NewCleanupError creates a new CleanupError
func NewCleanupError(phase string, errs ...error) *CleanupError {
	return &CleanupError{
		Phase: phase,
		Errs:  errs,
	}
}

// TimeoutError represents a timeout during an operation
type TimeoutError struct {
	Operation string
	Duration  string
	Details   string
}

func (e *TimeoutError) Error() string {
	msg := fmt.Sprintf("timeout after %s waiting for %s", e.Duration, e.Operation)
	if e.Details != "" {
		msg += ": " + e.Details
	}
	return msg
}

func (e *TimeoutError) Is(target error) bool {
	switch target {
	case ErrCRDeletionTimeout, ErrJobTimeout:
		return true
	}
	return false
}

// NewTimeoutError creates a new TimeoutError
func NewTimeoutError(operation, duration, details string) *TimeoutError {
	return &TimeoutError{
		Operation: operation,
		Duration:  duration,
		Details:   details,
	}
}

// IsNotFound returns true if the error indicates a resource was not found
func IsNotFound(err error) bool {
	return errors.Is(err, ErrResourceNotFound)
}

// IsTimeout returns true if the error is a timeout error
func IsTimeout(err error) bool {
	var te *TimeoutError
	if errors.As(err, &te) {
		return true
	}
	return errors.Is(err, ErrCRDeletionTimeout) || errors.Is(err, ErrJobTimeout)
}

// IsCancelled returns true if the error indicates cancellation
func IsCancelled(err error) bool {
	return errors.Is(err, ErrContextCancelled)
}
