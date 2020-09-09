package sqs_test

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	goformation "github.com/awslabs/goformation/v4"
	goformationiam "github.com/awslabs/goformation/v4/cloudformation/iam"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/domain"

	"context"

	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/alphagov/paas-sqs-broker/sqs"
	fakeClient "github.com/alphagov/paas-sqs-broker/sqs/fakes"
)

var _ = Describe("Provider", func() {
	var (
		fakeCfnClient *fakeClient.FakeClient
		sqsProvider   *sqs.Provider
	)

	BeforeEach(func() {
		fakeCfnClient = &fakeClient.FakeClient{}
		sqsProvider = &sqs.Provider{
			Client:         fakeCfnClient,
			Environment:    "test",
			ResourcePrefix: "testprefix",
		}
	})

	Context("Provision", func() {
		var (
			provisionData    provideriface.ProvisionData
			createStackInput *cloudformation.CreateStackInput
			queue            *goformationsqs.Queue
		)

		BeforeEach(func() {
			provisionData = provideriface.ProvisionData{
				InstanceID: "a5da1b66-da42-4c83-b806-f287bc589ab3",
				Plan: domain.ServicePlan{
					Name: "standard",
					ID:   "uuid-2",
				},
				Details: domain.ProvisionDetails{
					OrganizationGUID: "27b72d3f-9401-4b45-a7e7-40b17819954f",
				},
			}
			createStackInput = nil
			queue = nil
		})

		JustBeforeEach(func() {
			spec, err := sqsProvider.Provision(context.Background(), provisionData)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.DashboardURL).To(Equal(""))
			Expect(spec.OperationData).To(Equal("provision"))
			Expect(spec.IsAsync).To(BeTrue())

			Expect(fakeCfnClient.CreateStackWithContextCallCount()).To(Equal(1))

			var ctx context.Context
			ctx, createStackInput, _ = fakeCfnClient.CreateStackWithContextArgsForCall(0)
			Expect(ctx).ToNot(BeNil())

			Expect(createStackInput.TemplateBody).ToNot(BeNil())
			t, err := goformation.ParseYAML([]byte(*createStackInput.TemplateBody))
			Expect(err).ToNot(HaveOccurred())

			var ok bool

			Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
			queue, ok = t.Resources[sqs.SQSResourceName].(*goformationsqs.Queue)
			Expect(ok).To(BeTrue())

		})

		It("should have CAPABILITY_NAMED_IAM", func() {
			Expect(createStackInput.Capabilities).To(ConsistOf(
				aws.String("CAPABILITY_NAMED_IAM"),
			))
		})

		It("should use the correct stack prefix", func() {
			Expect(createStackInput.StackName).To(Equal(aws.String(fmt.Sprintf("testprefix-%s", provisionData.InstanceID))))
		})

		It("should have sensible default params", func() {
			Expect(queue.ContentBasedDeduplication).To(BeFalse())
			Expect(queue.DelaySeconds).To(Equal(0))
			Expect(queue.MaximumMessageSize).To(Equal(0))
			Expect(queue.MessageRetentionPeriod).To(Equal(0))
			Expect(queue.ReceiveMessageWaitTimeSeconds).To(Equal(0))
			Expect(queue.RedrivePolicy).To(Equal(""))
			Expect(queue.VisibilityTimeout).To(Equal(0))
		})

		Context("Standard queues", func() {
			It("Should not be a FIFO queue", func() {
				Expect(queue.FifoQueue).To(BeFalse())
			})
		})

		Context("FIFO queues", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					InstanceID: "a5da1b66-da42-4c83-b806-f287bc589ab3",
					Plan: domain.ServicePlan{
						Name: "fifo",
						ID:   "uuid-2",
					},
					Details: domain.ProvisionDetails{
						OrganizationGUID: "27b72d3f-9401-4b45-a7e7-40b17819954f",
					},
				}
			})

			It("Should be a FIFO queue", func() {
				Expect(queue.FifoQueue).To(BeTrue())
			})
		})

		Context("when content_based_deduplication provision param set to false", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					Details: domain.ProvisionDetails{
						RawParameters: json.RawMessage(`{
							"content_based_deduplication": false
						}`),
					},
				}
			})
			It("should not set content-based-deduplication", func() {
				Expect(queue.ContentBasedDeduplication).To(BeFalse())
			})
		})

		Context("when content_based_deduplication provision param set to true", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					Details: domain.ProvisionDetails{
						RawParameters: json.RawMessage(`{
							"content_based_deduplication": true
						}`),
					},
				}
			})

			It("should set content-based-deduplication", func() {
				Expect(queue.ContentBasedDeduplication).To(BeTrue())
			})
		})

		Context("when delay_seconds provision param set", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					Details: domain.ProvisionDetails{
						RawParameters: json.RawMessage(`{
							"delay_seconds": 5
						}`),
					},
				}
			})

			It("should set queue delay in seconds", func() {
				Expect(queue.DelaySeconds).To(Equal(5))
			})
		})

		Context("when maximum_message_size provision param set", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					Details: domain.ProvisionDetails{
						RawParameters: json.RawMessage(`{
							"maximum_message_size": 10
						}`),
					},
				}
			})

			It("should set the queue max message size", func() {
				Expect(queue.MaximumMessageSize).To(Equal(10))
			})
		})

		Context("when message_retention_period param set", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					Details: domain.ProvisionDetails{
						RawParameters: json.RawMessage(`{
							"message_retention_period": 3
						}`),
					},
				}
			})

			It("should set the queue retention period", func() {
				Expect(queue.MessageRetentionPeriod).To(Equal(3))
			})
		})

		Context("when receive_message_wait_time_seconds param set", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					Details: domain.ProvisionDetails{
						RawParameters: json.RawMessage(`{
							"receive_message_wait_time_seconds": 20
						}`),
					},
				}
			})

			It("should set the wait time in seconds", func() {
				Expect(queue.ReceiveMessageWaitTimeSeconds).To(Equal(20))
			})
		})

		Context("when redrive_max_receive_count param set", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					Details: domain.ProvisionDetails{
						RawParameters: json.RawMessage(`{
							"redrive_max_receive_count": 30
						}`),
					},
				}
			})

			It("should set redrive policy", func() {
				Expect(queue.RedrivePolicy).To(And(
					HaveKeyWithValue("maxReceiveCount", float64(30)),
					HaveKey("deadLetterTargetArn"), // Is that good enough?
				))
			})
		})

		Context("when visibility_timeout param set", func() {
			BeforeEach(func() {
				provisionData = provideriface.ProvisionData{
					Details: domain.ProvisionDetails{
						RawParameters: json.RawMessage(`{
							"visibility_timeout": 11
						}`),
					},
				}
			})

			It("should set queue visibility timeout", func() {
				Expect(queue.VisibilityTimeout).To(Equal(11))
			})
		})

		It("Should set appropriate tags", func() {
			Expect(queue.Tags).To(And(
				ContainElement(goformationtags.Tag{
					Key:   "Name",
					Value: provisionData.InstanceID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				}),
				ContainElement(goformationtags.Tag{
					Key:   "ServiceID",
					Value: provisionData.Details.ServiceID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Environment",
					Value: "test",
				}),
			))
		})

		It("Should construct queue name correctly", func() {
			// the queue resource should have had values from provisionData passed through
			queueName := fmt.Sprintf("testprefix-%s", provisionData.InstanceID)
			Expect(queue.QueueName).To(Equal(queueName))
		})

	})

	Describe("Provision", func() {
		It("creates a cloudformation stack", func() {
			provisionData := provideriface.ProvisionData{
				InstanceID: "a5da1b66-da42-4c83-b806-f287bc589ab3",
				Plan: domain.ServicePlan{
					Name: "standard",
					ID:   "uuid-2",
				},
				Details: domain.ProvisionDetails{
					OrganizationGUID: "27b72d3f-9401-4b45-a7e7-40b17819954f",
					RawParameters: json.RawMessage(`{
						"content_based_deduplication": true
					}`),
				},
			}
			spec, err := sqsProvider.Provision(context.Background(), provisionData)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.DashboardURL).To(Equal(""))
			Expect(spec.OperationData).To(Equal("provision"))
			Expect(spec.IsAsync).To(BeTrue())

			Expect(fakeCfnClient.CreateStackWithContextCallCount()).To(Equal(1))

			ctx, input, _ := fakeCfnClient.CreateStackWithContextArgsForCall(0)
			Expect(ctx).ToNot(BeNil())
			Expect(input.Capabilities).To(ConsistOf(
				aws.String("CAPABILITY_NAMED_IAM"),
			))
			Expect(input.StackName).To(Equal(aws.String(fmt.Sprintf("testprefix-%s", provisionData.InstanceID))))

			Expect(input.TemplateBody).ToNot(BeNil())
			t, err := goformation.ParseYAML([]byte(*input.TemplateBody))
			Expect(err).ToNot(HaveOccurred())

			// we should see a queue resource
			Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
			queue, ok := t.Resources[sqs.SQSResourceName].(*goformationsqs.Queue)
			Expect(ok).To(BeTrue())

			// fifo should be set to false because we asked for a standard queue
			Expect(queue.FifoQueue).To(BeFalse())

			// should have suitable tags set
			Expect(queue.Tags).To(And(
				ContainElement(goformationtags.Tag{
					Key:   "Name",
					Value: provisionData.InstanceID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				}),
				ContainElement(goformationtags.Tag{
					Key:   "ServiceID",
					Value: provisionData.Details.ServiceID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Environment",
					Value: "test",
				}),
			))

			// the queue resource should have had values from provisionData passed through
			queueName := fmt.Sprintf("testprefix-%s", provisionData.InstanceID)
			Expect(queue.QueueName).To(Equal(queueName))

			// we set this in provdata
			Expect(queue.ContentBasedDeduplication).To(BeTrue())
		})
	})

	DescribeTable("last operation fetches stack status",
		func(cloudformationStatus string, expectedServiceStatus domain.LastOperationState) {
			fakeCfnClient.DescribeStacksWithContextReturnsOnCall(0, &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackName:   aws.String("some stack"),
						StackStatus: aws.String(cloudformationStatus),
					},
				},
			}, nil)
			lastOperationData := provideriface.LastOperationData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			lastOp, err := sqsProvider.LastOperation(context.Background(), lastOperationData)
			Expect(err).NotTo(HaveOccurred())

			Expect(lastOp.State).To(Equal(expectedServiceStatus))
		},

		Entry("delete failed",
			cloudformation.StackStatusDeleteFailed,
			domain.Failed,
		),
		Entry("create failed",
			cloudformation.StackStatusCreateFailed,
			domain.Failed,
		),
		Entry("rollback failed",
			cloudformation.StackStatusRollbackFailed,
			domain.Failed,
		),
		Entry("update rollback failed",
			cloudformation.StackStatusUpdateRollbackFailed,
			domain.Failed,
		),
		Entry("rollback complete",
			cloudformation.StackStatusRollbackComplete,
			domain.Failed,
		),
		Entry("update rollback complete",
			cloudformation.StackStatusUpdateRollbackComplete,
			domain.Failed,
		),
		Entry("create complete",
			cloudformation.StackStatusCreateComplete,
			domain.Succeeded,
		),
		Entry("update complete",
			cloudformation.StackStatusUpdateComplete,
			domain.Succeeded,
		),
		Entry("update complete",
			cloudformation.StackStatusDeleteComplete,
			domain.Succeeded,
		),
		Entry("create in progress",
			cloudformation.StackStatusCreateInProgress,
			domain.InProgress,
		),
		Entry("update in progress",
			cloudformation.StackStatusUpdateInProgress,
			domain.InProgress,
		),
		Entry("delete in progress",
			cloudformation.StackStatusDeleteInProgress,
			domain.InProgress,
		),
		Entry("rollback in progress",
			cloudformation.StackStatusRollbackInProgress,
			domain.InProgress,
		),
	)

	DescribeTable("last binding operation fetches stack status",
		func(cloudformationStatus string, expectedServiceStatus domain.LastOperationState) {
			fakeCfnClient.DescribeStacksWithContextReturnsOnCall(0, &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackName:   aws.String("some stack"),
						StackStatus: aws.String(cloudformationStatus),
					},
				},
			}, nil)
			lastOperationData := provideriface.LastBindingOperationData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				BindingID:  "c6ea1339-7ade-4952-9247-e419b59e7b67",
			}
			lastOp, err := sqsProvider.LastBindingOperation(context.Background(), lastOperationData)
			Expect(err).NotTo(HaveOccurred())

			Expect(lastOp.State).To(Equal(expectedServiceStatus))
		},

		Entry("delete failed",
			cloudformation.StackStatusDeleteFailed,
			domain.Failed,
		),
		Entry("create failed",
			cloudformation.StackStatusCreateFailed,
			domain.Failed,
		),
		Entry("rollback failed",
			cloudformation.StackStatusRollbackFailed,
			domain.Failed,
		),
		Entry("update rollback failed",
			cloudformation.StackStatusUpdateRollbackFailed,
			domain.Failed,
		),
		Entry("rollback complete",
			cloudformation.StackStatusRollbackComplete,
			domain.Failed,
		),
		Entry("update rollback complete",
			cloudformation.StackStatusUpdateRollbackComplete,
			domain.Failed,
		),
		Entry("create complete",
			cloudformation.StackStatusCreateComplete,
			domain.Succeeded,
		),
		Entry("update complete",
			cloudformation.StackStatusUpdateComplete,
			domain.Succeeded,
		),
		Entry("update complete",
			cloudformation.StackStatusDeleteComplete,
			domain.Succeeded,
		),
		Entry("create in progress",
			cloudformation.StackStatusCreateInProgress,
			domain.InProgress,
		),
		Entry("update in progress",
			cloudformation.StackStatusUpdateInProgress,
			domain.InProgress,
		),
		Entry("delete in progress",
			cloudformation.StackStatusDeleteInProgress,
			domain.InProgress,
		),
		Entry("rollback in progress",
			cloudformation.StackStatusRollbackInProgress,
			domain.InProgress,
		),
	)

	Describe("GetBinding", func() {
		It("error when presented with a non-existent binding", func() {
			fakeCfnClient.DescribeStacksWithContextReturnsOnCall(
				0,
				&cloudformation.DescribeStacksOutput{
					Stacks: []*cloudformation.Stack{},
				},
				&fakeClient.MockAWSError{
					C: "ValidationError",
					M: "Stack with id testprefix-09E1993E-62E2-4040-ADF2-4D3EC741EFE6 does not exist",
				},
			)

			_, err := sqsProvider.GetBinding(context.Background(), provideriface.GetBindData{
				BindingID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			})
			Expect(err).To(MatchError(sqs.ErrStackNotFound))
		})

		It("returns some credentials", func() {
			instanceID := "09E1993E-62E2-4040-ADF2-4D3EC741EFE6"
			bindingID := "c6ea1339-7ade-4952-9247-e419b59e7b67"
			bindingStackOutput := &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
						Outputs: []*cloudformation.Output{
							{
								OutputKey:   aws.String(sqs.SQSOutputIAMAccessKeyID),
								OutputValue: aws.String("michael"),
							},
							{
								OutputKey:   aws.String(sqs.SQSOutputIAMSecretAccessKey),
								OutputValue: aws.String("phyllis"),
							},
						},
					},
				},
			}
			queueStackOutput := &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
						Outputs: []*cloudformation.Output{
							{
								OutputKey:   aws.String(sqs.SQSQueueARNOutputName),
								OutputValue: aws.String("orange"),
							},
							{
								OutputKey:   aws.String(sqs.SQSDLQueueARNOutputName),
								OutputValue: aws.String("apple"),
							},
							{
								OutputKey:   aws.String(sqs.SQSQueueURLOutputName),
								OutputValue: aws.String("https://flambouyant/"),
							},
							{
								OutputKey:   aws.String(sqs.SQSDLQueueURLOutputName),
								OutputValue: aws.String("https://pilchard/"),
							},
							{
								OutputKey:   aws.String(sqs.SQSRegionOutputName),
								OutputValue: aws.String("antartica"),
							},
						},
					},
				},
			}
			fakeCfnClient.DescribeStacksWithContextStub = func(ctx context.Context, input *cloudformation.DescribeStacksInput, opts ...request.Option) (*cloudformation.DescribeStacksOutput, error) {
				if *input.StackName == fmt.Sprintf("testprefix-%s", bindingID) {
					return bindingStackOutput, nil
				} else if *input.StackName == fmt.Sprintf("testprefix-%s", instanceID) {
					return queueStackOutput, nil
				} else {
					panic("describe stacks stub: expected stack name to match binding or instance")
				}
			}

			spec, err := sqsProvider.GetBinding(context.Background(), provideriface.GetBindData{
				InstanceID: instanceID,
				BindingID:  bindingID,
			})
			Expect(fakeCfnClient.DescribeStacksWithContextCallCount()).To(Equal(2))
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.Credentials).ToNot(BeNil())
			creds, err := json.Marshal(spec.Credentials)
			Expect(err).ToNot(HaveOccurred())
			Expect(creds).To(MatchJSON(`{
				"AccessKeyID": "michael",
				"SecretAccessKey": "phyllis",
				"Region": "antartica",
				"QueueURL": "https://flambouyant/",
				"DLQueueURL": "https://pilchard/"
			}`))
		})
	})

	Describe("Deprovision", func() {
		It("succeeds when deleting a non-existent stack", func() {
			fakeCfnClient.DescribeStacksWithContextReturnsOnCall(
				0,
				&cloudformation.DescribeStacksOutput{
					Stacks: []*cloudformation.Stack{},
				},
				&fakeClient.MockAWSError{
					C: "ValidationError",
					M: "Stack with id testprefix-09E1993E-62E2-4040-ADF2-4D3EC741EFE6 does not exist",
				},
			)

			deprovisionData := provideriface.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			_, err := sqsProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).NotTo(HaveOccurred())
		})
		It("deletes a cloudformation stack", func() {
			fakeCfnClient.DescribeStacksWithContextReturnsOnCall(0, &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackName:   aws.String("some stack"),
						StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
					},
				},
			}, nil)

			deprovisionData := provideriface.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			spec, err := sqsProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.OperationData).To(Equal(sqs.DeprovisionOperation))
			Expect(spec.IsAsync).To(BeTrue())

			Expect(fakeCfnClient.DeleteStackWithContextCallCount()).To(Equal(1))
			ctx, input, _ := fakeCfnClient.DeleteStackWithContextArgsForCall(0)
			Expect(ctx).ToNot(BeNil())
			Expect(input.StackName).To(Equal(aws.String(fmt.Sprintf("testprefix-%s", deprovisionData.InstanceID))))
		})
	})

	Describe("Unbind", func() {
		It("succeeds when deleting a non-existent stack", func() {
			fakeCfnClient.DescribeStacksWithContextReturnsOnCall(
				0,
				&cloudformation.DescribeStacksOutput{
					Stacks: []*cloudformation.Stack{},
				},
				&fakeClient.MockAWSError{
					C: "ValidationError",
					M: "Stack with id testprefix-09E1993E-62E2-4040-ADF2-4D3EC741EFE6 does not exist",
				},
			)

			unbindData := provideriface.UnbindData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				BindingID:  "c6ea1339-7ade-4952-9247-e419b59e7b67",
			}
			_, err := sqsProvider.Unbind(context.Background(), unbindData)
			Expect(err).NotTo(HaveOccurred())
		})
		It("deletes a cloudformation stack", func() {
			fakeCfnClient.DescribeStacksWithContextReturnsOnCall(0, &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackName:   aws.String("some stack"),
						StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
					},
				},
			}, nil)

			unbindData := provideriface.UnbindData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				BindingID:  "c6ea1339-7ade-4952-9247-e419b59e7b67",
			}
			spec, err := sqsProvider.Unbind(context.Background(), unbindData)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.OperationData).To(Equal(sqs.UnbindOperation))
			Expect(spec.IsAsync).To(BeTrue())

			Expect(fakeCfnClient.DeleteStackWithContextCallCount()).To(Equal(1))
			ctx, input, _ := fakeCfnClient.DeleteStackWithContextArgsForCall(0)
			Expect(ctx).ToNot(BeNil())
			Expect(input.StackName).To(Equal(aws.String(fmt.Sprintf("testprefix-%s", unbindData.BindingID))))
		})
	})

	Context("Bind", func() {
		var (
			bindData         provideriface.BindData
			createStackInput *cloudformation.CreateStackInput
			user             *goformationiam.User
			policy           *goformationiam.Policy
			arn1             = "arn-1"
			arn2             = "arn-2"
		)

		BeforeEach(func() {
			bindData = provideriface.BindData{
				InstanceID: "a5da1b66-da42-4c83-b806-f287bc589ab3",
				BindingID:  "f56687df-e3d0-452a-9755-1a6d6d9e248f",
			}
			fakeCfnClient.DescribeStacksWithContextReturnsOnCall(0, &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackName:   aws.String("some stack"),
						StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
						Outputs: []*cloudformation.Output{
							{
								OutputKey:   aws.String(sqs.SQSQueueARNOutputName),
								OutputValue: aws.String(arn1),
							},
							{
								OutputKey:   aws.String(sqs.SQSDLQueueARNOutputName),
								OutputValue: aws.String(arn2),
							},
						},
					},
				},
			}, nil)
			createStackInput = nil
			user = nil
		})

		JustBeforeEach(func() {
			spec, err := sqsProvider.Bind(context.Background(), bindData)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.OperationData).To(Equal(sqs.BindOperation))
			Expect(spec.IsAsync).To(BeTrue())

			Expect(fakeCfnClient.CreateStackWithContextCallCount()).To(Equal(1))

			var ctx context.Context
			ctx, createStackInput, _ = fakeCfnClient.CreateStackWithContextArgsForCall(0)
			Expect(ctx).ToNot(BeNil())

			Expect(createStackInput.TemplateBody).ToNot(BeNil())
			t, err := goformation.ParseYAML([]byte(*createStackInput.TemplateBody))
			Expect(err).ToNot(HaveOccurred())

			Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.User{})))
			var ok bool
			user, ok = t.Resources[sqs.SQSResourceIAMUserResourceName].(*goformationiam.User)
			Expect(ok).To(BeTrue())
			Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.Policy{})))
			policy, ok = t.Resources[sqs.SQSResourceIAMPolicyResourceName].(*goformationiam.Policy)
			Expect(ok).To(BeTrue())
		})

		It("should have CAPABILITY_NAMED_IAM", func() {
			Expect(createStackInput.Capabilities).To(ConsistOf(
				aws.String("CAPABILITY_NAMED_IAM"),
			))
		})

		It("should use the correct stack prefix", func() {
			Expect(createStackInput.StackName).To(Equal(aws.String(fmt.Sprintf("testprefix-%s", bindData.BindingID))))
		})

		It("Should set appropriate tags", func() {
			Expect(user.Tags).To(And(
				ContainElement(goformationtags.Tag{
					Key:   "Name",
					Value: bindData.BindingID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Environment",
					Value: "test",
				}),
			))
		})

		It("should use create user name with binding id", func() {
			Expect(user.UserName).To(HavePrefix("testprefix-"))
			Expect(user.UserName).To(HaveSuffix(bindData.BindingID))
		})

		It("should use create user with a path based on ResourcePrefix", func() {
			Expect(user.Path).To(Equal("/testprefix/"))
		})

		It("should not set permission boundary by default", func() {
			Expect(user.PermissionsBoundary).To(BeEmpty())
		})

		It("should extract arns from queue stack outputs", func() {
			Expect(policy.PolicyDocument).To(
				HaveKeyWithValue("Statement", ContainElement(
					HaveKeyWithValue("Resource", ConsistOf(arn1, arn2)),
				)),
			)
		})

		Context("when permission boundary is provided", func() {
			BeforeEach(func() {
				sqsProvider.PermissionsBoundary = "arn:fake:permission:boundary"
			})
			It("should create user with a permission boundary if provided", func() {
				Expect(user.PermissionsBoundary).To(Equal("arn:fake:permission:boundary"))
			})
		})
	})

	Describe("Update", func() {
		It("updates a cloudformation stack", func() {
			updateData := provideriface.UpdateData{
				InstanceID: "a5da1b66-da42-4c83-b806-f287bc589ab3",
				Plan: domain.ServicePlan{
					Name: "fifo",
					ID:   "uuid-2",
				},
				Details: domain.UpdateDetails{
					ServiceID: "27b72d3f-9401-4b45-a7e7-40b17819954f",
					RawParameters: json.RawMessage(`{
						"content_based_deduplication": true
					}`),
				},
			}
			spec, err := sqsProvider.Update(context.Background(), updateData)
			Expect(err).NotTo(HaveOccurred())
			Expect(spec.DashboardURL).To(Equal(""))
			Expect(spec.OperationData).To(Equal(sqs.UpdateOperation))
			Expect(spec.IsAsync).To(BeTrue())

			Expect(fakeCfnClient.UpdateStackWithContextCallCount()).To(Equal(1))

			ctx, input, _ := fakeCfnClient.UpdateStackWithContextArgsForCall(0)
			Expect(ctx).ToNot(BeNil())
			Expect(input.Capabilities).To(ConsistOf(
				aws.String("CAPABILITY_NAMED_IAM"),
			))
			Expect(input.StackName).To(Equal(aws.String(fmt.Sprintf("testprefix-%s", updateData.InstanceID))))

			Expect(input.TemplateBody).ToNot(BeNil())
			t, err := goformation.ParseYAML([]byte(*input.TemplateBody))
			Expect(err).ToNot(HaveOccurred())

			// we should see a queue resource
			Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
			queue, ok := t.Resources[sqs.SQSResourceName].(*goformationsqs.Queue)
			Expect(ok).To(BeTrue())

			// fifo should be set to true because we asked for a fifo queue
			Expect(queue.FifoQueue).To(BeTrue())

			// should have suitable tags set
			Expect(queue.Tags).To(And(
				ContainElement(goformationtags.Tag{
					Key:   "Name",
					Value: updateData.InstanceID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				}),
				ContainElement(goformationtags.Tag{
					Key:   "ServiceID",
					Value: updateData.Details.ServiceID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Environment",
					Value: "test",
				}),
			))

			// the queue resource should have had values from provisionData passed through
			queueName := fmt.Sprintf("testprefix-%s", updateData.InstanceID)
			Expect(queue.QueueName).To(Equal(queueName))

			// we set this in provdata
			Expect(queue.ContentBasedDeduplication).To(BeTrue())
		})
	})

})
