package main

import (
	"go.uber.org/fx"

	"mcp-gitlab-review/internal/app"
)

func main() {
	fx.New(app.Module).Run()
}
