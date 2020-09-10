module github.com/alphagov/paas-sqs-broker

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/alphagov/paas-service-broker-base v0.8.0
	github.com/aws/aws-sdk-go v1.34.20
	github.com/awslabs/goformation/v4 v4.15.0
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.3
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/pivotal-cf/brokerapi v6.4.2+incompatible
	github.com/satori/go.uuid v1.2.0
	golang.org/x/net v0.0.0-20200904194848-62affa334b73 // indirect
	golang.org/x/sys v0.0.0-20200909081042-eff7692f9009 // indirect
	google.golang.org/genproto v0.0.0-20200904004341-0bd0a958aa1d // indirect
)

go 1.14
