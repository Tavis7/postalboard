#!/bin/bash

RUNNING=1
while [ $RUNNING -eq 1 ]
do
    [ -e ./postalboard ] || (echo "Buildign..." && go build ./...) || break
    ./postalboard
    STATUS=$?
    if [ $STATUS -ne 2 ]
    then
        echo "Quitting: $STATUS"
        RUNNING=0
    fi
done
