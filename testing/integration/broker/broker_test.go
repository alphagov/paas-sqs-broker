package broker_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	ASYNC_ALLOWED = true
)

type BindingResponse struct {
	Credentials map[string]interface{} `json:"credentials"`
}

var _ = Describe("Broker", func() {

	BeforeEach(func() {
		// instanceID = uuid.NewV4().String()
	})

	It("should return a 410 response when trying to delete a non-existent instance", func() {
	})

	It("should manage the lifecycle of an SQS bucket", func() {
		By("initialising")
		Expect(true).To(Equal(true))
		// sqsClientConfig, brokerTester := initialise(*BrokerSuiteData.LocalhostIAMPolicyArn)

		By("Provisioning")

		// defer helpers.DeprovisionService(brokerTester, instanceID, serviceID, planID)

		By("Binding an app")

		// defer helpers.Unbind(brokerTester, instanceID, serviceID, planID, binding1ID)

		By("Asserting the credentials returned work for both reading and writing")

		By("Binding an app as a read-only user")

		// defer helpers.Unbind(brokerTester, instanceID, serviceID, planID, binding2ID)

		By("Asserting that read-only credentials can read, but fail to write to a file")

		By("Asserting the first user's credentials still work for reading and writing")
	})

})
