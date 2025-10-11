#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

# Temporary file to store the modified configuration
TEMP_JS_FILE="/usr/share/nginx/html/env-config.js.tmp"

# Create a copy of the template to modify
cp /usr/share/nginx/html/env-config.js "$TEMP_JS_FILE"

# Extract all placeholder names and loop through them
# Use a combination of grep and sed to extract the variable names
PLACEHOLDERS=$(grep -o '[A-Z_]*_PLACEHOLDER' "$TEMP_JS_FILE" | sed 's/_PLACEHOLDER//')

for var_name in $PLACEHOLDERS; do
    # Get the value of the environment variable
    var_value=$(eval echo "\$$var_name")
    
    if [ -n "$var_value" ]; then
        placeholder="${var_name}_PLACEHOLDER"
        # Perform the substitution using sed with a safe delimiter (@)
        sed -i "s@$placeholder@$var_value@g" "$TEMP_JS_FILE"
    fi
done

# Overwrite the original config file with the new one
mv "$TEMP_JS_FILE" /usr/share/nginx/html/env-config.js

# Check for BACKEND_SERVER_URL environment variable set or empty
if [ -z "${BACKEND_SERVER_URL:-}" ]; then
  echo "BACKEND_SERVER_URL environment variable not set.." >&2
  exit 1
fi

# Substitute BACKEND_SERVER_URL in the Nginx configuration template
envsubst '$BACKEND_SERVER_URL' < /etc/nginx/conf.d/ui.conf.template > /etc/nginx/conf.d/default.conf

# Start Nginx in the foreground
nginx -g "daemon off;"
