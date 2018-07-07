.PHONY: googlePProf


explore:
		go run main.go explore --level info

spartaPProf:
		go run main.go provision --s3Bucket $(S3_BUCKET) --level info

describeSpartaPProf:
		go run main.go describe --s3Bucket $(S3_BUCKET) --out ./google-pprof.html --level info

googlePProf:
		go run --tags googlepprof main.go provision --s3Bucket $(S3_BUCKET) --tags googlepprof --level info

describeGooglePProf:
		go run --tags googlepprof main.go describe --s3Bucket $(S3_BUCKET) --tags googlepprof --out ./google-pprof.html --level info