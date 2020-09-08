module github.com/alphagov/paas-sqs-broker

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/alphagov/paas-service-broker-base v0.7.1-0.20200908130742-3fe7ef5e9c07
	github.com/aws/aws-sdk-go v1.27.1
	github.com/awslabs/goformation/v4 v4.15.0
	github.com/cloudfoundry-community/go-cfclient v0.0.0-20190808214049-35bcce23fc5f // indirect
	github.com/olekukonko/tablewriter v0.0.4 // indirect
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/pivotal-cf/brokerapi v6.4.2+incompatible
	github.com/satori/go.uuid v1.2.0
)

go 1.13
