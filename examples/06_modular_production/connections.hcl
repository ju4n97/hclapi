connection "sqlite" "main" {
  url = "file:./data/app.db?mode=rwc"

  pool {
    max_open_conns = 1
    idle_timeout   = "5m"
  }
}