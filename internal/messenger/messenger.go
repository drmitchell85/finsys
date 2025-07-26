package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/drmitchell85/finsys/internal/config"
	"github.com/drmitchell85/finsys/internal/models"
	"github.com/drmitchell85/finsys/internal/utils"
	"github.com/google/uuid"
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

func (s *QueueService) EnqueueMessage(ctx context.Context, msgType string, payload any, idempKey string) (string, error) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return "", utils.NewInternalError(fmt.Errorf("error marshaling payload: %w", err))
	}

	msg := models.Message{
		Type:      msgType,
		Payload:   payloadData,
		Attempts:  1,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return "", utils.NewInternalError(fmt.Errorf("error marshaling message: %w", err))
	}

	// Determine which queue to use based on message type
	queueURL, err := s.getQueueURLForType(msgType)
	if err != nil {
		return "", utils.NewInternalError(fmt.Errorf("error determining queue url: %w", err))
	}

	res, err := s.client.SendMessageWithContext(ctx, &sqs.SendMessageInput{
		QueueUrl:               aws.String(queueURL),
		MessageBody:            aws.String(string(data)),
		MessageDeduplicationId: aws.String(idempKey), // for FIFO queues
		MessageGroupId:         aws.String(msgType),  // for FIFO queues
	})
	if err != nil {
		return "", utils.WrapError(err, utils.ErrInternal, "error sending message")
	}

	return *res.MessageId, nil
}

func (s *QueueService) getQueueURLForType(msgType string) (string, error) {
	switch msgType {
	case "transaction":
		return s.transactionQueueURL, nil
	case "notification":
		return s.notificationQueueURL, nil
	case "transactiondlq":
		return s.transactionDLQURL, nil
	case "notificationdlq":
		return s.notificationDLQURL, nil
	default:
		return "", utils.NewInternalError(fmt.Errorf("missing message type"))
	}
}

// Helper methods for common message types
func (s *QueueService) EnqueueTransaction(ctx context.Context, txID uuid.UUID, idempKey string, operation string) (string, error) {
	payload := models.TransactionPayload{
		TransactionID:  txID,
		IdempotencyKey: idempKey,
		Operation:      operation,
	}

	return s.EnqueueMessage(ctx, "transaction", payload, idempKey)
}

func (s *QueueService) EnqueueNotification(ctx context.Context, userID uuid.UUID, templateID string, destination string, data any) (string, error) {
	payload := models.NotificationPayload{
		UserID:      userID,
		TemplateID:  templateID,
		Destination: destination,
		Data:        data,
	}

	idempKey := fmt.Sprintf("notify:%s:%s:%s", userID, templateID, destination)
	return s.EnqueueMessage(ctx, "notification", payload, idempKey)
}

func (s *QueueService) ReceiveTransactions() ([]*sqs.Message, error) {
	res, err := s.client.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(s.transactionQueueURL),
		AttributeNames:      aws.StringSlice([]string{"All"}),
		MaxNumberOfMessages: aws.Int64(int64(s.maxNumberOfMessages)),
		WaitTimeSeconds:     aws.Int64(int64(s.waitTimeSeconds)),
	})
	if err != nil {
		return nil, utils.NewInternalError(fmt.Errorf("error receiving messages: %w", err))
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
		return nil, utils.NewInternalError(fmt.Errorf("error receiving messages: %w", err))
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
		return utils.NewInternalError(fmt.Errorf("unknown queue type: %s", queueType))
	}

	_, err := s.client.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})

	if err != nil {
		return utils.NewInternalError(fmt.Errorf("failed to delete message: %w", err))
	}

	return nil
}
