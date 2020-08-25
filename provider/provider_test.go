package provider_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"

	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/alphagov/paas-sqs-broker/provider"
	fakeClient "github.com/alphagov/paas-sqs-broker/sqs/fakes"
	"github.com/pivotal-cf/brokerapi"
)

var _ = Describe("Provider", func() {
	var (
		fakeSQSClient *fakeClient.FakeClient
		sqsProvider   *provider.SQSProvider
	)

	BeforeEach(func() {
		fakeSQSClient = &fakeClient.FakeClient{}
		sqsProvider = provider.NewSQSProvider(fakeSQSClient)
	})

	Describe("Provision", func() {
		It("passes the correct parameters to the client", func() {
			provisionData := provideriface.ProvisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			_, _, _, err := sqsProvider.Provision(context.Background(), provisionData)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Deprovision", func() {
	})

	Describe("Bind", func() {
	})

	Describe("Unbind", func() {
	})

	Describe("Update", func() {
		It("does not support updating a bucket", func() {
			updateData := provideriface.UpdateData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}

			_, _, err := sqsProvider.Update(context.Background(), updateData)
			Expect(err).To(MatchError(provider.ErrUpdateNotSupported))
		})
	})

	Describe("LastOperation", func() {
		It("returns success unconditionally", func() {
			state, description, err := sqsProvider.LastOperation(context.Background(), provideriface.LastOperationData{})
			Expect(err).NotTo(HaveOccurred())
			Expect(description).To(Equal("Last operation polling not required. All operations are synchronous."))
			Expect(state).To(Equal(brokerapi.Succeeded))
		})
	})
})
