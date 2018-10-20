// +build mage

package main

import (
	"os"

	spartaMage "github.com/mweagle/Sparta/magefile"
)

// ProvisionSparta the Sparta PPRof version
func ProvisionSparta() error {
	return spartaMage.SpartaCommand("provision",
		"--s3Bucket",
		os.Getenv("S3_BUCKET"))
}

// ProvisionGoogle the Google PProf version
func ProvisionGoogle() error {
	return spartaMage.SpartaCommand("provision",
		"--s3Bucket",
		os.Getenv("S3_BUCKET"),
		"--tags",
		"googlepprof")
}

// DescribeSparta produces a description of the Sparta version
func DescribeSparta() error {
	return spartaMage.SpartaCommand("describe",
		"--s3Bucket",
		os.Getenv("S3_BUCKET"),
		"--out",
		"sparta-pprof.html")
}

// DescribeGoogle produces a description of the Google version
func DescribeGoogle() error {
	return spartaMage.SpartaCommand("describe",
		"--s3Bucket",
		os.Getenv("S3_BUCKET"),
		"--out",
		"google-pprof.html")
}

// Delete the service, iff it exists
func Delete() error {
	return spartaMage.Delete()
}

// Status report if the stack has been provisioned
func Status() error {
	return spartaMage.Status(true)
}

// Version information
func Version() error {
	return spartaMage.Version()
}
