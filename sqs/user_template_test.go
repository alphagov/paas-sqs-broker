package sqs_test

import (
	"encoding/json"

	"github.com/alphagov/paas-sqs-broker/sqs"
	goformation "github.com/awslabs/goformation/v4"
	goformationiam "github.com/awslabs/goformation/v4/cloudformation/iam"
	goformationsecretsmanager "github.com/awslabs/goformation/v4/cloudformation/secretsmanager"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	"github.com/awslabs/goformation/v4/intrinsics"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("UserTemplate", func() {
	var user *goformationiam.User
	var policy *goformationiam.Policy
	var builder sqs.UserTemplateBuilder
	var rawText string

	BeforeEach(func() {
		builder = sqs.UserTemplateBuilder{}
	})

	JustBeforeEach(func() {
		var err error
		rawText, err = builder.Build()
		Expect(err).ToNot(HaveOccurred())
		template, err := goformation.ParseYAML([]byte(rawText))
		Expect(err).ToNot(HaveOccurred())
		Expect(template.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.User{})))
		Expect(template.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.AccessKey{})))
		Expect(template.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.Policy{})))
		Expect(template.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsecretsmanager.Secret{})))
		var ok bool
		user, ok = template.Resources[sqs.ResourceUser].(*goformationiam.User)
		Expect(ok).To(BeTrue())
		policy, ok = template.Resources[sqs.ResourcePolicy].(*goformationiam.Policy)
		Expect(ok).To(BeTrue())
		_, ok = template.Resources[sqs.ResourceCredentials].(*goformationsecretsmanager.Secret)
		Expect(ok).To(BeTrue())
	})

	It("should not have any input parameters", func() {
		text, err := builder.Build()
		Expect(err).ToNot(HaveOccurred())
		t, err := goformation.ParseYAML([]byte(text))
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Parameters).To(BeEmpty())
	})

	It("should create a template for a json blob containing provisioned credentials", func() {
		processed, err := intrinsics.ProcessYAML([]byte(rawText), nil)
		Expect(err).ToNot(HaveOccurred())
		var result map[string]interface{}
		err = json.Unmarshal(processed, &result)
		Expect(err).ToNot(HaveOccurred())
		resources := result["Resources"].(map[string]interface{})
		resource := resources[sqs.ResourceCredentials].(map[string]interface{})
		properties := resource["Properties"].(map[string]interface{})
		value := properties["SecretString"].(string)
		var credentials map[string]string
		err = json.Unmarshal([]byte(value), &credentials)
		Expect(err).ToNot(HaveOccurred())
		Expect(credentials).To(HaveKey("aws_access_key_id"))
		Expect(credentials).To(HaveKey("aws_secret_access_key"))
		Expect(credentials).To(HaveKey("aws_region"))
		Expect(credentials).To(HaveKey("primary_queue_url"))
		Expect(credentials).To(HaveKey("secondary_queue_url"))
	})

	Context("when binding id and prefix are set", func() {
		BeforeEach(func() {
			builder.BindingID = "xxxx-xxxx-xxxx"
			builder.ResourcePrefix = "prefixed-path"
		})
		It("should set the user name and path", func() {
			Expect(user.UserName).To(Equal("binding-xxxx-xxxx-xxxx"))
			Expect(user.Path).To(Equal("/prefixed-path/"))
		})

	})

	Context("when tags are set", func() {
		BeforeEach(func() {
			builder.Tags = map[string]string{
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

	Context("when the access policy is 'producer'", func() {
		BeforeEach(func() {
			builder.AccessPolicy = "producer"
		})
		It("should use the 'producer' canned access policy", func() {
			Expect(policy.PolicyDocument).To(
				HaveKeyWithValue("Statement", ConsistOf(
					HaveKeyWithValue("Action", ConsistOf(
						"sqs:GetQueueAttributes",
						"sqs:GetQueueUrl",
						"sqs:ListDeadLetterSourceQueues",
						"sqs:ListQueueTags",
						"sqs:SendMessage",
					)),
				)))
		})
	})

	Context("when the access policy is 'consumer'", func() {
		BeforeEach(func() {
			builder.AccessPolicy = "consumer"
		})
		It("should use the 'consumer' canned access policy", func() {
			Expect(policy.PolicyDocument).To(
				HaveKeyWithValue("Statement", ConsistOf(
					HaveKeyWithValue("Action", ConsistOf(
						"sqs:DeleteMessage",
						"sqs:GetQueueAttributes",
						"sqs:GetQueueUrl",
						"sqs:ListDeadLetterSourceQueues",
						"sqs:ListQueueTags",
						"sqs:PurgeQueue",
						"sqs:ReceiveMessage",
					)),
				)))
		})
	})

	Context("when the access policy is unspecified", func() {
		BeforeEach(func() {
			builder.AccessPolicy = ""
		})
		It("should use the 'full' canned access policy", func() {
			Expect(policy.PolicyDocument).To(
				HaveKeyWithValue("Statement", ConsistOf(
					HaveKeyWithValue("Action", ConsistOf(
						"sqs:ChangeMessageVisibility",
						"sqs:DeleteMessage",
						"sqs:GetQueueAttributes",
						"sqs:GetQueueUrl",
						"sqs:ListDeadLetterSourceQueues",
						"sqs:ListQueueTags",
						"sqs:PurgeQueue",
						"sqs:ReceiveMessage",
						"sqs:SendMessage",
					)),
				)))
		})
	})

	Context("when additional user policy is not set", func() {
		It("should not set an additional policy", func() {
			Expect(user.ManagedPolicyArns).To(BeEmpty())
		})
	})

	Context("when additional user policy is set", func() {
		BeforeEach(func() {
			builder.AdditionalUserPolicy = "lololol"
		})
		It("should set an additional policy", func() {
			Expect(user.ManagedPolicyArns).To(ConsistOf("lololol"))
		})
	})

	It("should return an error for unknown access policies", func() {
		t, err := sqs.UserTemplateBuilder{
			AccessPolicy: "bananas",
		}.Build()

		Expect(t).To(BeEmpty())
		Expect(err).To(MatchError("unknown access policy \"bananas\""))
	})

	Context("when queue ARNs are set", func() {
		BeforeEach(func() {
			builder.PrimaryQueueARN = "abc"
			builder.SecondaryQueueARN = "qwe"
		})
		It("the policy is scoped to the queue ARNs", func() {
			Expect(policy.PolicyDocument).To(
				HaveKeyWithValue("Statement", ConsistOf(
					And(
						HaveKeyWithValue("Effect", "Allow"),
						HaveKeyWithValue("Resource", ConsistOf("abc", "qwe")),
						HaveKeyWithValue("Action", ConsistOf(
							"sqs:ChangeMessageVisibility",
							"sqs:DeleteMessage",
							"sqs:GetQueueAttributes",
							"sqs:GetQueueUrl",
							"sqs:ListDeadLetterSourceQueues",
							"sqs:ListQueueTags",
							"sqs:PurgeQueue",
							"sqs:ReceiveMessage",
							"sqs:SendMessage",
						))),
				)))
		})
	})

	It("should create an active access key", func() {
		var result map[string]interface{}
		Expect(yaml.Unmarshal([]byte(rawText), &result)).To(Succeed())
		resources := result["Resources"].(map[interface{}]interface{})
		resource := resources[sqs.ResourceAccessKey].(map[interface{}]interface{})
		properties := resource["Properties"].(map[interface{}]interface{})

		Expect(properties).To(HaveKeyWithValue("Status", "Active"))
		Expect(properties).To(HaveKeyWithValue("UserName", HaveKeyWithValue("Ref", sqs.ResourceUser)))
	})

	It("should have an output for the secretsmanager path to credentials", func() {
		text, err := builder.Build()
		Expect(err).ToNot(HaveOccurred())
		t, err := goformation.ParseYAML([]byte(text))
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Outputs).To(And(
			HaveKey(sqs.OutputCredentialsARN),
		))
	})
})
