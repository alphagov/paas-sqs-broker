package broker_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/pivotal-cf/brokerapi/domain"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Suite")
}

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
