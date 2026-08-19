package frontend

import (
	"embed"
)

//go:embed static/* generated/*
var FS embed.FS
