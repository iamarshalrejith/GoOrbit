build:
	@go build -o bin/GoOrbit

run: build
	 @./bin/GoOrbit

test:
	@go test ./... -v
