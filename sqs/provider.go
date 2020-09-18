package sqs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
)

var (
	// capabilities required by cloudformation
	capabilities = []*string{
		aws.String("CAPABILITY_NAMED_IAM"),
	}
	// NoExistErrMatch is a string to match if stack does not exist
	NoExistErrMatch = "does not exist"
	// ErrStackNotFound returned when stack does not exist, or has been deleted
	ErrStackNotFound      = fmt.Errorf("cloudformation stack does not exist")
	ErrUpdateNotSupported = errors.New("Updating the SQS queue is currently not supported")
	// PollingInterval is the duration between calls to check state when waiting for apply/destroy to complete
	PollingInterval      = time.Second * 15
	ProvisionOperation   = "provision"
	DeprovisionOperation = "deprovision"
	UpdateOperation      = "update"
	BindOperation        = "bind"
	UnbindOperation      = "unbind"
)

type Provider struct {
	Environment         string // Name of environment to tag resources with
	Client              Client // AWS SDK compatible client
	ResourcePrefix      string // AWS resources with be named with this prefix
	PermissionsBoundary string // IAM users created on bind will have this boundary
	Timeout             time.Duration
	Logger              lager.Logger
}

func (s *Provider) Provision(ctx context.Context, provisionData provideriface.ProvisionData) (*domain.ProvisionedServiceSpec, error) {
	if provisionData.Plan.Name == "fifo" {
		return nil, apiresponses.NewFailureResponseBuilder(errors.New("FIFO plan unimplemented"), http.StatusNotImplemented, "not-implemented").WithEmptyResponse().Build()
	}
	tmplParams := QueueImmutableParams{}
	tmplParams.QueueName = s.getStackName(provisionData.InstanceID)
	tmplParams.Tags.Name = provisionData.InstanceID
	tmplParams.Tags.ServiceID = provisionData.Details.ServiceID
	tmplParams.Tags.Environment = s.Environment

	tmpl := QueueTemplate(tmplParams)

	params := QueueUpdatableParams{}
	if provisionData.Details.RawParameters != nil {
		if err := json.Unmarshal(provisionData.Details.RawParameters, &params); err != nil {
			return nil, err
		}
	}

	_, err := s.Client.CreateStackWithContext(ctx, &cloudformation.CreateStackInput{
		Capabilities: capabilities,
		TemplateBody: aws.String(tmpl),
		StackName:    aws.String(s.getStackName(provisionData.InstanceID)),
		Parameters:   params.CreateParams(),
	})
	if err != nil {
		return nil, err
	}

	return &domain.ProvisionedServiceSpec{
		OperationData: ProvisionOperation,
		IsAsync:       true,
	}, nil
}

func (s *Provider) Deprovision(ctx context.Context, deprovisionData provideriface.DeprovisionData) (*domain.DeprovisionServiceSpec, error) {
	stackName := s.getStackName(deprovisionData.InstanceID)
	stack, err := s.getStack(ctx, stackName)
	if err == ErrStackNotFound {
		// resource is already deleted (or never existsed)
		// so we're done here
		return &domain.DeprovisionServiceSpec{
			OperationData: DeprovisionOperation,
			IsAsync:       false,
		}, nil
	} else if err != nil {
		// failed to get stack status
		return nil, err // should this be async and checked later
	}
	if *stack.StackStatus == cloudformation.StackStatusDeleteComplete {
		// resource already deleted
		return &domain.DeprovisionServiceSpec{}, nil
	}
	// trigger a delete unless we're already in a deleting state
	if *stack.StackStatus != cloudformation.StackStatusDeleteInProgress {
		_, err := s.Client.DeleteStackWithContext(ctx, &cloudformation.DeleteStackInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			return nil, err
		}
	}

	return &domain.DeprovisionServiceSpec{
		OperationData: DeprovisionOperation,
		IsAsync:       true,
	}, nil
}

