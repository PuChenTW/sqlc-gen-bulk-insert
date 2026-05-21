package main

import (
	"github.com/sqlc-dev/plugin-sdk-go/codegen"

	"github.com/puchentw/sqlc-gen-bulk-insert/internal/gen"
)

func main() {
	codegen.Run(gen.Generate)
}
