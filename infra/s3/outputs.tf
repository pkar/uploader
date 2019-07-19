output "s3_files_arn" {
  value = "${aws_s3_bucket.files_bucket.arn}"
}

output "s3_artifact_arn2" {
  value = "${aws_s3_bucket.artifact_bucket.arn}"
}
