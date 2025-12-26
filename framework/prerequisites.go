package framework

import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrerequisiteStatus represents the status of a single prerequisite
type PrerequisiteStatus struct {
	Name      string
	Installed bool
	Message   string
}

// PrerequisitesResult contains the results of all prerequisite checks
type PrerequisitesResult struct {
	TempoOperator         PrerequisiteStatus
	OpenTelemetryOperator PrerequisiteStatus
	AllMet                bool
}

// Required CRDs for each operator
var (
	tempoCRDs = []string{
		"tempomonolithics.tempo.grafana.com",
		"tempostacks.tempo.grafana.com",
	}

	openTelemetryCRDs = []string{
		"opentelemetrycollectors.opentelemetry.io",
	}
)

// CheckPrerequisites verifies that required operators are installed in the cluster
func (f *Framework) CheckPrerequisites() (*PrerequisitesResult, error) {
	apiextClient, err := apiextensionsclient.NewForConfig(f.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create apiextensions client: %w", err)
	}

	result := &PrerequisitesResult{
		AllMet: true,
	}

	// Check Tempo Operator
	result.TempoOperator = checkCRDs(f.ctx, apiextClient, "Tempo Operator", tempoCRDs)
	if !result.TempoOperator.Installed {
		result.AllMet = false
	}

	// Check OpenTelemetry Operator
	result.OpenTelemetryOperator = checkCRDs(f.ctx, apiextClient, "OpenTelemetry Operator", openTelemetryCRDs)
	if !result.OpenTelemetryOperator.Installed {
		result.AllMet = false
	}

	return result, nil
}

// checkCRDs verifies that all required CRDs for an operator are installed
func checkCRDs(ctx context.Context, client apiextensionsclient.Interface, operatorName string, crds []string) PrerequisiteStatus {
	status := PrerequisiteStatus{
		Name:      operatorName,
		Installed: true,
	}

	var missing []string
	var found []string

	for _, crdName := range crds {
		crd, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			missing = append(missing, crdName)
			status.Installed = false
			continue
		}

		// Check if CRD is established
		if !isCRDEstablished(crd) {
			missing = append(missing, crdName+" (not established)")
			status.Installed = false
			continue
		}

		found = append(found, crdName)
	}

	if status.Installed {
		status.Message = fmt.Sprintf("All CRDs found: %v", found)
	} else {
		status.Message = fmt.Sprintf("Missing CRDs: %v", missing)
	}

	return status
}

// isCRDEstablished checks if the CRD has the Established condition set to True
func isCRDEstablished(crd *apiextensionsv1.CustomResourceDefinition) bool {
	for _, cond := range crd.Status.Conditions {
		if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
			return true
		}
	}
	return false
}

// String returns a human-readable summary of the prerequisites result
func (r *PrerequisitesResult) String() string {
	tempoStatus := "✓"
	if !r.TempoOperator.Installed {
		tempoStatus = "✗"
	}

	otelStatus := "✓"
	if !r.OpenTelemetryOperator.Installed {
		otelStatus = "✗"
	}

	return fmt.Sprintf(
		"Prerequisites Check:\n"+
			"  %s Tempo Operator: %s\n"+
			"  %s OpenTelemetry Operator: %s\n"+
			"  All prerequisites met: %v",
		tempoStatus, r.TempoOperator.Message,
		otelStatus, r.OpenTelemetryOperator.Message,
		r.AllMet,
	)
}
