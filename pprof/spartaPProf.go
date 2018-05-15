// +build !googlepprof

package pprof

import (
	"fmt"
	"os"
	"time"

	sparta "github.com/mweagle/Sparta"
)

// RegisterProfiler registers the profiler
func init() {
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		fmt.Printf("Attempting to register Sparta profiler")

		// Install the profiling hook. During `provision`, this will annotate
		// each lambda function with enough context to publish profile snapshots
		// Once the stack is deployed, use the /cmd/load.go file as in:
		// go run load.go ARN_TO_DEPLOYED_FUNCTION
		// to generate sample load. The lambda function will publish profile snapshots
		// to an S3 location which can then be interrogated locally by re-running
		// this application with the `profile` option
		sparta.ScheduleProfileLoop(nil,
			5*time.Second,
			30*time.Second,
			"goroutine",
			"heap",
			"threadcreate",
			"block",
			"mutex")
	}
}

// WorkflowHooks returns the workflow hooks to setup profiling
func WorkflowHooks() *sparta.WorkflowHooks {
	return &sparta.WorkflowHooks{}
}
