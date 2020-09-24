package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"fmt"
	"net"
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-service-broker-base/broker"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

var configFilePath string

func main() {
	flag.StringVar(&configFilePath, "config", "", "Location of the config file")
	flag.Parse()

	file, err := os.Open(configFilePath)
	if err != nil {
		log.Fatalf("Error opening config file %s: %s\n", configFilePath, err)
	}
	defer file.Close()

	config, err := broker.NewConfig(file)
	if err != nil {
		log.Fatalf("Error validating config file: %v\n", err)
	}

	err = json.Unmarshal(config.Provider, &config)
	if err != nil {
		log.Fatalf("Error parsing configuration: %v\n", err)
	}

	sqsClientConfig, err := sqs.NewConfig(config.Provider)
	if err != nil {
		log.Fatalf("Error parsing configuration: %v\n", err)
	}

	logger := lager.NewLogger("sqs-service-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, config.API.LagerLogLevel))

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(sqsClientConfig.AWSRegion),
	}))
	cfg := aws.NewConfig()
	cfg = cfg.WithRegion(sqsClientConfig.AWSRegion)

	sqsProvider := &sqs.Provider{
		Client: struct {
			*secretsmanager.SecretsManager
			*cloudformation.CloudFormation
		}{
			SecretsManager: secretsmanager.New(sess, cfg),
			CloudFormation: cloudformation.New(sess, cfg),
		},
		Environment:          sqsClientConfig.DeployEnvironment,
		ResourcePrefix:       sqsClientConfig.ResourcePrefix,
		AdditionalUserPolicy: sqsClientConfig.AdditionalUserPolicy,
		PermissionsBoundary:  sqsClientConfig.PermissionsBoundary,
		Timeout:              sqsClientConfig.Timeout,
		Logger:               logger,
	}

	serviceBroker, err := broker.New(config, sqsProvider, logger)
	if err != nil {
		log.Fatalf("Error creating service broker: %s", err)
	}

	brokerAPI := broker.NewAPI(serviceBroker, logger, config)

	listener, err := net.Listen("tcp", ":"+config.API.Port)
	if err != nil {
		log.Fatalf("Error listening to port %s: %s", config.API.Port, err)
	}
	fmt.Println("SQS Service Broker started on port " + config.API.Port + "...")
	if err := http.Serve(listener, brokerAPI); err != nil {
		log.Fatalf("Error opening config file %s: %s\n", configFilePath, err)
	}
}
