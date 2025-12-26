package metrics

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/gvr"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ClientConfig holds configuration for the Prometheus client
type ClientConfig struct {
	Namespace           string
	ThanosURL           string
	Token               string
	MonitoringNamespace string
	ServiceAccountName  string
	AutoDiscover        bool

	// KubeConfig is optional; if provided, it will be used for auto-discovery
	KubeConfig *rest.Config
}

// Client represents a Prometheus/Thanos client
type Client struct {
	config     *ClientConfig
	httpClient *http.Client
	baseURL    string
}

// PrometheusResponse represents the response from Prometheus API
type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string             `json:"resultType"`
		Result     []PrometheusResult `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

// PrometheusResult represents a single result from Prometheus
type PrometheusResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
	Value  []interface{}     `json:"value,omitempty"`
}

// NewClient creates a new Prometheus client
func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	client := &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}

	// Auto-discover Thanos URL and token if needed
	if config.AutoDiscover {
		if config.KubeConfig == nil {
			return nil, fmt.Errorf("KubeConfig is required for auto-discovery")
		}

		if config.ThanosURL == "" {
			url, err := client.discoverThanosURL(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to discover Thanos URL: %w", err)
			}
			client.config.ThanosURL = url
			fmt.Printf("✅ Discovered Thanos URL: %s\n", url)
		}

		if config.Token == "" {
			token, err := client.generateToken(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to generate token: %w", err)
			}
			client.config.Token = token
			fmt.Printf("✅ Generated authentication token\n")
		}
	}

	if client.config.ThanosURL == "" {
		return nil, fmt.Errorf("Thanos URL is required")
	}

	if client.config.Token == "" {
		return nil, fmt.Errorf("authentication token is required")
	}

	client.baseURL = client.config.ThanosURL

	return client, nil
}

// discoverThanosURL discovers the Thanos Querier URL from OpenShift using Kubernetes client
func (c *Client) discoverThanosURL(ctx context.Context) (string, error) {
	dynamicClient, err := dynamic.NewForConfig(c.config.KubeConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create dynamic client: %w", err)
	}

	namespace := c.config.MonitoringNamespace
	if namespace == "" {
		namespace = "openshift-monitoring"
	}

	route, err := dynamicClient.Resource(gvr.Route).Namespace(namespace).Get(ctx, "thanos-querier", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get thanos-querier route: %w", err)
	}

	host, found, err := unstructured.NestedString(route.Object, "spec", "host")
	if err != nil || !found {
		return "", fmt.Errorf("thanos-querier route host not found")
	}

	if host == "" {
		return "", fmt.Errorf("thanos-querier route host is empty")
	}

	return "https://" + host, nil
}

// generateToken generates a service account token using Kubernetes client
func (c *Client) generateToken(ctx context.Context) (string, error) {
	clientset, err := kubernetes.NewForConfig(c.config.KubeConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	namespace := c.config.MonitoringNamespace
	if namespace == "" {
		namespace = "openshift-monitoring"
	}

	saName := c.config.ServiceAccountName
	if saName == "" {
		saName = "prometheus-k8s"
	}

	// Token expiration: 1 hour
	expirationSeconds := int64(3600)

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
		},
	}

	tokenResponse, err := clientset.CoreV1().ServiceAccounts(namespace).CreateToken(
		ctx,
		saName,
		tokenRequest,
		metav1.CreateOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create token for %s/%s: %w", namespace, saName, err)
	}

	token := strings.TrimSpace(tokenResponse.Status.Token)
	if token == "" {
		return "", fmt.Errorf("empty token received")
	}

	return token, nil
}

// QueryRange executes a range query against Prometheus
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*PrometheusResponse, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", fmt.Sprintf("%d", start.Unix()))
	params.Add("end", fmt.Sprintf("%d", end.Unix()))
	params.Add("step", fmt.Sprintf("%d", int(step.Seconds())))

	apiURL := fmt.Sprintf("%s/api/v1/query_range?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s - %s", promResp.ErrorType, promResp.Error)
	}

	return &promResp, nil
}

// Query executes an instant query against Prometheus
func (c *Client) Query(ctx context.Context, query string, evalTime time.Time) (*PrometheusResponse, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("time", fmt.Sprintf("%d", evalTime.Unix()))

	apiURL := fmt.Sprintf("%s/api/v1/query?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s - %s", promResp.ErrorType, promResp.Error)
	}

	return &promResp, nil
}
