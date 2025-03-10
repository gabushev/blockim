#!/bin/sh

HOST=$(echo "$SERVER_ADDR" | cut -d':' -f1)
PORT=$(echo "$SERVER_ADDR" | cut -d':' -f2)

echo "Waiting for server to be ready at $HOST:$PORT..."
while ! nc -z "$HOST" "$PORT"; do
  sleep 1
  echo "Waiting for server..."
done

echo "Server is ready, starting client..."
./client --server "http://$SERVER_ADDR"
