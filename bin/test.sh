#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# API_KEY would normally be retrieved through a login endpoint first,
# this is a working hardcoded one for now.
API_KEY=c16f3b18-e43f-4216-b66e-62a26df8c683

FILEPATH=$DIR/test.txt

# note the aws url is subject to change
# URL="https://<apigateway id>.execute-api.us-east-1.amazonaws.com/stage"

# PUT an object by first retrieving an upload location and id, then uploading
# directly to the returned location.
resp=$(curl -s -X PUT -H "Content-Type: application/json" -d "{\"api_key\":\"$API_KEY\",\"filepath\":\"$FILEPATH\"}" "$URL")

# brew install jq
id=$(echo $resp | jq -r '.id')
echo file id is $id
upload_url=$(echo $resp | jq -r '.upload_url')
echo uploading to url $upload_url
curl --upload-file $FILEPATH "$upload_url"

# GET an object by id
echo retrieving file $id
curl -L "${URL}?api_key=${API_KEY}&id=$id"
echo
