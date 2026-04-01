build:
	CGO_ENABLED=0 go build -o billfold ./cmd/billfold/

run: build
	./billfold

test:
	go test ./...

clean:
	rm -f billfold

.PHONY: build run test clean
