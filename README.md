# vpcflow-digesterd #

**A service which creates, stores, and fetches digests for VPC flow logs**

## Overview ##

AWS VPC Flow Logs are a data source by which a team can detect anomalies in connection patterns, use of non-standard ports,
or even view the interconnections of systems. To assist in the consumption and analysis of these logs, vpcflow-digesterd
provides APIs for generating vpc flow log digests and for retrieving those digests.

A digests is defined by a window of time specified in the `start` and `stop` REST API query parameters. See [api.yaml]( https://bitbucket.org/atlassian/vpcflow-digesterd/src/master/api.yaml) for more information.

This project has two major components: an API to create and fetch digests, and a worker which performs the actual log compaction.
This allows for multiple setups depending on your use case. For example, for the simplest setup, this project can run as a standalone
service if `STREAM_APPLIANCE_ENDPOINT` is set to `0.0.0.0:<PORT>`. Another, more asynchronous setup would involve running vpcflow-digesterd
as two services, with the API component producing to some event bus, and configuring the event bus to POST into the worker component.

## Modules ##

The service struct in the digesterd package contains the modules used by this application. If none of these modules are configured,
the built-in modules will be used.

```
func main() {
	...

	// Service created with default modules
	service := &digesterd.Service{
		Middleware: middleware,
	}

	...
}
```

### Storage ###

This module is responsible for storing and retrieving the vpc log digests. The built-in storage module uses S3 as the store and
can be configured with the `DIGEST_STORAGE_BUCKET` and `DIGEST_STORAGE_BUCKET_REGION` environment variables. To use a custom storage
module, implement the `types.Storage` interface and set the Storage attribute on the `digesterd.Service` struct in your `main.go`.

### Marker ###

As previously described, the project components can be configured to run asynchronously. The Marker module is used to mark when a
digest is in progess of being created and when a digest is complete. The built-in Marker uses S3 as its backend and can be configured
with the `DIGEST_PROGRESS_BUCKET` and `DIGEST_PROGRESS_BUCKET_REGION` environment variables. To use a custom marker module, implement
the `types.Marker` interface and set the Marker attribute on the `digesterd.Service` struct in your `main.go`.

### Queuer ###

This module is responsible for queuing digester jobs which will eventually be consumed by the Produce handler. The built-in Queuer POSTs
to an HTTP endpoint. It can be configured with the `STREAM_APPLIANCE_ENDPOINT` environment variable. This
project can be configured to run asynchronously if the queuer POSTs to some event bus and returns immdetiately, so long as a 200 response
from that event bus indicates that the digest job will eventually be POSTed to the worker component of the project. To use a custom queuer
module, implement the `types.Queuer` interface and set the Queuer attribute on the `digesterd.Service` struct in your `main.go`.

### HTTPClient ###

This is the client to be used with the default Queuer module. If no client is provided, a default will be used. This project makes use of
the [transport](https://bitbucket.org/atlassian/transport) library which provides a thin layer of configuration on top of the `http.Client`
from the standard lib. While the HTTP client that is built-in to this project will be sufficient for most uses cases, a custom one can be
provided by setting the HTTPClient attribute on the `digesterd.Service` struct in your `main.go`.

### Logging ###

This project uses [logevent](https://bitbucket.org/atlassian/logevent) as its logging interface. Structured logs that this project emits
can be found in the `logs` package. This project comes with a couple of default logging implementations that can be found in the plugins
package. These loggers are injected via HTTP middleware on the request context.

```
func main() {
	router := chi.NewRouter()
	middleware := []func(http.Handler) http.Handler{
		plugins.DefaultLogMiddleware(), // injects a logger which sends to os.Stdout
	}
	service := &digesterd.Service{Middleware: middleware}
	if err := service.BindRoutes(router); err != nil {
		panic(err.Error())
	}
}
```
Please note that this project will not run without some sort of logger being installed. While it's not recommended, if you wish to omit
logging, use the `NopLogMiddleware`.

### Stats ###

This project uses [xstats](https://github.com/rs/xstats) as the stats client. It supports a decent range of backends. The default stats
backend for the project is statsd using the datadog tagging extensions. The default backend will send stats to "localhost:8126". To change
the destination or the backend install the `CustomStatMiddleware` with your own xstats client.

### ExitSignals ###

Exit signals in this project are used to signal the service to perform a graceful shutdown. The built-in exit signal listens for SIGTERM and SIGINT and signals to the main routine to shutdown the service.

## Setup ##

* configure AWS to [publish flow logs to S3](https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-s3.html)
* create a bucket in AWS to store the created digests
* create a bucket in AWS to store progress states for queued digests
* setup environment variables

| Name                        | Required | Description                                                                                                                     | Example                                         |
|-----------------------------|:--------:|---------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------|
| PORT                        |    No   | HTTP Port for application (defaults to 8080)                                                                                     | 8080                                            |
| VPC\_FLOW\_LOGS\_BUCKET        |    Yes   | Bucket Name which holds VPC flow logs                                                                                           | vpc-flow-logs                                   |
| VPC\_FLOW\_LOGS\_BUCKET\_REGION        |    Yes   | Bucket region for VPC\_FLOW\_LOGS\_BUCKET                                                                                          | us-west-2                                   |
| VPC\_FLOW\_LOGS\_BUCKET\_ROLE      |    No   | Role ARN to assume which grants read access to the VPC Flow Logs bucket                                                                     | arn:aws:iam::account-id:role/role-name                   |
| VPC\_FLOW\_LOGS\_SCAN\_REGIONS      |    No   | Comma separated list of regions to scan for VPC Flow Logs. If omitted, will scan all regions                                                 | us-west-2,us-east-2                   |
| VPC\_FLOW\_LOGS\_SCAN\_ACCOUNTS      |    No   | Comma separated list of AWS accounts to scan for VPC Flow Logs. If omitted, will scan all accounts                                                 | 123456789011,123456789012                    |
| VPC\_MAX\_BYTES\_PREFETCH      |    Yes   | When making the digest, the max number of bytes to prefetch from the bucket objects                                             | 150000000                                       |
| VPC\_MAX\_CONCURRENT\_PREFETCH |    Yes   | When making the digest, the max number of bucket objects to prefetch                                                            | 2                                               |
| DIGEST\_STORAGE\_BUCKET       |    Yes   | The name of the S3 bucket used to store digests                                                                                 | vpc-flow-digests                                |
| DIGEST\_STORAGE\_BUCKET\_REGION       |    Yes   | The region of the S3 bucket used to store digests                                                                                 | us-west-2                                |
| DIGEST\_STORAGE\_BUCKET\_ROLE      |    No   | Role ARN to assume which grants read access to the digest storage bucket                                                                     | arn:aws:iam::account-id:role/role-name                   |
| DIGEST\_PROGRESS\_BUCKET      |    Yes   | The name of the S3 bucket used to store digest progress states                                                                  | vpc-flow-digests-progress                       |
| DIGEST\_PROGRESS\_BUCKET\_REGION      |    Yes   | The region of the S3 bucket used to store digest progress states            | us-west-2                       |
| DIGEST\_PROGRESS\_BUCKET\_ROLE      |    No   | Role ARN to assume which grants read access to the digest progress bucket                                                                     | arn:aws:iam::account-id:role/role-name                   |
| DIGEST\_PROGRESS\_TIMEOUT			  |    Yes  | Time, in milliseconds, after which an in progress marker is considered invalid					| 100000 |
| STREAM\_APPLIANCE\_ENDPOINT   |    Yes   | Endpoint for the service which queues digests to be created. | http://ec2-event-bus.us-west-2.compute.amazonaws.com |
| STREAM\_APPLIANCE\_TOPIC      |    Yes   | Event bus name.                                                  | digest-queue                                    |
| USE\_IAM                     |    Yes   | true or false. Set this flag to true if your application will be assuming an IAM role to read and write to the S3 buckets. This is recommended if you are deploying your application to an ec2 instance.       | true                                            |
| AWS\_CREDENTIALS\_FILE        |    No    | If not using IAM, use this to specify a credential file                                                                         | ~/.aws/credentials                              |
| AWS\_CREDENTIALS\_PROFILE     |    No    | If not using IAM, use this to specify the credentials profile to use                                                            | default                                         |
| AWS\_ACCESS\_KEY\_ID           |    No    | If not using IAM, use this to specify an AWS access key ID                                                                      |                                                 |
| AWS\_SECRET\_ACCESS\_KEY       |    No    | If not using IAM, use this to specify an AWS secret key                                                                         |                                                 |



## Contributing ##

### License ###

This project is licensed under Apache 2.0. See LICENSE.txt for details.

### Contributing Agreement ###

Atlassian requires signing a contributor's agreement before we can accept a
patch. If you are an individual you can fill out the
[individual CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=3f94fbdc-2fbe-46ac-b14c-5d152700ae5d).
If you are contributing on behalf of your company then please fill out the
[corporate CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=e1c17c66-ca4d-4aab-a953-2c231af4a20b).
