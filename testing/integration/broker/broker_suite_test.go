package broker_test

import (
	"os"
	"testing"

	"github.com/alphagov/paas-service-broker-base/broker"
	"github.com/alphagov/paas-service-broker-base/testing/mock_locket_server"
	"github.com/alphagov/paas-sqs-broker/sqs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	BrokerSuiteData SuiteData
	mockLocket      *mock_locket_server.MockLocket
	locketFixtures  mock_locket_server.LocketFixtures
)

type SuiteData struct {
	AWSRegion string
}

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Suite")
}

var _ = BeforeSuite(func() {
	file, err := os.Open("../../fixtures/config.json")
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	config, err := broker.NewConfig(file)
	Expect(err).ToNot(HaveOccurred())
	sqsClientConfig, err := sqs.NewSQSClientConfig(config.Provider)
	Expect(err).ToNot(HaveOccurred())

	// sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(sqsClientConfig.AWSRegion)}))

	// Start test Locket server
	locketFixtures, err = mock_locket_server.SetupLocketFixtures()
	Expect(err).NotTo(HaveOccurred())
	mockLocket, err = mock_locket_server.New("keyBasedLock", locketFixtures.Filepath)
	Expect(err).NotTo(HaveOccurred())
	mockLocket.Start(mockLocket.Logger, mockLocket.ListenAddress, mockLocket.Certificate, mockLocket.Handler)

	BrokerSuiteData = SuiteData{
		AWSRegion: sqsClientConfig.AWSRegion,
	}
})

var _ = AfterSuite(func() {
	if mockLocket != nil {
		mockLocket.Stop()
	}
	locketFixtures.Cleanup()
})