func (s *Provider) Bind(ctx context.Context, bindData provideriface.BindData) (*domain.Binding, error) {
	queueStackName := s.getStackName(bindData.InstanceID)
	queueStack, err := s.getStack(ctx, queueStackName)
	if err == ErrStackNotFound {
		// resource is already deleted (or never existsed)
		// so we're done here
		return nil, brokerapi.ErrInstanceDoesNotExist
	} else if err != nil {
		// failed to get stack status
		return nil, err // should this be async and checked later
	}

	params := UserParams{
		BindingID:           bindData.BindingID,
		ResourcePrefix:      s.ResourcePrefix,
		PermissionsBoundary: s.PermissionsBoundary,
		Tags: map[string]string{
			"Name":        bindData.BindingID,
			"Service":     "sqs",
			"ServiceID":   bindData.Details.ServiceID,
			"Environment": s.Environment,
		},
		PrimaryQueueARN:   getStackOutput(queueStack, OutputPrimaryQueueARN),
		PrimaryQueueURL:   getStackOutput(queueStack, OutputPrimaryQueueURL),
		SecondaryQueueARN: getStackOutput(queueStack, OutputSecondaryQueueARN),
		SecondaryQueueURL: getStackOutput(queueStack, OutputSecondaryQueueURL),
	}

	tmpl, err := UserTemplate(params)
	if err != nil {
		return nil, err
	}

	yaml, err := tmpl.YAML()
	if err != nil {
		return nil, err
	}

	_, err = s.Client.CreateStackWithContext(ctx, &cloudformation.CreateStackInput{
		Capabilities: capabilities,
		TemplateBody: aws.String(string(yaml)),
		StackName:    aws.String(s.getStackName(bindData.BindingID)),
	})
	if err != nil {
		return nil, err
	}

	return &domain.Binding{
		IsAsync:       true,
		OperationData: BindOperation,
	}, nil
}

func (s *Provider) Unbind(ctx context.Context, unbindData provideriface.UnbindData) (*domain.UnbindSpec, error) {
	stackName := s.getStackName(unbindData.BindingID)
	stack, err := s.getStack(ctx, stackName)
	if err == ErrStackNotFound {
		// resource is already deleted (or never existsed)
		// so we're done here
		return &domain.UnbindSpec{
			OperationData: UnbindOperation,
			IsAsync:       false,
		}, nil
	} else if err != nil {
		// failed to get stack status
		return nil, err // should this be async and checked later
	}
	if *stack.StackStatus == cloudformation.StackStatusDeleteComplete {
		// resource already deleted
		return &domain.UnbindSpec{}, nil
	}
	// trigger a delete unless we're already in a deleting state
	if *stack.StackStatus != cloudformation.StackStatusDeleteInProgress {
		_, err := s.Client.DeleteStackWithContext(ctx, &cloudformation.DeleteStackInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			return nil, err
		}
	}

	return &domain.UnbindSpec{
		OperationData: UnbindOperation,
		IsAsync:       true,
	}, nil
}

