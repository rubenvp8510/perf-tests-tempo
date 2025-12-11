package framework

import "fmt"

// SetupTempo deploys Tempo (monolithic or stack) with optional resource configuration
// variant: "monolithic" or "stack"
// resources: optional resource configuration (only applies to monolithic)
func (f *Framework) SetupTempo(variant string, resources *ResourceConfig) error {
	switch variant {
	case "monolithic":
		return f.SetupTempoMonolithic(resources)
	case "stack":
		if resources != nil {
			// Resources are not supported for stack variant
			// Log a warning but continue
		}
		return f.SetupTempoStack()
	default:
		return fmt.Errorf("invalid tempo variant: %s (must be 'monolithic' or 'stack')", variant)
	}
}
