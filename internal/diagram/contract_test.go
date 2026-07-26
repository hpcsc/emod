//go:build unit

package diagram_test

import (
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

// Every exporter turns the same AST into a different rendering of it. The
// behaviour they share — nothing is dropped, order is preserved, output is
// well-formed — belongs to all four, so it is asserted once here against each
// of them rather than copied into every format's file.
type exporter struct {
	name string
	// fillOfLabel returns the fill colour of the shape drawn for a label; nil
	// for the text formats, which have no colours.
	fillOfLabel func(t *testing.T, output, label string) string
	// countConnections reports how many connections the output draws; nil for
	// formats that describe elements without drawing arrows between them.
	countConnections  func(output string) int
	export            func(*ast.Model, diagram.Style) ([]byte, error)
	requireWellFormed func(t *testing.T, output string)
}

func requireValidXML(t *testing.T, output string) {
	t.Helper()
	require.NoError(t, xml.Unmarshal([]byte(output), new(any)), "output must be well-formed XML")
}

func requireNonEmptyText(t *testing.T, output string) {
	t.Helper()
	require.NotEmpty(t, strings.TrimSpace(output), "output must not be blank")
}

func exporters() []exporter {
	return []exporter{
		{
			name:              "drawio",
			fillOfLabel:       drawioFillOfLabel,
			countConnections:  func(output string) int { return strings.Count(output, `edge="1"`) },
			export:            diagram.ExportDrawio,
			requireWellFormed: requireValidXML,
		},
		{
			name:              "svg",
			fillOfLabel:       svgFillOfLabel,
			countConnections:  func(output string) int { return strings.Count(output, "marker-end") },
			export:            diagram.ExportSVG,
			requireWellFormed: requireValidXML,
		},
		{
			name:              "mermaid",
			export:            diagram.ExportMermaid,
			requireWellFormed: requireNonEmptyText,
		},
		{
			name:              "ascii",
			countConnections:  func(output string) int { return strings.Count(output, " -> ") },
			export:            diagram.ExportASCII,
			requireWellFormed: requireNonEmptyText,
		},
	}
}

var (
	drawioStyleFill = regexp.MustCompile(`fillColor=(#[0-9a-fA-F]{6})`)
	svgRectFill     = regexp.MustCompile(`<rect[^>]*\bfill="(#[0-9a-fA-F]{6})"`)
)

// drawioFillOfLabel returns the fill of the shape whose label contains label.
func drawioFillOfLabel(t *testing.T, output, label string) string {
	t.Helper()

	for _, shape := range drawioShapes(t, output) {
		if !strings.Contains(shape.label, label) {
			continue
		}
		if m := drawioStyleFill.FindStringSubmatch(shape.style); m != nil {
			return strings.ToLower(m[1])
		}
	}

	require.Fail(t, fmt.Sprintf("no drawio cell labelled %q in output", label))
	return ""
}

// svgFillOfLabel returns the fill of the rect drawn immediately before the text
// element carrying label — the shape that label sits inside.
func svgFillOfLabel(t *testing.T, output, label string) string {
	t.Helper()

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "<text") || !strings.Contains(line, ">"+label+"<") {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			if m := svgRectFill.FindStringSubmatch(lines[j]); m != nil {
				return strings.ToLower(m[1])
			}
		}
	}

	require.Fail(t, fmt.Sprintf("no svg shape labelled %q in output", label))
	return ""
}

func (e exporter) run(t *testing.T, model *ast.Model, style diagram.Style) string {
	t.Helper()

	raw, err := e.export(model, style)
	require.NoError(t, err)

	return string(raw)
}

