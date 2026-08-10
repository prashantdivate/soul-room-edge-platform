terraform {
  required_version = ">= 1.6.0"
}

variable "name" {
  type    = string
  default = "soul-room"
}

# Cloud-neutral skeleton: wire to your provider module for managed PostgreSQL,
# S3-compatible object storage, load balancer, TLS certificates, secrets,
# private networking, observability, backup, and disaster recovery.
output "deployment_name" {
  value = var.name
}
