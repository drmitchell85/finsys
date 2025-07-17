package messenger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/drmitchell85/finsys/internal/config"
)

type QueueService struct {
	client               *sqs.SQS
	transactionQueueURL  string
	transactionDLQURL    string
	notificationQueueURL string
	notificationDLQURL   string
	maxNumberOfMessages  int
	waitTimeSeconds      int
}

type Message struct {
	Id        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Attempts  int             `json:"attempts"`
	Timestamp int64           `json:"timestamp"`
}

func NewQueueService(config config.Config) *QueueService {
	client := initSQSClient(config)

	// construct queue URLs - format differs between localstack and aws
	prefix := ""
	if os.Getenv("LOCAL_DEV") == "true" {
		// localstack format: http://localhost:4566/000000000000/queue-name
		prefix = fmt.Sprintf("%s/000000000000/", config.AWS.Host)
	}

	return &QueueService{
		client:               client,
		transactionQueueURL:  prefix + config.SQS.TransactionQueue,
		transactionDLQURL:    prefix + config.SQS.TransactionDLQ,
		notificationQueueURL: prefix + config.SQS.NotificationQueue,
		notificationDLQURL:   prefix + config.SQS.NotificationDLQ,
		maxNumberOfMessages:  config.SQS.MaxNumberOfMessages,
		waitTimeSeconds:      config.SQS.WaitTimeSeconds,
	}
}

func initSQSClient(config config.Config) *sqs.SQS {
	var sess *session.Session

	if os.Getenv("LOCAL_DEV") == "true" {
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Endpoint:    aws.String(config.AWS.Host),
				Region:      aws.String(config.AWS.Region),
				DisableSSL:  aws.Bool(true),
				Credentials: credentials.NewStaticCredentials("test", "test", ""),
			},
		}))
	} else {
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
	}

	sqsClient := sqs.New(sess)

	log.Println("connected to sqs")

	return sqsClient
}

func (s *QueueService) GetClient() *sqs.SQS {
	return s.client
}

func (s *QueueService) EnqueueTransaction(data []byte) (string, error) {
	res, err := s.client.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &s.transactionQueueURL,
		MessageBody: aws.String(string(data)),
	})
	if err != nil {
		return "", fmt.Errorf("error sending message: %w", err)
	}

	return *res.MessageId, nil
}

func (s *QueueService) EnqueueNotification(data []byte) (string, error) {
	res, err := s.client.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &s.notificationQueueURL,
		MessageBody: aws.String(string(data)),
	})
	if err != nil {
		return "", fmt.Errorf("error sending message: %w", err)
	}

	return *res.MessageId, nil
}

func (s *QueueService) EnqueueToDeadLetter(messageType string, data []byte, reason string) error {
	var queueURL string

	switch messageType {
	case "transaction":
		queueURL = s.transactionDLQURL
	case "notification":
		queueURL = s.notificationDLQURL
	default:
		return fmt.Errorf("unknown message type: %s", messageType)
	}

	_, err := s.client.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(data)),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"FailureReason": {
				DataType:    aws.String("String"),
				StringValue: aws.String(reason),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send to DLQ: %w", err)
	}
	return nil
}

func (s *QueueService) ReceiveTransactions() ([]*sqs.Message, error) {
	res, err := s.client.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(s.transactionQueueURL),
		AttributeNames:      aws.StringSlice([]string{"All"}),
		MaxNumberOfMessages: aws.Int64(int64(s.maxNumberOfMessages)),
		WaitTimeSeconds:     aws.Int64(int64(s.waitTimeSeconds)),
	})
	if err != nil {
		return nil, fmt.Errorf("error receiving messages: %w", err)
	}

	return res.Messages, nil
}

func (s *QueueService) ReceiveNotifications() ([]*sqs.Message, error) {
	res, err := s.client.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(s.notificationQueueURL),
		AttributeNames:      aws.StringSlice([]string{"All"}),
		MaxNumberOfMessages: aws.Int64(int64(s.maxNumberOfMessages)),
		WaitTimeSeconds:     aws.Int64(int64(s.waitTimeSeconds)),
	})
	if err != nil {
		return nil, fmt.Errorf("error receiving messages: %w", err)
	}

	return res.Messages, nil
}

func (s *QueueService) DeleteMessage(queueType string, receiptHandle string) error {
	var queueURL string

	switch queueType {
	case "transaction":
		queueURL = s.transactionQueueURL
	case "notification":
		queueURL = s.notificationQueueURL
	default:
		return fmt.Errorf("unknown queue type: %s", queueType)
	}

	_, err := s.client.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})

	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}
