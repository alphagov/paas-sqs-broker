# PaaS SQS Broker

A broker for AWS SQS queues conforming to the [Open Service Broker API
specification](https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/spec.md).

The implementation uses CloudFormation to create an SQS queue for every service instance and
bindings are implemented (again through CloudFormation) as an IAM user with access keys.
Permissions boundaries are used to ensure that the broker can only create users with access to SQS
and not other things.
AWS Secrets Manager is used to store binding credentials, and access to it can be restricted to
just the SQS broker's own prefix.

## Requirements

The IAM role for the broker must include at least the following policy (substituting ${account_id} for your account ID):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "cloudformation:*"
      ],
      "Effect": "Allow",
      "Resource": "arn:aws:cloudformation:*:*:stack/paas-sqs-broker-*"
    },
    {
      "Action": [
        "sqs:*"
      ],
      "Effect": "Allow",
      "Resource": "arn:aws:sqs:*:*:paas-sqs-broker-*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:PutUserPolicy",
        "iam:AttachUserPolicy",
        "iam:CreateUser"
      ],
      "Resource": "arn:aws:iam::${account_id}:user/*",
      "Condition": {
        "StringEquals": {
          "iam:PermissionsBoundary": "arn:aws:iam::${account_id}:policy/SQSBrokerUserPermissionsBoundary"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:CreatePolicy",
        "iam:DeletePolicy"
      ],
      "Resource": "arn:aws:iam::*:policy/paas-sqs-broker-*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:*AccessKey*",
        "iam:DeleteUser",
        "iam:DeleteUserPolicy",
        "iam:GetUser",
        "iam:GetUserPolicy",
        "iam:ListAttachedUserPolicies",
        "iam:TagUser",
        "iam:UntagUser",
        "iam:UpdateUser"
      ],
      "Resource": "arn:aws:iam::${account_id}:user/paas-sqs-broker/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:GetUser"
      ],
      "Resource": "arn:aws:iam::${account_id}:user/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:*"
      ],
      "Resource": "arn:aws:secretsmanager:*:${account_id}:secret:paas-sqs-broker-*"
    }
  ]
}
```

A policy must exist with at least these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sqs:*",
      "Resource": "arn:aws:sqs:*:*:paas-sqs-broker-*"
    }
  ]
}
```

And this must match the name used in the iam:PermissionsBoundary condition above (SQSBrokerUserPermissionsBoundary in this example).

Additionally, you may provide an additional IAM Policy that will be
attached to all IAM Users managed by this broker.  For example, you
could use the following policy to restrict access to a particular set
of egress IPs:


```json
{
   "Version": "2012-10-17",
   "Id": "Policy1415115909153",
   "Statement": [
     {
       "Sid": "Access-to-specific-VPC-only",
       "Principal": "*",
       "Action": "*",
       "Effect": "Deny",
       "Resource": "*",
       "Condition": {
         "NotIpAddress": {
           "aws:SourceIp": ["192.0.2.1", "192.0.2.7"]
         }
       }
     }
   ]
}
```

### Configuration options

The following options can be added to the configuration file:

| Field                            | Default value | Type   | Values                                                                     |
| -------------------------------- | ------------- | ------ | -------------------------------------------------------------------------- |
| `basic_auth_username`            | empty string  | string | any non-empty string                                                       |
| `basic_auth_password`            | empty string  | string | any non-empty string                                                       |
| `port`                           | 3000          | string | any free port                                                              |
| `log_level`                      | debug         | string | debug,info,error,fatal                                                     |
| `aws_region`                     | empty string  | string | any [AWS region](https://docs.aws.amazon.com/general/latest/gr/rande.html) |
| `resource_prefix`                | empty string  | string | any                                                                        |
| `additional_user_policy`         | empty string  | string | an ARN of an IAM Policy                                                    |
| `permissions_boundary`           | empty string  | string | an ARN of an IAM Policy                                                    |
| `deploy_env`                     | empty string  | string |                                                                            |

## Running tests

You can use the standard go tooling to execute tests:

```
go test -v ./...
```

To run integration tests against a real AWS environment you must have AWS
credentials in your environment and you must set the ENABLE_INTEGRATION_TESTS
environment variable to `true`.

It may also be benefical to use the ginko test runner to enable parallel tests
when working with the integration tests:

```
ENABLE_INTEGRATION_TESTS=true go run github.com/onsi/ginkgo/ginkgo -v -mod=vendor -nodes=2 -stream ./...
```

If you have access to the GOV.UK PaaS build CI then you test with a permission boundary set using:

```
fly -t paas-ci execute -c ci/integration.yml --input repo=.
```

(this will upload your current modifications to concourse and execute the integration tests).

## Patching an existing bosh environment

If you want to patch an existing bosh environment you can run the following command:

```
make bosh_scp
```

This requires an existing bosh session to be established beforehand.