package framework

import (
	"errors"
	"testing"
)

func TestResourceError(t *testing.T) {
	baseErr := errors.New("base error")
	resErr := NewResourceError("Pod", "default", "my-pod", baseErr)

	expected := "Pod default/my-pod: base error"
	if resErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, resErr.Error())
	}

	if !errors.Is(resErr, baseErr) {
		t.Error("expected ResourceError to wrap base error")
	}
}

func TestResourceError_ClusterScoped(t *testing.T) {
	baseErr := errors.New("base error")
	resErr := NewResourceError("ClusterRole", "", "my-role", baseErr)

	expected := "ClusterRole my-role: base error"
	if resErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, resErr.Error())
	}
}

func TestPrerequisiteError(t *testing.T) {
	baseErr := errors.New("CRD not found")
	preErr := NewPrerequisiteError("TempoOperator", baseErr)

	expected := "prerequisite check failed for TempoOperator: CRD not found"
	if preErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, preErr.Error())
	}

	if !errors.Is(preErr, baseErr) {
		t.Error("expected PrerequisiteError to wrap base error")
	}
}

func TestCleanupError(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	cleanupErr := NewCleanupError("CR deletion", err1, err2)

	if cleanupErr.Phase != "CR deletion" {
		t.Errorf("expected phase 'CR deletion', got %q", cleanupErr.Phase)
	}

	// Should contain both errors
	errStr := cleanupErr.Error()
	if errStr == "" {
		t.Error("expected non-empty error string")
	}
}

func TestTimeoutError(t *testing.T) {
	timeoutErr := NewTimeoutError("CR deletion", "120s", "remaining: Pod/my-pod")

	expected := "timeout after 120s waiting for CR deletion: remaining: Pod/my-pod"
	if timeoutErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, timeoutErr.Error())
	}
}

func TestTimeoutError_NoDetails(t *testing.T) {
	timeoutErr := NewTimeoutError("pod readiness", "60s", "")

	expected := "timeout after 60s waiting for pod readiness"
	if timeoutErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, timeoutErr.Error())
	}
}

func TestTimeoutError_Is(t *testing.T) {
	timeoutErr := NewTimeoutError("CR deletion", "120s", "")

	if !errors.Is(timeoutErr, ErrCRDeletionTimeout) {
		t.Error("expected TimeoutError to match ErrCRDeletionTimeout")
	}

	if !errors.Is(timeoutErr, ErrJobTimeout) {
		t.Error("expected TimeoutError to match ErrJobTimeout")
	}
}

func TestIsNotFound(t *testing.T) {
	if IsNotFound(errors.New("random error")) {
		t.Error("random error should not be NotFound")
	}

	if !IsNotFound(ErrResourceNotFound) {
		t.Error("ErrResourceNotFound should be NotFound")
	}
}

func TestIsTimeout(t *testing.T) {
	if IsTimeout(errors.New("random error")) {
		t.Error("random error should not be Timeout")
	}

	if !IsTimeout(ErrCRDeletionTimeout) {
		t.Error("ErrCRDeletionTimeout should be Timeout")
	}

	if !IsTimeout(ErrJobTimeout) {
		t.Error("ErrJobTimeout should be Timeout")
	}

	timeoutErr := NewTimeoutError("test", "10s", "")
	if !IsTimeout(timeoutErr) {
		t.Error("TimeoutError should be Timeout")
	}
}

func TestIsCancelled(t *testing.T) {
	if IsCancelled(errors.New("random error")) {
		t.Error("random error should not be Cancelled")
	}

	if !IsCancelled(ErrContextCancelled) {
		t.Error("ErrContextCancelled should be Cancelled")
	}
}

func TestSentinelErrors(t *testing.T) {
	// Ensure sentinel errors are distinct
	errs := []error{
		ErrNamespaceRequired,
		ErrCRDeletionTimeout,
		ErrFinalizerRemoval,
		ErrOperatorNotInstalled,
		ErrCRDNotEstablished,
		ErrPodNotReady,
		ErrJobFailed,
		ErrJobTimeout,
		ErrMetricsCollection,
		ErrPrometheusQuery,
		ErrResourceNotFound,
		ErrClusterConnection,
		ErrContextCancelled,
	}

	for i, err1 := range errs {
		for j, err2 := range errs {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("sentinel errors %v and %v should be distinct", err1, err2)
			}
		}
	}
}
