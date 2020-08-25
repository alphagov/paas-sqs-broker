package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-s3-broker/s3/policy"
	"github.com/alphagov/paas-service-broker-base/provider"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

const (
	awsMaxWaitAttempts = 15
	awsWaitDelay       = 3 * time.Second
)

//go:generate counterfeiter -o fakes/fake_sqs_client.go . Client
type Client interface {
	CreateQueue(provisionData provider.ProvisionData) error
	DeleteQueue(name string) error
	AddUserToQueue(bindData provider.BindData) (QueueCredentials, error)
	RemoveUserFromQueueAndDeleteUser(bindingID, queueName string) error
}

type QueueCredentials struct {
	QueueName          string `json:"queue_name"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	AWSRegion          string `json:"aws_region"`
	DeployEnvironment  string `json:"deploy_env"`
}

type Config struct {
	AWSRegion              string `json:"aws_region"`
	ResourcePrefix         string `json:"resource_prefix"`
	IAMUserPath            string `json:"iam_user_path"`
	DeployEnvironment      string `json:"deploy_env"`
	IpRestrictionPolicyARN string `json:"iam_ip_restriction_policy_arn"`
	Timeout                time.Duration
}

func NewSQSClientConfig(configJSON []byte) (*Config, error) {
	config := &Config{}
	err := json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

type SQSClient struct {
	queuePrefix            string
	iamUserPath            string
	ipRestrictionPolicyArn string
	awsRegion              string
	deployEnvironment      string
	timeout                time.Duration
	sqsClient              sqsiface.SQSAPI
	iamClient              iamiface.IAMAPI
	logger                 lager.Logger
	context                context.Context
}

type BindParams struct {
}

type ProvisionParams struct {
}

func NewSQSClient(
	config *Config,
	sqsClient sqsiface.SQSAPI,
	iamClient iamiface.IAMAPI,
	logger lager.Logger,
	ctx context.Context,
) *SQSClient {
	timeout := config.Timeout
	if timeout == time.Duration(0) {
		timeout = 30 * time.Second
	}

	return &SQSClient{
		queuePrefix:            config.ResourcePrefix,
		iamUserPath:            fmt.Sprintf("/%s/", strings.Trim(config.IAMUserPath, "/")),
		ipRestrictionPolicyArn: config.IpRestrictionPolicyARN,
		awsRegion:              config.AWSRegion,
		deployEnvironment:      config.DeployEnvironment,
		timeout:                timeout,
		sqsClient:              sqsClient,
		iamClient:              iamClient,
		logger:                 logger,
		context:                ctx,
	}
}

func (s *SQSClient) CreateQueue(provisionData provider.ProvisionData) error {
	logger := s.logger.Session("create-queue")
	queueName := s.buildQueueName(provisionData.InstanceID)

	logger.Info("create-queue", lager.Data{"queue": queueName})
	_, err := s.sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})

	if err != nil {
		logger.Error("create-queue", err)
		return err
	}

	return err
}

func (s *S3Client) DeleteQueue(name string) error {
	logger := s.logger.Session("delete-queue")
	fullQueueName := s.buildQueueName(name)

	logger.Info("delete-queue", lager.Data{"queue": fullQueueName})
	_, err := s.sqsClient.DeleteQueue(&sqs.DeleteQueueInput{
		QueueName: aws.String(fullQueueName),
	})
	return err
}

func (s *S3Client) AddUserToQueue(bindData provider.BindData) (QueueCredentials, error) {
	logger := s.logger.Session("add-user-to-queue")
	var permissions policy.Permissions = policy.ReadWritePermissions{}

	bindParams := BindParams{
		AllowExternalAccess: false,
		Permissions:         policy.ReadWritePermissionsName, // Required, as if another bind parameter is set, `ValidatePermissions` is called below.
	}
	if bindData.Details.RawParameters != nil {
		logger.Info("parse-raw-params")
		err := json.Unmarshal(bindData.Details.RawParameters, &bindParams)
		if err != nil {
			logger.Error("parse-raw-params", err)
			return BucketCredentials{}, err
		}

		permissions, err = policy.ValidatePermissions(bindParams.Permissions)
		if err != nil {
			logger.Error("invalid-permissions", err)
			return BucketCredentials{}, err
		}
	}

	fullQueueName := s.buildQueueName(bindData.InstanceID)
	username := s.buildBindingUsername(bindData.BindingID)
	userTags := []*iam.Tag{
		{
			Key:   aws.String("service_instance_guid"),
			Value: aws.String(bindData.InstanceID),
		},
		{
			Key:   aws.String("created_by"),
			Value: aws.String("paas-sqs-broker"),
		},
		{
			Key:   aws.String("deploy_env"),
			Value: aws.String(s.deployEnvironment),
		},
	}

	user := &iam.CreateUserInput{
		Path:     aws.String(s.iamUserPath),
		UserName: aws.String(username),
		Tags:     userTags,
	}
	logger.Info("create-user", lager.Data{"bucket": fullBucketName, "user": user})
	createUserOutput, err := s.iamClient.CreateUser(user)
	if err != nil {
		logger.Error("create-user", err)
		return BucketCredentials{}, err
	}

	err = s.iamClient.WaitUntilUserExistsWithContext(
		s.context,
		&iam.GetUserInput{UserName: aws.String(username)},

		request.WithWaiterDelay(request.ConstantWaiterDelay(awsWaitDelay)),
		request.WithWaiterMaxAttempts(awsMaxWaitAttempts),
	)

	if err != nil {
		logger.Error("wait-for-user-exist", err)
		return BucketCredentials{}, err
	}

	if !bindParams.AllowExternalAccess {
		logger.Info("allow-external-access", lager.Data{"bucket": fullBucketName})
		_, err = s.iamClient.AttachUserPolicy(&iam.AttachUserPolicyInput{
			PolicyArn: aws.String(s.ipRestrictionPolicyArn),
			UserName:  aws.String(username),
		})
		if err != nil {
			logger.Error("allow-external-access", err)
			s.deleteUserWithoutError(username)
			return BucketCredentials{}, err
		}
	}

	logger.Info("create-access-key", lager.Data{"bucket": fullBucketName, "username": username})
	createAccessKeyOutput, err := s.iamClient.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(username),
	})
	if err != nil {
		logger.Error("create-access-key", err)
		s.deleteUserWithoutError(username)
		return BucketCredentials{}, err
	}

	return BucketCredentials{
		QueueName:          fullQueueName,
		AWSAccessKeyID:     *createAccessKeyOutput.AccessKey.AccessKeyId,
		AWSSecretAccessKey: *createAccessKeyOutput.AccessKey.SecretAccessKey,
		AWSRegion:          s.awsRegion,
	}, nil
}

func (s *S3Client) deleteUserWithoutError(username string) {
	err := s.deleteUser(username)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Deleted User %s, and suppressed error", username), err)
	}
}

func (s *S3Client) deleteUser(username string) error {
	keys, err := s.iamClient.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(username),
	})
	policies, err := s.iamClient.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return err
	}
	if keys != nil {
		for _, k := range keys.AccessKeyMetadata {
			_, err := s.iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				UserName:    aws.String(username),
				AccessKeyId: k.AccessKeyId,
			})
			if err != nil {
				return err
			}
		}
	}
	if policies != nil {
		for _, p := range policies.AttachedPolicies {
			_, err := s.iamClient.DetachUserPolicy(&iam.DetachUserPolicyInput{
				UserName:  aws.String(username),
				PolicyArn: p.PolicyArn,
			})
			if err != nil {
				return err
			}
		}
	}
	_, err = s.iamClient.DeleteUser(&iam.DeleteUserInput{
		UserName: aws.String(username),
	})
	return err
}

func (s *S3Client) buildBucketName(instanceID string) string {
	return fmt.Sprintf("%s%s", s.queuePrefix, instanceID)
}

func (s *S3Client) buildBindingUsername(bindingID string) string {
	return fmt.Sprintf("%s%s", s.queuePrefix, bindingID)
}

func (s *S3Client) RemoveUserFromQueueAndDeleteUser(bindingID, bucketName string) error {
	logger := s.logger.Session("remove-user-from-queue")

	username := s.buildBindingUsername(bindingID)
	fullQueueName := s.buildQueueName(bucketName)

	logger.Info("delete-user", lager.Data{"username": username})
	err = s.deleteUser(username)
	if err != nil {
		logger.Error("delete-user", err)
		return err
	}

	return nil
}
