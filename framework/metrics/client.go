package metrics

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// ClientConfig holds configuration for the Prometheus client
type ClientConfig struct {
	Namespace           string
	ThanosURL           string
	Token               string
	MonitoringNamespace string
	ServiceAccountName  string
	AutoDiscover        bool
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
		if config.ThanosURL == "" {
			url, err := client.discoverThanosURL()
			if err != nil {
				return nil, fmt.Errorf("failed to discover Thanos URL: %w", err)
			}
			client.config.ThanosURL = url
			fmt.Printf("✅ Discovered Thanos URL: %s\n", url)
		}

		if config.Token == "" {
			token, err := client.generateToken()
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

// discoverThanosURL discovers the Thanos Querier URL from OpenShift
func (c *Client) discoverThanosURL() (string, error) {
	cmd := exec.Command("oc", "get", "route", "thanos-querier",
		"-n", c.config.MonitoringNamespace,
		"-o", "jsonpath={.spec.host}")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get thanos-querier route: %w", err)
	}

	host := strings.TrimSpace(string(output))
	if host == "" {
		return "", fmt.Errorf("thanos-querier route not found")
	}

	return "https://" + host, nil
}

// generateToken generates a service account token
func (c *Client) generateToken() (string, error) {
	cmd := exec.Command("oc", "create", "token", c.config.ServiceAccountName,
		"-n", c.config.MonitoringNamespace,
		"--duration=1h")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	token := strings.TrimSpace(string(output))
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
