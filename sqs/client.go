package sqs

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fakes/fake_sqs_client.go . Client
type Client interface {
	DescribeStacksWithContext(aws.Context, *cloudformation.DescribeStacksInput, ...request.Option) (*cloudformation.DescribeStacksOutput, error)
	CreateStackWithContext(aws.Context, *cloudformation.CreateStackInput, ...request.Option) (*cloudformation.CreateStackOutput, error)
	UpdateStackWithContext(aws.Context, *cloudformation.UpdateStackInput, ...request.Option) (*cloudformation.UpdateStackOutput, error)
	DeleteStackWithContext(aws.Context, *cloudformation.DeleteStackInput, ...request.Option) (*cloudformation.DeleteStackOutput, error)
	GetSecretValueWithContext(aws.Context, *secretsmanager.GetSecretValueInput, ...request.Option) (*secretsmanager.GetSecretValueOutput, error)
}

type Config struct {
	AWSRegion         string `json:"aws_region"`
	ResourcePrefix    string `json:"resource_prefix"`
	DeployEnvironment string `json:"deploy_env"`
	Timeout           time.Duration
	// AdditionalUserPolicy is optionally the ARN of an IAM Policy that
	// will be attached to each IAM User created by the broker.  The
	// intended use case is, for example, to restrict all access to be
	// from a particular VPC, source IP, or via a particular VPC
	// Endpoint.
	AdditionalUserPolicy string `json:"additional_user_policy"`
	PermissionsBoundary  string `json:"permissions_boundary"`
}

func NewConfig(configJSON []byte) (*Config, error) {
	config := &Config{}
	err := json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
