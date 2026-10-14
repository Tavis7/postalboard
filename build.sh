#!/bin/bash

echo "sqlc generate"
sqlc generate

echo "building"
go build ./... &&
    go test -count=1 ./... &&
    (curl --silent -X POST "localhost:8080/admin/restart" || go build ./cmd/postalboard)
