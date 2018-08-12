# SpartaPProf
Sparta-based application that demonstrates how to add `pprof` support to your lambda application.

See the [profiling](http://gosparta.io/docs/profiling/) documentation for complete details.

## Sparta Target

`make spartaPProf`

The Sparta target performs snapshoting and posting to S3. These snapshots are then visualized via the `profile` option of your application.

## Google Target

`make googlePProf`

This target uses the Google Stackdriver profiler to send snapshots of your lambda functions to Stackdriver for visualization.  It also creates a Lambda function that subscribes to the logs of the primary load generation function (`helloWorld`). The subscription is mediated by a Kinesis stream and the records are delivered to StackDriver logs.

<div align="center"><img src="https://raw.githubusercontent.com/mweagle/SpartaPProf/master/site/Logs_Viewer-SpartaPProf.jpg" />
</div>

See [Centralised logging for AWS Lambda](https://theburningmonk.com/2018/07/centralised-logging-for-aws-lambda-revised-2018/) for other options for centralizing your lambda function logs.