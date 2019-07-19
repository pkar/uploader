package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(tb testing.TB, exp, act interface{}, msg ...string) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		if len(msg) > 0 {
			fmt.Printf(
				"\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n%v\n\n",
				filepath.Base(file), line, exp, act, msg)
		} else {
			fmt.Printf(
				"\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n",
				filepath.Base(file), line, exp, act)
		}
		tb.FailNow()
	}
}

func Test_validateAPIKey(t *testing.T) {
	u, err := validateAPIKey("c16f3b18-e43f-4216-b66e-62a26df8c683")
	Ok(t, err)
	Equals(t, "useridpaul", u)
	u, err = validateAPIKey("s")
	Equals(t, err, ErrUnauthorized)
	Equals(t, "", u)
}

func Test_uploader(t *testing.T) {
	db = dynamodb.New(session.New(), aws.NewConfig().WithRegion("us-east-1"))
	dynamoFileTableName = "Files"
	region = "us-east-1"
	s3Bucket = "magic-uploader-files"
	tests := []struct {
		name             string
		request          events.APIGatewayProxyRequest
		expectStatusCode int
		expect           string
		err              error
	}{
		{
			name: "00 create put",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPut,
				Body:       `{"api_key":"c16f3b18-e43f-4216-b66e-62a26df8c683","filepath":"magic/thegathering.txt"}`,
			},
			expectStatusCode: http.StatusCreated,
			expect:           `"upload_url":"https://magic-uploader-files`,
			err:              nil,
		},
		{
			name: "01 invalid put json",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPut,
				Body:       `{`,
			},
			expectStatusCode: http.StatusBadRequest,
			expect:           "invalid json",
			err:              nil,
		},
		{
			name: "02 invalid method",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       "",
			},
			expectStatusCode: http.StatusMethodNotAllowed,
			expect:           http.StatusText(http.StatusMethodNotAllowed),
			err:              nil,
		},
		{
			name: "03 get file",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				QueryStringParameters: map[string]string{
					"api_key": "c16f3b18-e43f-4216-b66e-62a26df8c683",
					"id":      "2a7a90d63f8486d4a6c2ffe7c3a290b0",
				},
			},
			expectStatusCode: http.StatusMovedPermanently,
			expect:           ``,
			err:              nil,
		},
		{
			name: "03 get file not found",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				QueryStringParameters: map[string]string{
					"api_key": "c16f3b18-e43f-4216-b66e-62a26df8c683",
					"id":      "2a7a90d3f8486d4a6c2ffe7c3a290b0",
				},
			},
			expectStatusCode: http.StatusNotFound,
			expect:           `not found`,
			err:              nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response, err := uploader(tc.request)
			Equals(t, tc.err, err)
			Equals(t, tc.expectStatusCode, response.StatusCode, response.Body)
			if tc.expectStatusCode == http.StatusMovedPermanently {
				if l, ok := response.Headers["Location"]; !ok {
					if !strings.Contains(l, "https://"+s3Bucket) {
						t.Fatal("location not set")
					}
				}
			}
			Equals(t, true, strings.Contains(response.Body, tc.expect), tc.expect+"\n"+response.Body)
		})
	}
}
