#!/bin/bash

go build ./... && curl "localhost:8080/admin/restart"
