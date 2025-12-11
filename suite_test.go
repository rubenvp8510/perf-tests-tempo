package test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPerformanceTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tempo Performance Tests Suite")
}

var _ = BeforeSuite(func() {
	// Verify cluster connection
	// Check for required CRDs (TempoMonolithic, OpenTelemetryCollector)
	// This can be done by trying to list the CRDs or checking if the framework can be created
})
