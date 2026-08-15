#!/bin/sh
set -eu

umask 077
mkdir -p /keys

if [ ! -s /keys/api_private_key ]; then
  openssl genpkey -algorithm RSA -out /keys/api_private_key -pkeyopt rsa_keygen_bits:2048
fi
if [ ! -s /keys/api_public_key ]; then
  openssl rsa -in /keys/api_private_key -out /keys/api_public_key -pubout
fi
if [ ! -s /keys/ssh_private_key ]; then
  openssl genpkey -algorithm RSA -out /keys/ssh_private_key -pkeyopt rsa_keygen_bits:2048
fi

chmod 0600 /keys/api_private_key /keys/api_public_key /keys/ssh_private_key
