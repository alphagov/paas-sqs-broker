package sqs_test

import (
	"encoding/json"

	"github.com/alphagov/paas-sqs-broker/sqs"
	goformation "github.com/awslabs/goformation/v4/cloudformation"
	goformationiam "github.com/awslabs/goformation/v4/cloudformation/iam"
	goformationsecretsmanager "github.com/awslabs/goformation/v4/cloudformation/secretsmanager"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	"github.com/awslabs/goformation/v4/intrinsics"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserTemplate", func() {
	var user *goformationiam.User
	var accessKey *goformationiam.AccessKey
	var policy *goformationiam.Policy
	var params sqs.UserParams
	var template *goformation.Template

	BeforeEach(func() {
		params = sqs.UserParams{}
	})

	JustBeforeEach(func() {
		var err error
		template, err = sqs.UserTemplate(params)
		Expect(err).ToNot(HaveOccurred())
		Expect(template.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.User{})))
		Expect(template.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.AccessKey{})))
		Expect(template.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationiam.Policy{})))
		Expect(template.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsecretsmanager.Secret{})))
		var ok bool
		user, ok = template.Resources[sqs.ResourceUser].(*goformationiam.User)
		Expect(ok).To(BeTrue())
		accessKey, ok = template.Resources[sqs.ResourceAccessKey].(*goformationiam.AccessKey)
		Expect(ok).To(BeTrue())
		policy, ok = template.Resources[sqs.ResourcePolicy].(*goformationiam.Policy)
		Expect(ok).To(BeTrue())
		_, ok = template.Resources[sqs.ResourceCredentials].(*goformationsecretsmanager.Secret)
		Expect(ok).To(BeTrue())
	})

	It("should not have any input parameters", func() {
		t, err := sqs.UserTemplate(sqs.UserParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Parameters).To(BeEmpty())
	})

	It("should create a template for a json blob containing provisioned credentials", func() {
		out, err := template.JSON()
		Expect(err).ToNot(HaveOccurred())
		processed, err := intrinsics.ProcessJSON(out, nil)
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
			params.BindingID = "xxxx-xxxx-xxxx"
			params.ResourcePrefix = "prefixed-path"
		})
		It("should set the user name and path", func() {
			Expect(user.UserName).To(Equal("binding-xxxx-xxxx-xxxx"))
			Expect(user.Path).To(Equal("/prefixed-path/"))
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

	Context("when the access policy is 'producer'", func() {
		BeforeEach(func() {
			params.AccessPolicy = "producer"
		})
		It("should use the 'producer' canned access policy", func() {
			Expect(policy.PolicyDocument).To(BeAssignableToTypeOf(sqs.PolicyDocument{}))
			policyDoc := policy.PolicyDocument.(sqs.PolicyDocument)
			Expect(policyDoc.Statement[0].Action).To(ConsistOf(
				"sqs:GetQueueAttributes",
				"sqs:GetQueueUrl",
				"sqs:ListDeadLetterSourceQueues",
				"sqs:ListQueueTags",
				"sqs:SendMessage",
			))
		})
	})

	Context("when the access policy is 'consumer'", func() {
		BeforeEach(func() {
			params.AccessPolicy = "consumer"
		})
		It("should use the 'consumer' canned access policy", func() {
			Expect(policy.PolicyDocument).To(BeAssignableToTypeOf(sqs.PolicyDocument{}))
			policyDoc := policy.PolicyDocument.(sqs.PolicyDocument)
			Expect(policyDoc.Statement[0].Action).To(ConsistOf(
				"sqs:DeleteMessage",
				"sqs:GetQueueAttributes",
				"sqs:GetQueueUrl",
				"sqs:ListDeadLetterSourceQueues",
				"sqs:ListQueueTags",
				"sqs:PurgeQueue",
				"sqs:ReceiveMessage",
			))
		})
	})

	Context("when the access policy is unspecified", func() {
		BeforeEach(func() {
			params.AccessPolicy = ""
		})
		It("should use the 'full' canned access policy", func() {
			Expect(policy.PolicyDocument).To(BeAssignableToTypeOf(sqs.PolicyDocument{}))
			policyDoc := policy.PolicyDocument.(sqs.PolicyDocument)
			Expect(policyDoc.Statement[0].Action).To(ConsistOf(
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
		})
	})

	It("should return an error for unknown access policies", func() {
		t, err := sqs.UserTemplate(sqs.UserParams{
			AccessPolicy: "bananas",
		})
		Expect(t).To(BeNil())
		Expect(err).To(MatchError("unknown access policy \"bananas\""))
	})

	Context("when queue ARNs are set", func() {
		BeforeEach(func() {
			params.PrimaryQueueARN = "abc"
			params.SecondaryQueueARN = "qwe"
		})
		It("the policy is scoped to the queue ARNs", func() {
			Expect(policy.PolicyDocument).To(BeAssignableToTypeOf(sqs.PolicyDocument{}))
			policyDoc := policy.PolicyDocument.(sqs.PolicyDocument)
			Expect(policyDoc.Statement).To(HaveLen(1))
			Expect(policyDoc.Statement[0].Effect).To(Equal("Allow"))
			Expect(policyDoc.Statement[0].Action).To(ConsistOf(
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

	It("should create an active access key", func() {
		Expect(accessKey.Status).To(Equal("Active"))
		Expect(accessKey.UserName).ToNot(BeEmpty())
	})

	It("should have an output for the secretsmanager path to credentials", func() {
		t, err := sqs.UserTemplate(sqs.UserParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Outputs).To(And(
			HaveKey(sqs.OutputCredentialsARN),
		))
	})
})
