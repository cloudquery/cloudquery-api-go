//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.3.0 --config=./models.yaml ./spec.json
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.3.0 --config=./client.yaml ./spec.json
package cloudquery_api
