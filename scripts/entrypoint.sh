#!/bin/sh

ENV_VARS_FILE=".env"

if [ -f "$ENV_VARS_FILE" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    cleaned_line=$(echo "$line" | sed 's/#.*//;s/^[[:space:]]*//;s/[[:space:]]*$//')
    
    if [ -z "$cleaned_line" ]; then
      continue
    fi
    
    export "$cleaned_line"
  done < "$ENV_VARS_FILE"
else
  exit 1
fi

exec "$@"