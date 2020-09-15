package sqs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	goformation "github.com/awslabs/goformation/v4/cloudformation"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
)

const (
	ParamDelaySeconds                  = "DelaySeconds"
	ParamMaximumMessageSize            = "MaximumMessageSize"
	ParamMessageRetentionPeriod        = "MessageRetentionPeriod"
	ParamReceiveMessageWaitTimeSeconds = "ReceiveMessageWaitTimeSeconds"
	ParamRedriveMaxReceiveCount        = "RedriveMaxReceiveCount"
	ParamVisibilityTimeout             = "VisibilityTimeout"
)

const (
	ConditionShouldNotUseDLQ = "ShouldNotUseDLQ"
)

const (
	ResourcePrimaryQueue   = "PrimaryQueue"
	ResourceSecondaryQueue = "SecondaryQueue"
)

const (
	OutputPrimaryQueueURL   = "PrimaryQueueURL"
	OutputPrimaryQueueARN   = "PrimaryQueueARN"
	OutputSecondaryQueueURL = "SecondaryQueueURL"
	OutputSecondaryQueueARN = "SecondaryQueueARN"
)

// TemplateParams is the set of actual CloudFormation template
// parameters that can be passed to the stack.  If it comes from user
// configuration such as:
//     cf create-service foo -c '{"my-config": "bar"}'`
// then it should be in TemplateParams so that CloudFormation can keep
// track of its value across updates.
type TemplateParams struct {
	// ContentBasedDeduplication For first-in-first-out (FIFO) queues,
	// specifies whether to enable content-based deduplication. During the
	// deduplication interval, Amazon SQS treats messages that are sent with
	// identical content as duplicates and delivers only one copy of the
	// message.  (FIXME unimplemented)
	ContentBasedDeduplication *bool `json:"content_based_deduplication,omitempty"`
	// DelaySeconds The time in seconds for which the delivery of all messages
	// in the queue is delayed. You can specify an integer value of 0 to 900
	// (15 minutes).
	DelaySeconds *int `json:"delay_seconds,omitempty"`
	// MaximumMessageSize is the limit of how many bytes that a message can
	// contain before Amazon SQS rejects it. You can specify an integer value
	// from 1,024 bytes (1 KiB) to 262,144 bytes (256 KiB). The default value
	// is 262,144 (256 KiB).
	MaximumMessageSize *int `json:"maximum_message_size,omitempty"`
	// MessageRetentionPeriod The number of seconds
	// that Amazon SQS retains a message. You can
	// specify an integer value from 60 seconds (1
	// minute) to 1,209,600 seconds (14 days). The
	// default value is 345,600 seconds (4 days).
	MessageRetentionPeriod *int `json:"message_retention_period,omitempty"`
	// ReceiveMessageWaitTimeSeconds Specifies the
	// duration, in seconds, that the ReceiveMessage
	// action call waits until a message is in the
	// queue in order to include it in the response,
	// rather than returning an empty response if a
	// message isn't yet available. You can specify an
	// integer from 1 to 20. Short polling is used as
	// the default or when you specify 0 for this
	// property.
	ReceiveMessageWaitTimeSeconds *int `json:"receive_message_wait_time_seconds,omitempty"`
	// RedriveMaxReceiveCount  The number of times a
	// message is delivered to the source queue before
	// being moved to the dead-letter queue.  (FIXME unimplemented)
	RedriveMaxReceiveCount *int `json:"redrive_max_receive_count,omitempty"`
	// VisibilityTimeout The length of time during
	// which a message will be unavailable after a
	// message is delivered from the queue. This blocks
	// other components from receiving the same message
	// and gives the initial component time to process
	// and delete the message from the queue.
	// Values must be from 0 to 43,200 seconds (12 hours). If you don't specify a value, AWS CloudFormation uses the default value of 30 seconds.
	VisibilityTimeout *int `json:"visibility_timeout,omitempty"`
}

