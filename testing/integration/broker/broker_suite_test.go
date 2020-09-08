package broker_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alphagov/paas-service-broker-base/broker"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/pivotal-cf/brokerapi/domain"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var (
	BrokerSuiteData SuiteData
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
	sqsClientConfig, err := sqs.NewConfig(config.Provider)
	Expect(err).ToNot(HaveOccurred())

	// by default the integration tests run without a permission boundary so
	// that there are no dependencies on setting up external IAM policies
	// to run the test with a predefined permission boundary policy set the following environment variable:
	//
	// PERMISSIONS_BOUNDARY_ARN="arn:aws:iam::ACCOUNT-ID:policy/SQSBrokerUserPermissionsBoundary"
	//
	optionalPermissionsBoundary := os.Getenv("PERMISSIONS_BOUNDARY_ARN")
	if optionalPermissionsBoundary != "" {
		sqsClientConfig.PermissionsBoundary = optionalPermissionsBoundary
	}

	BrokerSuiteData = SuiteData{
		AWSRegion: sqsClientConfig.AWSRegion,
	}
})

func HaveLastOperationState(expectedState domain.LastOperationState) types.GomegaMatcher {
	return &haveLastOperationStateMatcher{
		expectedState: expectedState,
	}
}

type haveLastOperationStateMatcher struct {
	expectedState domain.LastOperationState
}

func (matcher *haveLastOperationStateMatcher) state(actual interface{}) (domain.LastOperationState, error) {
	res, ok := actual.(*httptest.ResponseRecorder)
	if !ok {
		return "", fmt.Errorf("HaveLastOperationState matcher expects an httptest.ResponseRecorder")
	}
	var ret struct {
		State domain.LastOperationState `json:"state"`
	}
	_ = json.NewDecoder(res.Result().Body).Decode(&ret)
	return ret.State, nil
}

func (matcher *haveLastOperationStateMatcher) Match(actual interface{}) (success bool, err error) {
	actualState, err := matcher.state(actual)
	if err != nil {
		return false, err
	}
	return actualState == matcher.expectedState, nil
}

func (matcher *haveLastOperationStateMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto have last operation state of\n\t%#v", actual, matcher.expectedState)
}

func (matcher *haveLastOperationStateMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to have last operation state of\n\t%#v", actual, matcher.expectedState)
}

func (matcher *haveLastOperationStateMatcher) MatchMayChangeInTheFuture(actual interface{}) bool {
	actualState, err := matcher.state(actual)
	if err != nil {
		return false
	}
	switch actualState {
	case domain.InProgress:
		return true
	default:
		return false
	}
}
