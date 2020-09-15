package broker_test

import (
	"os"
	"testing"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"

	brokerbase "github.com/alphagov/paas-service-broker-base/broker"
	brokertesting "github.com/alphagov/paas-service-broker-base/testing"
	"github.com/alphagov/paas-sqs-broker/sqs"
)

var (
	broker brokertesting.BrokerTester
)

var _ = BeforeSuite(func() {

	// integration tests are slow by nature, globally set the timeout
	SetDefaultEventuallyTimeout(10 * time.Minute)

	file, err := os.Open("../../fixtures/config.json")
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	config, err := brokerbase.NewConfig(file)
	Expect(err).ToNot(HaveOccurred())

	sqsClientConfig, err := sqs.NewConfig(config.Provider)
	Expect(err).ToNot(HaveOccurred())

	// by default the integration tests run without a permission boundary so
	// that there are no dependencies on setting up external IAM policies
	// to run the test with a predefined permission boundary policy set the following environment variable:
	//
	// PERMISSIONS_BOUNDARY_ARN="arn:aws:iam::ACCOUNT-ID:policy/SQSBrokerUserPermissionsBoundary"
	//
	optionalPermissionsBoundary := os.Getenv("PERMISSIONS_BOUNDARY_ARN")
	if optionalPermissionsBoundary != "" {
		sqsClientConfig.PermissionsBoundary = optionalPermissionsBoundary
	}

	logger := lager.NewLogger("sqs-service-broker-test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, config.API.LagerLogLevel))

	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(sqsClientConfig.AWSRegion)}))

	sqsProvider := &sqs.Provider{
		Client: struct {
			*secretsmanager.SecretsManager
			*cloudformation.CloudFormation
		}{
			SecretsManager: secretsmanager.New(sess),
			CloudFormation: cloudformation.New(sess),
		},
		Environment:         sqsClientConfig.DeployEnvironment,
		ResourcePrefix:      sqsClientConfig.ResourcePrefix,
		PermissionsBoundary: sqsClientConfig.PermissionsBoundary,
		Timeout:             sqsClientConfig.Timeout,
		Logger:              logger,
	}

	serviceBroker, err := brokerbase.New(config, sqsProvider, logger)
	Expect(err).ToNot(HaveOccurred())
	brokerAPI := brokerbase.NewAPI(serviceBroker, logger, config)

	broker = brokertesting.New(brokerapi.BrokerCredentials{
		Username: "username",
		Password: "password",
	}, brokerAPI)
})

// DescribeIntegrationTest acts like Describe but only conditionally runs tests
// based on if the ENABLE_INTEGRATION_TESTS enviironment variable is set.  This
// results in tests showing up as "pending" rather than as "passed" (which can
// be misleading when viewing the test output)
func DescribeIntegrationTest(desc string, fn func()) (ok bool) {
	if os.Getenv("ENABLE_INTEGRATION_TESTS") == "true" {
		ok = Describe(desc, fn)
	} else {
		ok = PDescribe(desc, fn)
	}
	return ok
}

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Suite")
}
