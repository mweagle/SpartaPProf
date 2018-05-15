.PHONY: googlePProf

spartaPProf:
	GCLOUD_KEYS_PATH=$(GOPATH)/src/github.com/mweagle/SpartaPProf/pprof/gcloud-spartapprof-keys.json \
		go run main.go provision --s3Bucket weagle --level info

describeSpartaPProf:
	GCLOUD_KEYS_PATH=$(GOPATH)/src/github.com/mweagle/SpartaPProf/pprof/gcloud-spartapprof-keys.json \
		go run main.go describe --s3Bucket weagle --out ./google-pprof.html --level info

googlePProf:
	GCLOUD_KEYS_PATH=$(GOPATH)/src/github.com/mweagle/SpartaPProf/pprof/gcloud-spartapprof-keys.json \
		go run --tags googlepprof main.go provision --s3Bucket weagle --tags googlepprof --level info

describeGooglePProf:
	GCLOUD_KEYS_PATH=$(GOPATH)/src/github.com/mweagle/SpartaPProf/pprof/gcloud-spartapprof-keys.json \
		go run --tags googlepprof main.go describe --s3Bucket weagle --tags googlepprof --out ./google-pprof.html --level info