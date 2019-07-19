package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type File struct {
	ID     string
	Date   time.Time
	Name   string
	S3Path string
}

var db *dynamodb.DynamoDB

func getItem(id string) (*File, error) {
	// Prepare the input for the query.
	input := &dynamodb.GetItemInput{
		TableName: aws.String(dynamoFileTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
	}

	result, err := db.GetItem(input)
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, ErrNotFound
	}

	f := &File{}
	err = dynamodbattribute.UnmarshalMap(result.Item, f)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func putItem(f *File) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String(dynamoFileTableName),
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(f.ID),
			},
			"name": {
				S: aws.String(f.Name),
			},
			"s3path": {
				S: aws.String(f.S3Path),
			},
			"date": {
				S: aws.String(f.Date.Format(time.RFC3339)),
			},
		},
	}

	_, err := db.PutItem(input)
	return err
}
