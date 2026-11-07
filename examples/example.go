package examples

import (
	_ "embed"
)

//go:embed all_patterns.emod
var allPatterns string

//go:embed dcb_model.emod
var dcbModel string

//go:embed specs_hotel.emod
var specsHotel string

type Example struct {
	Name        string
	Description string
	Source      string
}

func All() []Example {
	return []Example{
		{
			Name:        "all_patterns",
			Description: "A hotel model that shows the four slice patterns: command, view, automation and translation.",
			Source:      allPatterns,
		},
		{
			Name:        "dcb_model",
			Description: "An order model in a dcb context, with tags on events and commands that read them with decides_on.",
			Source:      dcbModel,
		},
		{
			Name:        "specs_hotel",
			Description: "A hotel model with Given-When-Then specs, with example payloads and rejections.",
			Source:      specsHotel,
		},
	}
}
