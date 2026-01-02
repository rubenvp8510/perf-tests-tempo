package dashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

// GetTemplateFuncs returns the template function map
func GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatBytes":    formatBytes,
		"formatDuration": formatDuration,
		"formatPercent":  formatPercent,
		"formatTime":     formatTime,
		"formatValue":    formatValue,
		"toJSON":         toJSON,
		"getRunColor":    getRunColor,
		"sub":            sub,
	}
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes float64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%.0f B", bytes)
	}
	div, exp := float64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", bytes/div, "KMGTPE"[exp])
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", mins, secs)
		}
		return fmt.Sprintf("%dm", mins)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dh", hours)
}

// formatPercent formats a ratio as a percentage
func formatPercent(ratio float64) string {
	return fmt.Sprintf("%.1f%%", ratio*100)
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("15:04:05")
}

// formatValue formats a value with its unit
func formatValue(value float64, unit string) string {
	switch unit {
	case "bytes":
		return formatBytes(value)
	case "seconds":
		if value < 0.001 {
			return fmt.Sprintf("%.0f µs", value*1e6)
		}
		if value < 1 {
			return fmt.Sprintf("%.2f ms", value*1000)
		}
		return fmt.Sprintf("%.3f s", value)
	case "percent":
		return formatPercent(value)
	default:
		if value >= 1e6 {
			return fmt.Sprintf("%.2fM", value/1e6)
		}
		if value >= 1e3 {
			return fmt.Sprintf("%.2fK", value/1e3)
		}
		return fmt.Sprintf("%.2f", value)
	}
}

// toJSON converts a value to JSON for embedding in templates
func toJSON(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(b)
}

// getRunColor returns a color for a given run index
func getRunColor(index int) string {
	colors := []string{
		"rgba(233, 69, 96, 1)",  // red
		"rgba(52, 152, 219, 1)", // blue
		"rgba(46, 204, 113, 1)", // green
		"rgba(241, 196, 15, 1)", // yellow
	}
	return colors[index%len(colors)]
}

// sub subtracts b from a (for template use)
func sub(a, b int) int {
	return a - b
}
