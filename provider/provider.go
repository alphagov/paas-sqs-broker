package provider

import (
	"context"
	"errors"

	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/pivotal-cf/brokerapi"
)

type SQSProvider struct {
	client sqs.Client
}

func NewSQSProvider(sqsClient sqs.Client) *SQSProvider {
	return &SQSProvider{
		client: sqsClient,
	}
}

func (s *SQSProvider) Provision(ctx context.Context, provisionData provideriface.ProvisionData) (
	dashboardURL, operationData string, isAsync bool, err error) {

	return "", "", false, err
}

func (s *SQSProvider) Deprovision(ctx context.Context, deprovisionData provideriface.DeprovisionData) (
	operationData string, isAsync bool, err error) {

	return "", false, err
}

func (s *SQSProvider) Bind(ctx context.Context, bindData provideriface.BindData) (
	binding brokerapi.Binding, err error) {

	return brokerapi.Binding{
		IsAsync:     false,
		Credentials: brokerapi.Binding{},
	}, nil
}

func (s *SQSProvider) Unbind(ctx context.Context, unbindData provideriface.UnbindData) (
	unbinding brokerapi.UnbindSpec, err error) {

	return brokerapi.UnbindSpec{
		IsAsync: false,
	}, nil
}

var ErrUpdateNotSupported = errors.New("Updating the SQS queue is currently not supported")

func (s *SQSProvider) Update(ctx context.Context, updateData provideriface.UpdateData) (
	operationData string, isAsync bool, err error) {
	return "", false, ErrUpdateNotSupported
}

func (s *SQSProvider) LastOperation(ctx context.Context, lastOperationData provideriface.LastOperationData) (
	state brokerapi.LastOperationState, description string, err error) {
	return brokerapi.Succeeded, "Last operation polling not required. All operations are synchronous.", nil
}
