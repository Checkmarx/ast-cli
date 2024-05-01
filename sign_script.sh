#!/bin/bash

# Assuming private_key.pem and public_key.pem are in the same directory as this script
PRIVATE_KEY="private_key.pem"
PUBLIC_KEY="public_key.pem"
INPUT_EXE="cx.exe"
OUTPUT_EXE="signed-cx.exe"

# Sign the executable using osslsigncode
osslsigncode sign -key "$PRIVATE_KEY" -cert "$PUBLIC_KEY" -in "$INPUT_EXE" -out "$OUTPUT_EXE"
