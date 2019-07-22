package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var (
	ErrNoBody           = errors.New("no json body provided")
	ErrUnauthorized     = errors.New("not authorized")
	ErrNotFound         = errors.New("not found")
	dynamoFileTableName string
	region              string
	s3Bucket            string
	// TODO move to sessions table
	validAPIKeys = map[string]string{
		"c16f3b18-e43f-4216-b66e-62a26df8c683": "useridpaul",
		"88954e19-c9ef-4350-bca5-bc896885f249": "useriduploader",
	}
	validMu = &sync.Mutex{}
)

const (
	errTemplate     = `{"error":"%s"}`
	respPutTemplate = `{"upload_url":"%s","id":"%s"}`
	respGetTemplate = `{"url":"%s"}`
)

type Event struct {
	APIKey   string `json:"api_key"`
	Filepath string `json:"filepath"`
}

func validateAPIKey(apiKey string) (string, error) {
	// TODO validate the user api key.
	// This requires the user to login to an endpoint(using Amazon Incognito or some other RBAC)
	// and lookup the stored apiKey in a user session table with expiring keys.
	validMu.Lock()
	defer validMu.Unlock()
	if user, ok := validAPIKeys[apiKey]; ok {
		return user, nil
	}
	return "", ErrUnauthorized
}

func get(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	apiKey, ok := request.QueryStringParameters["api_key"]
	if !ok {
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "api_key parameter missing"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusBadRequest,
		}, nil
	}
	if _, err := validateAPIKey(apiKey); err != nil {
		if err == ErrUnauthorized {
			return events.APIGatewayProxyResponse{
				Body:       fmt.Sprintf(errTemplate, apiKey+" unauthorized "),
				Headers:    map[string]string{"Content-Type": "application/json"},
				StatusCode: http.StatusUnauthorized,
			}, nil
		}
		log.Println("ERRO:", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, apiKey+" unauthorized"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusUnauthorized,
		}, nil
	}

	id, ok := request.QueryStringParameters["id"]
	if !ok {
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "id parameter missing"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusBadRequest,
		}, nil
	}
	// lookup s3path in dynamo and redirect to presigned url
	f, err := getItem(id)
	if err != nil {
		if err == ErrNotFound {
			return events.APIGatewayProxyResponse{
				Body:       fmt.Sprintf(errTemplate, id+" not found"),
				Headers:    map[string]string{"Content-Type": "application/json"},
				StatusCode: http.StatusNotFound,
			}, nil
		}
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "error getting file"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	presign, err := presignedURL(region, s3Bucket, f.S3Path, http.MethodGet)
	if err != nil {
		log.Println("ERRO:", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "error getting presigned url"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		Body: "",
		Headers: map[string]string{
			"Location": presign,
		},
		StatusCode: http.StatusMovedPermanently,
	}, nil
}

func put(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if len(request.Body) < 1 {
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "api_key not provided"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusBadRequest,
		}, nil
	}
	event := &Event{}
	if err := json.Unmarshal([]byte(request.Body), &event); err != nil {
		log.Println("ERRO", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "invalid json"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusBadRequest,
		}, nil
	}
	username, err := validateAPIKey(event.APIKey)
	if err != nil {
		if err == ErrUnauthorized {
			return events.APIGatewayProxyResponse{
				Body:       fmt.Sprintf(errTemplate, "unauthorized"),
				Headers:    map[string]string{"Content-Type": "application/json"},
				StatusCode: http.StatusUnauthorized,
			}, nil
		}
		log.Println("ERRO", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "unauthorized"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusUnauthorized,
		}, nil
	}
	if event.Filepath == "" {
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "filepath required"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	// insert to db
	// TODO use something other than md5 for name collisions
	h := md5.New()
	io.WriteString(h, filepath.Join(username, event.Filepath))
	// FIXME auto generate an id. Currently just using the filepath
	// md5 allows for overwriting files with the same path.
	now := time.Now()
	// io.WriteString(h, now.String())
	f := &File{
		ID:   fmt.Sprintf("%x", h.Sum(nil)),
		Date: now,
		Name: filepath.Base(event.Filepath),
		// TODO randomize with subdirectory for user and or by date
		// if we don't want to overwrite files.
		S3Path: filepath.Join(username, event.Filepath),
	}
	if err := putItem(f); err != nil {
		log.Println("ERRO", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "error putting to db"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// TODO have lambda check for completed upload within timeframe and remove if not.

	presign, err := presignedURL(region, s3Bucket, f.S3Path, http.MethodPut)
	if err != nil {
		log.Println("ERRO", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf(errTemplate, "error getting pre signed url"),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf(respPutTemplate, presign, f.ID),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: http.StatusCreated,
	}, nil
}

func uploader(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodPut:
		return put(request)
	case http.MethodGet:
		return get(request)
	}
	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(http.StatusMethodNotAllowed),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: http.StatusMethodNotAllowed,
	}, nil
}

func main() {
	dynamoFileTableName = os.Getenv("file_table_name")
	region = os.Getenv("region")
	s3Bucket = os.Getenv("bucket")
	db = dynamodb.New(session.New(), aws.NewConfig().WithRegion(region))
	// Start takes a handler and talks to an internal Lambda endpoint to pass requests to the handler
	lambda.Start(uploader)
}
