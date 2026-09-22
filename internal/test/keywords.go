package test

// Keywords are the words the language gives a meaning to: the block types, the
// attribute names and the calls. None of them is reserved — a field or a
// construct may be named after any of them — and the tests that prove it read
// the list from here.
func Keywords() []string {
	return []string{
		"emod",
		"model",
		"actor",
		"context",
		"mode",
		"aggregate",
		"invariants",
		"slice",
		"description",
		"trigger",
		"command",
		"event",
		"view",
		"automation",
		"translation",
		"spec",
		"fields",
		"flow",
		"tags",
		"decides_on",
		"target",
		"type",
		"source",
		"subscribes",
		"on",
		"every",
		"after",
		"reads",
		"external_system",
		"events",
		"where",
		"given",
		"when",
		"then",
		"required",
		"optional",
		"external",
		"rejected",
		"tag",
	}
}
