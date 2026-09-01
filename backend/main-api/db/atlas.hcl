// Atlas config for main-api's versioned migrations.
env "local" {
  src = "ent://schema"
  dev = "docker://postgres/17/dev"
  migration {
    dir = "file://migrations"
  }
}
