package viewer

import (
	"embed"
)

//go:embed static/* generated/*
var ViewerFS embed.FS
