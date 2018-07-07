// +build googlepprof

package pprof

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"strings"

	"cloud.google.com/go/profiler"
	"github.com/mweagle/Sparta"
	"github.com/mweagle/ssm-cache"
	"github.com/sirupsen/logrus"
)

// InitializeProfiler sets up the profiler
func InitializeProfiler(cacheClient ssmcache.Client, logger *logrus.Logger) {
	logger.Info("Initializing Google profiler from SSM keys")

	// https://docs.aws.amazon.com/lambda/latest/dg/current-supported-versions.html
	// Setup a ssmClient with a 5 min expiry
	stringVal, stringErr := cacheClient.GetSecureString("GoogleCloudPProfEncryptedCredentials")
	if stringErr != nil {
		logger.Error("Failed to access Google credentials data. Error: ", stringErr)
		return
	} else {
		// Great, let's go ahead and write it out...
		googleCredsPath := filepath.Join("/tmp", "googleCredsPProf.json")
		errWrite := ioutil.WriteFile(googleCredsPath, []byte(stringVal), os.ModePerm)
		if errWrite != nil {
			logger.Error("Failed to save Google credentials file. Error: ", errWrite)
			return
		} else {
			// Sweet, set the env var and move on...
			logger.Info("Saved Google creds to path: ", googleCredsPath)
			setEnvErr := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", googleCredsPath)
			if setEnvErr != nil {
				logger.Error("Failed to set JSON path. Error: ", setEnvErr)
			} else {
				logger.Info("Set credentials path to: ", googleCredsPath)
				// So the service name has to match this regexp:
				// ^[a-z]([-a-z0-9_.]{0,253}[a-z0-9])?$
				// Per: https://cloud.google.com/profiler/docs/profiling-go
				stackDriverName := strings.ToLower(os.Getenv("AWS_LAMBDA_FUNCTION_NAME"))
				profileErr := profiler.Start(profiler.Config{
					Service:        stackDriverName,
					ServiceVersion: sparta.StampedBuildID,
					ProjectID:      "spartapprof",
					DebugLogging:   true,
				})
				if profileErr != nil {
					logger.Error("Failed to start the Google profiler: ", profileErr)
				} else {
					logger.Info("Registered Google Profiler")
				}
			}
		}
	}
}