func (s *Provider) Update(ctx context.Context, updateData provideriface.UpdateData) (*domain.UpdateServiceSpec, error) {
	params := QueueUpdatableParams{}
	if updateData.Details.RawParameters != nil {
		if err := json.Unmarshal(updateData.Details.RawParameters, &params); err != nil {
			return nil, err
		}
	}

	_, err := s.Client.UpdateStackWithContext(ctx, &cloudformation.UpdateStackInput{
		Capabilities:        capabilities,
		StackName:           aws.String(s.getStackName(updateData.InstanceID)),
		Parameters:          params.UpdateParams(),
		UsePreviousTemplate: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return &domain.UpdateServiceSpec{
		OperationData: UpdateOperation,
		IsAsync:       true,
	}, nil
}

func (s *Provider) LastOperation(ctx context.Context, lastOperationData provideriface.LastOperationData) (*domain.LastOperation, error) {
	stackName := s.getStackName(lastOperationData.InstanceID)
	stack, err := s.getStack(ctx, stackName)
	if err == ErrStackNotFound {
		if lastOperationData.PollDetails.OperationData == DeprovisionOperation {
			return &domain.LastOperation{
				State:       domain.Succeeded,
				Description: "done",
			}, nil
		}
		return &domain.LastOperation{
			State:       domain.Failed,
			Description: "failed: cloudformation stack does not exist",
		}, nil
	} else if err != nil {
		// failed to get stack status
		return nil, err
	}

	switch *stack.StackStatus {
	case cloudformation.StackStatusDeleteFailed, cloudformation.StackStatusCreateFailed, cloudformation.StackStatusRollbackFailed, cloudformation.StackStatusUpdateRollbackFailed, cloudformation.StackStatusRollbackComplete, cloudformation.StackStatusUpdateRollbackComplete:
		return &domain.LastOperation{
			State:       domain.Failed,
			Description: fmt.Sprintf("failed: %s", *stack.StackStatus),
		}, nil
	case cloudformation.StackStatusCreateComplete, cloudformation.StackStatusUpdateComplete, cloudformation.StackStatusDeleteComplete:
		return &domain.LastOperation{
			State:       domain.Succeeded,
			Description: "done",
		}, nil
	default:
		return &domain.LastOperation{
			State:       domain.InProgress,
			Description: "pending",
		}, nil
	}
}

func (s *Provider) LastBindingOperation(ctx context.Context, lastBindingOperationData provideriface.LastBindingOperationData) (*domain.LastOperation, error) {
	stackName := s.getStackName(lastBindingOperationData.BindingID)
	stack, err := s.getStack(ctx, stackName)
	if err == ErrStackNotFound {
		if lastBindingOperationData.PollDetails.OperationData == UnbindOperation {
			return &domain.LastOperation{
				State:       domain.Succeeded,
				Description: "done",
			}, nil
		}
		return &domain.LastOperation{
			State:       domain.Failed,
			Description: "failed: cloudformation stack does not exist",
		}, nil
	} else if err != nil {
		// failed to get stack status
		return nil, err
	}

	switch *stack.StackStatus {
	case cloudformation.StackStatusDeleteFailed, cloudformation.StackStatusCreateFailed, cloudformation.StackStatusRollbackFailed, cloudformation.StackStatusUpdateRollbackFailed, cloudformation.StackStatusRollbackComplete, cloudformation.StackStatusUpdateRollbackComplete:
		return &domain.LastOperation{
			State:       domain.Failed,
			Description: fmt.Sprintf("failed: %s", *stack.StackStatus),
		}, nil
	case cloudformation.StackStatusCreateComplete, cloudformation.StackStatusUpdateComplete, cloudformation.StackStatusDeleteComplete:
		return &domain.LastOperation{
			State:       domain.Succeeded,
			Description: "ready",
		}, nil
	default:
		return &domain.LastOperation{
			State:       domain.InProgress,
			Description: "pending",
		}, nil
	}
}

func (s *Provider) GetBinding(ctx context.Context, getBindingData provideriface.GetBindData) (*domain.GetBindingSpec, error) {
	userStackName := s.getStackName(getBindingData.BindingID)
	userStack, err := s.getStack(ctx, userStackName)
	if err == ErrStackNotFound {
		return nil, ErrStackNotFound
	} else if err != nil {
		// failed to get stack status
		return nil, err
	}

	credentialsARN := getStackOutput(userStack, OutputCredentialsARN)
	res, err := s.Client.GetSecretValueWithContext(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(credentialsARN),
	})
	if err != nil {
		return nil, err
	} else if res.SecretString == nil {
		return nil, fmt.Errorf("invalid response from secrets manager")
	}

	var creds interface{}
	if err := json.Unmarshal([]byte(*res.SecretString), &creds); err != nil {
		return nil, err
	}

	return &domain.GetBindingSpec{
		Credentials: creds,
	}, nil
}

func (s *Provider) getStack(ctx context.Context, stackName string) (*cloudformation.Stack, error) {
	describeOutput, err := s.Client.DescribeStacksWithContext(ctx, &cloudformation.DescribeStacksInput{
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

func (s *Provider) getStackName(instanceID string) string {
	return fmt.Sprintf("%s-%s", s.ResourcePrefix, instanceID)
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

func getStackOutput(stack *cloudformation.Stack, key string) string {
	for _, item := range stack.Outputs {
		if item.OutputKey == nil || item.OutputValue == nil {
			continue
		}
		if *item.OutputKey == key {
			return *item.OutputValue
		}
	}
	return ""
}
