terraform {
  required_version = "~> 1.12"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "3.0"
    }
  }
}

provider "helm" {
  kubernetes = {
    config_path = "~/.kube/config"
  }
}
