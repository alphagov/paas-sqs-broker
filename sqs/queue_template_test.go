package sqs_test

import (
	"github.com/alphagov/paas-sqs-broker/sqs"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("QueueTemplate", func() {
	var queue *goformationsqs.Queue
	var dlqueue *goformationsqs.Queue
	var params sqs.QueueParams

	BeforeEach(func() {
		params = sqs.QueueParams{}
	})

	JustBeforeEach(func() {
		t, err := sqs.QueueTemplate(params)
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
		var ok bool
		queue, ok = t.Resources[sqs.SQSResourceName].(*goformationsqs.Queue)
		Expect(ok).To(BeTrue())
		dlqueue, ok = t.Resources[sqs.SQSDLQResourceName].(*goformationsqs.Queue)
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
		It("should set queue names", func() {
			Expect(queue.QueueName).To(Equal("q-name-a"))
			Expect(dlqueue.QueueName).To(Equal("q-name-a-dl"))
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
			Expect(queue.Tags).To(ConsistOf(
				goformationtags.Tag{ // auto-injected
					Key:   "QueueType",
					Value: "Main",
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
			Expect(dlqueue.Tags).To(ConsistOf(
				goformationtags.Tag{ // auto-injected
					Key:   "QueueType",
					Value: "Dead-Letter",
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
		Expect(queue.ContentBasedDeduplication).To(BeFalse())
		Expect(queue.DelaySeconds).To(BeZero())
		Expect(queue.FifoQueue).To(BeFalse())
		Expect(queue.MaximumMessageSize).To(BeZero())
		Expect(queue.MessageRetentionPeriod).To(BeZero())
		Expect(queue.ReceiveMessageWaitTimeSeconds).To(BeZero())
		Expect(queue.RedrivePolicy).To(BeEmpty())
		Expect(queue.VisibilityTimeout).To(BeZero())

		Expect(dlqueue.ContentBasedDeduplication).To(BeFalse())
		Expect(dlqueue.FifoQueue).To(BeFalse())
		Expect(dlqueue.MessageRetentionPeriod).To(BeZero())
		Expect(dlqueue.VisibilityTimeout).To(BeZero())
	})

	Context("when contentBasedDeduplication is set", func() {
		BeforeEach(func() {
			params.ContentBasedDeduplication = true
		})
		It("should set queue ContentBasedDeduplication from spec", func() {
			Expect(queue.ContentBasedDeduplication).To(BeTrue())
			Expect(dlqueue.ContentBasedDeduplication).To(BeTrue())
		})
	})

	Context("when delaySeconds is set", func() {
		BeforeEach(func() {
			params.DelaySeconds = 600
		})
		It("should set queue DelaySeconds from spec", func() {
			Expect(queue.DelaySeconds).To(Equal(600))
		})
	})

	Context("when fifoQueue is set", func() {
		BeforeEach(func() {
			params.FifoQueue = true
		})
		It("should set queue FifoQueue from spec", func() {
			Expect(queue.FifoQueue).To(BeTrue())
			Expect(dlqueue.FifoQueue).To(BeTrue())
		})
	})

	Context("when maximumMessageSize is set", func() {
		BeforeEach(func() {
			params.MaximumMessageSize = 300
		})
		It("should set queue MaximumMessageSize from spec", func() {
			Expect(queue.MaximumMessageSize).To(Equal(300))
		})
	})

	Context("when messageRetentionPeriod is set", func() {
		BeforeEach(func() {
			params.MessageRetentionPeriod = 20
		})
		It("should set queue MessageRetentionPeriod from spec", func() {
			Expect(queue.MessageRetentionPeriod).To(Equal(20))
			Expect(dlqueue.MessageRetentionPeriod).To(Equal(20))
		})
	})

	Context("when receiveMessageWaitTimeSeconds is set", func() {
		BeforeEach(func() {
			params.ReceiveMessageWaitTimeSeconds = 1200
		})
		It("should set queue ReceiveMessageWaitTimeSeconds from spec", func() {
			Expect(queue.ReceiveMessageWaitTimeSeconds).To(Equal(1200))
		})
	})

	Context("when RedriveMaxReceiveCount is set", func() {
		BeforeEach(func() {
			params.RedriveMaxReceiveCount = 1
		})
		It("should set queue ContentBasedDeduplication from spec", func() {
			policy, ok := queue.RedrivePolicy.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(policy["maxReceiveCount"]).To(Equal(1))
		})
	})

	Context("when VisibilityTimeout is set", func() {
		BeforeEach(func() {
			params.VisibilityTimeout = 30
		})
		It("should set queue VisibilityTimeout from spec", func() {
			Expect(queue.VisibilityTimeout).To(Equal(30))
			Expect(dlqueue.VisibilityTimeout).To(Equal(30))
		})
	})

	It("should have outputs for connection details", func() {
		t, err := sqs.QueueTemplate(sqs.QueueParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Outputs).To(And(
			HaveKey("QueueURL"),
			HaveKey("DLQueueURL"),
			HaveKey("QueueARN"),
			HaveKey("DLQueueARN"),
		))
	})
})
