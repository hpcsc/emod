package viewer

import (
	"embed"
)

//go:embed viewer.html viewer.js store.js config.js bus.js layout.js renderer.js interaction.js ui.js model.js emod-export.js wasm_exec.js emod.wasm
var ViewerFS embed.FS
