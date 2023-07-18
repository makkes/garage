package matchers

import (
	"errors"
	"fmt"

	"github.com/onsi/gomega/types"
)

type beAssignableToError struct {
	expected interface{}
}

func BeAssignableToError(expected error) types.GomegaMatcher {
	return beAssignableToError{
		expected: &expected,
	}
}

func (m beAssignableToError) Match(actual interface{}) (bool, error) {
	actualErr, ok := actual.(error)
	if !ok {
		return false, fmt.Errorf("BeAssignableToError matcher expects an error")
	}

	return errors.As(actualErr, m.expected), nil
}

func (m beAssignableToError) FailureMessage(actual interface{}) (message string) {
	panic("not implemented") // TODO: Implement
}

func (m beAssignableToError) NegatedFailureMessage(actual interface{}) (message string) {
	panic("not implemented") // TODO: Implement
}
