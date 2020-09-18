package broker_test

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/alphagov/paas-sqs-broker/testing/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"

	brokertesting "github.com/alphagov/paas-service-broker-base/testing"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pivotal-cf/brokerapi/domain"
	uuid "github.com/satori/go.uuid"
)

const (
	ASYNC_ALLOWED = true
)

var _ = DescribeIntegrationTest("broker integration tests", func() {

	var (
		instanceID      string
		provisionValues brokertesting.RequestBody
		bindingID       string
		binding         struct {
			Credentials struct {
				AWSAccessKeyID     string `json:"aws_access_key_id"`
				AWSSecretAccessKey string `json:"aws_secret_access_key"`
				AWSRegion          string `json:"aws_region"`
				PrimaryQueueURL    string `json:"primary_queue_url"`
				SecondaryQueueURL  string `json:"secondary_queue_url"`
			}
		}
	)

	BeforeEach(func() {
		instanceID = uuid.NewV4().String()
		bindingID = uuid.NewV4().String()

		provisionValues = brokertesting.RequestBody{
			ServiceID:        "uuid-1",
			PlanID:           "uuid-2", // standard (non-FIFO) queue plan
			OrganizationGUID: uuid.NewV4().String(),
			Parameters: &brokertesting.ConfigurationValues{
				"message_retention_period": 60,
			},
		}
	})

	It("should manage the lifecycle of an SQS queue", func() {

		By("provisioning", func() {
			res := broker.Provision(
				instanceID,
				provisionValues,
				ASYNC_ALLOWED,
			)

			Expect(res.Code).To(Equal(http.StatusAccepted))
		})

		By("waiting for provision process to complete", func() {
			provisionState := lastServiceOperationChan(
				broker,
				instanceID,
				provisionValues.ServiceID,
				provisionValues.PlanID,
				sqs.ProvisionOperation,
			)

			Eventually(provisionState).Should(BeSuccessState())
		})

		defer By("waiting for deprovision process to complete", func() {
			deprovisionState := lastServiceOperationChan(
				broker,
				instanceID,
				provisionValues.ServiceID,
				provisionValues.PlanID,
				sqs.DeprovisionOperation,
			)

			Eventually(deprovisionState).Should(BeSuccessState())
		})

		defer By("deprovisioning", func() {
			res := broker.Deprovision(
				instanceID,
				provisionValues.ServiceID,
				provisionValues.PlanID,
				ASYNC_ALLOWED,
			)

			Expect(res.Code).To(Equal(http.StatusAccepted))
		})

		By("binding", func() {
			res := broker.Bind(instanceID, bindingID, brokertesting.RequestBody{
				ServiceID:        provisionValues.ServiceID,
				PlanID:           provisionValues.PlanID,
				OrganizationGUID: "some customer",
			}, ASYNC_ALLOWED)

			Expect(res.Code).To(Equal(http.StatusAccepted))
		})

		By("waiting for bind operation to complete", func() {
			bindingState := lastBindingOperationChan(
				broker,
				instanceID,
				provisionValues.ServiceID,
				provisionValues.PlanID,
				bindingID,
				sqs.BindOperation,
			)

			Eventually(bindingState).Should(BeSuccessState())
		})

		defer By("waiting for unbind operation to complete", func() {
			unbindingState := lastBindingOperationChan(
				broker,
				instanceID,
				provisionValues.ServiceID,
				provisionValues.PlanID,
				bindingID,
				sqs.UnbindOperation,
			)

			Eventually(unbindingState).Should(BeSuccessState())
		})

		defer By("unbinding", func() {
			res := broker.Unbind(
				instanceID,
				provisionValues.ServiceID,
				provisionValues.PlanID,
				bindingID,
				ASYNC_ALLOWED,
			)

			Expect(res.Code).To(Equal(http.StatusAccepted))
		})

		By("fetching the binding credentials", func() {
			res := broker.GetBinding(
				instanceID,
				bindingID,
				provisionValues.ServiceID,
				provisionValues.PlanID,
			)
			Expect(res.Code).To(Equal(http.StatusOK))
			err := json.NewDecoder(res.Result().Body).Decode(&binding)
			Expect(err).ToNot(HaveOccurred())

			Expect(binding.Credentials.AWSAccessKeyID).ToNot(BeEmpty())
			Expect(binding.Credentials.AWSSecretAccessKey).ToNot(BeEmpty())
			Expect(binding.Credentials.AWSRegion).ToNot(BeEmpty())
			Expect(binding.Credentials.PrimaryQueueURL).ToNot(BeEmpty())
			Expect(binding.Credentials.SecondaryQueueURL).ToNot(BeEmpty())
		})

		By("checking we see expected configuration", func() {
			output, err := sqsAdminClient.GetQueueAttributes(&awssqs.GetQueueAttributesInput{
				QueueUrl:       aws.String(binding.Credentials.PrimaryQueueURL),
				AttributeNames: []*string{aws.String(awssqs.QueueAttributeNameAll)},
			})
			Expect(err).ToNot(HaveOccurred())
			// We haven't set redrive_max_receive_count so there shouldn't be a redrive policy
			Expect(output.Attributes).ToNot(HaveKey(awssqs.QueueAttributeNameRedrivePolicy))
			Expect(output.Attributes).To(
				HaveKeyWithValue(awssqs.QueueAttributeNameMessageRetentionPeriod, aws.String("60")))
		})

		By("updating", func() {
			res := broker.Update(instanceID, brokertesting.RequestBody{
				ServiceID: provisionValues.ServiceID,
				PlanID:    provisionValues.PlanID,
				Parameters: &brokertesting.ConfigurationValues{
					"delay_seconds":             30,
					"redrive_max_receive_count": 3,
				},
				PreviousValues: &provisionValues,
			}, ASYNC_ALLOWED)

			Expect(res.Code).To(Equal(http.StatusAccepted))
		})

		By("waiting for update process to complete", func() {
			updateState := lastServiceOperationChan(
				broker,
				instanceID,
				provisionValues.ServiceID,
				provisionValues.PlanID,
				sqs.UpdateOperation,
			)

			Eventually(updateState).Should(BeSuccessState())
		})

		By("using binding credentials to access the service", func() {
			sess := session.Must(session.NewSession(&aws.Config{
				Region: aws.String(binding.Credentials.AWSRegion),
				Credentials: credentials.NewStaticCredentials(
					binding.Credentials.AWSAccessKeyID,
					binding.Credentials.AWSSecretAccessKey,
					"",
				),
			}))
			sqsClient := awssqs.New(sess)

			_, err := sqsClient.SendMessage(&awssqs.SendMessageInput{
				MessageBody: aws.String("Hello World."),
				QueueUrl:    aws.String(binding.Credentials.PrimaryQueueURL),
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = sqsClient.ReceiveMessage(&awssqs.ReceiveMessageInput{
				QueueUrl:            aws.String(binding.Credentials.PrimaryQueueURL),
				MaxNumberOfMessages: aws.Int64(10),
			})
			Expect(err).ToNot(HaveOccurred())
		})

		By("checking we see expected new configuration", func() {
			output, err := sqsAdminClient.GetQueueAttributes(&awssqs.GetQueueAttributesInput{
				QueueUrl:       aws.String(binding.Credentials.PrimaryQueueURL),
				AttributeNames: []*string{aws.String(awssqs.QueueAttributeNameAll)},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Attributes).To(
				HaveKeyWithValue(awssqs.QueueAttributeNameDelaySeconds, aws.String("30")))
			Expect(output.Attributes).To(
				HaveKeyWithValue(awssqs.QueueAttributeNameMessageRetentionPeriod, aws.String("60")))

			Expect(output.Attributes[awssqs.QueueAttributeNameRedrivePolicy]).ToNot(BeNil())

			var redrivePolicy map[string]interface{}
			Expect(
				json.Unmarshal(
					[]byte(*output.Attributes[awssqs.QueueAttributeNameRedrivePolicy]),
					&redrivePolicy,
				),
			).To(Succeed())
			Expect(redrivePolicy).To(And(
				HaveKeyWithValue("deadLetterTargetArn", ContainSubstring("-sec")),
				// JSON unmarshals numbers float64 rather than int, so
				// we use BeNumerically() to be numeric-type-agnostic
				HaveKeyWithValue("maxReceiveCount", BeNumerically("==", 3))))
		})
	})

})

// lastServiceOperationChan returns a channel that repeatedly receives the latest provision
// state.  Each time a value is pulled off the channel, the "last service operation" endpoint gets
// polled and sent to the channel.  This makes writing assertions cleaner.
func lastServiceOperationChan(broker brokertesting.BrokerTester, instanceID string, serviceID string, planID string, opData string) chan domain.LastOperationState {
	ch := make(chan domain.LastOperationState)
	go func() {
		for {
			res := broker.LastOperation(instanceID, serviceID, planID, opData)
			var ret struct {
				State domain.LastOperationState `json:"state"`
			}
			result := res.Result()
			_ = json.NewDecoder(result.Body).Decode(&ret)
			result.Body.Close()
			ch <- ret.State
			time.Sleep(5 * time.Second)
		}
	}()
	return ch
}

// lastBindingOperationChan returns a channel that repeatedly receives the latest binding
// state.  Each time a value is pulled off the channel, the "last binding operation" endpoint gets
// polled and sent to the channel.  This makes writing assertions cleaner.
func lastBindingOperationChan(broker brokertesting.BrokerTester, instanceID string, serviceID string, planID string, bindingID string, opData string) chan domain.LastOperationState {
	ch := make(chan domain.LastOperationState)
	go func() {
		for {
			res := broker.LastBindingOperation(instanceID, bindingID, serviceID, planID, opData)
			var ret struct {
				State domain.LastOperationState `json:"state"`
			}
			result := res.Result()
			_ = json.NewDecoder(result.Body).Decode(&ret)
			result.Body.Close()
			ch <- ret.State
			time.Sleep(5 * time.Second)
		}
	}()
	return ch
}
