package sqs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
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
	AccessPolicyFull     AccessPolicy = "full"
	AccessPolicyProducer AccessPolicy = "producer"
	AccessPolicyConsumer AccessPolicy = "consumer"
)

type UserTemplateBuilder struct {
	BindingID           string            `json:"-"`
	ResourcePrefix      string            `json:"-"`
	UserPath            string            `json:"-"`
	PrimaryQueueURL     string            `json:"-"`
	PrimaryQueueARN     string            `json:"-"`
	SecondaryQueueURL   string            `json:"-"`
	SecondaryQueueARN   string            `json:"-"`
	Tags                map[string]string `json:"-"`
	PermissionsBoundary string            `json:"-"`
	AccessPolicy        AccessPolicy      `json:"access_policy"`
	AccessPolicyActions []string
}

type Credentials struct {
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	AWSRegion          string `json:"aws_region"`
	PrimaryQueueURL    string `json:"primary_queue_url"`
	SecondaryQueueURL  string `json:"secondary_queue_url"`
}

func (builder UserTemplateBuilder) CredentialsJSON() (string, error) {
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
		PrimaryQueueURL:    builder.PrimaryQueueURL,
		SecondaryQueueURL:  builder.SecondaryQueueURL,
	}
	credentialsTemplate, err := json.Marshal(credentialsPlaceholders)
	if err != nil {
		return "", err
	}
	return string(credentialsTemplate), nil
}

func (builder UserTemplateBuilder) Build() (string, error) {
	if builder.AccessPolicy == "" {
		builder.AccessPolicy = "full"
	}
	var err error
	builder.AccessPolicyActions, err = builder.GetAccessPolicy()
	if err != nil {
		return "", err
	}
	t, err := template.New("user-template").Parse(userTemplateFormat)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)

	err = t.Execute(buf, builder)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (builder UserTemplateBuilder) GetAccessPolicy() ([]string, error) {
	switch builder.AccessPolicy {
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
			fmt.Errorf("unknown access policy %#v", builder.AccessPolicy),
			http.StatusBadRequest,
			"unknown-access-policy",
		)
	}
}