// GetStackTemplate returns a cloudformation Template for provisioning an SQS queue
func QueueTemplate(queueName string, isFIFO bool, tags map[string]string) (*goformation.Template, error) {
	template := goformation.NewTemplate()

	templateTags := []goformationtags.Tag{}
	for k, v := range tags {
		templateTags = append(templateTags, goformationtags.Tag{
			Key:   k,
			Value: v,
		})
	}

	template.Parameters = map[string]goformation.Parameter{
		ParamDelaySeconds: {
			Description: `The time in seconds for which the delivery of all messages in the queue is delayed. You can specify an integer value of 0 to 900 (15 minutes).`,
			Type:        "Number",
			Default:     0,
			MinValue:    0,
			MaxValue:    900,
		},
		ParamMaximumMessageSize: {
			Description: `The limit of how many bytes that a message can contain before Amazon SQS rejects it. You can specify an integer value from 1,024 bytes (1 KiB) to 262,144 bytes (256 KiB). The default value is 262,144 (256 KiB).`,
			Type:        "Number",
			Default:     262144,
			MinValue:    1024,
			MaxValue:    262144,
		},
		ParamMessageRetentionPeriod: {
			Description: `The number of seconds that Amazon SQS retains a message. You can specify an integer value from 60 seconds (1 minute) to 1,209,600 seconds (14 days). The default value is 345,600 seconds (4 days).`,
			Type:        "Number",
			Default:     345600,
			MinValue:    60,
			MaxValue:    1209600,
		},
		ParamReceiveMessageWaitTimeSeconds: {
			Description: `Specifies the duration, in seconds, that the ReceiveMessage action call waits until a message is in the queue in order to include it in the response, rather than returning an empty response if a message isn't yet available. You can specify an integer from 1 to 20. Short polling is used as the default or when you specify 0 for this property.`,
			Type:        "Number",
			Default:     0,
			MinValue:    0,
			MaxValue:    20,
		},
		ParamRedriveMaxReceiveCount: {
			Description: `The number of times a message is delivered to the source queue before being moved to the dead-letter queue.  A value of 0 disables the dead-letter queue.`,
			Type:        "Number",
			Default:     0,
			MinValue:    0,
		},
		ParamVisibilityTimeout: {
			Description: `The length of time during which a message will be unavailable after a message is delivered from the queue. This blocks other components from receiving the same message and gives the initial component time to process and delete the message from the queue.  Values must be from 0 to 43,200 seconds (12 hours). If you don't specify a value, AWS CloudFormation uses the default value of 30 seconds.`,
			Type:        "Number",
			Default:     30,
			MinValue:    0,
			MaxValue:    43200,
		},
	}

	// This is broken.  It uses the wrong function name Fn::Equal
	// (should be Fn::Equals).  If I use the right function name, the
	// whole thing gets replaced with `false` when the template is
	// rendered.  wtf
	template.Conditions = map[string]interface{}{ConditionShouldNotUseDLQ: json.RawMessage(
		fmt.Sprintf(`{"Fn::Equal":["%s",0]}`,
			goformation.Ref(ParamRedriveMaxReceiveCount)))}

	template.Resources[ResourcePrimaryQueue] = &Queue{
		QueueName: fmt.Sprintf("%s-pri", queueName),
		Tags: append(templateTags, goformationtags.Tag{
			Key:   "QueueType",
			Value: "Primary",
		}),
		// ContentBasedDeduplication:     params.ContentBasedDeduplication,
		DelaySeconds:                  goformation.Ref(ParamDelaySeconds),
		FifoQueue:                     isFIFO,
		MaximumMessageSize:            goformation.Ref(ParamMaximumMessageSize),
		MessageRetentionPeriod:        goformation.Ref(ParamMessageRetentionPeriod),
		ReceiveMessageWaitTimeSeconds: goformation.Ref(ParamReceiveMessageWaitTimeSeconds),
		RedrivePolicy: goformation.If(
			ConditionShouldNotUseDLQ,
			"AWS::NoValue",
			// goformation.Sub doesn't support the two-parameter
			// version so we have to reimplement it inline here,
			// including the weird base64 encoding thing
			base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`
              {"Fn::Sub":
                [%q,
                {
                  "DeadLetterTargetArn": %q,
                  "MaxReceiveCount": %q
                }]}`,
				`{"deadLetterTargetArn":"${DeadLetterTargetArn}", "maxReceiveCount": "${MaxReceiveCount}"}`,
				goformation.GetAtt(ResourceSecondaryQueue, "Arn"),
				goformation.Ref(ParamRedriveMaxReceiveCount),
			)))),
		VisibilityTimeout: goformation.Ref(ParamVisibilityTimeout),
	}

	template.Resources[ResourceSecondaryQueue] = &Queue{
		QueueName: fmt.Sprintf("%s-sec", queueName),
		Tags: append(templateTags, goformationtags.Tag{
			Key:   "QueueType",
			Value: "Secondary",
		}),
		FifoQueue:              isFIFO,
		MessageRetentionPeriod: goformation.Ref(ParamMessageRetentionPeriod),
		// ContentBasedDeduplication: params.ContentBasedDeduplication,
		VisibilityTimeout: goformation.Ref(ParamVisibilityTimeout),
	}

	template.Outputs[OutputPrimaryQueueURL] = goformation.Output{
		Description: "Primary queue URL",
		Value:       goformation.Ref(ResourcePrimaryQueue),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", queueName, OutputPrimaryQueueURL),
		},
	}

	template.Outputs[OutputPrimaryQueueARN] = goformation.Output{
		Description: "Primary queue ARN",
		Value:       goformation.GetAtt(ResourcePrimaryQueue, "Arn"),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", queueName, OutputPrimaryQueueARN),
		},
	}

	template.Outputs[OutputSecondaryQueueURL] = goformation.Output{
		Description: "Secondary queue URL",
		Value:       goformation.Ref(ResourceSecondaryQueue),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", queueName, OutputSecondaryQueueURL),
		},
	}

	template.Outputs[OutputSecondaryQueueARN] = goformation.Output{
		Description: "Secondary queue ARN",
		Value:       goformation.GetAtt(ResourceSecondaryQueue, "Arn"),
		Export: goformation.Export{
			Name: fmt.Sprintf("%s-%s", queueName, OutputSecondaryQueueARN),
		},
	}

	return template, nil
}
