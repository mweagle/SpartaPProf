package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	sparta "github.com/mweagle/Sparta"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	iamBuilder "github.com/mweagle/Sparta/aws/iam/builder"
	pprofImpl "github.com/mweagle/SpartaPProf/pprof"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/mweagle/ssm-cache"
	"github.com/sirupsen/logrus"
)

var cacheClient ssmcache.Client

func init() {
	cacheClient = ssmcache.NewClient(5 * time.Minute)
}

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
	expString, _ := cacheClient.GetExpiringString("RotatingString", 30*time.Second)
	return fmt.Sprintf("Hi there: %s 🌍", expString), nil
}

//----------------------------------------------------------------------------//

func main() {
	lambdaFn := sparta.HandleAWSLambda("Hello World",
		helloWorld,
		sparta.IAMRoleDefinition{})
	lambdaFn.Options.Timeout = 60
	lambdaFn.Options.MemorySize = 256

	// Enable the lambda function to access the Parameter Store
	lambdaFn.RoleDefinition.Privileges = append(lambdaFn.RoleDefinition.Privileges,
		iamBuilder.Allow("ssm:GetParameter", "ssm:GetParametersByPath").
			ForResource().
			Literal("arn:aws:ssm:").
			Region(":").
			AccountID(":").
			Literal("*").
			ToPrivilege())

	// Need to debug the gRPC connection? Set an env var
	// according to https://golang.org/pkg/net/http/
	// lambdaFn.Options.Environment = map[string]*gocf.StringExpr{
	// 	"GODEBUG": gocf.String("http2debug=2"),
	// }

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

	// Initialize the profiler
	if sparta.IsExecutingInLambda() {
		initLogger, initLoggerErr := sparta.NewLogger("info")
		if initLoggerErr != nil {
			panic("Failed to initialize logger: " + initLoggerErr.Error())
		}
		pprofImpl.InitializeProfiler(cacheClient, initLogger)
	}

	// Startup
	err := sparta.Main(pprofStackName,
		"Sparta application that demonstrates how to profile in AWS Lambda",
		lambdaFunctions,
		nil,
		nil)
	if err != nil {
		os.Exit(1)
	}
}
