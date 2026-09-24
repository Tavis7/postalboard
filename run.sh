#!/bin/bash

RUNNING=1
while [ $RUNNING -eq 1 ]
do
    echo "Building..."
    go build ./cmd/postalboard || break
    ./postalboard
    STATUS=$?
    if [ $STATUS -ne 2 ]
    then
        echo "Quitting: $STATUS"
        RUNNING=0
    fi
done
