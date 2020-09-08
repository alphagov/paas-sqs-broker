package sqs_test

import (
	"github.com/alphagov/paas-sqs-broker/sqs"
	goformationiam "github.com/awslabs/goformation/v4/cloudformation/iam"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserTemplate", func() {
	var user *goformationiam.User
	var accessKey *goformationiam.AccessKey
	var policy *goformationiam.Policy
	var params sqs.UserParams

	BeforeEach(func() {
		params = sqs.UserParams{}
	})

	JustBeforeEach(func() {
		t, err := sqs.UserTemplate(params)
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.User{})))
		Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.AccessKey{})))
		Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.Policy{})))
		var ok bool
		user, ok = t.Resources[sqs.SQSResourceIAMUserResourceName].(*goformationiam.User)
		Expect(ok).To(BeTrue())
		accessKey, ok = t.Resources[sqs.SQSResourceIAMAccessKeyResourceName].(*goformationiam.AccessKey)
		Expect(ok).To(BeTrue())
		policy, ok = t.Resources[sqs.SQSResourceIAMPolicyResourceName].(*goformationiam.Policy)
		Expect(ok).To(BeTrue())
	})

	It("should not have any input parameters", func() {
		t, err := sqs.UserTemplate(sqs.UserParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Parameters).To(BeEmpty())
	})

	Context("when userName is set", func() {
		BeforeEach(func() {
			params.UserName = "paas-sqs-broker-a"
		})
		It("should have a user name prefixed with broker prefix", func() {
			Expect(user.UserName).To(Equal("paas-sqs-broker-a"))
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
			Expect(user.Tags).To(ConsistOf(
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

	Context("when QueueARN is set", func() {
		BeforeEach(func() {
			params.QueueARN = "abc"
			params.DLQueueARN = "qwe"
		})
		It("the policy is scoped to the queue ARN", func() {
			Expect(policy.PolicyDocument).To(BeAssignableToTypeOf(sqs.PolicyDocument{}))
			policyDoc := policy.PolicyDocument.(sqs.PolicyDocument)
			Expect(policyDoc.Statement).To(HaveLen(1))
			Expect(policyDoc.Statement[0].Effect).To(Equal("Allow"))
			Expect(policyDoc.Statement[0].Action).To(ContainElements(
				"sqs:ChangeMessageVisibility",
				"sqs:DeleteMessage",
				"sqs:GetQueueAttributes",
				"sqs:GetQueueUrl",
				"sqs:ListDeadLetterSourceQueues",
				"sqs:ListQueueTags",
				"sqs:PurgeQueue",
				"sqs:ReceiveMessage",
				"sqs:SendMessage",
			))
			Expect(policyDoc.Statement[0].Resource).To(ConsistOf("abc", "qwe"))
		})
	})

	It("should create an active access key", func () {
		Expect(accessKey.Status).To(Equal("Active"))
		Expect(accessKey.UserName).ToNot(BeEmpty())
	})

	It("should have outputs for connection details", func() {
		t, err := sqs.UserTemplate(sqs.UserParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Outputs).To(And(
			HaveKey("IAMAccessKeyID"),
			HaveKey("IAMSecretsAccessKey"),
		))
	})
})
