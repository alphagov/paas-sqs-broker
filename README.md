# PaaS SQS Broker

A broker for AWS SQS queues conforming to the [Open Service Broker API
specification](https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/spec.md).

The implementation creates an SQS queue for every service instance and bindings
are implemented as an IAM user with access keys.

## Running tests

You can use the standard go tooling to execute tests:

```
go test -v ./...
```

To run integration tests against a real AWS environment you must have AWS
credentials in your environment and you must set the ENABLE_INTEGRATION_TESTS
environment variable to `true`.

```
ENABLE_INTEGRATION_TESTS=true go test -v ./integration/...
```

If you have access to the GOV.UK PaaS build CI then you test with a permission boundary set using:

```
fly -t paas-ci execute -c ci/integration.yml --input repo=.
```

(this will upload your current modifications to concourse and execute the integration tests).
