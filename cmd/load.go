package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

const totalRequests = 60

func main() {
	sess := session.Must(session.NewSession())
	lambaSvc := lambda.New(sess)
	if len(os.Args) < 2 {
		fmt.Printf("Please provide a lambda ARN command line argument")
		os.Exit(1)
	}
	lambdaARN := os.Args[1]
	invokeInput := &lambda.InvokeInput{
		FunctionName: aws.String(lambdaARN),
	}
	for i := 0; i < totalRequests; i++ {
		lambdaResponse, lambdaResponseErr := lambaSvc.Invoke(invokeInput)
		if lambdaResponseErr != nil {
			fmt.Printf("Failed to invoke function: %s\n", lambdaResponseErr.Error())
		} else {
			fmt.Printf("Lambda response (%d of %d): %s\n", i, totalRequests, string(lambdaResponse.Payload))
		}
		time.Sleep(10 * time.Millisecond)
	}
}
