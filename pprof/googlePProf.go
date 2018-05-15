// +build googlepprof

package pprof

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/profiler"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/mweagle/Sparta"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	gcloudPrivateKeyFilename = "gcloud-keys.json"
)

func init() {
	// Set the env var to the location of the JSON file and then setup the
	// profiler...
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		grpc.EnableTracing = true

		log.Printf("Attempting to register Google profiler")
		pathToCreds := filepath.Join(os.Getenv("LAMBDA_TASK_ROOT"), gcloudPrivateKeyFilename)
		setEnvErr := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", pathToCreds)
		if setEnvErr != nil {
			fmt.Printf("Failed to set JSON path: " + setEnvErr.Error())
		} else {
			log.Printf("Set credentials path to: " + pathToCreds)
			profileErr := profiler.Start(profiler.Config{
				Service:        "awslambda-pprof",
				ServiceVersion: sparta.StampedBuildID,
				ProjectID:      "spartapprof",
				DebugLogging:   true,
			})
			if profileErr != nil {
				log.Fatalf("Cannot start the profiler: %v", profileErr)
			} else {
				log.Printf("Registered Google Profiler")
			}
		}
	}
}

// WorkflowHooks returns the workflow hooks to setup profiling
func WorkflowHooks() *sparta.WorkflowHooks {
	// ArchiveHook is responsible for injecting the Google
	// JSON file into the archive
	// TODO: Migrate this to an SSM store
	archiveHook := func(context map[string]interface{},
		serviceName string,
		zipWriter *zip.Writer,
		awsSession *session.Session,
		noop bool,
		logger *logrus.Logger) error {

		logger.Info("Adding Google Stackdriver credentials")
		binaryWriter, binaryWriterErr := zipWriter.Create(gcloudPrivateKeyFilename)
		if nil != binaryWriterErr {
			return binaryWriterErr
		}
		jsonPath := os.Getenv("GCLOUD_KEYS_PATH")
		if jsonPath == "" {
			return errors.Errorf("Please provide env.GCLOUD_KEYS_PATH that points to your JSON file")
		}
		content, contentErr := ioutil.ReadFile(jsonPath)
		if contentErr != nil {
			return contentErr
		}
		_, copyErr := io.Copy(binaryWriter, bytes.NewReader(content))
		return copyErr
	}
	return &sparta.WorkflowHooks{
		Archives: []sparta.ArchiveHookHandler{sparta.ArchiveHookFunc(archiveHook)},
	}
}
