endpoint "GET /health/live" {
  description = "Public health check endpoint for load balancers."

  # EXPLICIT OVERRIDE: Empty array removes the global JWT requirement
  auth = []

  pipeline {
    respond {
      status = 200
      body   = { status = "OK", timestamp = ctx.timestamp_epoch }
    }
  }
}

endpoint "POST /webhooks/stripe" {
  description = "Public webhook receiver with strict header validation."

  # EXPLICIT OVERRIDE: Must be public to receive external Stripe webhooks
  auth = []

  request {
    headers {
      field "stripe-signature" {
        type     = string
        required = true
      }
    }
    body {
      field "type" { type = string, required = true }
      field "data" { type = any, required = true }
    }
  }

  pipeline {
    respond {
      status = 200
      body   = { received = true }
    }
  }
}