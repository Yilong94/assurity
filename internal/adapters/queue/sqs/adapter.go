package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

// Adapter implements ports.JobQueue using AWS SQS.
type Adapter struct {
	client     *sqs.Client
	queueURL   string
	visTimeout int32
	waitTime   int32
}

var _ ports.JobQueue = (*Adapter)(nil)

// New builds an SQS adapter from environment variables.
func New(ctx context.Context) (*Adapter, error) {
	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		return nil, fmt.Errorf("SQS_QUEUE_URL is required")
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "us-east-1"
	}
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		endpoint = os.Getenv("SQS_ENDPOINT")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	sqsClient := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	vis := int32(60)
	if v := os.Getenv("SQS_VISIBILITY_TIMEOUT"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("SQS_VISIBILITY_TIMEOUT: %w", err)
		}
		vis = int32(n)
	}

	wait := int32(20)
	if v := os.Getenv("SQS_WAIT_TIME_SECONDS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("SQS_WAIT_TIME_SECONDS: %w", err)
		}
		wait = int32(n)
	}

	return &Adapter{
		client:     sqsClient,
		queueURL:   queueURL,
		visTimeout: vis,
		waitTime:   wait,
	}, nil
}

// Close implements ports.JobQueue.
func (a *Adapter) Close() error { return nil }

// Enqueue implements ports.JobQueue.
func (a *Adapter) Send(ctx context.Context, job domain.ProbeJob) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = a.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(a.queueURL),
		MessageBody: aws.String(string(b)),
	})
	return err
}

// Receive implements ports.JobQueue.
func (a *Adapter) Receive(ctx context.Context) (domain.ReceivedProbeJob, error) {
	for {
		select {
		case <-ctx.Done():
			return domain.ReceivedProbeJob{}, ctx.Err()
		default:
		}

		out, err := a.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(a.queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     a.waitTime,
			VisibilityTimeout:   a.visTimeout,
		})
		if err != nil {
			return domain.ReceivedProbeJob{}, err
		}

		if len(out.Messages) == 0 {
			continue
		}

		m := out.Messages[0]
		if m.Body == nil || m.ReceiptHandle == nil {
			continue
		}

		var job domain.ProbeJob
		if err := json.Unmarshal([]byte(*m.Body), &job); err != nil {
			return domain.ReceivedProbeJob{}, fmt.Errorf("decode job: %w", err)
		}

		return domain.ReceivedProbeJob{Job: job, ReceiptHandle: *m.ReceiptHandle}, nil
	}
}

// Delete implements ports.JobQueue.
func (a *Adapter) Delete(ctx context.Context, receiptHandle string) error {
	_, err := a.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(a.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	return err
}
