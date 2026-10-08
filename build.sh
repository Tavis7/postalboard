#!/bin/bash

echo "sqlc generate"
sqlc generate

echo "building"
go build ./... && (curl -X POST "localhost:8080/admin/restart" || go build ./cmd/postalboard)
