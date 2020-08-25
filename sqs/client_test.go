package sqs_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alphagov/paas-sqs-broker/sqs/policy"
	"github.com/pivotal-cf/brokerapi"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-sqs-broker/sqs"
	fakeClient "github.com/alphagov/paas-sqs-broker/sqs/fakes"
	"github.com/alphagov/paas-service-broker-base/provider"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	awsSQS "github.com/aws/aws-sdk-go/service/sqs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		sqsAPI          *fakeClient.FakeSQSAPI
		iamAPI          *fakeClient.FakeIAMAPI
		sqsClient       *sqs.SQSClient
		sqsClientConfig *sqs.Config
		logger          lager.Logger
	)

	BeforeEach(func() {
		sqsAPI = &fakeClient.FakeSQSAPI{}
		iamAPI = &fakeClient.FakeIAMAPI{}
		logger = lager.NewLogger("sqs-service-broker-test")
		sqsClientConfig = &sqs.Config{
			AWSRegion:              "eu-west-2",
			ResourcePrefix:         "test-queue-prefix-",
			IAMUserPath:            "/test-iam-path/",
			DeployEnvironment:      "test-env",
			Timeout:                2 * time.Second,
			IpRestrictionPolicyARN: "test-ip-restriction-policy-arn",
		}
		sqsClient = sqs.NewSQSClient(
			sqsClientConfig,
			sqsAPI,
			iamAPI,
			logger,
			context.Background(),
		)
	})
	Describe("CreateQueue", func() {
	})
	Describe("AddUserToBucket", func() {
		BeforeEach(func() {
			// Set up fake API
			iamAPI.CreateUserReturnsOnCall(0, &iam.CreateUserOutput{
				User: &iam.User{
					Arn: aws.String("arn"),
				},
			}, nil)
			iamAPI.CreateAccessKeyReturnsOnCall(0, &iam.CreateAccessKeyOutput{
				AccessKey: &iam.AccessKey{
					AccessKeyId:     aws.String("access-key-id"),
					SecretAccessKey: aws.String("secret-access-key"),
				},
			}, nil)

		})
		It("manages the user and bucket policy", func() {
			bindData := provider.BindData{
				InstanceID: "test-instance-id",
				BindingID:  "test-binding-id",
			}
			bucketCredentials, err := sqsClient.AddUserToBucket(bindData)
			Expect(err).NotTo(HaveOccurred())

			By("creating a user")
			Expect(iamAPI.CreateUserCallCount()).To(Equal(1))

			By("creating access keys for the user")
			Expect(iamAPI.CreateAccessKeyCallCount()).To(Equal(1))

			By("returning the bucket credentials")
			Expect(bucketCredentials).To(Equal(sqs.QueueCredentials{
				QueueName:          sqsClientConfig.ResourcePrefix + bindData.InstanceID,
				AWSAccessKeyID:     "access-key-id",
				AWSSecretAccessKey: "secret-access-key",
				AWSRegion:          sqsClientConfig.AWSRegion,
			}))
		})

		It("returns an error if the permissions requested aren't known", func() {
			bindData := provider.BindData{
				InstanceID: "test-instance-id",
				BindingID:  "test-binding-id",
				Details: brokerapi.BindDetails{
					RawParameters: json.RawMessage(`{"permissions": "read-write-banana"}`),
				},
			}

			_, err := sqsClient.AddUserToBucket(bindData)
			Expect(err).To(HaveOccurred())
		})

		Context("when not allowing external access", func() {
			Context("by omitting the parameter", func() {
				It("attaches the IP-Restriction policy", func() {
					bindData := provider.BindData{
						BindingID: "test-instance-id",
						Details: brokerapi.BindDetails{
							RawParameters: nil,
						},
					}
					sqsClient.AddUserToBucket(bindData)
					createUserInput := iamAPI.CreateUserArgsForCall(0)
					Expect(iamAPI.AttachUserPolicyCallCount()).To(Equal(1))
					attachPolicyArgs := iamAPI.AttachUserPolicyArgsForCall(0)

					Expect(*attachPolicyArgs.PolicyArn).To(Equal(sqsClientConfig.IpRestrictionPolicyARN))
					Expect(*attachPolicyArgs.UserName).To(Equal(*createUserInput.UserName))
				})
			})
			Context("by setting the parameter to false", func() {
				It("attaches the IP-Restriction policy", func() {
					bindData := provider.BindData{
						BindingID: "test-instance-id",
						Details: brokerapi.BindDetails{
							RawParameters: json.RawMessage(`{"allow_external_access": false}`),
						},
					}
					sqsClient.AddUserToBucket(bindData)
					createUserInput := iamAPI.CreateUserArgsForCall(0)
					Expect(iamAPI.AttachUserPolicyCallCount()).To(Equal(1))
					attachPolicyArgs := iamAPI.AttachUserPolicyArgsForCall(0)

					Expect(*attachPolicyArgs.PolicyArn).To(Equal(sqsClientConfig.IpRestrictionPolicyARN))
					Expect(*attachPolicyArgs.UserName).To(Equal(*createUserInput.UserName))
				})
			})
		})

		Context("when allowing external access by setting the parameter to true", func() {
			It("does not attach the IP-Restriction policy", func() {
				bindData := provider.BindData{
					BindingID: "test-instance-id",
					Details: brokerapi.BindDetails{
						RawParameters: json.RawMessage(`{"allow_external_access": true}`),
					},
				}
				sqsClient.AddUserToBucket(bindData)
				Expect(iamAPI.AttachUserPolicyCallCount()).To(Equal(0))
			})
		})

		Context("when creating an access key fails", func() {
			It("deletes the user", func() {
				// Set up fake API
				iamAPI.CreateUserReturnsOnCall(0, &iam.CreateUserOutput{
					User: &iam.User{},
				}, nil)
				iamAPI.CreateAccessKeyReturnsOnCall(0, &iam.CreateAccessKeyOutput{}, errors.New("some-error"))
				bindData := provider.BindData{
					InstanceID: "test-instance-id",
					BindingID:  "test-binding-id",
				}
				_, err := sqsClient.AddUserToBucket(bindData)
				Expect(err).To(HaveOccurred())
				Expect(iamAPI.DeleteUserCallCount()).To(Equal(1))
			})
		})
	})

	Describe("RemoveUserFromQueueAndDeleteUser", func() {
		It("manages the user and bucket policy", func() {
			// Set up fake API
			userArn := "arn:aws:iam::account-number:user/sqs-broker/" + sqsClientConfig.ResourcePrefix + "some-user"
			iamAPI.ListAccessKeysReturnsOnCall(0, &iam.ListAccessKeysOutput{
				AccessKeyMetadata: []*iam.AccessKeyMetadata{{AccessKeyId: aws.String("key")}},
			}, nil)
			iamAPI.DeleteAccessKeyReturnsOnCall(0, nil, nil)

			err := sqsClient.RemoveUserFromBucketAndDeleteUser("some-user", "bucketName")
			Expect(err).NotTo(HaveOccurred())

			By("deleting the user and keys")
			Expect(iamAPI.DeleteUserCallCount()).To(Equal(1))
			Expect(iamAPI.DeleteAccessKeyCallCount()).To(Equal(1))
		})

		Context("when deleting the user fails", func() {
			It("returns an error", func() {
				// Set up fake API
				userArn := "arn:aws:iam::account-number:user/sqs-broker/" + sqsClientConfig.ResourcePrefix + "some-user"
				errDeletingUser := errors.New("error-deleting-user")
				iamAPI.DeleteUserReturnsOnCall(0, &iam.DeleteUserOutput{}, errDeletingUser)

				err := sqsClient.RemoveUserFromBucketAndDeleteUser("some-user", "bucketName")
				Expect(err).To(MatchError(errDeletingUser))
			})
		})
	})
})

func hasTag(tags []*awsSQS.Tag, key string, value string) bool {
	for _, tag := range tags {
		if *tag.Key == key && *tag.Value == value {
			return true
		}
	}
	return false
}
