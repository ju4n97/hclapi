server {
  host = "127.0.0.1"
  port = 8080
}

endpoint "GET /openapi.json" {
  openapi {
    format = "json"
  }
}

endpoint "GET /docs" {
  openapi {
    ui = "elements"
  }
}

endpoint "GET /api/v1/weather/{city}" {
  description = "Fetches live weather conditions via a native Go step handler."

  request {
    path {
      field "city" {
        type     = string
        required = true
      }
    }
  }

  pipeline {
    go "fetch_weather" {
      use = "services.get_weather"

      args = {
        city = ctx.request.path.city
      }
    }

    respond {
      status = 200
      body   = steps.fetch_weather.result
    }
  }
}