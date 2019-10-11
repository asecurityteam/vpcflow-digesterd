<a id="markdown-vpcflow-digesterd-a-service-which-creates-stores-and-fetches-digests-for-vpc-flow-logs" name="vpcflow-digesterd-a-service-which-creates-stores-and-fetches-digests-for-vpc-flow-logs"></a>
# vpcflow-digesterd A service which creates, stores, and fetches digests for VPC flow logs #
[![GoDoc](https://godoc.org/github.com/asecurityteam/vpcflow-digesterd?status.svg)](https://godoc.org/github.com/asecurityteam/vpcflow-digesterd)
[![Build Status](https://travis-ci.com/asecurityteam/vpcflow-digesterd.png?branch=master)](https://travis-ci.com/asecurityteam/vpcflow-digesterd)
[![codecov.io](https://codecov.io/github/asecurityteam/vpcflow-digesterd/coverage.svg?branch=master)](https://codecov.io/github/asecurityteam/vpcflow-digesterd?branch=master)

*Status: Incubation*

<!-- TOC -->

- [vpcflow-digesterd A service which creates, stores, and fetches digests for VPC flow logs](#vpcflow-digesterd-a-service-which-creates-stores-and-fetches-digests-for-vpc-flow-logs)
    - [Overview](#overview)
    - [Modules](#modules)
        - [Storage](#storage)
        - [Marker](#marker)
        - [Queuer](#queuer)
        - [HTTPClient](#httpclient)
        - [Logging](#logging)
        - [Stats](#stats)
        - [ExitSignals](#exitsignals)
    - [Setup](#setup)
    - [Contributing](#contributing)
        - [License](#license)
        - [Contributing Agreement](#contributing-agreement)

<!-- /TOC -->

<a id="markdown-overview" name="overview"></a>
## Overview ##

AWS VPC Flow Logs are a data source by which a team can detect anomalies in connection patterns, use of non-standard ports,
or even view the interconnections of systems. To assist in the consumption and analysis of these logs, vpcflow-digesterd
provides APIs for generating vpc flow log digests and for retrieving those digests.

A digests is defined by a window of time specified in the `start` and `stop` REST API query parameters. See [api.yaml]( https://github.com/asecurityteam/vpcflow-digesterd/src/master/api.yaml) for more information.

This project has two major components: an API to create and fetch digests, and a worker which performs the actual log compaction.
This allows for multiple setups depending on your use case. For example, for the simplest setup, this project can run as a standalone
service if `STREAM_APPLIANCE_ENDPOINT` is set to `<RUNTIME_HTTPSERVER_ADDRESS>`. Another, more asynchronous setup would involve running vpcflow-digesterd
as two services, with the API component producing to some event bus, and configuring the event bus to POST into the worker component.

<a id="markdown-modules" name="modules"></a>
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

<a id="markdown-storage" name="storage"></a>
### Storage ###

This module is responsible for storing and retrieving the vpc log digests. The built-in storage module uses S3 as the store and
can be configured with the `DIGEST_STORAGE_BUCKET` and `DIGEST_STORAGE_BUCKET_REGION` environment variables. To use a custom storage
module, implement the `types.Storage` interface and set the Storage attribute on the `digesterd.Service` struct in your `main.go`.

<a id="markdown-marker" name="marker"></a>
### Marker ###

As previously described, the project components can be configured to run asynchronously. The Marker module is used to mark when a
digest is in progess of being created and when a digest is complete. The built-in Marker uses S3 as its backend and can be configured
with the `DIGEST_PROGRESS_BUCKET` and `DIGEST_PROGRESS_BUCKET_REGION` environment variables. To use a custom marker module, implement
the `types.Marker` interface and set the Marker attribute on the `digesterd.Service` struct in your `main.go`.

<a id="markdown-queuer" name="queuer"></a>
### Queuer ###

This module is responsible for queuing digester jobs which will eventually be consumed by the Produce handler. The built-in Queuer POSTs
to an HTTP endpoint. It can be configured with the `STREAM_APPLIANCE_ENDPOINT` environment variable. This
project can be configured to run asynchronously if the queuer POSTs to some event bus and returns immdetiately, so long as a 200 response
from that event bus indicates that the digest job will eventually be POSTed to the worker component of the project. To use a custom queuer
module, implement the `types.Queuer` interface and set the Queuer attribute on the `digesterd.Service` struct in your `main.go`.

<a id="markdown-httpclient" name="httpclient"></a>
### HTTPClient ###

This is the client to be used with the default Queuer module. If no client is provided, a default will be used. This project makes use of
the [transport](https://github.com/asecurityteam/transport) library which provides a thin layer of configuration on top of the `http.Client`
from the standard lib. While the HTTP client that is built-in to this project will be sufficient for most uses cases, a custom one can be
provided by setting the HTTPClient attribute on the `digesterd.Service` struct in your `main.go`.

<a id="markdown-logging" name="logging"></a>
### Logging ###

This project uses [runhttp's Logger](https://github.com/asecurityteam/runhttp/blob/master/domain.go#L13) as its logging interface. Structured logs that this project emits
can be found in the `logs` package. The runhttp runtime injects loggers via HTTP middleware on the request context.

<a id="markdown-stats" name="stats"></a>
### Stats ###

This project uses [runhttp's Stat](https://github.com/asecurityteam/runhttp/blob/master/domain.go#L25) as the stats client. It supports a decent range of backends. The default stats
backend for the project is statsd using the datadog tagging extensions. The default backend will send stats to "localhost:8125". To change
the destination, modify the `RUNTIME_STATS_OUTPUT` environment variable.

<a id="markdown-exitsignals" name="exitsignals"></a>
### ExitSignals ###

Exit signals in this project are used to signal the service to perform a graceful shutdown. The built-in exit signal listens for SIGTERM and SIGINT and signals to the main routine to shutdown the service.

<a id="markdown-setup" name="setup"></a>
## Setup ##

* configure AWS to [publish flow logs to S3](https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-s3.html)
* create a bucket in AWS to store the created digests
* create a bucket in AWS to store progress states for queued digests
* setup environment variables

| Name                                | Required | Description                                                                                                                                                                                              | Example                                              |
|-------------------------------------|:--------:|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------|
| VPC\_FLOW\_LOGS\_BUCKET             |   Yes    | Bucket Name which holds VPC flow logs                                                                                                                                                                    | vpc-flow-logs                                        |
| VPC\_FLOW\_LOGS\_BUCKET\_REGION     |   Yes    | Bucket region for VPC\_FLOW\_LOGS\_BUCKET                                                                                                                                                                | us-west-2                                            |
| VPC\_FLOW\_LOGS\_BUCKET\_ROLE       |    No    | Role ARN to assume which grants read access to the VPC Flow Logs bucket                                                                                                                                  | arn:aws:iam::account-id:role/role-name               |
| VPC\_FLOW\_LOGS\_SCAN\_REGIONS      |    No    | Comma separated list of regions to scan for VPC Flow Logs. If omitted, will scan all regions                                                                                                             | us-west-2,us-east-2                                  |
| VPC\_FLOW\_LOGS\_SCAN\_ACCOUNTS     |    No    | Comma separated list of AWS accounts to scan for VPC Flow Logs. If omitted, will scan all accounts                                                                                                       | 123456789011,123456789012                            |
| VPC\_MAX\_BYTES\_PREFETCH           |   Yes    | When making the digest, the max number of bytes to prefetch from the bucket objects                                                                                                                      | 150000000                                            |
| VPC\_MAX\_CONCURRENT\_PREFETCH      |   Yes    | When making the digest, the max number of bucket objects to prefetch                                                                                                                                     | 2                                                    |
| DIGEST\_STORAGE\_BUCKET             |   Yes    | The name of the S3 bucket used to store digests                                                                                                                                                          | vpc-flow-digests                                     |
| DIGEST\_STORAGE\_BUCKET\_REGION     |   Yes    | The region of the S3 bucket used to store digests                                                                                                                                                        | us-west-2                                            |
| DIGEST\_STORAGE\_BUCKET\_ROLE       |    No    | Role ARN to assume which grants read access to the digest storage bucket                                                                                                                                 | arn:aws:iam::account-id:role/role-name               |
| DIGEST\_PROGRESS\_BUCKET            |   Yes    | The name of the S3 bucket used to store digest progress states                                                                                                                                           | vpc-flow-digests-progress                            |
| DIGEST\_PROGRESS\_BUCKET\_REGION    |   Yes    | The region of the S3 bucket used to store digest progress states                                                                                                                                         | us-west-2                                            |
| DIGEST\_PROGRESS\_BUCKET\_ROLE      |    No    | Role ARN to assume which grants read access to the digest progress bucket                                                                                                                                | arn:aws:iam::account-id:role/role-name               |
| DIGEST\_PROGRESS\_TIMEOUT           |   Yes    | Time, in milliseconds, after which an in progress marker is considered invalid                                                                                                                           | 100000                                               |
| STREAM\_APPLIANCE\_ENDPOINT         |   Yes    | Endpoint for the service which queues digests to be created.                                                                                                                                             | http://ec2-event-bus.us-west-2.compute.amazonaws.com |
| STREAM\_APPLIANCE\_TOPIC            |   Yes    | Event bus name.                                                                                                                                                                                          | digest-queue                                         |
| USE\_IAM                            |   Yes    | true or false. Set this flag to true if your application will be assuming an IAM role to read and write to the S3 buckets. This is recommended if you are deploying your application to an ec2 instance. | true                                                 |
| AWS\_CREDENTIALS\_FILE              |    No    | If not using IAM, use this to specify a credential file                                                                                                                                                  | ~/.aws/credentials                                   |
| AWS\_CREDENTIALS\_PROFILE           |    No    | If not using IAM, use this to specify the credentials profile to use                                                                                                                                     | default                                              |
| AWS\_ACCESS\_KEY\_ID                |    No    | If not using IAM, use this to specify an AWS access key ID                                                                                                                                               |                                                      |
| AWS\_SECRET\_ACCESS\_KEY            |    No    | If not using IAM, use this to specify an AWS secret key                                                                                                                                                  |                                                      |
| RUNTIME_HTTPSERVER_ADDRESS          |   Yes    | (string) The listening address of the server.                                                                                                                                                            | :8080                                                |
| RUNTIME_CONNSTATE_REPORTINTERVAL    |   YES    | (time.Duration) Interval on which gauges are reported.                                                                                                                                                   | 5s                                                   |
| RUNTIME_CONNSTATE_HIJACKEDCOUNTER   |   YES    | (string) Name of the counter metric tracking hijacked clients.                                                                                                                                           | http.server.connstate.hijacked                       |
| RUNTIME_CONNSTATE_CLOSEDCOUNTER     |   YES    | (string) Name of the counter metric tracking closed clients.                                                                                                                                             | http.server.connstate.closed                         |
| RUNTIME_CONNSTATE_IDLEGAUGE         |   YES    | (string) Name of the gauge metric tracking idle clients.                                                                                                                                                 | http.server.connstate.idle.gauge                     |
| RUNTIME_CONNSTATE_IDLECOUNTER       |   YES    | (string) Name of the counter metric tracking idle clients.                                                                                                                                               | http.server.connstate.idle                           |
| RUNTIME_CONNSTATE_ACTIVEGAUGE       |   YES    | string) Name of the gauge metric tracking active clients.                                                                                                                                                | http.server.connstate.active.gauge                   |
| RUNTIME_CONNSTATE_ACTIVECOUNTER     |   YES    | (string) Name of the counter metric tracking active clients.                                                                                                                                             | http.server.connstate.active                         |
| RUNTIME_CONNSTATE_NEWGAUGE          |   YES    | (string) Name of the gauge metric tracking new clients.                                                                                                                                                  | http.server.connstate.new.gauge                      |
| RUNTIME_CONNSTATE_NEWCOUNTER        |   YES    | (string) Name of the counter metric tracking new clients.                                                                                                                                                | http.server.connstate.new                            |
| RUNTIME_LOGGER_OUTPUT               |   YES    | (string) Destination stream of the logs. One of STDOUT, NULL.                                                                                                                                            | STDOUT                                               |
| RUNTIME_LOGGER_LEVEL                |   YES    | (string) The minimum level of logs to emit. One of DEBUG, INFO, WARN, ERROR.                                                                                                                             | INFO                                                 |
| RUNTIME_STATS_OUTPUT                |   YES    | (string) Destination stream of the stats. One of NULLSTAT, DATADOG.                                                                                                                                      | DATADOG                                              |
| RUNTIME_STATS_DATADOG_PACKETSIZE    |   YES    | (int) Max packet size to send.                                                                                                                                                                           | 32768                                                |
| RUNTIME_STATS_DATADOG_TAGS          |   YES    | ([]string) Any static tags for all metrics.                                                                                                                                                              | ""                                                   |
| RUNTIME_STATS_DATADOG_FLUSHINTERVAL |   YES    | (time.Duration) Frequencing of sending metrics to listener.                                                                                                                                              | 10s                                                  |
| RUNTIME_STATS_DATADOG_ADDRESS       |   YES    | (string) Listener address to use when sending metrics.                                                                                                                                                   | localhost:8125                                       |
| RUNTIME_SIGNALS_INSTALLED           |   YES    | ([]string) Which signal handlers are installed. Choices are OS.                                                                                                                                          | OS                                                   |
| RUNTIME_SIGNALS_OS_SIGNALS          |   YES    | ([]int) Which signals to listen for.                                                                                                                                                                     | 15 2                                                 |


<a id="markdown-contributing" name="contributing"></a>
## Contributing ##

<a id="markdown-license" name="license"></a>
### License ###

This project is licensed under Apache 2.0. See LICENSE.txt for details.

<a id="markdown-contributing-agreement" name="contributing-agreement"></a>
### Contributing Agreement ###

Atlassian requires signing a contributor's agreement before we can accept a
patch. If you are an individual you can fill out the
[individual CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=3f94fbdc-2fbe-46ac-b14c-5d152700ae5d).
If you are contributing on behalf of your company then please fill out the
[corporate CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=e1c17c66-ca4d-4aab-a953-2c231af4a20b).
