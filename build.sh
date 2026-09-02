#!/bin/bash

go build ./... && curl -X POST "localhost:8080/admin/restart"
