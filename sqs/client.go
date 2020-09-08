package sqs

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

//go:generate counterfeiter -o fakes/fake_sqs_client.go . Client
type Client interface {
	DescribeStacksWithContext(aws.Context, *cloudformation.DescribeStacksInput, ...request.Option) (*cloudformation.DescribeStacksOutput, error)
	// DescribeStackEventsWithContext(aws.Context, *cloudformation.DescribeStackEventsInput, ...request.Option) (*cloudformation.DescribeStackEventsOutput, error)
	CreateStackWithContext(aws.Context, *cloudformation.CreateStackInput, ...request.Option) (*cloudformation.CreateStackOutput, error)
	UpdateStackWithContext(aws.Context, *cloudformation.UpdateStackInput, ...request.Option) (*cloudformation.UpdateStackOutput, error)
	DeleteStackWithContext(aws.Context, *cloudformation.DeleteStackInput, ...request.Option) (*cloudformation.DeleteStackOutput, error)
}

type Config struct {
	AWSRegion           string `json:"aws_region"`
	ResourcePrefix      string `json:"resource_prefix"`
	DeployEnvironment   string `json:"deploy_env"`
	Timeout             time.Duration
	PermissionsBoundary string `json:"permissions_boundary"`
}

func NewConfig(configJSON []byte) (*Config, error) {
	config := &Config{}
	err := json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
