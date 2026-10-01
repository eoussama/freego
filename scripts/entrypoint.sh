#!/bin/sh
# Exports the variables of .env, if present, then runs the given command.

ENV_VARS_FILE=".env"

if [ -f "$ENV_VARS_FILE" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    # Trim surrounding whitespace; skip blank lines and comments.
    cleaned_line=$(printf '%s' "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

    case "$cleaned_line" in
      '' | '#'*) continue ;;
    esac

    export "$cleaned_line"
  done < "$ENV_VARS_FILE"
else
  echo "warning: no $ENV_VARS_FILE file found, see .env.example" >&2
fi

exec "$@"
