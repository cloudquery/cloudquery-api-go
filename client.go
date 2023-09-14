//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.13.3 --config=./models.yaml ./spec.json
//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.13.3 --config=./client.yaml ./spec.json
package cloudquery_api
