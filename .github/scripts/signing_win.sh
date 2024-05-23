#!/bin/bash

# The purpose of this script is to sign Windows binaries on the remote Checkmarx signing server as part of the CI process

# Check if FILEPATH parameter is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <filename>"
  exit 1
fi

FILEPATH=$1
OS_TYPE=$2
FILENAME=$(basename "$FILEPATH")
FILENAME_SIGNED="$FILENAME"_signed

# Define remote server details
REMOTE_USER=$SIGNING_REMOTE_SSH_USER
REMOTE_HOST=$SIGNING_REMOTE_SSH_HOST
REMOTE_PATH="/tmp"

# HSM credentials
HSM_CREDS=$SIGNING_HSM_CREDS

# Check if OS is windows
if [ "$OS_TYPE" != "windows" ]; then
  echo "The artifact is not a windows binary file, exiting."
  exit 0
fi

# Check if required variables are set
if [ -z "$REMOTE_USER" ] || [ -z "$REMOTE_HOST" ] || [ -z "$HSM_CREDS" ] || [ -z "$SIGNING_REMOTE_SSH_PRIVATE_KEY" ]; then
  echo "Required environment variables are not set"
  exit 1
fi

# Create an SSH key file from the secret
SSH_KEY_PATH=$(mktemp)
echo "$SIGNING_REMOTE_SSH_PRIVATE_KEY" > "$SSH_KEY_PATH"
chmod 600 "$SSH_KEY_PATH"

# Ensure cleanup of temporary files on exit
trap 'rm -f "$SSH_KEY_PATH"' EXIT

# Be sure we don't have already uploaded filess on the remote server
ssh -n -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "rm -f '$REMOTE_PATH/$FILENAME' && rm -f '$REMOTE_PATH/$FILENAME_SIGNED'"

# Upload file via scp
scp -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no "$FILEPATH" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH"
# Check if file was uploaded
if [ $? -ne 0 ]; then
    echo "Failed to copy $FILEPATH"
    exit 1
fi

# Sign
ssh -n -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "osslsigncode sign -certs /home/ubuntu/checkmarx.crt  -key 'pkcs11:object=CNGRSAPriv-cx-signing' -pass $HSM_CREDS -pkcs11module /opt/cloudhsm/lib/libcloudhsm_pkcs11.so -t http://timestamp.digicert.com -in '$REMOTE_PATH/$FILENAME' -out '$REMOTE_PATH/$FILENAME_SIGNED'"
# Check remote command status
if [ $? -ne 0 ]; then
    echo "Failed to sign file $FILENAME"
    ssh -n -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "rm -f '$REMOTE_PATH/$FILENAME'"
    exit 1
fi

# Download signed file via scp
scp -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH/$FILENAME_SIGNED" "/tmp/$FILENAME_SIGNED"
# Check the status
if [ $? -ne 0 ]; then
    echo "Failed to download signed file $FILENAME_SIGNED"
    ssh -n -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "rm -f '$REMOTE_PATH/$FILENAME' && rm -f '$REMOTE_PATH/$FILENAME_SIGNED'"
    exit 1
fi

# Replace original file with the signed
rm -f "$FILEPATH" && mv "/tmp/$FILENAME_SIGNED" "$FILEPATH"

# Cleanup remote server
ssh -n -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "rm -f '$REMOTE_PATH/$FILENAME' && rm -f '$REMOTE_PATH/$FILENAME_SIGNED'"
# Cleanup
rm -f "$SSH_KEY_PATH"
echo "Signing process completed successfully."
