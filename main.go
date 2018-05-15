package main

import (
	"context"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	sparta "github.com/mweagle/Sparta"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	pprofImpl "github.com/mweagle/SpartaPProf/pprof"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/sirupsen/logrus"
)

//----------------------------------------------------------------------------//
// Throwaway functions to generate load
//

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
	for i := 0; i < 128; i++ {
		s = append(s, "magical pandas")
		if (i % 32) == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func artificialLoad() {
	var once sync.Once
	once.Do(func() {
		go emptySelect()
	})
	go leakyFunction()
	load()
}

func load() {
	for i := 0; i < (1 << 7); i++ {
		rand.Int63()
	}
}

//----------------------------------------------------------------------------//

// Standard Sparta lambda function
func helloWorld(ctx context.Context) (string, error) {
	artificialLoad()
	return "Hi there 🌍", nil
}

//----------------------------------------------------------------------------//

func main() {
	lambdaFn := sparta.HandleAWSLambda("Hello World",
		helloWorld,
		sparta.IAMRoleDefinition{})
	lambdaFn.Options.Timeout = 60
	lambdaFn.Options.MemorySize = 256

	// How to get the build id inside the lambda function?
	lambdaFn.Options.Environment = map[string]*gocf.StringExpr{
		"GODEBUG": gocf.String("http2debug=2"),
	}
	arnDecorator := func(serviceName string,
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

	lambdaFn.Decorators = append(lambdaFn.Decorators,
		sparta.TemplateDecoratorHookFunc(arnDecorator))

	// Sanitize the name so that it doesn't have any spaces
	//stackName := spartaCF.UserScopedStackName("SpartaHello")
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFunctions = append(lambdaFunctions, lambdaFn)
	pprofStackName := spartaCF.UserScopedStackName("SpartaPProf")
	workflowHooks := pprofImpl.WorkflowHooks()

	err := sparta.MainEx(pprofStackName,
		"Sparta application that demonstrates sparta.ScheduleProfileLoop usage",
		lambdaFunctions,
		nil,
		nil,
		workflowHooks,
		false)
	if err != nil {
		os.Exit(1)
	}
}
