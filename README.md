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
      "Action": [
        "sqs:SendMessage",
        "sqs:ReceiveMessage",
        "sqs:GetQueueAttributes",
        "sqs:DeleteMessage"
      ],
      "Resource": "arn:aws:sqs:*:*:paas-sqs-broker-*"
    }
  ]
}
```
And this must match the name used in the iam:PermissionsBoundary condition above (SQSBrokerUserPermissionsBoundary in this example).

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
| `permissions_boundary`                | empty string  | string | any                                                                        |
| `deploy_env`                     | empty string  | string |                                                                            |

## Running tests

You can use the standard go tooling to execute tests:

```
go test -v ./...
```

To run integration tests against a real AWS environment you must have AWS
credentials in your environment and you must set the ENABLE_INTEGRATION_TESTS
environment variable to `true`.

```
ENABLE_INTEGRATION_TESTS=true go test -v ./...
```

If you have access to the GOV.UK PaaS build CI then you test with a permission boundary set using:

```
fly -t paas-ci execute -c ci/integration.yml --input repo=.
```

(this will upload your current modifications to concourse and execute the integration tests).
