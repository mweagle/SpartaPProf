package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	sparta "github.com/mweagle/Sparta"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	gocf "github.com/mweagle/go-cloudformation"
)

func emptySelect() {
	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)

	quit := make(chan bool)

	for i := 0; i < n; i++ {
		go func() {
			for {
				select {
				case <-quit:
					return
				default:
				}
			}
		}()
	}
	time.Sleep(20 * time.Second)
	for i := 0; i < n; i++ {
		quit <- true
	}
}

// Adapted from https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/
func leakyFunction() {
	s := make([]string, 3)
	for i := 0; i < 1000; i++ {
		s = append(s, "magical pandas")
		if (i % 100) == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Standard Sparta lambda function
func helloWorld(w http.ResponseWriter, r *http.Request) {
	go leakyFunction()
	var once sync.Once
	once.Do(func() {
		go emptySelect()
	})
	fmt.Fprintf(w, "Hi there 🌍")
}

func main() {
	// Install the profiling hook. During `provision`, this will annotate
	// each lambda function with enough context to publish profile snapshots
	// Once the stack is deployed, use the /cmd/load.go file as in:
	// go run load.go ARN_TO_DEPLOYED_FUNCTION
	// to generate sample load. The lambda function will publish profile snapshots
	// to an S3 location which can then be interrogated locally by re-running
	// this application with the `profile` option
	sparta.ScheduleProfileLoop(nil, 5*time.Second, 30*time.Second,
		"goroutine",
		"heap",
		"threadcreate",
		"block",
		"mutex")

	lambdaFn := sparta.HandleAWSLambda("Hello World",
		http.HandlerFunc(helloWorld),
		sparta.IAMRoleDefinition{})
	lambdaFn.Options.Timeout = 60
	lambdaFn.Options.MemorySize = 256
	lambdaFn.Decorator = func(serviceName string,
		lambdaResourceName string,
		lambdaResource gocf.LambdaFunction,
		resourceMetadata map[string]interface{},
		S3Bucket string,
		S3Key string,
		buildID string,
		template *gocf.Template,
		context map[string]interface{},
		logger *logrus.Logger) error {

		// Add the function ARN as a stack output
		template.Outputs["FunctionARN"] = &gocf.Output{
			Description: "Lambda function ARN",
			Value:       gocf.GetAtt(lambdaResourceName, "Arn"),
		}
		return nil
	}
	// Sanitize the name so that it doesn't have any spaces
	//stackName := spartaCF.UserScopedStackName("SpartaHello")
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFunctions = append(lambdaFunctions, lambdaFn)
	pprofStackName := spartaCF.UserScopedStackName("SpartaPProf")
	err := sparta.Main(pprofStackName,
		"Sparta application that demonstrates sparta.ScheduleProfileLoop usage",
		lambdaFunctions,
		nil,
		nil)
	if err != nil {
		os.Exit(1)
	}
}
