#!/bin/bash

if ! pgrep -f "/usr/local/bin/main"; then
  echo "Application is not running"
  exit 1
fi

echo "Application is running"
exit 0
