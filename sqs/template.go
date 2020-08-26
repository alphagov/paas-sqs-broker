package sqs

import (
	"fmt"

	goformation "github.com/awslabs/goformation/v4/cloudformation"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
)

const (
	SQSResourceName      = "SQSQueue"
	SQSDLQResourceName   = "SQSDLQueue"
	SQSOutputURL         = "QueueURL"
	SQSDLQOutputURL      = "DLQueueURL"
	SQSResourceIAMPolicy = "SQSSIAMPolicy"
	IAMRoleParameterName = "IAMRoleName"
)

type QueueParams struct {
	QueueName                     string `json:"queueName,omitempty"`
	ContentBasedDeduplication     bool   `json:"contentBasedDeduplication,omitempty"`
	DelaySeconds                  int    `json:"delaySeconds,omitempty"`
	FifoQueue                     bool   `json:"fifoQueue,omitempty"`
	MaximumMessageSize            int    `json:"maximumMessageSize,omitempty"`
	MessageRetentionPeriod        int    `json:"messageRetentionPeriod,omitempty"`
	ReceiveMessageWaitTimeSeconds int    `json:"receiveMessageWaitTimeSeconds,omitempty"`
	RedriveMaxReceiveCount        int    `json:"redriveMaxReceiveCount,omitempty"`
	VisibilityTimeout             int    `json:"visibilityTimeout,omitempty"`
}

// GetStackTemplate returns a cloudformation Template for provisioning an SQS queue
func QueueTemplate(params QueueParams) (*goformation.Template, error) {
	template := goformation.NewTemplate()

	template.Parameters[IAMRoleParameterName] = goformation.Parameter{
		Type: "String",
	}

	tags := []goformationtags.Tag{
		{
			Key:   "Service",
			Value: "sqs",
		},
		{
			Key:   "Name",
			Value: "????",
		},
		{
			Key:   "DeployEnv",
			Value: "????",
		},
	}

	var redrivePolicy interface{}
	if params.RedriveMaxReceiveCount > 0 {
		redrivePolicy = map[string]interface{}{
			"deadLetterTargetArn": goformation.GetAtt(SQSDLQResourceName, "Arn"),
			"maxReceiveCount":     params.RedriveMaxReceiveCount,
		}
	} else {
		redrivePolicy = ""
	}

	template.Resources[SQSResourceName] = &goformationsqs.Queue{
		QueueName: params.QueueName,
		Tags: append(tags, goformationtags.Tag{
			Key:   "QueueType",
			Value: "Main",
		}),
		ContentBasedDeduplication:     params.ContentBasedDeduplication,
		DelaySeconds:                  params.DelaySeconds,
		FifoQueue:                     params.FifoQueue,
		MaximumMessageSize:            params.MaximumMessageSize,
		MessageRetentionPeriod:        params.MessageRetentionPeriod,
		ReceiveMessageWaitTimeSeconds: params.ReceiveMessageWaitTimeSeconds,
		RedrivePolicy:                 redrivePolicy,
		VisibilityTimeout:             params.VisibilityTimeout,
	}

	dlQueueName := fmt.Sprintf("%s-dl", params.QueueName)
	template.Resources[SQSDLQResourceName] = &goformationsqs.Queue{
		QueueName: dlQueueName,
		Tags: append(tags, goformationtags.Tag{
			Key:   "QueueType",
			Value: "Dead-Letter",
		}),
		FifoQueue:                 params.FifoQueue,
		MessageRetentionPeriod:    params.MessageRetentionPeriod,
		ContentBasedDeduplication: params.ContentBasedDeduplication,
		VisibilityTimeout:         params.VisibilityTimeout,
	}

	template.Outputs[SQSOutputURL] = goformation.Output{
		Description: "SQSQueue URL",
		Value:       goformation.Ref(SQSResourceName),
	}

	template.Outputs[SQSDLQOutputURL] = goformation.Output{
		Description: "SQSQueue DLQ URL",
		Value:       goformation.Ref(SQSDLQResourceName),
	}

	return template, nil
}

func UserTemplate() (*goformation.Template, error) {
	// policy := PolicyDocument{
	// 	Version: "2012-10-17",
	// 	Statement: []PolicyStatement{
	// 		{
	// 			Effect: "Allow",
	// 			Action: []string{
	// 				"sqs:ChangeMessageVisibility",
	// 				"sqs:DeleteMessage",
	// 				"sqs:GetQueueAttributes",
	// 				"sqs:GetQueueUrl",
	// 				"sqs:ListDeadLetterSourceQueues",
	// 				"sqs:ListQueueTags",
	// 				"sqs:PurgeQueue",
	// 				"sqs:ReceiveMessage",
	// 				"sqs:SendMessage",
	// 			},
	// 			Resource: []string{
	// 				goformation.GetAtt(SQSResourceName, "Arn"),
	// 				goformation.GetAtt(SQSDLQResourceName, "Arn"),
	// 			},
	// 		},
	// 	},
	// }

	// template.Resources[SQSResourceIAMPolicy] = &goformationiam.Policy{
	// 	PolicyName:     goformation.Join("-", []string{
	// 		"sqs",
	// 		"access",
	// 		goformation.GetAtt(SQSResourceName, "QueueName"),
	// 	}),
	// 	PolicyDocument: policy,
	// 	Roles: []string{
	// 		goformation.Ref(IAMRoleParameterName),
	// 	},
	// }
	return nil, fmt.Errorf("not imp")
}

// helpers for building iam documents in cloudformation

type PolicyDocument struct {
	Version   string
	Statement []PolicyStatement
}

type PolicyStatement struct {
	Effect   string
	Action   []string
	Resource []string
}

type AssumeRolePolicyDocument struct {
	Version   string
	Statement []AssumeRolePolicyStatement
}

type AssumeRolePolicyStatement struct {
	Effect    string
	Principal PolicyPrincipal
	Action    []string
	Condition PolicyCondition `json:"Condition,omitempty"`
}

type PolicyPrincipal struct {
	AWS       []string `json:"AWS,omitempty"`
	Federated []string `json:"Federated,omitempty"`
}

type PolicyCondition struct {
	StringEquals map[string]string `json:"StringEquals,omitempty"`
}

func NewRolePolicyDocument(resources, actions []string) PolicyDocument {
	return PolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect:   "Allow",
				Action:   actions,
				Resource: resources,
			},
		},
	}
}
