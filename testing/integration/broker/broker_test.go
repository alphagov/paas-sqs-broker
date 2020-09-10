package broker_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/ssm"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-service-broker-base/broker"
	brokertesting "github.com/alphagov/paas-service-broker-base/testing"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain"
	uuid "github.com/satori/go.uuid"
)

const (
	ASYNC_ALLOWED = true
)

var (
	PollingTimeout = time.Minute * 10
)

var _ = Describe("Broker", func() {
	var (
		brokerTester brokertesting.BrokerTester
	)

	var (
		instanceID string
		binding1ID string
		serviceID  = "uuid-1"
		planID     = "uuid-2"
	)

	BeforeEach(func() {
		if os.Getenv("ENABLE_INTEGRATION_TESTS") != "true" {
			Skip("Skipping integration tests as ENABLE_INTEGRATION_TESTS is not set to 'true'")
		}
		instanceID = uuid.NewV4().String()
		binding1ID = uuid.NewV4().String()
		_, brokerTester = initialise()
	})

	It("should manage the lifecycle of an SQS queue", func() {
		By("Provisioning")
		provisionValues := brokertesting.RequestBody{
			ServiceID:        serviceID,
			PlanID:           planID,
			OrganizationGUID: "some customer",
			Parameters: &brokertesting.ConfigurationValues{
				"message_retention_period": 60,
			},
		}
		res := brokerTester.Provision(instanceID, provisionValues, ASYNC_ALLOWED)
		Expect(res.Code).To(Equal(http.StatusAccepted))

		By("waiting for the provision to succeed")
		Eventually(func() *httptest.ResponseRecorder {
			return brokerTester.LastOperation(instanceID, serviceID, planID, sqs.ProvisionOperation)
		}, PollingTimeout, 5*time.Second).Should(HaveLastOperationState(domain.Succeeded))

		defer DeprovisionService(brokerTester, instanceID, serviceID, planID)

		By("Binding an app")
		res = brokerTester.Bind(instanceID, binding1ID, brokertesting.RequestBody{
			ServiceID:        serviceID,
			PlanID:           planID,
			OrganizationGUID: "some customer",
		}, ASYNC_ALLOWED)
		Expect(res.Code).To(Equal(http.StatusAccepted))

		defer Unbind(brokerTester, instanceID, serviceID, planID, binding1ID)

		By("waiting for the bind to complete")
		Eventually(func() *httptest.ResponseRecorder {
			return brokerTester.LastBindingOperation(instanceID, binding1ID, serviceID, planID, sqs.BindOperation)
		}, PollingTimeout, 5*time.Second).Should(HaveLastOperationState(domain.Succeeded))

		By("getting the binding")
		res = brokerTester.GetBinding(instanceID, binding1ID, serviceID, planID)
		Expect(res.Code).To(Equal(http.StatusOK))

		By("Asserting the credentials returned work for both reading and writing")
		var ret struct {
			Credentials struct {
				AWSAccessKeyID     string `json:"aws_access_key_id"`
				AWSSecretAccessKey string `json:"aws_secret_access_key"`
				AWSRegion          string `json:"aws_region"`
				PrimaryQueueURL    string `json:"primary_queue_url"`
				SecondaryQueueURL  string `json:"secondary_queue_url"`
			}
		}
		err := json.NewDecoder(res.Result().Body).Decode(&ret)
		Expect(err).ToNot(HaveOccurred())

		sess := session.Must(session.NewSession(&aws.Config{
			Region:      aws.String(ret.Credentials.AWSRegion),
			Credentials: credentials.NewStaticCredentials(ret.Credentials.AWSAccessKeyID, ret.Credentials.AWSSecretAccessKey, ""),
		}))
		sqsClient := awssqs.New(sess)

		_, err = sqsClient.SendMessage(&awssqs.SendMessageInput{
			MessageBody: aws.String("Hello World."),
			QueueUrl:    aws.String(ret.Credentials.PrimaryQueueURL),
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = sqsClient.ReceiveMessage(&awssqs.ReceiveMessageInput{
			QueueUrl:            aws.String(ret.Credentials.PrimaryQueueURL),
			MaxNumberOfMessages: aws.Int64(10),
		})
		Expect(err).ToNot(HaveOccurred())

		By("updating the message retention period")
		res = brokerTester.Update(instanceID, brokertesting.RequestBody{
			ServiceID: serviceID,
			PlanID:    planID,
			Parameters: &brokertesting.ConfigurationValues{
				"message_retention_period": 120,
			},
			PreviousValues: &provisionValues,
		}, ASYNC_ALLOWED)
		Expect(res.Code).To(Equal(http.StatusAccepted))

		By("waiting for the update to succeed")
		Eventually(func() *httptest.ResponseRecorder {
			return brokerTester.LastOperation(instanceID, serviceID, planID, sqs.UpdateOperation)
		}, PollingTimeout, 5*time.Second).Should(HaveLastOperationState(domain.Succeeded))

	})

})

func DeprovisionService(brokerTester brokertesting.BrokerTester, instanceID, serviceID, planID string) {
	By("Deprovisioning")
	res := brokerTester.Deprovision(instanceID, serviceID, planID, true)
	Expect(res.Code).To(Equal(http.StatusAccepted))

	By("waiting for the deprovision to succeed")
	Eventually(func() *httptest.ResponseRecorder {
		return brokerTester.LastOperation(instanceID, serviceID, planID, sqs.DeprovisionOperation)
	}, PollingTimeout, 5*time.Second).Should(HaveLastOperationState(domain.Succeeded))
}

func Unbind(brokerTester brokertesting.BrokerTester, instanceID string, serviceID string, planID string, bindingID string) {
	By(fmt.Sprintf("Deferred: Unbinding the %s binding", bindingID))
	res := brokerTester.Unbind(instanceID, serviceID, planID, bindingID, true)
	Expect(res.Code).To(Equal(http.StatusAccepted))

	By("waiting for the unbind to complete")
	Eventually(func() *httptest.ResponseRecorder {
		return brokerTester.LastBindingOperation(instanceID, bindingID, serviceID, planID, sqs.UnbindOperation)
	}, PollingTimeout, 5*time.Second).Should(HaveLastOperationState(domain.Succeeded))
}

func initialise() (*sqs.Config, brokertesting.BrokerTester) {
	file, err := os.Open("../../fixtures/config.json")
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	config, err := broker.NewConfig(file)
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
			*ssm.SSM
			*cloudformation.CloudFormation
		}{
			SSM:            ssm.New(sess),
			CloudFormation: cloudformation.New(sess),
		},
		Environment:         sqsClientConfig.DeployEnvironment,
		ResourcePrefix:      sqsClientConfig.ResourcePrefix,
		PermissionsBoundary: sqsClientConfig.PermissionsBoundary,
		Timeout:             sqsClientConfig.Timeout,
		Logger:              logger,
	}

	serviceBroker, err := broker.New(config, sqsProvider, logger)
	Expect(err).ToNot(HaveOccurred())
	brokerAPI := broker.NewAPI(serviceBroker, logger, config)

	return sqsClientConfig, brokertesting.New(brokerapi.BrokerCredentials{
		Username: "username",
		Password: "password",
	}, brokerAPI)
}