func TestExporterContract(t *testing.T) {
	for _, e := range exporters() {
		t.Run(e.name, func(t *testing.T) {
			t.Run("nil model produces no output and no error", func(t *testing.T) {
				raw, err := e.export(nil, diagram.StyleAuto)

				require.NoError(t, err)
				require.Empty(t, raw)
			})

			t.Run("a model with no contexts still produces well-formed output", func(t *testing.T) {
				output := e.run(t, &ast.Model{Name: "Empty"}, diagram.StyleAuto)

				e.requireWellFormed(t, output)
			})

			t.Run("every named element in the model appears in the output", func(t *testing.T) {
				output := e.run(t, fullModel(), diagram.StyleAuto)

				e.requireWellFormed(t, output)
				for _, label := range fullModelLabels {
					require.Contains(t, output, label)
				}
			})

			t.Run("slices are rendered in model order", func(t *testing.T) {
				model := &ast.Model{
					Name: "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Slices: []*ast.Slice{
								{Name: "First", Commands: []*ast.Command{{Name: "CmdA"}}},
								{Name: "Second", Commands: []*ast.Command{{Name: "CmdB"}}},
								{Name: "Third", Commands: []*ast.Command{{Name: "CmdC"}}},
							},
						}},
					}},
				}

				output := e.run(t, model, diagram.StyleAuto)

				require.Equal(t, []string{"CmdA", "CmdB", "CmdC"}, appearanceOrder(t, output, "CmdA", "CmdB", "CmdC"))
			})

			t.Run("slices declared directly under a DCB context are rendered", func(t *testing.T) {
				model := &ast.Model{
					Name: "DCBTest",
					Contexts: []*ast.Context{{
						Name:   "DCBCtx",
						Slices: []*ast.Slice{{Name: "DirectSlice", Commands: []*ast.Command{{Name: "DirectCmd"}}}},
					}},
				}

				output := e.run(t, model, diagram.StyleDCB)

				e.requireWellFormed(t, output)
				require.Contains(t, output, "DirectCmd")
			})

			t.Run("an aggregate-only context renders under the DCB style too", func(t *testing.T) {
				output := e.run(t, singleSliceModel("Test", "S", command("Cmd")), diagram.StyleDCB)

				e.requireWellFormed(t, output)
				require.Contains(t, output, "Cmd")
			})
		})
	}
}

// TestExporterTranslationEdges covers the arrow a translation implies between
// its command and its nested event. Drawing it unconditionally duplicated any
// flow that already declared the same pair, which put two identical arrows
// between the same two boxes.
func TestExporterTranslationEdges(t *testing.T) {
	for _, e := range exporters() {
		if e.countConnections == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			t.Run("draws the implied command to event arrow when no flow declares it", func(t *testing.T) {
				output := e.run(t, translationModel(false), diagram.StyleAuto)

				// external system -> reactor, reactor -> command, command -> event
				require.Equal(t, 3, e.countConnections(output))
			})

			t.Run("draws it once when a flow declares the same pair", func(t *testing.T) {
				implied := e.run(t, translationModel(false), diagram.StyleAuto)
				declared := e.run(t, translationModel(true), diagram.StyleAuto)

				require.Equal(t, e.countConnections(implied), e.countConnections(declared),
					"spelling the flow out must not add a second arrow for the same pair")
			})

			t.Run("still draws a flow that the translation does not imply", func(t *testing.T) {
				model := translationModel(false)
				slice := model.Contexts[0].Aggregates[0].Slices[0]
				slice.Commands = append(slice.Commands, command("Refund"))
				slice.Events = append(slice.Events, event("Refunded"))
				slice.Flows = []*ast.Flow{{CommandName: "Refund", EventName: "Refunded"}}

				output := e.run(t, model, diagram.StyleAuto)

				require.Equal(t, 4, e.countConnections(output))
			})
		})
	}
}

// translationModel holds a translation whose command emits its nested event.
// With declareFlow set, the model also states that connection as a flow — the
// same fact said twice.
func translationModel(declareFlow bool) *ast.Model {
	model := singleSliceModel("Payments", "Take Payment", command("Charge"), event("Charged"))
	slice := model.Contexts[0].Aggregates[0].Slices[0]
	slice.Translations = []*ast.Translation{{
		Name:           "StripeWebhook",
		ExternalSystem: "Stripe",
		Command:        "Charge",
		Event:          &ast.Event{Name: "Charged"},
	}}
	if declareFlow {
		slice.Flows = []*ast.Flow{{CommandName: "Charge", EventName: "Charged"}}
	}
	return model
}

