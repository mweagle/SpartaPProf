// +build googlepprof

package pprof

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	googleLogging "cloud.google.com/go/logging"
	"cloud.google.com/go/profiler"
	awsLambdaEvents "github.com/aws/aws-lambda-go/events"
	"github.com/mweagle/Sparta"
	"github.com/mweagle/ssm-cache"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
)

const GoogleProjectID = "spartapprof"

var cacheClient ssmcache.Client
var googleLogClient *googleLogging.Client
var googleLogger *googleLogging.Logger
var once sync.Once
var googleCredsPath = filepath.Join("/tmp", "googleCredsPProf.json")

////////////////////////////////////////////////////////////////////////////////
//
//  _____ ____  _   _ _______ ________   _________
// / ____/ __ \| \ | |__   __|  ____\ \ / /__   __|
// | |   | |  | |  \| |  | |  | |__   \ V /   | |
// | |   | |  | | . ` |  | |  |  __|   > <    | |
// | |___| |__| | |\  |  | |  | |____ / . \   | |
//  \_____\____/|_| \_|  |_|  |______/_/ \_\  |_|
//
////////////////////////////////////////////////////////////////////////////////
func init() {
	cacheClient = ssmcache.NewClient(5 * time.Minute)

	logger := log.New(os.Stdout, "logger: ", log.Llongfile|log.LUTC)

	// Read the logging client creds
	stringVal, stringErr := cacheClient.GetSecureString("GoogleCloudPProfEncryptedCredentials")
	if stringErr != nil {
		logger.Fatal("Failed to access Google credentials data. Error: ", stringErr)
		return
	} else {
		// Great, let's go ahead and write it out...
		errWrite := ioutil.WriteFile(googleCredsPath, []byte(stringVal), os.ModePerm)
		if errWrite != nil {
			logger.Fatal("Failed to save Google credentials file. Error: ", errWrite)
			return
		} else {
			// Excellent, set the env var and move on...
			logger.Printf("Saved Google creds to path: %s", googleCredsPath)
			setEnvErr := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", googleCredsPath)
			if setEnvErr != nil {
				logger.Fatal("Failed to set JSON path. Error: ", setEnvErr)
			} else {
				logger.Printf("Set env.GOOGLE_APPLICATION_CREDENTIALS to: %s", googleCredsPath)
			}
		}
	}
}
func oneTimeGoogleClientInit() {
	// TODO - consider parameterizing
	logger := log.New(os.Stdout, "logger: ", log.Llongfile|log.LUTC)

	loggingClient, loggingClientErr := googleLogging.NewClient(context.Background(),
		fmt.Sprintf("projects/%s", GoogleProjectID),
		option.WithCredentialsFile(googleCredsPath))
	if loggingClientErr != nil {
		logger.Fatalf("Failed to initialize Google logging client: %s", loggingClientErr)
	} else {
		// Setup the logging client
		googleLogClient = loggingClient
		googleLogger = googleLogClient.Logger("sparta", googleLogging.CommonResource(
			&mrpb.MonitoredResource{
				Type: "project",
				Labels: map[string]string{
					"project_id": GoogleProjectID,
				},
			}),
		)
		logger.Printf("Initialized Stackdriver Log")
	}
}

// LogRelayFunction is the mock relay function that ships
// logs to an external service. In the Google case, it ships the logfiles
// to StackDriver
func LogRelayFunction(ctx context.Context, kinesisEvent awsLambdaEvents.KinesisEvent) error {
	once.Do(oneTimeGoogleClientInit)
	if googleLogClient == nil {
		return nil
	}
	logger, _ := ctx.Value(sparta.ContextKeyLogger).(*logrus.Logger)

	// https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/SubscriptionFilters.html
	// Base64 encoded and compressed with the gzip format
	for _, eachRecord := range kinesisEvent.Records {
		b := bytes.NewBuffer(eachRecord.Kinesis.Data)
		reader, err := gzip.NewReader(b)
		if err != nil {
			logger.Error("Failed to read: %s", err.Error())
			continue
		}
		var gunzipBuffer bytes.Buffer
		_, err = gunzipBuffer.ReadFrom(reader)
		if err != nil {
			logger.Error("Failed to ReadFrom: %s", err.Error())
			continue
		}
		// Unmarshall that into a log
		var target awsLambdaEvents.CloudwatchLogsData
		unmarshalErr := json.Unmarshal(gunzipBuffer.Bytes(), &target)
		if unmarshalErr != nil {
			logger.Error("Failed to unmarshal: %s", unmarshalErr.Error())
		} else {
			for _, eachEvent := range target.LogEvents {
				// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
				syncErr := googleLogger.LogSync(ctx, googleLogging.Entry{
					Payload: json.RawMessage([]byte(eachEvent.Message)),
				})
				if syncErr != nil {
					if strings.Contains(syncErr.Error(), "json.Unmarshal") {
						syncErr = googleLogger.LogSync(ctx, googleLogging.Entry{
							Payload: eachEvent.Message,
						})
					}
				}
				if syncErr != nil {
					logger.Error("Failed to send <%s>. Error: %s", eachEvent.Message, syncErr.Error())
				}
			}
		}
	}
	return nil
}

// InitializeProfiler sets up the profiler
func InitializeProfiler() {
	logger := log.New(os.Stdout, "logger: ", log.Llongfile|log.LUTC)
	logger.Print("Initializing Google profiler from SSM keys")

	// So the service name has to match this regexp:
	// ^[a-z]([-a-z0-9_.]{0,253}[a-z0-9])?$
	// Per: https://cloud.google.com/profiler/docs/profiling-go
	stackDriverName := strings.ToLower(os.Getenv("AWS_LAMBDA_FUNCTION_NAME"))
	profileErr := profiler.Start(profiler.Config{
		ProjectID:      GoogleProjectID,
		Service:        stackDriverName,
		ServiceVersion: sparta.StampedBuildID,
		DebugLogging:   true,
	})
	if profileErr != nil {
		logger.Fatalf("Failed to start the Google profiler: %s", profileErr.Error())
	} else {
		logger.Print("Registered Google Profiler")
	}
}
