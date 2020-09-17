package sqs

// queueTemplateFormat is a raw format string for generating a
// CloudFormation template for an SQS queue.  The names in here
// carefully match constants in queue_template.go, and there are
// format specifiers such as %[1]s that pull in numbered arguments to
// a fmt.Sprintf() call.
const queueTemplateFormat = `
AWSTemplateFormatVersion: 2010-09-09
Parameters:
  DelaySeconds:
    Default: 0
    Description: |
      The time in seconds for which the delivery of all messages in
      the queue is delayed. You can specify an integer value of 0 to
      900 (15 minutes).
    MaxValue: 900
    Type: Number
  MaximumMessageSize:
    Default: 262144
    Description: |
      The limit of how many bytes that a message can contain before
      Amazon SQS rejects it. You can specify an integer value from
      1,024 bytes (1 KiB) to 262,144 bytes (256 KiB). The default
      value is 262,144 (256 KiB).
    MaxValue: 262144
    MinValue: 1024
    Type: Number
  MessageRetentionPeriod:
    Default: 345600
    Description: |
      The number of seconds that Amazon SQS retains a message. You can
      specify an integer value from 60 seconds (1 minute) to 1,209,600
      seconds (14 days). The default value is 345,600 seconds (4
      days).
    MaxValue: 1209600
    MinValue: 60
    Type: Number
  ReceiveMessageWaitTimeSeconds:
    Default: 0
    Description: |
      Specifies the duration, in seconds, that the ReceiveMessage
      action call waits until a message is in the queue in order to
      include it in the response, rather than returning an empty
      response if a message isn't yet available. You can specify an
      integer from 1 to 20. Short polling is used as the default or
      when you specify 0 for this property.
    MaxValue: 20
    Type: Number
  RedriveMaxReceiveCount:
    Default: 0
    Description: |
      The number of times a message is delivered to the source queue
      before being moved to the dead-letter queue.  A value of 0
      disables the dead-letter queue.
    Type: Number
  VisibilityTimeout:
    Default: 30
    Description: |
      The length of time during which a message will be unavailable
      after a message is delivered from the queue. This blocks other
      components from receiving the same message and gives the initial
      component time to process and delete the message from the queue.
      Values must be from 0 to 43,200 seconds (12 hours).  If you
      don't specify a value, AWS CloudFormation uses the default value
      of 30 seconds.
    MaxValue: 43200
    Type: Number
Conditions:
  ShouldNotUseDLQ:
    Fn::Equals:
    - !Ref RedriveMaxReceiveCount
    - 0
Resources:
  PrimaryQueue:
    Properties:
      QueueName: %[1]s-pri
      Tags:
      - Key: QueueType
        Value: Primary
      - Key: Name
        Value: %[2]s
      - Key: Service
        Value: sqs
      - Key: ServiceID
        Value: %[3]s
      - Key: Environment
        Value: %[4]s
      DelaySeconds: !Ref DelaySeconds
      MaximumMessageSize: !Ref MaximumMessageSize
      MessageRetentionPeriod: !Ref MessageRetentionPeriod
      ReceiveMessageWaitTimeSeconds: !Ref ReceiveMessageWaitTimeSeconds
      RedrivePolicy: !If
        - ShouldNotUseDLQ
        - !Ref "AWS::NoValue"
        - deadLetterTargetArn:
            Fn::GetAtt:
            - SecondaryQueue
            - Arn
          maxReceiveCount: !Ref RedriveMaxReceiveCount
      VisibilityTimeout: !Ref VisibilityTimeout
    Type: AWS::SQS::Queue
  SecondaryQueue:
    Properties:
      QueueName: %[1]s-sec
      Tags:
      - Key: QueueType
        Value: Secondary
      - Key: Name
        Value: %[2]s
      - Key: Service
        Value: sqs
      - Key: ServiceID
        Value: %[3]s
      - Key: Environment
        Value: %[4]s
      MessageRetentionPeriod: !Ref MessageRetentionPeriod
      VisibilityTimeout: !Ref VisibilityTimeout
    Type: AWS::SQS::Queue
Outputs:
  PrimaryQueueARN:
    Description: Primary queue ARN
    Export:
      Name: %[1]s-PrimaryQueueARN
    Value:
      Fn::GetAtt:
      - PrimaryQueue
      - Arn
  PrimaryQueueURL:
    Description: Primary queue URL
    Export:
      Name: %[1]s-PrimaryQueueURL
    Value: !Ref PrimaryQueue
  SecondaryQueueARN:
    Description: Secondary queue ARN
    Export:
      Name: %[1]s-SecondaryQueueARN
    Value:
      Fn::GetAtt:
      - SecondaryQueue
      - Arn
  SecondaryQueueURL:
    Description: Secondary queue URL
    Export:
      Name: %[1]s-SecondaryQueueURL
    Value: !Ref SecondaryQueue
`
