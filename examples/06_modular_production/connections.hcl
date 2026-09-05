connection "sqlite" "main" {
  source = "file:./data/app.db?mode=rwc"

  pool {
    max_open     = 1
    idle_timeout = "5m"
  }
}