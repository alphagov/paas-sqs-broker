package sqs_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var ()

	BeforeEach(func() {
		// logger = lager.NewLogger("sqs-service-broker-test")
	})

	It("Does a thing", func() {
		Expect(true).To(Equal(true))
	})
})
