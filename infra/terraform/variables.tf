variable "project_id" {
  type        = string
  description = "GCP project id"
}

variable "region" {
  type        = string
  default     = "us-central1"
  description = "GCP region"
}

variable "name_prefix" {
  type        = string
  default     = "codigo-sre"
  description = "Resource name prefix"
}

variable "gke_num_nodes" {
  type        = number
  default     = 2
  description = "Default node count"
}
