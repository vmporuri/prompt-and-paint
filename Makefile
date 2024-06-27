build:
	go build -o bin/app cmd/web/*.go

run: build
	./bin/app
