package sqs

import (
	"fmt"
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

// TemplateParams is the data that gets baked into the template
// itself, not passed in as a stack parameter at create/update time.
type TemplateParams struct {
	QueueName string
	IsFIFO    bool
	Tags      struct {
		Name        string
		ServiceID   string
		Environment string
	}
}

// StackParams is the set of actual CloudFormation template
// parameters that can be passed to the stack.  If it comes from user
// configuration such as:
//     cf create-service foo -c '{"my-config": "bar"}'`
// then it should be in StackParams so that CloudFormation can keep
// track of its value across updates.
type StackParams struct {
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
	// being moved to the dead-letter queue.
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
func QueueTemplate(templateParams TemplateParams) string {
	return fmt.Sprintf(queueTemplateFormat, templateParams.QueueName, templateParams.IsFIFO, templateParams.Tags.Name, templateParams.Tags.ServiceID, templateParams.Tags.Environment)
}