// TestExporterPalette pins the event-modeling sticky-note convention — orange
// events, blue commands, green read models, white triggers, grey external
// systems — without pinning the exact palette values, which are free to change.
func TestExporterPalette(t *testing.T) {
	for _, e := range exporters() {
		if e.fillOfLabel == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			t.Run("follows the sticky-note colour convention", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				require.Equal(t, "orange", colorFamily(t, e.fillOfLabel(t, output, "Evt")))
				require.Equal(t, "blue", colorFamily(t, e.fillOfLabel(t, output, "Cmd")))
				require.Equal(t, "green", colorFamily(t, e.fillOfLabel(t, output, "Rmo")))
				require.Equal(t, "white", colorFamily(t, e.fillOfLabel(t, output, "Form")))
				require.Equal(t, "grey", colorFamily(t, e.fillOfLabel(t, output, "Stripe")))
			})

			t.Run("gives each element type a distinguishable fill", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				fills := []string{
					e.fillOfLabel(t, output, "Evt"),
					e.fillOfLabel(t, output, "Cmd"),
					e.fillOfLabel(t, output, "Rmo"),
					e.fillOfLabel(t, output, "Stripe"),
				}

				require.Len(t, unique(fills), len(fills), "each element type needs its own fill")
			})

			t.Run("draws an external system with a dashed outline", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				require.Contains(t, output, "dash", "an external system is drawn dashed")
			})
		})
	}
}

// paletteModel holds one element of every coloured kind, each with a distinct
// label so a test can ask for its fill. Every element is described, so asking
// for a shape by its label goes through whatever a format does with prose.
func paletteModel() *ast.Model {
	model := singleSliceModel("Palette", "S",
		&ast.Trigger{Kind: "UI", Name: "Form", Description: "Where the order is placed"},
		&ast.Command{Name: "Cmd", Description: "Asks for the order to be placed"},
		&ast.Event{Name: "Evt", Description: "The order was placed"},
		&ast.View{Name: "Rmo", Description: "Every order placed so far"},
	)
	model.Contexts[0].Aggregates[0].Slices[0].Translations = []*ast.Translation{{
		Name:           "Import",
		Description:    "Restates a payment provider callback in our own language",
		ExternalSystem: "Stripe",
		Command:        "Cmd",
		Event:          &ast.Event{Name: "Evt"},
	}}
	return model
}

// colorFamily classifies a hex colour by hue, so the convention is asserted
// rather than the specific shade.
func colorFamily(t *testing.T, hex string) string {
	t.Helper()

	hue, saturation, value := hsv(t, hex)
	switch {
	case saturation < 0.05 && value >= 0.99:
		return "white"
	case saturation < 0.05:
		return "grey"
	case hue >= 20 && hue < 50:
		return "orange"
	case hue >= 70 && hue < 170:
		return "green"
	case hue >= 170 && hue < 260:
		return "blue"
	case hue >= 260 && hue < 320:
		return "purple"
	default:
		return fmt.Sprintf("unclassified(hue %.0f)", hue)
	}
}

func hsv(t *testing.T, hex string) (hue, saturation, value float64) {
	t.Helper()
	require.Len(t, hex, 7, "expected a #rrggbb colour, got %q", hex)

	channel := func(offset int) float64 {
		v, err := strconv.ParseUint(hex[offset:offset+2], 16, 8)
		require.NoError(t, err, "parsing colour %q", hex)
		return float64(v) / 255
	}
	r, g, b := channel(1), channel(3), channel(5)

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	switch {
	case delta == 0:
		hue = 0
	case max == r:
		hue = 60 * math.Mod((g-b)/delta+6, 6)
	case max == g:
		hue = 60 * ((b-r)/delta + 2)
	default:
		hue = 60 * ((r-g)/delta + 4)
	}

	if max > 0 {
		saturation = delta / max
	}

	return hue, saturation, max
}

