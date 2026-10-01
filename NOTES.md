Standard go project layout: <https://github.com/golang-standards/project-layout>

Goose: <https://github.com/pressly/goose>

From project root:

```
. .env
cd sql/schema/
goose up
```

SQLC: <https://sqlc.dev/>

Run `sqlc generate` from project root to regenerate `internal/db` package
