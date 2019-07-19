resource "aws_dynamodb_table" "uploader-dynamodb-table" {
  name = var.file_table_name
  read_capacity = 5
  write_capacity = 5
  hash_key = "id"

  attribute {
    name = "id"
    type = "S"
  }
}
