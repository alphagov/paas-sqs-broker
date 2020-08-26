package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pivotal-cf/brokerapi"
)

var (
	// capabilities required by cloudformation
	capabilities = []*string{
		aws.String("CAPABILITY_NAMED_IAM"),
	}
	// NoExistErrMatch is a string to match if stack does not exist
	NoExistErrMatch = "does not exist"
	// ErrStackNotFound returned when stack does not exist, or has been deleted
	ErrStackNotFound = fmt.Errorf("STACK_NOT_FOUND")
	// PollingInterval is the duration between calls to check state when waiting for apply/destroy to complete
	PollingInterval = time.Second * 15
)

type SQSProvider struct {
	client sqs.Client
}

func NewSQSProvider(sqsClient sqs.Client) *SQSProvider {
	return &SQSProvider{
		client: sqsClient,
	}
}

func (s *SQSProvider) Provision(ctx context.Context, provisionData provideriface.ProvisionData) (dashboardURL, operationData string, isAsync bool, err error) {

	tmpl, err := sqs.QueueTemplate(sqs.QueueParams{
		QueueName: "??",
		FifoQueue: false, // set based on provisionData
	})
	if err != nil {
		return "", "", false, err
	}

	yaml, err := tmpl.YAML()
	if err != nil {
		return "", "", false, err
	}

	params := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("thing"),
			ParameterValue: aws.String("value"),
		},
	}

	_, err = s.client.CreateStackWithContext(ctx, &cloudformation.CreateStackInput{
		Capabilities: capabilities,
		TemplateBody: aws.String(string(yaml)),
		StackName:    aws.String(s.getStackName(provisionData.InstanceID)),
		Parameters:   params,
	})
	if err != nil {
		return "", "", true, err
	}

	return "", "", true, nil
}

func (s *SQSProvider) Deprovision(ctx context.Context, deprovisionData provideriface.DeprovisionData) (operationData string, isAsync bool, err error) {
	stackName := s.getStackName(deprovisionData.InstanceID)
	stack, err := s.getStack(ctx, deprovisionData.InstanceID)
	if err == ErrStackNotFound {
		// resource is already deleted (or never existsed)
		// so we're done here
		return "", false, nil
	} else if err != nil {
		// failed to get stack status
		return "", false, err // should this be async and checked later
	}
	if *stack.StackStatus == cloudformation.StackStatusDeleteComplete {
		// resource already deleted
		return "", false, nil
	}
	// trigger a delete unless we're already in a deleting state
	if *stack.StackStatus != cloudformation.StackStatusDeleteInProgress {
		_, err := s.client.DeleteStackWithContext(ctx, &cloudformation.DeleteStackInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			return "", false, err
		}
	}

	return "", true, nil
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

func (s *SQSProvider) Update(ctx context.Context, updateData provideriface.UpdateData) (operationData string, isAsync bool, err error) {
	// _, err = r.Client.UpdateStackWithContext(ctx, &UpdateStackInput{
	// 	Capabilities:    capabilities,
	// 	TemplateBody:    aws.String(string(yaml)),
	// 	StackName:       aws.String(stack.GetStackName()),
	// 	StackPolicyBody: stackPolicy,
	// 	Parameters:      params,
	// })
	// if err != nil && !IsNoUpdateError(err) {
	// 	return err
	// }
	// return "", true, nil
	return "", false, ErrUpdateNotSupported
}

func (s *SQSProvider) LastOperation(ctx context.Context, lastOperationData provideriface.LastOperationData) (state brokerapi.LastOperationState, description string, err error) {
	stackName := s.getStackName(lastOperationData.InstanceID)
	stack, err := s.getStack(ctx, stackName)
	if err != nil {
		return "", "", err
	}

	switch *stack.StackStatus {
	case cloudformation.StackStatusDeleteFailed, cloudformation.StackStatusCreateFailed, cloudformation.StackStatusRollbackFailed, cloudformation.StackStatusUpdateRollbackFailed, cloudformation.StackStatusRollbackComplete, cloudformation.StackStatusUpdateRollbackComplete:
		return brokerapi.Succeeded, fmt.Sprintf("failed: %s", *stack.StackStatus), nil
	case cloudformation.StackStatusCreateComplete, cloudformation.StackStatusUpdateComplete, cloudformation.StackStatusDeleteComplete:
		return brokerapi.Succeeded, "ready", nil
	default:
		return brokerapi.InProgress, "pending", nil
	}
}

func (s *SQSProvider) getStack(ctx context.Context, stackName string) (*cloudformation.Stack, error) {
	describeOutput, err := s.client.DescribeStacksWithContext(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		if IsNotFoundError(err) {
			return nil, ErrStackNotFound
		}
		return nil, err
	}
	if describeOutput == nil {
		return nil, fmt.Errorf("describeOutput was nil, potential issue with AWS Client")
	}
	if len(describeOutput.Stacks) == 0 {
		return nil, fmt.Errorf("describeOutput contained no Stacks, potential issue with AWS Client")
	}
	if len(describeOutput.Stacks) > 1 {
		return nil, fmt.Errorf("describeOutput contained multiple Stacks which is unexpected when calling with StackName, potential issue with AWS Client")
	}
	state := describeOutput.Stacks[0]
	if state.StackStatus == nil {
		return nil, fmt.Errorf("describeOutput contained a nil StackStatus, potential issue with AWS Client")
	}
	return state, nil
}

func (s *SQSProvider) getStackName(instanceID string) string {
	return instanceID
}

func IsNotFoundError(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "ResourceNotFoundException" {
			return true
		} else if awsErr.Code() == "ValidationError" && strings.Contains(awsErr.Message(), NoExistErrMatch) {
			return true
		}
	}
	return false
}

func in(needle string, haystack []string) bool {
	for _, s := range haystack {
		if needle == s {
			return true
		}
	}
	return false
}
