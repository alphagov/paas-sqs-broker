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

	Context("when QueueName is set", func() {
		BeforeEach(func() {
			builder.QueueName = "q-name-a"
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
			builder.Tags.Name = "instance-1234"
			builder.Tags.ServiceID = "service-abcd"
			builder.Tags.Environment = "autom8"
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

	XContext("when IsFIFO is set", func() {
		BeforeEach(func() {
			// builder.IsFIFO = true
		})
		It("should set queue FifoQueue from spec", func() {
			Expect(primaryQueue.FifoQueue).To(BeTrue())
			Expect(secondaryQueue.FifoQueue).To(BeTrue())
		})
	})

	It("should have outputs for connection details", func() {
		text, err := sqs.QueueTemplateBuilder{}.Build()
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
