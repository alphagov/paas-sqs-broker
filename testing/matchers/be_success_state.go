package matchers

import (
	"fmt"

	"github.com/onsi/gomega/types"
	"github.com/pivotal-cf/brokerapi/domain"
)

const MaxFailures = 2

// BeSuccessState is a gomega custom matcher for checking that the result of
// receiving from a `chan domain.LastOperationState` is a successful state. It
// mostly works like the `Receive(Equal(state))` matcher, but with the addition
// that it understands how to bail out early from terminal conditions, making
// it useful for use in Eventually calls polling the state chan
func BeSuccessState() types.GomegaMatcher {
	return &haveLastOperationStateMatcher{
		expectedState: domain.Succeeded,
	}
}

type haveLastOperationStateMatcher struct {
	expectedState   domain.LastOperationState
	inTerminalState bool
	failures        int
}

func (matcher *haveLastOperationStateMatcher) Match(actual interface{}) (success bool, err error) {
	ch, ok := actual.(chan domain.LastOperationState)
	if !ok {
		return false, fmt.Errorf("HaveLastOperationState matcher expects an chan domain.LastOperationState")
	}
	select {
	case actualState, ok := <-ch:
		if !ok { // channel got closed, bail out
			matcher.inTerminalState = true
			return false, nil
		} else if actualState == domain.Failed { // if we see multiple failures in a row bail out
			matcher.failures++
			if matcher.failures > MaxFailures {
				matcher.inTerminalState = true
				return false, nil
			}
		} else { // reset fail count as something good happened
			matcher.failures = 0
		}
		return actualState == matcher.expectedState, nil
	default:
		return false, nil // nothing on channel, return immeditely
	}
}

func (matcher *haveLastOperationStateMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto have last operation state of\n\t%#v", actual, matcher.expectedState)
}

func (matcher *haveLastOperationStateMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to have last operation state of\n\t%#v", actual, matcher.expectedState)
}

func (matcher *haveLastOperationStateMatcher) MatchMayChangeInTheFuture(actual interface{}) bool {
	return !matcher.inTerminalState
}
