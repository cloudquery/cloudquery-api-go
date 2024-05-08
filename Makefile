.PHONY: test
test:
	go test -race ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: gen-client
gen-client:
	@command -v openapi-generator >/dev/null 2>&1 || { \
		echo "Error: 'openapi-generator' command not found. Please install it before running convert-spec."; \
		echo "On MacOS you can use Homebrew: brew install openapi-generator"; \
		echo "You can install it by following the instructions at: https://github.com/OpenAPITools/openapi-generator?tab=readme-ov-file#1---installation"; \
		exit 1; \
	}
	openapi-generator generate -g openapi --skip-validate-spec -i spec.json -o .openapi-tmp
	mv .openapi-tmp/openapi.json spec.json
	rm -rf .openapi-tmp
	go generate ./...