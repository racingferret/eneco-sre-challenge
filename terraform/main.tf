resource "helm_release" "sre_challenge" {
  name       = var.chart_name
  chart      = "./charts/${var.chart_name}"
  namespace  = var.namespace

  set = [
    {
      name  = "ENVIRONMENT"
      value = var.environment
    },
    {
      name  = "MAX_CONNECTIONS"
      value = var.max_connections
    },
    {
      name  = "LOG_LEVEL"
      value = var.log_level
    },
    {
      name  = "API_BASE_URL"
      value = var.api_base_url
    }
  ]
}

