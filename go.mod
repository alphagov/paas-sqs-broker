module github.com/alphagov/paas-sqs-broker

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/alphagov/gsp/components/service-operator v0.0.0-20200824170823-0419eed5d281
	github.com/alphagov/paas-service-broker-base v0.6.0
	github.com/aws/aws-sdk-go v1.33.11
	github.com/awslabs/goformation/v4 v4.15.0
	github.com/cloudfoundry-community/go-cfclient v0.0.0-20190808214049-35bcce23fc5f
	github.com/go-logr/logr v0.2.0
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.13.0
	github.com/onsi/gomega v1.10.1
	github.com/pivotal-cf/brokerapi v6.4.2+incompatible
	github.com/sanathkr/yaml v0.0.0-20170819201035-0056894fa522
	istio.io/istio v0.0.0-20200826011009-d717f95a86c5
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	sigs.k8s.io/controller-runtime v0.6.2
)

go 1.13
