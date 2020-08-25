package sqs

import (
	"encoding/json"
	"time"
)

//go:generate counterfeiter -o fakes/fake_sqs_client.go . Client
type Client interface{}

type Config struct {
	AWSRegion         string `json:"aws_region"`
	ResourcePrefix    string `json:"resource_prefix"`
	IAMUserPath       string `json:"iam_user_path"`
	DeployEnvironment string `json:"deploy_env"`
	Timeout           time.Duration
}

func NewSQSClientConfig(configJSON []byte) (*Config, error) {
	config := &Config{}
	err := json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

type SQSClient struct{}
