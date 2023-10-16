.PHONY: test
test:
	go test -tags=assert -race ./...

