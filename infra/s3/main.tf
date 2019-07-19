provider "aws" {
  region = "${var.region}"
}

resource "aws_s3_bucket" "files_bucket" {
  bucket = "${var.files_bucket}"
  acl    = "private"
}

resource "aws_s3_bucket" "artifact_bucket" {
  bucket = "${var.artifact_bucket}"
  acl    = "private"
}
