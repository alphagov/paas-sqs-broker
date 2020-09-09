package sqs

import (
	"fmt"

	goformation "github.com/awslabs/goformation/v4/cloudformation"
	goformationiam "github.com/awslabs/goformation/v4/cloudformation/iam"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
)

const (
	ResourceUser      = "IAMUser"
	ResourceAccessKey = "IAMAccessKey"
	ResourcePolicy    = "IAMPolicy"
)

const (
	OutputAccessKeyID     = "IAMAccessKeyID"
	OutputSecretAccessKey = "IAMSecretAccessKey"
)

type UserParams struct {
	UserName            string            `json:"-"`
	UserPath            string            `json:"-"`
	QueueARN            string            `json:"queueARN,omitempty"`
	DLQueueARN          string            `json:"dlqueueARN,omitempty"`
	Tags                map[string]string `json:"tags,omitempty"`
	PermissionsBoundary string            `json:"-"`
}

func UserTemplate(params UserParams) (*goformation.Template, error) {
	template := goformation.NewTemplate()

	tags := []goformationtags.Tag{}
	for k, v := range params.Tags {
		tags = append(tags, goformationtags.Tag{
			Key:   k,
			Value: v,
		})
	}

	policy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect: "Allow",
				Action: []string{
					"sqs:ChangeMessageVisibility",
					"sqs:DeleteMessage",
					"sqs:GetQueueAttributes",
					"sqs:GetQueueUrl",
					"sqs:ListDeadLetterSourceQueues",
					"sqs:ListQueueTags",
					"sqs:PurgeQueue",
					"sqs:ReceiveMessage",
					"sqs:SendMessage",
				},
				Resource: []string{
					params.QueueARN,
					params.DLQueueARN,
				},
			},
		},
	}

	template.Resources[ResourceUser] = &goformationiam.User{
		UserName:            params.UserName,
		Path:                params.UserPath,
		Tags:                tags,
		PermissionsBoundary: params.PermissionsBoundary,
	}

	template.Resources[ResourceAccessKey] = &goformationiam.AccessKey{
		Serial:   1,
		Status:   "Active",
		UserName: goformation.Ref(ResourceUser),
	}

	template.Resources[ResourcePolicy] = &goformationiam.Policy{
		PolicyName:     params.UserName,
		PolicyDocument: policy,
		Users: []string{
			goformation.Ref(ResourceUser),
		},
	}

	template.Outputs[OutputAccessKeyID] = goformation.Output{
		Description: "Access Key ID",
		Value:       goformation.Ref(ResourceAccessKey),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", params.UserName, OutputAccessKeyID),
		},
	}

	template.Outputs[OutputSecretAccessKey] = goformation.Output{ // TODO: do we need to do the whole secrets manager thing here?
		Description: "Secret Access Key",
		Value:       goformation.GetAtt(ResourceAccessKey, "SecretAccessKey"),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", params.UserName, OutputSecretAccessKey),
		},
	}
	return template, nil
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
