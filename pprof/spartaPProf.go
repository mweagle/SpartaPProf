// +build !googlepprof

package pprof

import (
	"time"

	sparta "github.com/mweagle/Sparta"
	ssmcache "github.com/mweagle/ssm-cache"
	"github.com/sirupsen/logrus"
)


// LogRelayFunction is the mock relay function that ships
// logs to an external service. In the default case it's a nop.
func LogRelayFunction(ctx context.Context) error
	return nil
}

// InitializeProfiler registers the profiler
func InitializeProfiler() {
	log.Info("Attempting to register Sparta profiler")

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
