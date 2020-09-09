package sqs

import (
	"fmt"

	goformation "github.com/awslabs/goformation/v4/cloudformation"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
)

const (
	ResourceMainQueue       = "MainQueue"
	ResourceDeadletterQueue = "DeadletterQueue"
)

const (
	OutputMainQueueURL       = "MainQueueURL"
	OutputMainQueueARN       = "MainQueueARN"
	OutputDeadletterQueueURL = "DeadletterQueueURL"
	OutputDeadletterQueueARN = "DeadletterQueueARN"
	OutputRegion             = "Region"
)

type QueueParams struct {
	// QueueName is the name of the aws queue resource
	QueueName string `json:"-"`
	// ContentBasedDeduplication For first-in-first-out (FIFO) queues,
	// specifies whether to enable content-based deduplication. During the
	// deduplication interval, Amazon SQS treats messages that are sent with
	// identical content as duplicates and delivers only one copy of the
	// message.
	ContentBasedDeduplication bool `json:"content_based_deduplication,omitempty"`
	// DelaySeconds The time in seconds for which the delivery of all messages
	// in the queue is delayed. You can specify an integer value of 0 to 900
	// (15 minutes).
	DelaySeconds int `json:"delay_seconds,omitempty"`
	// FifoQueue If set to true, creates a FIFO queue. If you don't specify
	// this property, Amazon SQS creates a standard queue.
	FifoQueue bool `json:"-"`
	// MaximumMessageSize is the limit of how many bytes that a message can
	// contain before Amazon SQS rejects it. You can specify an integer value
	// from 1,024 bytes (1 KiB) to 262,144 bytes (256 KiB). The default value
	// is 262,144 (256 KiB).
	MaximumMessageSize int `json:"maximum_message_size,omitempty"`
	// MessageRetentionPeriod The number of seconds
	// that Amazon SQS retains a message. You can
	// specify an integer value from 60 seconds (1
	// minute) to 1,209,600 seconds (14 days). The
	// default value is 345,600 seconds (4 days).
	MessageRetentionPeriod int `json:"message_retention_period,omitempty"`
	// ReceiveMessageWaitTimeSeconds Specifies the
	// duration, in seconds, that the ReceiveMessage
	// action call waits until a message is in the
	// queue in order to include it in the response,
	// rather than returning an empty response if a
	// message isn't yet available. You can specify an
	// integer from 1 to 20. Short polling is used as
	// the default or when you specify 0 for this
	// property.
	ReceiveMessageWaitTimeSeconds int `json:"receive_message_wait_time_seconds,omitempty"`
	// RedriveMaxReceiveCount  The number of times a
	// message is delivered to the source queue before
	// being moved to the dead-letter queue.
	RedriveMaxReceiveCount int `json:"redrive_max_receive_count,omitempty"`
	// VisibilityTimeout The length of time during
	// which a message will be unavailable after a
	// message is delivered from the queue. This blocks
	// other components from receiving the same message
	// and gives the initial component time to process
	// and delete the message from the queue.
	// Values must be from 0 to 43,200 seconds (12 hours). If you don't specify a value, AWS CloudFormation uses the default value of 30 seconds.
	VisibilityTimeout int `json:"visibility_timeout,omitempty"`
	// Tags are the AWS resource tags to attach to this queue.
	Tags map[string]string `json:"-"`
}

// GetStackTemplate returns a cloudformation Template for provisioning an SQS queue
func QueueTemplate(params QueueParams) (*goformation.Template, error) {
	template := goformation.NewTemplate()

	tags := []goformationtags.Tag{}
	for k, v := range params.Tags {
		tags = append(tags, goformationtags.Tag{
			Key:   k,
			Value: v,
		})
	}

	var redrivePolicy interface{}
	if params.RedriveMaxReceiveCount > 0 {
		redrivePolicy = map[string]interface{}{
			"deadLetterTargetArn": goformation.GetAtt(ResourceDeadletterQueue, "Arn"),
			"maxReceiveCount":     params.RedriveMaxReceiveCount,
		}
	} else {
		redrivePolicy = ""
	}

	template.Resources[ResourceMainQueue] = &goformationsqs.Queue{
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
	template.Resources[ResourceDeadletterQueue] = &goformationsqs.Queue{
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

	template.Outputs[OutputMainQueueURL] = goformation.Output{
		Description: "Main queue URL",
		Value:       goformation.Ref(ResourceMainQueue),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", params.QueueName, OutputMainQueueURL),
		},
	}

	template.Outputs[OutputMainQueueARN] = goformation.Output{
		Description: "Main queue ARN",
		Value:       goformation.GetAtt(ResourceMainQueue, "Arn"),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", params.QueueName, OutputMainQueueARN),
		},
	}

	template.Outputs[OutputDeadletterQueueURL] = goformation.Output{
		Description: "Deadletter queue URL",
		Value:       goformation.Ref(ResourceDeadletterQueue),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", params.QueueName, OutputDeadletterQueueURL),
		},
	}

	template.Outputs[OutputDeadletterQueueARN] = goformation.Output{
		Description: "Deadletter queue ARN",
		Value:       goformation.GetAtt(ResourceDeadletterQueue, "Arn"),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", params.QueueName, OutputDeadletterQueueARN),
		},
	}

	template.Outputs[OutputRegion] = goformation.Output{
		Description: "Region",
		Value:       goformation.Ref("AWS::Region"),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", params.QueueName, OutputRegion),
		},
	}

	return template, nil
}
