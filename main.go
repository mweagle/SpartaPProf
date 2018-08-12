package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	sparta "github.com/mweagle/Sparta"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	iamBuilder "github.com/mweagle/Sparta/aws/iam/builder"
	spartaDecorators "github.com/mweagle/Sparta/decorator"
	"github.com/mweagle/SpartaPProf/pprof"
	pprofImpl "github.com/mweagle/SpartaPProf/pprof"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/sirupsen/logrus"
)

var once sync.Once

////////////////////////////////////////////////////////////////////////////////
// _   _ _____ ___ _    ___
// | | | |_   _|_ _| |  / __|
// | |_| | | |  | || |__\__ \
//  \___/  |_| |___|____|___/
//
////////////////////////////////////////////////////////////////////////////////

func spartaLambdaMaker(functionName string,
	handler interface{}) *sparta.LambdaAWSInfo {

	lambdaFn := sparta.HandleAWSLambda(functionName,
		handler,
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
	lambdaFn.Options.Environment = map[string]*gocf.StringExpr{
		"GODEBUG": gocf.String("http2debug=0"),
	}

	lambdaFn.Decorators = append(lambdaFn.Decorators,
		spartaDecorators.PublishAttOutputDecorator(fmt.Sprintf("%s FunctionARN", functionName),
			fmt.Sprintf("%s Lambda ARN", functionName), "Arn"))

	return lambdaFn
}

////////////////////////////////////////////////////////////////////////////////
//  _      _   __  __ ___ ___   _
// | |    /_\ |  \/  | _ )   \ /_\
// | |__ / _ \| |\/| | _ \ |) / _ \
// |____/_/ \_\_|  |_|___/___/_/ \_\
///
////////////////////////////////////////////////////////////////////////////////

// Standard Sparta lambda function
func helloWorld(ctx context.Context) (string, error) {
	if sparta.IsExecutingInLambda() {
		once.Do(pprofImpl.InitializeProfiler)
	}

	//load.GenerateArtificialLoad()
	logger, _ := ctx.Value(sparta.ContextKeyLogger).(*logrus.Logger)
	logger.Info("Hello from AWS Lambda 👋")
	return fmt.Sprintf("Hi there: %s 🌍", "World"), nil
}

////////////////////////////////////////////////////////////////////////////////
//  __  __   _   ___ _  _
// |  \/  | /_\ |_ _| \| |
// | |\/| |/ _ \ | || .` |
// |_|  |_/_/ \_\___|_|\_|
//
////////////////////////////////////////////////////////////////////////////////
func main() {
	workflowHooks := &sparta.WorkflowHooks{}

	helloWorldLambda := spartaLambdaMaker("Hello World", helloWorld)
	loggingRelay := spartaLambdaMaker("KinesisLogConsumer", pprof.LogRelayFunction)

	// Create the Kinesis Log Aggregator resources
	kinesisResource := &gocf.KinesisStream{
		ShardCount: gocf.Integer(1),
	}
	kinesisMapping := &sparta.EventSourceMapping{
		StartingPosition: "TRIM_HORIZON",
		BatchSize:        10,
	}
	// Create the decorator
	decorator := spartaDecorators.NewLogAggregatorDecorator(kinesisResource,
		kinesisMapping,
		loggingRelay)

	// Make sure the LoggingRelay has privileges to read from Kinesis
	for _, eachStatement := range sparta.CommonIAMStatements.Kinesis {
		loggingRelay.RoleDefinition.Privileges = append(loggingRelay.RoleDefinition.Privileges, sparta.IAMRolePrivilege{
			Actions:  eachStatement.Action,
			Resource: gocf.GetAtt(decorator.KinesisLogicalResourceName(), "Arn"),
		})
	}

	// Two parts...
	lambdaFunctions := []*sparta.LambdaAWSInfo{helloWorldLambda, loggingRelay}

	// 1. Include the decorator for each lambda function. This
	// includes the function that relays events off of the Kinesis
	// stream. The reason for this is that the relay function needs a
	// EventSourceMapping attached to it.
	for _, eachLambda := range lambdaFunctions {
		if eachLambda.Decorators == nil {
			eachLambda.Decorators = make([]sparta.TemplateDecoratorHandler, 0)
		}
		eachLambda.Decorators = append(eachLambda.Decorators, decorator)
	}

	// 2. Add the decorator to the Service so that the single Kinesis
	// stream can be attached
	workflowHooks.ServiceDecorators = []sparta.ServiceDecoratorHookHandler{decorator}

	// Create the stack
	pprofStackName := spartaCF.UserScopedStackName("SpartaPProf")

	err := sparta.MainEx(pprofStackName,
		"Sparta application that demonstrates how to profile in AWS Lambda",
		lambdaFunctions,
		nil,
		nil,
		workflowHooks,
		false)
	if err != nil {
		os.Exit(1)
	}
}
