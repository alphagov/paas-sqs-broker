# PaaS SQS Broker

A broker for AWS SQS queues conforming to the [Open Service Broker API specification](https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/spec.md).

The implementation creates an SQS queue for every service instance and bindings are implemented as an IAM user with access keys.
