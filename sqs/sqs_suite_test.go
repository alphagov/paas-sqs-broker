package sqs_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSQS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SQS Suite")
}
