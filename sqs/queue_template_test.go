package sqs_test

import (
	"github.com/alphagov/paas-sqs-broker/sqs"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("QueueTemplate", func() {
	var primaryQueue *goformationsqs.Queue
	var secondaryQueue *goformationsqs.Queue
	var params sqs.QueueParams

	BeforeEach(func() {
		params = sqs.QueueParams{}
	})

	JustBeforeEach(func() {
		t, err := sqs.QueueTemplate(params)
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
		var ok bool
		primaryQueue, ok = t.Resources[sqs.ResourcePrimaryQueue].(*goformationsqs.Queue)
		Expect(ok).To(BeTrue())
		secondaryQueue, ok = t.Resources[sqs.ResourceSecondaryQueue].(*goformationsqs.Queue)
		Expect(ok).To(BeTrue())
	})

	It("should not have any input parameters", func() {
		t, err := sqs.QueueTemplate(sqs.QueueParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Parameters).To(BeEmpty())
	})

	Context("when queueName is set", func() {
		BeforeEach(func() {
			params.QueueName = "q-name-a"
		})
		It("should set primary queue name", func() {
			Expect(primaryQueue.QueueName).To(HavePrefix("q-name-a"))
			Expect(primaryQueue.QueueName).To(HaveSuffix("-pri"))
		})
		It("should set secondary queue name", func() {
			Expect(secondaryQueue.QueueName).To(HavePrefix("q-name-a"))
			Expect(secondaryQueue.QueueName).To(HaveSuffix("-sec"))
		})
	})

	Context("when tags are set", func() {
		BeforeEach(func() {
			params.Tags = map[string]string{
				"Service":   "sqs",
				"DeployEnv": "autom8",
			}
		})
		It("should have suitable tags", func() {
			Expect(primaryQueue.Tags).To(ConsistOf(
				goformationtags.Tag{ // auto-injected
					Key:   "QueueType",
					Value: "Primary",
				},
				goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				},
				goformationtags.Tag{
					Key:   "DeployEnv",
					Value: "autom8",
				},
			))
			Expect(secondaryQueue.Tags).To(ConsistOf(
				goformationtags.Tag{ // auto-injected
					Key:   "QueueType",
					Value: "Secondary",
				},
				goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				},
				goformationtags.Tag{
					Key:   "DeployEnv",
					Value: "autom8",
				},
			))
		})
	})

	It("should have sensible default values", func() {
		Expect(primaryQueue.ContentBasedDeduplication).To(BeFalse())
		Expect(primaryQueue.DelaySeconds).To(BeZero())
		Expect(primaryQueue.FifoQueue).To(BeFalse())
		Expect(primaryQueue.MaximumMessageSize).To(BeZero())
		Expect(primaryQueue.MessageRetentionPeriod).To(BeZero())
		Expect(primaryQueue.ReceiveMessageWaitTimeSeconds).To(BeZero())
		Expect(primaryQueue.RedrivePolicy).To(BeEmpty())
		Expect(primaryQueue.VisibilityTimeout).To(BeZero())

		Expect(secondaryQueue.ContentBasedDeduplication).To(BeFalse())
		Expect(secondaryQueue.FifoQueue).To(BeFalse())
		Expect(secondaryQueue.MessageRetentionPeriod).To(BeZero())
		Expect(secondaryQueue.VisibilityTimeout).To(BeZero())
	})

	Context("when contentBasedDeduplication is set", func() {
		BeforeEach(func() {
			params.ContentBasedDeduplication = true
		})
		It("should set queue ContentBasedDeduplication from spec", func() {
			Expect(primaryQueue.ContentBasedDeduplication).To(BeTrue())
			Expect(secondaryQueue.ContentBasedDeduplication).To(BeTrue())
		})
	})

	Context("when delaySeconds is set", func() {
		BeforeEach(func() {
			params.DelaySeconds = 600
		})
		It("should set queue DelaySeconds from spec", func() {
			Expect(primaryQueue.DelaySeconds).To(Equal(600))
		})
	})

	Context("when fifoQueue is set", func() {
		BeforeEach(func() {
			params.FifoQueue = true
		})
		It("should set queue FifoQueue from spec", func() {
			Expect(primaryQueue.FifoQueue).To(BeTrue())
			Expect(secondaryQueue.FifoQueue).To(BeTrue())
		})
	})

	Context("when maximumMessageSize is set", func() {
		BeforeEach(func() {
			params.MaximumMessageSize = 300
		})
		It("should set queue MaximumMessageSize from spec", func() {
			Expect(primaryQueue.MaximumMessageSize).To(Equal(300))
		})
	})

	Context("when messageRetentionPeriod is set", func() {
		BeforeEach(func() {
			params.MessageRetentionPeriod = 20
		})
		It("should set queue MessageRetentionPeriod from spec", func() {
			Expect(primaryQueue.MessageRetentionPeriod).To(Equal(20))
			Expect(secondaryQueue.MessageRetentionPeriod).To(Equal(20))
		})
	})

	Context("when receiveMessageWaitTimeSeconds is set", func() {
		BeforeEach(func() {
			params.ReceiveMessageWaitTimeSeconds = 1200
		})
		It("should set queue ReceiveMessageWaitTimeSeconds from spec", func() {
			Expect(primaryQueue.ReceiveMessageWaitTimeSeconds).To(Equal(1200))
		})
	})

	Context("when RedriveMaxReceiveCount is set", func() {
		BeforeEach(func() {
			params.RedriveMaxReceiveCount = 1
		})
		It("should set queue ContentBasedDeduplication from spec", func() {
			policy, ok := primaryQueue.RedrivePolicy.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(policy["maxReceiveCount"]).To(Equal(1))
		})
	})

	Context("when VisibilityTimeout is set", func() {
		BeforeEach(func() {
			params.VisibilityTimeout = 30
		})
		It("should set queue VisibilityTimeout from spec", func() {
			Expect(primaryQueue.VisibilityTimeout).To(Equal(30))
			Expect(secondaryQueue.VisibilityTimeout).To(Equal(30))
		})
	})

	It("should have outputs for connection details", func() {
		t, err := sqs.QueueTemplate(sqs.QueueParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Outputs).To(And(
			HaveKey(sqs.OutputPrimaryQueueARN),
			HaveKey(sqs.OutputPrimaryQueueURL),
			HaveKey(sqs.OutputSecondaryQueueARN),
			HaveKey(sqs.OutputSecondaryQueueURL),
		))
	})
})
