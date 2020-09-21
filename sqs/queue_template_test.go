package sqs_test

import (
	"github.com/alphagov/paas-sqs-broker/sqs"
	goformation "github.com/awslabs/goformation/v4"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("QueueTemplateBuilder", func() {
	var primaryQueue *goformationsqs.Queue
	var secondaryQueue *goformationsqs.Queue
	var builder sqs.QueueTemplateBuilder

	BeforeEach(func() {
		builder = sqs.QueueTemplateBuilder{}
	})

	JustBeforeEach(func() {
		text, err := builder.Build()
		Expect(err).ToNot(HaveOccurred())
		t, err := goformation.ParseYAML([]byte(text))
		Expect(err).ToNot(HaveOccurred())

		Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
		var ok bool
		primaryQueue, ok = t.Resources[sqs.ResourcePrimaryQueue].(*goformationsqs.Queue)
		Expect(ok).To(BeTrue())
		secondaryQueue, ok = t.Resources[sqs.ResourceSecondaryQueue].(*goformationsqs.Queue)
		Expect(ok).To(BeTrue())
	})

	Context("when QueueName is set for a non-FIFO queue", func() {
		BeforeEach(func() {
			builder.QueueName = "q-name-a"
		})
		It("should set primary queue name with a .stnd extension", func() {
			Expect(primaryQueue.QueueName).To(HavePrefix("q-name-a"))
			Expect(primaryQueue.QueueName).To(HaveSuffix("-pri"))
		})
		It("should set secondary queue name with a .stnd extension", func() {
			Expect(secondaryQueue.QueueName).To(HavePrefix("q-name-a"))
			Expect(secondaryQueue.QueueName).To(HaveSuffix("-sec"))
		})
		It("should not break the 80 character limit for queue name", func() {
			Expect(len(primaryQueue.QueueName)).To(BeNumerically("<", 80))
			Expect(len(secondaryQueue.QueueName)).To(BeNumerically("<", 80))
		})
	})

	Context("when QueueName is set for a FIFO queue", func() {
		BeforeEach(func() {
			builder.QueueName = "q-name-a"
			builder.FIFOQueue = true
		})
		It("should set primary queue name with a .fifo extension", func() {
			Expect(primaryQueue.QueueName).To(HavePrefix("q-name-a"))
			Expect(primaryQueue.QueueName).To(HaveSuffix("-pri.fifo"))
		})
		It("should set secondary queue name with a .fifo extension", func() {
			Expect(secondaryQueue.QueueName).To(HavePrefix("q-name-a"))
			Expect(secondaryQueue.QueueName).To(HaveSuffix("-sec.fifo"))
		})
		It("should not break the 80 character limit for queue name", func() {
			Expect(len(primaryQueue.QueueName)).To(BeNumerically("<", 80))
			Expect(len(secondaryQueue.QueueName)).To(BeNumerically("<", 80))
		})
	})

	Context("when tags are set", func() {
		BeforeEach(func() {
			builder.Tags = map[string]string{
				"Name":        "instance-1234",
				"Service":     "sqs",
				"ServiceID":   "service-abcd",
				"Environment": "autom8",
			}
		})
		It("should have suitable tags", func() {
			Expect(primaryQueue.Tags).To(ConsistOf(
				goformationtags.Tag{ // auto-injected
					Key:   "QueueType",
					Value: "Primary",
				},
				goformationtags.Tag{
					Key:   "Name",
					Value: "instance-1234",
				},
				goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				},
				goformationtags.Tag{
					Key:   "ServiceID",
					Value: "service-abcd",
				},
				goformationtags.Tag{
					Key:   "Environment",
					Value: "autom8",
				},
			))
			Expect(secondaryQueue.Tags).To(ConsistOf(
				goformationtags.Tag{ // auto-injected
					Key:   "QueueType",
					Value: "Secondary",
				},
				goformationtags.Tag{
					Key:   "Name",
					Value: "instance-1234",
				},
				goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				},
				goformationtags.Tag{
					Key:   "ServiceID",
					Value: "service-abcd",
				},
				goformationtags.Tag{
					Key:   "Environment",
					Value: "autom8",
				},
			))
		})
	})

	It("should default to Standard (non-FIFO) queues", func() {
		Expect(primaryQueue.FifoQueue).To(BeFalse())
		Expect(secondaryQueue.FifoQueue).To(BeFalse())
	})

	Context("when FIFO queue is configured", func() {
		BeforeEach(func() {
			builder.FIFOQueue = true
		})
		It("should set queue FifoQueue from spec", func() {
			Expect(primaryQueue.FifoQueue).To(BeTrue())
			Expect(secondaryQueue.FifoQueue).To(BeTrue())
		})
	})

	It("should have outputs for connection details", func() {
		builder := &sqs.QueueTemplateBuilder{}
		text, err := builder.Build()
		Expect(err).ToNot(HaveOccurred())
		t, err := goformation.ParseYAML([]byte(text))
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Outputs).To(And(
			HaveKey(sqs.OutputPrimaryQueueARN),
			HaveKey(sqs.OutputPrimaryQueueURL),
			HaveKey(sqs.OutputSecondaryQueueARN),
			HaveKey(sqs.OutputSecondaryQueueURL),
		))
	})
})
