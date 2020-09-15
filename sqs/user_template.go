package sqs

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pivotal-cf/brokerapi/domain/apiresponses"

	goformation "github.com/awslabs/goformation/v4/cloudformation"
	goformationiam "github.com/awslabs/goformation/v4/cloudformation/iam"
	goformationsecretsmanager "github.com/awslabs/goformation/v4/cloudformation/secretsmanager"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
)

const (
	ResourceUser        = "IAMUser"
	ResourceAccessKey   = "IAMAccessKey"
	ResourcePolicy      = "IAMPolicy"
	ResourceCredentials = "BindingCredentials"
)

const (
	OutputCredentialsARN = "CredentialsARN"
)

type AccessPolicy = string

const (
    AccessPolicyFull AccessPolicy = "full"
    AccessPolicyProducer AccessPolicy = "producer"
    AccessPolicyConsumer AccessPolicy = "consumer"
)


type UserParams struct {
	BindingID           string            `json:"-"`
	ResourcePrefix      string            `json:"-"`
	UserPath            string            `json:"-"`
	PrimaryQueueURL     string            `json:"-"`
	PrimaryQueueARN     string            `json:"-"`
	SecondaryQueueURL   string            `json:"-"`
	SecondaryQueueARN   string            `json:"-"`
	Tags                map[string]string `json:"-"`
	PermissionsBoundary string            `json:"-"`
	AccessPolicy        AccessPolicy      `json:"-"`
}

type Credentials struct {
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	AWSRegion          string `json:"aws_region"`
	PrimaryQueueURL    string `json:"primary_queue_url"`
	SecondaryQueueURL  string `json:"secondary_queue_url"`
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

	if params.AccessPolicy == "" {
		params.AccessPolicy = "full"
	}

	cannedPolicy, err := getCannedAccessPolicy(params.AccessPolicy)
	if err != nil {
		return nil, err
	}

	policy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect: "Allow",
				Action: cannedPolicy,
				Resource: []string{
					params.PrimaryQueueARN,
					params.SecondaryQueueARN,
				},
			},
		},
	}

	// this is a template representing the json credential for the binding.
	// the values get interpolated with values from cloudformation
	// once they are available.
	//
	// ${res} is equivilent to cloudformation.Ref("res")
	// ${res.arn} is equivilent to cloudformation.GetAtt("res", "arn")
	//
	credentialsPlaceholders := Credentials{
		AWSAccessKeyID:     fmt.Sprintf("${%s}", ResourceAccessKey),
		AWSSecretAccessKey: fmt.Sprintf("${%s.SecretAccessKey}", ResourceAccessKey),
		AWSRegion:          "${AWS::Region}",
		PrimaryQueueURL:    params.PrimaryQueueURL,
		SecondaryQueueURL:  params.SecondaryQueueURL,
	}
	credentialsTemplate, err := json.Marshal(credentialsPlaceholders)
	if err != nil {
		return nil, err
	}

	template.Resources[ResourceUser] = &goformationiam.User{
		UserName:            fmt.Sprintf("binding-%s", params.BindingID),
		Path:                fmt.Sprintf("/%s/", params.ResourcePrefix),
		Tags:                tags,
		PermissionsBoundary: params.PermissionsBoundary,
	}

	template.Resources[ResourceAccessKey] = &goformationiam.AccessKey{
		Serial:   1,
		Status:   "Active",
		UserName: goformation.Ref(ResourceUser),
	}

	template.Resources[ResourcePolicy] = &goformationiam.Policy{
		PolicyName:     fmt.Sprintf("%s-%s", params.ResourcePrefix, params.BindingID),
		PolicyDocument: policy,
		Users: []string{
			goformation.Ref(ResourceUser),
		},
	}

	template.Resources[ResourceCredentials] = &goformationsecretsmanager.Secret{
		Description:  "Binding credentials",
		Name:         fmt.Sprintf("%s-%s", params.ResourcePrefix, params.BindingID),
		SecretString: goformation.Sub(credentialsTemplate),
	}

	template.Outputs[OutputCredentialsARN] = goformation.Output{
		Description: "Path to the binding credentials",
		Value:       goformation.Ref(ResourceCredentials),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", params.BindingID, OutputCredentialsARN), // export should not be required, this is a goformation bug
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

func getCannedAccessPolicy(policyName string) ([]string, error) {
	switch policyName {
		case AccessPolicyFull:
			return []string{
				"sqs:ChangeMessageVisibility",
				"sqs:DeleteMessage",
				"sqs:GetQueueAttributes",
				"sqs:GetQueueUrl",
				"sqs:ListDeadLetterSourceQueues",
				"sqs:ListQueueTags",
				"sqs:PurgeQueue",
				"sqs:ReceiveMessage",
				"sqs:SendMessage",
			}, nil
		case AccessPolicyProducer:
			return []string{
				"sqs:GetQueueAttributes",
				"sqs:GetQueueUrl",
				"sqs:ListDeadLetterSourceQueues",
				"sqs:ListQueueTags",
				"sqs:SendMessage",
			}, nil
		case AccessPolicyConsumer:
			return []string{
				"sqs:DeleteMessage",
				"sqs:GetQueueAttributes",
				"sqs:GetQueueUrl",
				"sqs:ListDeadLetterSourceQueues",
				"sqs:ListQueueTags",
				"sqs:PurgeQueue",
				"sqs:ReceiveMessage",
			}, nil

		default:
			return nil, apiresponses.NewFailureResponse(
				fmt.Errorf("unknown access policy %#v", policyName),
				http.StatusBadRequest,
				"unknown-access-policy",
			)
	}
}
