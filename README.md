# File updloader using AWS Lambda, API Gateway, Dynamodb, S3, and Go. Deployed with Terraform and ron.

This is a simple file uploader and retrieval system. The flow would be
the user gets an API key from some login endpoint and uses it for future
requests.

The first step would be to do a PUT to an endpoint(for this case it's just
the AWS API Gateway returned url) that returns JSON with fields giving
the file id, and where to upload. This creates an entry in Dynamodb and
returns a pre signed url for the upload which offloads the uploading
process to S3. Of course this creates a problem if the user decides
not to upload, there would be a dangling entry for the file in
Dynamodb. To solve that there would need to be a process verifying
that the uploads finished within the 5 minutes which the pre signed
url is valid for, or a cleanup process/Lambda that checks the validity of files.

With the upload_url, the client would then just upload, within 5 minutes,
the file.

To retrieve the file, the user would just use the api key and file id from
the first step, following redirects since it will return an HTTP 301 with
a Location header.


### Usage from bash

	# install jq for processing JSON
	# API_KEY would normally be retrieved through a login endpoint first,
	# this is a working hardcoded one for now.
	export API_KEY=c16f3b18-e43f-4216-b66e-62a26df8c683

	# FILEPATH will end up being the full path given within the users folder
	# Future PUT's with the same file path will overwrite the previous file
	# for the user.
	export FILEPATH=bin/test.txt

    # note the aws url is subject to change. after a deploy it will
	# be listed towards the end as Outputs:
	export URL="https://<apigateway url>.execute-api.us-east-1.amazonaws.com/stage"

	# PUT an object by first retrieving an upload location and id, then uploading
	# directly to the returned location.
	resp=$(curl -X PUT -H "Content-Type: application/json" -d "{\"api_key\":\"$API_KEY\",\"filepath\":\"$FILEPATH\"}" "$URL")
	id=$(echo $resp | jq -r '.id')
	echo $id
	upload_url=$(echo $resp | jq -r '.upload_url')
	curl --upload-file $FILEPATH "$upload_url"

	# GET an object by id
	curl -L "${URL}?api_key=${API_KEY}&id=$id"

### Requirements:

This was all done in macOS so the rest of this assumes so,
but a linux environment would work similarly.

	# /usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"
	brew install terraform awscli go jq
	# terraform: stable 0.12.4 (bottled), HEAD
	# awscli: stable 1.16.190 (bottled), HEAD
	# go: stable 1.12.7 (bottled), HEAD
    # docker(optional for local testing)
	# jq(optional for local testing with bash)
	# ron for keeping track of commands https://github.com/upsight/ron, it's built for macOS and linux in ./bin/


### Initial Setup

For initialization, add AWS credentials to ~/.aws/credentials with `aws configure`,
making sure the IAM credentials can access Lambda, S3, Dynamodb, API Gateway, and Cloudwatch.

	$ ./bin/ron t infra:init


### Deploying

The deploy command will compile the go binary and zip it for Lambda, then
upload it to S3. It will then apply any infrastructure changes with Terraform.

	# NOTE this will cost money in your amazon account.
	VERSION=0.0.1-a4 ./bin/ron t ron:deploy

	# to just apply infrastructure changes
    ./bin/ron t infra:apply

	# to teardown everything
    ./bin/ron t infra:destroy

Versioning is set in ron.yaml but can be passed in via the VERSION env variable.
Updating the code version can be done with:

	NEW_VERSION=0.0.1-a2 ./bin/ron t ron:update_version

The changes can then be checked into version control.
Rollbacks can be done by just running a deploy with the previous VERSION.


### Notes

This was more of a learning exercise to work with serverless deployments for
me. Using this will of course cost money and vendor lock-in is kind of high,
though still kind of portable.

References:
	- [Serverless Applications with AWS Lambda and API Gateway](https://learn.hashicorp.com/terraform/aws/lambda-api-gateway)
	- aws docs for each service
