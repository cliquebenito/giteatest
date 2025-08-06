#!/bin/bash

changedBranch=$(git symbolic-ref HEAD | sed -e 's,.*/\(.*\),\1,')
blockedUsers=(admin_test)
emailRegexp="^.*@(sbertech.ru)|(sberbank.ru)|(sc.sbt.ru)$"

if [[ ! "$SC_PUSHER_EMAIL" =~ $emailRegexp ]]; then
  echo "You are not allowed commit changes"
  exit 1
fi

if [[ ${blockedUsers[*]} =~ $SC_PUSHER_NAME ]]; then
    if [ $changedBranch == "main" ]; then
        echo "You are not allowed commit changes in this branch"
        exit 1
    fi
fi
