# Word of Wisdom TCP Server

TCP-server protected by PoW (Hashcash) from DDOS

## Description

This service produces a nice quotes if the client solves the PoW-based challenge.

The project contains two parts: server and client. You can use curl or whatever, but what the reason for that?

`Hashcash` was chosen as the algo for "protecting" because it's easy to sign, calculate, the complexity may vary by settings (the amount of leading zeros in solution). 

## Configuration

The file config.yaml contains configuration data may be overwritten by ENV variables.

Example:


```yaml
server:
  port: 8080
  secret: "your-secret-key"

pow:
  difficulty: 20

api:
  url: "https://the-one-api.dev/v2/quote"
  key: "your_api_key"
```
Configuration is required by docker images, but a few words below:

1. API Key for the-one-api.dev - it's a datasource for the quotes. But you can skip it.

   `API_KEY=someKeyNeedToBePlaced`
2. Server secret used for POV

   `SERVER_SECRET=1234567`
3. POW difficulty. Default value is 20 - it is enough to provide effective level but stay in second delays.

    `POW_DIFFICULTY=20`
4. Server address for client service. You can override it when the infrastructure went too complicated (don't do this)
   
    `SERVER_ADDR=localhost:8080`

## How to run and build

1. docker-compose

    `API_KEY=someKeyNeedToBePlaced SERVER_SECRET=123 docker-compose up --build`

    It would start a new server and provide api with [Swagger doc](http://localhost:8080/swagger/index.html) and the rest
    of API functionality.
    And the container with the client starts right after the server is ready, it would connect take a new challenge, solve it
    and die with glory.

2. run docker images separately
   
   1. Run the server first
   
      It starts, prepare the quotes and serves requests

    `API_KEY=someKeyNeedToBePlaced SERVER_SECRET=123 make build-server && make run-server`

    2. Run the client
   
       Each run would get one new task and tries to solve it
   
    `SERVER_ADDR=localhost:8080 make build-client && make run-client`
