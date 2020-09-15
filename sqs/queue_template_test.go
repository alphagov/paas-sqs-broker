package sqs_test

import (
	"github.com/alphagov/paas-sqs-broker/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("QueueTemplate", func() {
	var primaryQueue *sqs.Queue
	var secondaryQueue *sqs.Queue
	var queueName string
	var isFIFO bool
	var tags map[string]string

	BeforeEach(func() {
		queueName = ""
		isFIFO = false
		tags = nil
	})

	JustBeforeEach(func() {
		t, err := sqs.QueueTemplate(queueName, isFIFO, tags)
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&sqs.Queue{})))
		var ok bool
		primaryQueue, ok = t.Resources[sqs.ResourcePrimaryQueue].(*sqs.Queue)
		Expect(ok).To(BeTrue())
		secondaryQueue, ok = t.Resources[sqs.ResourceSecondaryQueue].(*sqs.Queue)
		Expect(ok).To(BeTrue())
	})

	Context("when queueName is set", func() {
		BeforeEach(func() {
			queueName = "q-name-a"
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
			tags = map[string]string{
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

	It("should default to Standard (non-FIFO) queues", func() {
		Expect(primaryQueue.FifoQueue).To(BeFalse())
		Expect(secondaryQueue.FifoQueue).To(BeFalse())
	})

	Context("when fifoQueue is set", func() {
		BeforeEach(func() {
			isFIFO = true
		})
		It("should set queue FifoQueue from spec", func() {
			Expect(primaryQueue.FifoQueue).To(BeTrue())
			Expect(secondaryQueue.FifoQueue).To(BeTrue())
		})
	})

	It("should have outputs for connection details", func() {
		t, err := sqs.QueueTemplate("", false, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Outputs).To(And(
			HaveKey(sqs.OutputPrimaryQueueARN),
			HaveKey(sqs.OutputPrimaryQueueURL),
			HaveKey(sqs.OutputSecondaryQueueARN),
			HaveKey(sqs.OutputSecondaryQueueURL),
		))
	})
})
