package examples

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat/perf-tests-tempo/test/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("Basic Performance Test", func() {
	var fw *framework.Framework

	BeforeEach(func() {
		var err error
		fw, err = framework.New()
		Expect(err).NotTo(HaveOccurred())

		// Deploy MinIO
		err = fw.SetupMinIO()
		Expect(err).NotTo(HaveOccurred())

		// Deploy Tempo with medium resources
		resourceConfig := &framework.ResourceConfig{
			Profile: "medium", // Uses preset profile
		}
		err = fw.SetupTempo("monolithic", resourceConfig)
		Expect(err).NotTo(HaveOccurred())

		// Deploy OpenTelemetry Collector
		err = fw.SetupOTelCollector()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle medium load", func() {
		// TODO: Add your performance test here
		// This is where you would:
		// - Deploy trace generators
		// - Deploy query generators
		// - Run load tests
		// - Collect metrics
		// - Assert on performance metrics
	})

	AfterEach(func() {
		if fw != nil {
			err := fw.Cleanup()
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

var _ = Describe("Performance Test with Custom Resources", func() {
	var fw *framework.Framework

	BeforeEach(func() {
		var err error
		fw, err = framework.New()
		Expect(err).NotTo(HaveOccurred())

		// Deploy MinIO
		err = fw.SetupMinIO()
		Expect(err).NotTo(HaveOccurred())

		// Deploy Tempo with custom resources
		resourceConfig := &framework.ResourceConfig{
			Resources: &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("6Gi"),
					corev1.ResourceCPU:    resource.MustParse("750m"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("6Gi"),
					corev1.ResourceCPU:    resource.MustParse("750m"),
				},
			},
		}
		err = fw.SetupTempo("monolithic", resourceConfig)
		Expect(err).NotTo(HaveOccurred())

		// Deploy OpenTelemetry Collector
		err = fw.SetupOTelCollector()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle custom resource configuration", func() {
		// TODO: Add your performance test here
	})

	AfterEach(func() {
		if fw != nil {
			err := fw.Cleanup()
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

var _ = Describe("Performance Test with Tempo Stack", func() {
	var fw *framework.Framework

	BeforeEach(func() {
		var err error
		fw, err = framework.New()
		Expect(err).NotTo(HaveOccurred())

		// Deploy MinIO
		err = fw.SetupMinIO()
		Expect(err).NotTo(HaveOccurred())

		// Deploy Tempo Stack (resources not supported for stack)
		err = fw.SetupTempo("stack", nil)
		Expect(err).NotTo(HaveOccurred())

		// Deploy OpenTelemetry Collector
		err = fw.SetupOTelCollector()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle stack deployment", func() {
		// TODO: Add your performance test here
	})

	AfterEach(func() {
		if fw != nil {
			err := fw.Cleanup()
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

var _ = Describe("Performance Test without Resources", func() {
	var fw *framework.Framework

	BeforeEach(func() {
		var err error
		fw, err = framework.New()
		Expect(err).NotTo(HaveOccurred())

		// Deploy MinIO
		err = fw.SetupMinIO()
		Expect(err).NotTo(HaveOccurred())

		// Deploy Tempo without resource configuration (uses defaults)
		err = fw.SetupTempo("monolithic", nil)
		Expect(err).NotTo(HaveOccurred())

		// Deploy OpenTelemetry Collector
		err = fw.SetupOTelCollector()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should use default resources", func() {
		// TODO: Add your performance test here
	})

	AfterEach(func() {
		if fw != nil {
			err := fw.Cleanup()
			Expect(err).NotTo(HaveOccurred())
		}
	})
})
