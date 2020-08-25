package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"fmt"
	"net"
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-sqs-broker/provider"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/alphagov/paas-service-broker-base/broker"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
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

	sqsClientConfig, err := sqs.NewSQSClientConfig(config.Provider)
	if err != nil {
		log.Fatalf("Error parsing configuration: %v\n", err)
	}

	logger := lager.NewLogger("sqs-service-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, config.API.LagerLogLevel))

	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(sqsClientConfig.AWSRegion)}))
	sqsClient := sqs.NewSQSClient(sqsClientConfig, aws_sqs.New(sess), iam.New(sess), logger, context.Background())

	sqsProvider := provider.NewSQSProvider(sqsClient)
	if err != nil {
		log.Fatalf("Error creating SQS Provider: %v\n", err)
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
	http.Serve(listener, brokerAPI)
}
