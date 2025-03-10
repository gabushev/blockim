.PHONY: build-server build-client run-server run-client restart-server swagger test

swagger:
	go install github.com/swaggo/swag/cmd/swag@v1.8.4
	cd cmd/server && $${HOME}/go/bin/swag init -g main.go --parseDependency --parseInternal --output ../../docs

test:
	go test -v ./internal/...

build-server:
	docker build -t blockim-server -f Dockerfile.server .

run-server:
	@if [ -z "$$SERVER_SECRET" ]; then \
		echo "Error: SERVER_SECRET is not set"; \
		exit 1; \
	fi
	@if [ -z "$$API_KEY" ]; then \
		echo "Error: API_KEY is not set"; \
		exit 1; \
	fi
	docker run -d \
		--name blockim-server \
		-p 8080:8080 \
		-e BLOCKIM_API_KEY=$${API_KEY} \
		-e BLOCKIM_SERVER_SECRET=$${SERVER_SECRET} \
		-e BLOCKIM_POW_DIFFICULTY=$${POW_DIFFICULTY:-20} \
		blockim-server

build-client:
	docker build -t blockim-client -f Dockerfile.client .

run-client:
	@if [ -z "$$SERVER_ADDR" ]; then \
		echo "Using default server address localhost:8080"; \
		export SERVER_ADDR=localhost:8080; \
	fi
	docker run -it --rm \
		--name blockim-client \
		--network host \
		-e SERVER_ADDR=$${SERVER_ADDR} \
		blockim-client

stop-server:
	docker stop blockim-server || true
	docker rm blockim-server || true

restart-server: swagger
	docker stop blockim-server || true
	docker rm blockim-server || true
	make swagger
	make build-server
	make run-server

rebuild: clean swagger build-server build-client