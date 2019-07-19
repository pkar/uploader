output "base_url" {
  value = aws_api_gateway_deployment.uploader.invoke_url
}