// appearanceOrder returns the given substrings sorted by where they first occur
// in output, failing if any is missing.
func appearanceOrder(t *testing.T, output string, substrings ...string) []string {
	t.Helper()

	ordered := make([]string, len(substrings))
	copy(ordered, substrings)

	positions := make(map[string]int, len(substrings))
	for _, s := range substrings {
		idx := strings.Index(output, s)
		require.NotEqual(t, -1, idx, "output does not contain %q", s)
		positions[s] = idx
	}

	sort.Slice(ordered, func(i, j int) bool { return positions[ordered[i]] < positions[ordered[j]] })

	return ordered
}

func unique(values []string) []string {
	seen := make(map[string]bool, len(values))
	var out []string
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// --- model builders shared by every exporter's tests ---

func minimalModel(name, sliceName string) *ast.Model {
	return &ast.Model{
		Name: name,
		Contexts: []*ast.Context{{
			Name: "Ctx",
			Aggregates: []*ast.Aggregate{{
				Name: "Agg",
				Slices: []*ast.Slice{{
					Name: sliceName,
				}},
			}},
		}},
	}
}

func singleSliceModel(modelName, sliceName string, opts ...any) *ast.Model {
	m := minimalModel(modelName, sliceName)
	s := m.Contexts[0].Aggregates[0].Slices[0]
	for _, opt := range opts {
		switch v := opt.(type) {
		case *ast.Command:
			s.Commands = append(s.Commands, v)
		case *ast.Event:
			s.Events = append(s.Events, v)
		case *ast.View:
			s.Views = append(s.Views, v)
		case *ast.Trigger:
			s.Trigger = v
		}
	}
	return m
}

func command(name string) *ast.Command {
	return &ast.Command{Name: name}
}

func event(name string) *ast.Event {
	return &ast.Event{Name: name}
}

func eventWithSource(name, source, externalName string) *ast.Event {
	return &ast.Event{Name: name, Source: source, ExternalName: externalName}
}

func view(name string) *ast.View {
	return &ast.View{Name: name}
}

// fullModelLabels lists the element names every format renders. Structural
// names (model, context, slice) and external systems appear in some formats
// only, so those stay in the per-format tests.
var fullModelLabels = []string{
	"PlaceOrderForm", "ShipTimer",
	"CreateOrder", "ValidatePayment", "ShipOrder",
	"OrderCreated", "PaymentValidated", "OrderShipped",
	"OrderSummary",
	"InventoryUpdater",
	"PaymentGW",
}

// describedModel carries a distinct description on every construct that accepts
// one — including the aggregate and the slices, which no format draws a shape
// for — so a test can tell whose prose reached which shape. Its context mixes an
// aggregate with a directly declared slice, so each layout style has work to do.
func describedModel() *ast.Model {
	return &ast.Model{
		Name:        "Described",
		Description: "How the hotel takes and imports room bookings",
		Actors: []*ast.Actor{
			{Name: "Guest", Description: "A person booking a room, not always the one staying in it"},
		},
		Contexts: []*ast.Context{{
			Name:        "Bookings",
			Description: "Everything the hotel knows about a stay before the guest arrives",
			Aggregates: []*ast.Aggregate{{
				Name:        "Booking",
				Description: "One guest holding one room over one date range",
				Slices: []*ast.Slice{{
					Name:        "Hold a room",
					Description: "A guest books a room from the public site",
					Trigger: &ast.Trigger{
						Kind:        "UI",
						Name:        "BookingForm",
						Actor:       "Guest",
						Description: "The booking form on the public site",
					},
					Commands: []*ast.Command{{
						Name:        "HoldRoom",
						Description: "Ask the hotel to hold a room over a date range",
					}},
					Events: []*ast.Event{{
						Name:        "RoomHeld",
						Description: "A room is held for a guest",
					}},
					Views: []*ast.View{{
						Name:        "StayList",
						Description: "Every booking with the stage it has reached",
						Subscribes:  []string{"RoomHeld"},
					}},
					Flows: []*ast.Flow{{CommandName: "HoldRoom", EventName: "RoomHeld"}},
					Automations: []*ast.Automation{{
						Name:         "AutoConfirm",
						Description:  "Confirms every booking the moment it is made",
						TriggerEvent: "RoomHeld",
						Command:      "HoldRoom",
					}},
					Translations: []*ast.Translation{{
						Name:           "PartnerWebhook",
						Description:    "Restates a partner webhook in the hotel's own language",
						ExternalSystem: "PartnerAPI",
						Command:        "HoldRoom",
						Event: &ast.Event{
							Name:        "PartnerBookingReceived",
							Description: "A partner site reported a booking",
						},
					}},
				}},
			}},
			Slices: []*ast.Slice{{
				Name:        "Settle the stay",
				Description: "The guest pays on the morning they leave",
				Events: []*ast.Event{{
					Name:        "StaySettled",
					Description: "The guest has paid for the whole stay",
					Tags:        []ast.TagEntry{{Key: "stay", FieldRef: "stayId"}},
				}},
			}},
		}},
	}
}

// withoutDescriptions strips the prose out of a model in place, so a test can
// compare a described model's diagram against the one it draws without prose.
func withoutDescriptions(model *ast.Model) *ast.Model {
	model.Description = ""
	for _, actor := range model.Actors {
		actor.Description = ""
	}
	for _, ctx := range model.Contexts {
		ctx.Description = ""
		for _, agg := range ctx.Aggregates {
			agg.Description = ""
			undescribeSlices(agg.Slices)
		}
		undescribeSlices(ctx.Slices)
	}
	return model
}

func undescribeSlices(slices []*ast.Slice) {
	for _, s := range slices {
		s.Description = ""
		if s.Trigger != nil {
			s.Trigger.Description = ""
		}
		for _, cmd := range s.Commands {
			cmd.Description = ""
		}
		for _, evt := range s.Events {
			evt.Description = ""
		}
		for _, v := range s.Views {
			v.Description = ""
		}
		for _, auto := range s.Automations {
			auto.Description = ""
		}
		for _, tr := range s.Translations {
			tr.Description = ""
			if tr.Event != nil {
				tr.Event.Description = ""
			}
		}
	}
}

func fullModel() *ast.Model {
	return &ast.Model{
		Name: "FullModel",
		Contexts: []*ast.Context{{
			Name: "Orders",
			Aggregates: []*ast.Aggregate{{
				Name: "Order",
				Slices: []*ast.Slice{
					{
						Name: "Create Order",
						Trigger: &ast.Trigger{
							Kind:  "UI",
							Name:  "PlaceOrderForm",
							Actor: "Customer",
						},
						Commands: []*ast.Command{
							{Name: "CreateOrder"},
							{Name: "ValidatePayment"},
						},
						Events: []*ast.Event{
							{Name: "OrderCreated"},
							{Name: "PaymentValidated"},
						},
						Views: []*ast.View{
							{Name: "OrderSummary", Subscribes: []string{"OrderCreated"}},
						},
						Flows: []*ast.Flow{
							{CommandName: "CreateOrder", EventName: "OrderCreated"},
							{CommandName: "ValidatePayment", EventName: "PaymentValidated"},
						},
						Automations: []*ast.Automation{
							{
								Name:         "InventoryUpdater",
								TriggerEvent: "OrderCreated",
								Command:      "CreateOrder",
							},
						},
						Translations: []*ast.Translation{
							{
								Name:           "PaymentGW",
								ExternalSystem: "Stripe",
								Command:        "ValidatePayment",
								Event:          &ast.Event{Name: "PaymentValidated"},
							},
						},
					},
					{
						Name: "Ship Order",
						Trigger: &ast.Trigger{
							Kind: "Schedule",
							Name: "ShipTimer",
						},
						Commands: []*ast.Command{
							{Name: "ShipOrder"},
						},
						Events: []*ast.Event{
							{Name: "OrderShipped"},
						},
						Flows: []*ast.Flow{
							{CommandName: "ShipOrder", EventName: "OrderShipped"},
						},
					},
				},
			}},
		}},
	}
}
