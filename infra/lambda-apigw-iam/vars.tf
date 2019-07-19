variable "region" {
  description = "specifies aws region"
  default     = "us-east-1"
}

variable "artifact_bucket" {
  description = "the bucket for fetching the artifact"
  default     = "magic-uploader-artifact"
}

variable "artifact_zip_name" {
  description = "name of the zip file"
  default     = "0.0.1-a4/uploader.zip"
}

variable "uploader_name" {
  description = "name of the binary"
  default     = "uploader"
}

variable "files_bucket" {
  default = "magic-uploader-files"
}

variable "file_table_name" {
  default = "Files"
}
