# vpcflow-digesterd #

**A service which creates, stores, and fetches digests for VPC flow logs**

## Overview ##

AWS Flow Logs are a data source by which a team can detect anomalies in connection patterns, use of non-standard ports, or even view the interconnections of systems.
To assist in the consumption and analysis of these logs, vpcflow-digesterd provides APIs for generating digests and for retrieving those digests.

A digests is defined by a window of time specified in the `start` and `stop` REST API query parameters. See [api.yaml]( https://bitbucket.org/atlassian/vpcflow-digesterd/src/master/api.yaml) for more information.

## Setup ##

* configure AWS to [publish flow logs to S3](https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-s3.html)
* create a bucket in AWS to store the created digests
* create a bucket in AWS to store progress states for queued digests
* setup environment variables

| Name                        | Required | Description                                                                                                                     | Example                                         |
|-----------------------------|:--------:|---------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------|
| PORT                        |    No   | HTTP Port for application (defaults to 8080)                                                                                     | 8080                                            |
| VPC\_FLOW\_LOGS\_BUCKET        |    Yes   | Bucket Name which holds VPC flow logs                                                                                           | vpc-flow-logs                                   |
| VPC\_MAX\_BYTES\_PREFETCH      |    Yes   | When making the digest, the max number of bytes to prefetch from the bucket objects                                             | 150000000                                       |
| VPC\_MAX\_CONCURRENT\_PREFETCH |    Yes   | When making the digest, the max number of bucket objects to prefetch                                                            | 2                                               |
| DIGEST\_STORAGE\_BUCKET       |    Yes   | The name of the S3 bucket used to store digests                                                                                 | vpc-flow-digests                                |
| DIGEST\_PROGRESS\_BUCKET      |    Yes   | The name of the S3 bucket used to store digest progress states                                                                  | vpc-flow-digests-progress                       |
| STREAM\_APPLIANCE\_ENDPOINT   |    Yes   | Endpoint for the service which queues digests to be created. See [sqsd](https://bitbucket.org/atlassian/sqsd) for more details. | http://ec2-sqsd.us-west-2.compute.amazonaws.com |
| STREAM\_APPLIANCE\_TOPIC      |    Yes   | Queue name. See [sqsd](https://bitbucket.org/atlassian/sqsd) for more details.                                                  | digest-queue                                    |
| REGION                      |    Yes   | The AWS region for the S3 buckets                                                                                                 | us-west-2                                       |
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