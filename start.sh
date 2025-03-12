#!/bin/sh

# this is needed for running as a separated docker container (to avoid passing extra obvious configuration)
SERVER_ADDR=${SERVER_ADDR:-"localhost:8080"}

HOST=$(echo "$SERVER_ADDR" | cut -d':' -f1)
PORT=$(echo "$SERVER_ADDR" | cut -d':' -f2)
echo "HOST: $HOST"
echo "port: $PORT"

echo "Waiting for server to be ready at $HOST:$PORT..."
while true; do
    if echo 'health' | nc -N "$HOST" "$PORT" | grep -q '"status":"ok"'; then
        break
    fi
    echo "Waiting for server..."
    sleep 1
done

echo "Server is ready, starting client..."
exec ./client -server "$SERVER_ADDR"
