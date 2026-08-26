connection "postgres" "primary" {
  url = env("DATABASE_PRIMARY_URL")

  pool {
    max_open_conns    = 50
    max_idle_conns    = 10
    conn_max_lifetime = "30m"
  }
}

connection "postgres" "replica" {
  url = env("DATABASE_REPLICA_URL")

  pool {
    max_open_conns    = 100
    max_idle_conns    = 20
    conn_max_lifetime = "30m"
  }
}