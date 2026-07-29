//go:build unit

package export_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestExport(t *testing.T) {
	t.Run("model json", func(t *testing.T) {
		t.Run("serializes model name and empty collections as omitted fields", func(t *testing.T) {
			model := &ast.Model{Name: "TestModel"}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)
			require.True(t, json.Valid(raw))

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)
			require.Equal(t, "TestModel", doc["name"])
			_, hasActors := doc["actors"]
			require.False(t, hasActors, "nil actors should be omitted")
			_, hasContexts := doc["contexts"]
			require.False(t, hasContexts, "nil contexts should be omitted")
		})

		t.Run("serializes actors", func(t *testing.T) {
			model := &ast.Model{
				Name: "App",
				Actors: []*ast.Actor{
					{Name: "Guest"},
					{Name: "Admin"},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			actors, ok := doc["actors"].([]any)
			require.True(t, ok)
			require.Len(t, actors, 2)
			require.Equal(t, "Guest", actors[0].(map[string]any)["name"])
			require.Equal(t, "Admin", actors[1].(map[string]any)["name"])
		})

		t.Run("serializes context with aggregate and slice containing command with typed fields", func(t *testing.T) {
			model := &ast.Model{
				Name: "Hotel",
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name: "Make Reservation",
										Commands: []*ast.Command{
											{
												Name: "MakeReservation",
												Fields: []*ast.Field{
													{Name: "guestId", Type: "string", Modifier: "required"},
													{Name: "roomType", Type: "string", Modifier: "required"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			contexts := doc["contexts"].([]any)
			c := contexts[0].(map[string]any)
			require.Equal(t, "Reservations", c["name"])

			aggs := c["aggregates"].([]any)
			agg := aggs[0].(map[string]any)
			require.Equal(t, "Reservation", agg["name"])

			slices := agg["slices"].([]any)
			s := slices[0].(map[string]any)
			require.Equal(t, "Make Reservation", s["name"])

			cmds := s["commands"].([]any)
			cmd := cmds[0].(map[string]any)
			require.Equal(t, "MakeReservation", cmd["name"])

			fields := cmd["fields"].([]any)
			require.Len(t, fields, 2)

			f0 := fields[0].(map[string]any)
			require.Equal(t, "guestId", f0["name"])
			require.Equal(t, "string", f0["type"])
			require.Equal(t, "required", f0["modifier"])

			f1 := fields[1].(map[string]any)
			require.Equal(t, "roomType", f1["name"])
			require.Equal(t, "string", f1["type"])
			require.Equal(t, "required", f1["modifier"])
		})

		t.Run("serializes trigger with actor and reads", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "My Slice",
										Trigger: &ast.Trigger{
											Kind:  "UI",
											Name:  "Form",
											Actor: "Guest",
											Reads: "MyView",
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			tr := s["trigger"].(map[string]any)
			require.Equal(t, "UI", tr["kind"])
			require.Equal(t, "Form", tr["name"])
			require.Equal(t, "Guest", tr["actor"])
			require.Equal(t, "MyView", tr["reads"])
		})

		t.Run("serializes event with source external and ExternalName", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "Receive",
										Events: []*ast.Event{
											{
												Name:         "PaymentReceived",
												Source:       "external",
												ExternalName: "Stripe",
												Fields: []*ast.Field{
													{Name: "paymentId", Type: "string", Modifier: "required"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			events := s["events"].([]any)
			e := events[0].(map[string]any)
			require.Equal(t, "PaymentReceived", e["name"])
			require.Equal(t, "external", e["source"])
			require.Equal(t, "Stripe", e["external_name"])
		})

		t.Run("serializes view with fields and subscribes", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Views: []*ast.View{
											{
												Name: "RoomsView",
												Fields: []*ast.Field{
													{Name: "roomId", Type: "string", Modifier: "required"},
													{Name: "status", Type: "string", Modifier: "optional"},
												},
												Subscribes: []string{"RoomReserved", "GuestCheckedOut"},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			views := s["views"].([]any)
			v := views[0].(map[string]any)
			require.Equal(t, "RoomsView", v["name"])

			fields := v["fields"].([]any)
			require.Len(t, fields, 2)

			f0 := fields[0].(map[string]any)
			require.Equal(t, "roomId", f0["name"])
			require.Equal(t, "string", f0["type"])
			require.Equal(t, "required", f0["modifier"])

			f1 := fields[1].(map[string]any)
			require.Equal(t, "status", f1["name"])
			require.Equal(t, "string", f1["type"])
			require.Equal(t, "optional", f1["modifier"])

			subs := v["subscribes"].([]any)
			require.Len(t, subs, 2)
			require.Equal(t, "RoomReserved", subs[0])
			require.Equal(t, "GuestCheckedOut", subs[1])
		})

		t.Run("serializes automation with cross-context target", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "Notify",
										Automations: []*ast.Automation{
											{
												Name:          "OrderNotifier",
												TriggerEvent:  "OrderPlaced",
												Command:       "SendNotification",
												TargetContext: "Notifications",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			autos := s["automations"].([]any)
			a := autos[0].(map[string]any)
			require.Equal(t, "OrderNotifier", a["name"])
			require.Equal(t, "OrderPlaced", a["trigger_event"])
			require.Equal(t, "SendNotification", a["command"])
			require.Equal(t, "Notifications", a["target_context"])
		})

		t.Run("serializes translation with external system and nested event", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "Import",
										Translations: []*ast.Translation{
											{
												Name:           "BookingImport",
												ExternalSystem: "Booking.com API",
												Reads:          "WebhookView",
												Command:        "ImportBooking",
												Event: &ast.Event{
													Name: "BookingImported",
													Fields: []*ast.Field{
														{Name: "bookingId", Type: "string", Modifier: "required"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			trans := s["translations"].([]any)
			tl := trans[0].(map[string]any)
			require.Equal(t, "BookingImport", tl["name"])
			require.Equal(t, "Booking.com API", tl["external_system"])
			require.Equal(t, "WebhookView", tl["reads"])
			require.Equal(t, "ImportBooking", tl["command"])

			nestedEvent := tl["event"].(map[string]any)
			require.Equal(t, "BookingImported", nestedEvent["name"])
			fields := nestedEvent["fields"].([]any)
			require.Len(t, fields, 1)
			require.Equal(t, "bookingId", fields[0].(map[string]any)["name"])
		})

		t.Run("serializes flows", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Flows: []*ast.Flow{
											{
												CommandName: "MakeReservation",
												EventName:   "ReservationMade",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			flows := s["flows"].([]any)
			f := flows[0].(map[string]any)
			require.Equal(t, "MakeReservation", f["command_name"])
			require.Equal(t, "ReservationMade", f["event_name"])
		})

		t.Run("field modifier omitted when empty", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{
												Name: "Cmd",
												Fields: []*ast.Field{
													{Name: "name", Type: "string"},
													{Name: "age", Type: "int", Modifier: "optional"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			cmd := s["commands"].([]any)[0].(map[string]any)
			fields := cmd["fields"].([]any)

			f0 := fields[0].(map[string]any)
			require.Equal(t, "name", f0["name"])
			require.Equal(t, "string", f0["type"])
			_, hasMod := f0["modifier"]
			require.False(t, hasMod, "empty modifier should be omitted")

			f1 := fields[1].(map[string]any)
			require.Equal(t, "age", f1["name"])
			require.Equal(t, "int", f1["type"])
			require.Equal(t, "optional", f1["modifier"])
		})

		t.Run("multiple contexts and actors produce full document", func(t *testing.T) {
			model := &ast.Model{
				Name: "App",
				Actors: []*ast.Actor{
					{Name: "Guest"},
					{Name: "Admin"},
				},
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Aggregates: []*ast.Aggregate{
							{Name: "Order"},
						},
					},
					{
						Name: "Payments",
						Aggregates: []*ast.Aggregate{
							{Name: "Payment"},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			require.Equal(t, "App", doc["name"])

			actors := doc["actors"].([]any)
			require.Len(t, actors, 2)

			contexts := doc["contexts"].([]any)
			require.Len(t, contexts, 2)
			require.Equal(t, "Orders", contexts[0].(map[string]any)["name"])
			require.Equal(t, "Payments", contexts[1].(map[string]any)["name"])
		})

		t.Run("comments are serialized on all node types", func(t *testing.T) {
			model := &ast.Model{
				Comments: []*ast.Comment{{Text: "# System model"}},
				Name:     "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Invariants: []*ast.Invariant{
									{
										Comments:  []*ast.Comment{{Text: "# Invariant comment"}},
										Name:      "OneThingPerAgg",
										Statement: "An aggregate holds one thing",
									},
								},
								Slices: []*ast.Slice{
									{
										Name:     "S",
										Comments: []*ast.Comment{{Text: "# Slice comment"}},
										Commands: []*ast.Command{
											{
												Comments: []*ast.Comment{{Text: "# Cmd comment"}},
												Name:     "DoThing",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			modelComments := doc["comments"].([]any)
			require.Len(t, modelComments, 1)
			require.Equal(t, "# System model", modelComments[0].(map[string]any)["text"])

			aggregate := firstOf(t, firstOf(t, doc, "contexts"), "aggregates")
			invariant := firstOf(t, aggregate, "invariants")
			require.Equal(t, []any{map[string]any{"text": "# Invariant comment"}}, invariant["comments"])

			s := firstSliceOf(t, doc)
			sliceComments := s["comments"].([]any)
			require.Len(t, sliceComments, 1)
			require.Equal(t, "# Slice comment", sliceComments[0].(map[string]any)["text"])

			cmd := s["commands"].([]any)[0].(map[string]any)
			cmdComments := cmd["comments"].([]any)
			require.Len(t, cmdComments, 1)
			require.Equal(t, "# Cmd comment", cmdComments[0].(map[string]any)["text"])
		})

		t.Run("nil model returns null-JSON", func(t *testing.T) {
			raw, err := export.ExportJSON(nil)
			require.NoError(t, err)
			require.Equal(t, "null", string(raw))
		})

		t.Run("output is valid json with correct key structure", func(t *testing.T) {
			model := &ast.Model{
				Name: "Full",
				Actors: []*ast.Actor{
					{Name: "User"},
				},
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Trigger: &ast.Trigger{
											Kind: "UI", Name: "Form", Actor: "User", Reads: "V",
										},
										Commands: []*ast.Command{
											{Name: "Cmd", Fields: []*ast.Field{{Name: "id", Type: "string"}}},
										},
										Events: []*ast.Event{
											{
												Name:         "Evt",
												Source:       "external",
												ExternalName: "Provider",
											},
										},
										Views: []*ast.View{
											{Name: "MyView", Subscribes: []string{"Evt"}},
										},
										Automations: []*ast.Automation{
											{Name: "Auto", TriggerEvent: "Evt", Command: "DoIt"},
										},
										Flows: []*ast.Flow{
											{CommandName: "Cmd", EventName: "Evt"},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)
			require.True(t, json.Valid(raw))

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			require.Equal(t, "Full", doc["name"])
			require.Len(t, doc["actors"], 1)
			require.Len(t, doc["contexts"], 1)

			c := doc["contexts"].([]any)[0].(map[string]any)
			agg := c["aggregates"].([]any)[0].(map[string]any)
			s := agg["slices"].([]any)[0].(map[string]any)

			require.Equal(t, "Form", s["trigger"].(map[string]any)["name"])
			require.Len(t, s["commands"], 1)
			require.Len(t, s["events"], 1)
			require.Len(t, s["views"], 1)
			require.Len(t, s["automations"], 1)
			require.Len(t, s["flows"], 1)
		})

		t.Run("includes position for model name", func(t *testing.T) {
			model := &ast.Model{
				Name:    "TestModel",
				NamePos: ast.Position{Filename: "test.cue", Line: 1, Column: 1},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			pos := doc["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(1), pos["line"])
			require.Equal(t, float64(1), pos["column"])
		})

		t.Run("includes position for actor name", func(t *testing.T) {
			model := &ast.Model{
				Name: "App",
				Actors: []*ast.Actor{
					{Name: "Guest", NamePos: ast.Position{Filename: "test.cue", Line: 2, Column: 3}},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			actors := doc["actors"].([]any)
			a := actors[0].(map[string]any)
			pos := a["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(2), pos["line"])
			require.Equal(t, float64(3), pos["column"])
		})

		t.Run("includes positions for context name and braces", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name:     "Ctx",
						NamePos:  ast.Position{Filename: "test.cue", Line: 5, Column: 1},
						OpenPos:  ast.Position{Filename: "test.cue", Line: 5, Column: 6},
						ClosePos: ast.Position{Filename: "test.cue", Line: 10, Column: 1},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			ctxs := doc["contexts"].([]any)
			c := ctxs[0].(map[string]any)

			pos := c["position"].(map[string]any)
			require.Equal(t, float64(5), pos["line"])

			openPos := c["open_position"].(map[string]any)
			require.Equal(t, float64(5), openPos["line"])
			require.Equal(t, float64(6), openPos["column"])

			closePos := c["close_position"].(map[string]any)
			require.Equal(t, float64(10), closePos["line"])
		})

		t.Run("includes positions for aggregate name and braces", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name:     "Agg",
								NamePos:  ast.Position{Filename: "test.cue", Line: 3, Column: 3},
								OpenPos:  ast.Position{Filename: "test.cue", Line: 3, Column: 8},
								ClosePos: ast.Position{Filename: "test.cue", Line: 7, Column: 1},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			ctxs := doc["contexts"].([]any)
			c := ctxs[0].(map[string]any)
			aggs := c["aggregates"].([]any)
			agg := aggs[0].(map[string]any)

			require.Equal(t, float64(3), agg["position"].(map[string]any)["line"])
			require.Equal(t, float64(8), agg["open_position"].(map[string]any)["column"])
			require.Equal(t, float64(7), agg["close_position"].(map[string]any)["line"])
		})

		t.Run("includes positions for slice name and braces", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name:     "S",
										NamePos:  ast.Position{Filename: "test.cue", Line: 4, Column: 5},
										OpenPos:  ast.Position{Filename: "test.cue", Line: 4, Column: 8},
										ClosePos: ast.Position{Filename: "test.cue", Line: 8, Column: 3},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			require.Equal(t, float64(4), s["position"].(map[string]any)["line"])
			require.Equal(t, float64(8), s["close_position"].(map[string]any)["line"])
		})

		t.Run("includes positions for command name and braces", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{
												Name:     "Cmd",
												NamePos:  ast.Position{Filename: "test.cue", Line: 5, Column: 7},
												OpenPos:  ast.Position{Filename: "test.cue", Line: 5, Column: 12},
												ClosePos: ast.Position{Filename: "test.cue", Line: 7, Column: 5},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			cmds := s["commands"].([]any)
			cmd := cmds[0].(map[string]any)
			require.Equal(t, float64(5), cmd["position"].(map[string]any)["line"])
			require.Equal(t, float64(12), cmd["open_position"].(map[string]any)["column"])
			require.Equal(t, float64(7), cmd["close_position"].(map[string]any)["line"])
		})

		t.Run("includes positions for event name/source/external and braces", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Events: []*ast.Event{
											{
												Name:            "Evt",
												NamePos:         ast.Position{Filename: "test.cue", Line: 6, Column: 7},
												Source:          "external",
												SourcePos:       ast.Position{Filename: "test.cue", Line: 6, Column: 13},
												ExternalName:    "Provider",
												ExternalNamePos: ast.Position{Filename: "test.cue", Line: 6, Column: 23},
												OpenPos:         ast.Position{Filename: "test.cue", Line: 6, Column: 28},
												ClosePos:        ast.Position{Filename: "test.cue", Line: 8, Column: 5},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			events := s["events"].([]any)
			e := events[0].(map[string]any)

			require.Equal(t, float64(6), e["position"].(map[string]any)["line"])
			require.Equal(t, float64(13), e["source_position"].(map[string]any)["column"])
			require.Equal(t, float64(23), e["external_name_position"].(map[string]any)["column"])
			require.Equal(t, float64(28), e["open_position"].(map[string]any)["column"])
			require.Equal(t, float64(8), e["close_position"].(map[string]any)["line"])
		})

		t.Run("includes positions for field name/type/modifier", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{
												Name: "Cmd",
												Fields: []*ast.Field{
													{
														Name:     "id",
														NamePos:  ast.Position{Filename: "test.cue", Line: 7, Column: 9},
														Type:     "string",
														TypePos:  ast.Position{Filename: "test.cue", Line: 7, Column: 13},
														Modifier: "required",
														ModPos:   ast.Position{Filename: "test.cue", Line: 7, Column: 21},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			cmds := s["commands"].([]any)
			cmd := cmds[0].(map[string]any)
			fields := cmd["fields"].([]any)
			f := fields[0].(map[string]any)

			require.Equal(t, float64(7), f["position"].(map[string]any)["line"])
			require.Equal(t, float64(13), f["type_position"].(map[string]any)["column"])
			require.Equal(t, float64(21), f["modifier_position"].(map[string]any)["column"])
		})

		t.Run("includes positions for flow command and event", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Flows: []*ast.Flow{
											{
												CommandName: "MakeReservation",
												CommandPos:  ast.Position{Filename: "test.cue", Line: 9, Column: 11},
												EventName:   "ReservationMade",
												EventPos:    ast.Position{Filename: "test.cue", Line: 9, Column: 28},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			flows := s["flows"].([]any)
			f := flows[0].(map[string]any)
			require.Equal(t, float64(11), f["command_position"].(map[string]any)["column"])
			require.Equal(t, float64(28), f["event_position"].(map[string]any)["column"])
		})

		t.Run("includes positions for trigger kind/name/actor/reads and braces", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Trigger: &ast.Trigger{
											Kind:     "UI",
											KindPos:  ast.Position{Filename: "test.cue", Line: 3, Column: 3},
											Name:     "Form",
											NamePos:  ast.Position{Filename: "test.cue", Line: 3, Column: 7},
											Actor:    "Guest",
											ActorPos: ast.Position{Filename: "test.cue", Line: 3, Column: 13},
											Reads:    "MyView",
											ReadsPos: ast.Position{Filename: "test.cue", Line: 3, Column: 20},
											OpenPos:  ast.Position{Filename: "test.cue", Line: 3, Column: 28},
											ClosePos: ast.Position{Filename: "test.cue", Line: 5, Column: 3},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			tr := s["trigger"].(map[string]any)
			require.Equal(t, float64(3), tr["kind_position"].(map[string]any)["column"])
			require.Equal(t, float64(7), tr["position"].(map[string]any)["column"])
			require.Equal(t, float64(13), tr["actor_position"].(map[string]any)["column"])
			require.Equal(t, float64(20), tr["reads_position"].(map[string]any)["column"])
			require.Equal(t, float64(28), tr["open_position"].(map[string]any)["column"])
			require.Equal(t, float64(5), tr["close_position"].(map[string]any)["line"])
		})

		t.Run("includes positions for view name and braces", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Views: []*ast.View{
											{
												Name:     "MyView",
												NamePos:  ast.Position{Filename: "test.cue", Line: 4, Column: 5},
												OpenPos:  ast.Position{Filename: "test.cue", Line: 4, Column: 13},
												ClosePos: ast.Position{Filename: "test.cue", Line: 6, Column: 3},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			views := s["views"].([]any)
			v := views[0].(map[string]any)
			require.Equal(t, float64(4), v["position"].(map[string]any)["line"])
			require.Equal(t, float64(6), v["close_position"].(map[string]any)["line"])
		})

		t.Run("includes positions for automation name/trigger/command/target and braces", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Automations: []*ast.Automation{
											{
												Name:             "Auto",
												NamePos:          ast.Position{Filename: "test.cue", Line: 5, Column: 5},
												TriggerEvent:     "Evt",
												TriggerEventPos:  ast.Position{Filename: "test.cue", Line: 5, Column: 11},
												Command:          "DoIt",
												CommandPos:       ast.Position{Filename: "test.cue", Line: 5, Column: 16},
												TargetContext:    "Other",
												TargetContextPos: ast.Position{Filename: "test.cue", Line: 5, Column: 22},
												OpenPos:          ast.Position{Filename: "test.cue", Line: 5, Column: 28},
												ClosePos:         ast.Position{Filename: "test.cue", Line: 7, Column: 3},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			autos := s["automations"].([]any)
			a := autos[0].(map[string]any)
			require.Equal(t, float64(5), a["position"].(map[string]any)["line"])
			require.Equal(t, float64(11), a["trigger_event_position"].(map[string]any)["column"])
			require.Equal(t, float64(16), a["command_position"].(map[string]any)["column"])
			require.Equal(t, float64(22), a["target_context_position"].(map[string]any)["column"])
			require.Equal(t, float64(7), a["close_position"].(map[string]any)["line"])
		})

		t.Run("includes positions for translation name/external/reads/command and braces with nested event", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Translations: []*ast.Translation{
											{
												Name:           "Import",
												NamePos:        ast.Position{Filename: "test.cue", Line: 6, Column: 5},
												ExternalSystem: "API",
												ExternalPos:    ast.Position{Filename: "test.cue", Line: 6, Column: 13},
												Reads:          "V",
												ReadsPos:       ast.Position{Filename: "test.cue", Line: 6, Column: 18},
												Command:        "Do",
												CommandPos:     ast.Position{Filename: "test.cue", Line: 6, Column: 21},
												OpenPos:        ast.Position{Filename: "test.cue", Line: 6, Column: 26},
												ClosePos:       ast.Position{Filename: "test.cue", Line: 9, Column: 3},
												Event: &ast.Event{
													Name:    "Done",
													NamePos: ast.Position{Filename: "test.cue", Line: 7, Column: 7},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			s := firstSliceOf(t, doc)
			trans := s["translations"].([]any)
			tl := trans[0].(map[string]any)
			require.Equal(t, float64(6), tl["position"].(map[string]any)["line"])
			require.Equal(t, float64(13), tl["external_position"].(map[string]any)["column"])
			require.Equal(t, float64(18), tl["reads_position"].(map[string]any)["column"])
			require.Equal(t, float64(21), tl["command_position"].(map[string]any)["column"])
			require.Equal(t, float64(26), tl["open_position"].(map[string]any)["column"])
			require.Equal(t, float64(9), tl["close_position"].(map[string]any)["line"])

			nestedEvent := tl["event"].(map[string]any)
			require.Equal(t, float64(7), nestedEvent["position"].(map[string]any)["line"])
		})

		t.Run("includes position for comments", func(t *testing.T) {
			model := &ast.Model{
				Comments: []*ast.Comment{
					{Text: "# comment", Position: ast.Position{Filename: "test.cue", Line: 1, Column: 1}},
				},
				Name: "Test",
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			comments := doc["comments"].([]any)
			c := comments[0].(map[string]any)
			pos := c["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(1), pos["line"])
			require.Equal(t, float64(1), pos["column"])
		})

		t.Run("zero-value positions are omitted from output", func(t *testing.T) {
			model := &ast.Model{
				Name: "NoPos",
				Actors: []*ast.Actor{
					{Name: "Guest"},
				},
			}

			raw, err := export.ExportJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			_, hasModelPos := doc["position"]
			require.False(t, hasModelPos, "zero-value position should be omitted")

			actors := doc["actors"].([]any)
			a := actors[0].(map[string]any)
			_, hasActorPos := a["position"]
			require.False(t, hasActorPos, "zero-value actor position should be omitted")
		})

		t.Run("carries the description of every construct that accepts one", func(t *testing.T) {
			raw, err := export.ExportJSON(buildFullModel())
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, json.Unmarshal(raw, &doc))

			require.Equal(t, map[string]any{
				"model":             "How the whole system earns its keep",
				"actor":             "The person the system serves",
				"context":           "The language boundary the model lives inside",
				"aggregate":         "The thing decisions are made about",
				"slice":             "One decision and everything it needs",
				"trigger":           "Where the decision starts",
				"command":           "The request the actor makes",
				"event":             "The fact the decision leaves behind",
				"view":              "What the actor reads before deciding",
				"automation":        "The decision nobody has to make by hand",
				"translation":       "How a partner's words become ours",
				"translation event": "The fact a partner reported",
			}, descriptionsByConstruct(t, doc))
		})

		t.Run("omits the description key entirely for a model that describes nothing", func(t *testing.T) {
			raw, err := export.ExportJSON(test.HotelReservationModel(t))
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, json.Unmarshal(raw, &doc))

			require.Empty(t, descriptionsAnywhere(doc),
				"an undescribed model must not carry the key, not even empty-valued")
		})

		t.Run("keeps a description's exact text", func(t *testing.T) {
			raw, err := export.ExportJSON(awkwardlyDescribedModel())
			require.NoError(t, err)
			require.True(t, json.Valid(raw))

			var doc map[string]any
			require.NoError(t, json.Unmarshal(raw, &doc))

			event := firstSliceOf(t, doc)["events"].([]any)[0].(map[string]any)
			require.Equal(t, awkwardDescription, doc["description"])
			require.Equal(t, awkwardDescription, event["description"])
		})

		t.Run("keeps a field named after a keyword under the name it was declared with", func(t *testing.T) {
			raw, err := export.ExportJSON(test.KeywordFieldSearchCatalogModel(t))
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, json.Unmarshal(raw, &doc))

			require.Equal(t, keywordFieldsByOwner, fieldsByOwner(doc))
		})

		t.Run("files every invariant under the scope that declared it, in declaration order", func(t *testing.T) {
			require.Equal(t, libraryLendingInvariants, invariantsByOwner(modelDocOf(t, test.InvariantLibraryLendingModel(t))))
		})
	})

	t.Run("model json with diagnostics", func(t *testing.T) {
		t.Run("empty diagnostics produces diagnostics: []", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			raw, err := export.ExportJSONDiagnostics(model, nil)
			require.NoError(t, err)
			require.True(t, json.Valid(raw))

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			diags, ok := doc["diagnostics"].([]any)
			require.True(t, ok, "diagnostics should be present")
			require.Empty(t, diags, "nil diagnostics should produce empty array")

			modelVal, ok := doc["model"].(map[string]any)
			require.True(t, ok, "model should be present")
			require.Equal(t, "Test", modelVal["name"])
		})

		t.Run("empty diagnostics slice produces diagnostics: []", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			raw, err := export.ExportJSONDiagnostics(model, []*diagnostic.Entry{})
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			diags, ok := doc["diagnostics"].([]any)
			require.True(t, ok)
			require.Empty(t, diags)
		})

		t.Run("single warning diagnostic with rule name", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			diags := []*diagnostic.Entry{
				{
					Filename: "test.cue",
					Line:     10,
					Column:   5,
					Message:  "unused actor",
					Severity: diagnostic.Warning,
					RuleName: "unused-actor",
				},
			}

			raw, err := export.ExportJSONDiagnostics(model, diags)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			d := doc["diagnostics"].([]any)[0].(map[string]any)
			require.Equal(t, "test.cue", d["file"])
			require.Equal(t, float64(10), d["line"])
			require.Equal(t, float64(5), d["column"])
			require.Equal(t, "unused actor", d["message"])
			require.Equal(t, "warning", d["severity"])
			require.Equal(t, "unused-actor", d["rule_name"])
		})

		t.Run("multiple diagnostics with different severities", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			diags := []*diagnostic.Entry{
				{
					Filename: "test.cue",
					Line:     5,
					Column:   1,
					Message:  "syntax error",
					Severity: diagnostic.Error,
				},
				{
					Filename: "test.cue",
					Line:     10,
					Column:   3,
					Message:  "unused field",
					Severity: diagnostic.Warning,
					RuleName: "unused-field",
				},
			}

			raw, err := export.ExportJSONDiagnostics(model, diags)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			diagList := doc["diagnostics"].([]any)
			require.Len(t, diagList, 2)

			d0 := diagList[0].(map[string]any)
			require.Equal(t, "syntax error", d0["message"])
			require.Equal(t, "error", d0["severity"])
			_, hasRule0 := d0["rule_name"]
			require.False(t, hasRule0, "diagnostic without rule_name should omit the field")

			d1 := diagList[1].(map[string]any)
			require.Equal(t, "unused field", d1["message"])
			require.Equal(t, "warning", d1["severity"])
			require.Equal(t, "unused-field", d1["rule_name"])
		})

		t.Run("nil model produces model: null", func(t *testing.T) {
			raw, err := export.ExportJSONDiagnostics(nil, nil)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			diags, ok := doc["diagnostics"].([]any)
			require.True(t, ok)
			require.Empty(t, diags)

			require.Nil(t, doc["model"], "nil model should produce null")
		})

		t.Run("full valid model produces diagnostics: [] with complete model", func(t *testing.T) {
			model := &ast.Model{
				Name: "Full",
				Actors: []*ast.Actor{
					{Name: "User"},
				},
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{Name: "Agg"},
						},
					},
				},
			}

			raw, err := export.ExportJSONDiagnostics(model, nil)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			diags, ok := doc["diagnostics"].([]any)
			require.True(t, ok)
			require.Empty(t, diags)

			modelVal, ok := doc["model"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "Full", modelVal["name"])
			require.Len(t, modelVal["actors"], 1)
			require.Len(t, modelVal["contexts"], 1)

			// Verify ExportJSON is unchanged — bare model JSON is still valid
			bareRaw, err := export.ExportJSON(model)
			require.NoError(t, err)
			var bareDoc map[string]any
			err = json.Unmarshal(bareRaw, &bareDoc)
			require.NoError(t, err)
			require.Equal(t, "Full", bareDoc["name"])
			_, hasDiags := bareDoc["diagnostics"]
			require.False(t, hasDiags, "ExportJSON should not include diagnostics")
		})

		t.Run("diagnostic without rule_name omits rule_name field", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			diags := []*diagnostic.Entry{
				{
					Filename: "test.cue",
					Line:     1,
					Column:   1,
					Message:  "error message",
					Severity: diagnostic.Error,
				},
			}

			raw, err := export.ExportJSONDiagnostics(model, diags)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			d := doc["diagnostics"].([]any)[0].(map[string]any)
			require.Equal(t, "error message", d["message"])
			require.Equal(t, "error", d["severity"])
			_, hasRule := d["rule_name"]
			require.False(t, hasRule, "empty rule_name should be omitted")
		})
	})

	t.Run("diagram json", func(t *testing.T) {
		t.Run("minimal model produces valid JSON with empty arrays", func(t *testing.T) {
			model := &ast.Model{Name: "TestModel"}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)
			require.True(t, json.Valid(raw))

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)
			require.Equal(t, "TestModel", doc["model_name"])

			nodes, ok := doc["nodes"].([]any)
			require.True(t, ok)
			require.Empty(t, nodes)

			edges, ok := doc["edges"].([]any)
			require.True(t, ok)
			require.Empty(t, edges)
		})

		t.Run("full model with all node types produces correct nodes", func(t *testing.T) {
			model := &ast.Model{
				Name: "Full",
				Actors: []*ast.Actor{
					{Name: "User"},
				},
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Trigger: &ast.Trigger{
											Kind: "UI", Name: "Form", Actor: "User", Reads: "V",
										},
										Commands: []*ast.Command{
											{Name: "Cmd"},
										},
										Events: []*ast.Event{
											{Name: "Evt"},
										},
										Views: []*ast.View{
											{Name: "MyView", Subscribes: []string{"Evt"}},
										},
										Automations: []*ast.Automation{
											{Name: "Auto", TriggerEvent: "Evt", Command: "DoIt"},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			require.Len(t, nodes, 9)

			types := make([]string, 0, len(nodes))
			for _, n := range nodes {
				types = append(types, n.(map[string]any)["type"].(string))
			}
			require.Equal(t, []string{"actor", "context", "aggregate", "slice", "command", "event", "trigger", "view", "automation"}, types)
		})

		t.Run("parentId chains are correct for hierarchical nodes", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Actors: []*ast.Actor{
					{Name: "Guest"},
				},
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{Name: "Cmd"},
										},
										Events: []*ast.Event{
											{Name: "Evt"},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)

			// actor-1 and context-1 should have nil parentId
			actor := nodes[0].(map[string]any)
			require.Equal(t, "actor", actor["type"])
			require.Nil(t, actor["parentId"])

			contextNode := nodes[1].(map[string]any)
			require.Equal(t, "context", contextNode["type"])
			require.Nil(t, contextNode["parentId"])

			// aggregate-1 should reference context-1
			aggregate := nodes[2].(map[string]any)
			require.Equal(t, "aggregate", aggregate["type"])
			require.Equal(t, "context-1", aggregate["parentId"])

			// slice-1 should reference aggregate-1
			sliceNode := nodes[3].(map[string]any)
			require.Equal(t, "slice", sliceNode["type"])
			require.Equal(t, "aggregate-1", sliceNode["parentId"])

			// command-1 should reference slice-1
			command := nodes[4].(map[string]any)
			require.Equal(t, "command", command["type"])
			require.Equal(t, "slice-1", command["parentId"])

			// event-1 should reference slice-1
			event := nodes[5].(map[string]any)
			require.Equal(t, "event", event["type"])
			require.Equal(t, "slice-1", event["parentId"])
		})

		t.Run("trigger node with kind/actor/reads", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Trigger: &ast.Trigger{
											Kind:  "UI",
											Name:  "FormSubmit",
											Actor: "Guest",
											Reads: "MyView",
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			// context, aggregate, slice, trigger
			require.Len(t, nodes, 4)

			trigger := nodes[3].(map[string]any)
			require.Equal(t, "trigger", trigger["type"])
			require.Equal(t, "slice-1", trigger["parentId"])
			require.Equal(t, "FormSubmit", trigger["label"])
			require.Equal(t, "UI", trigger["kind"])
			require.Equal(t, "Guest", trigger["actor"])
			require.Equal(t, "MyView", trigger["reads"])
		})

		t.Run("view node with fields and subscribes", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Views: []*ast.View{
											{
												Name: "RoomsView",
												Fields: []*ast.Field{
													{Name: "roomId", Type: "string", Modifier: "required"},
													{Name: "status", Type: "string", Modifier: "optional"},
												},
												Subscribes: []string{"RoomReserved", "GuestCheckedOut"},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			// context, aggregate, slice, view
			require.Len(t, nodes, 4)

			view := nodes[3].(map[string]any)
			require.Equal(t, "view", view["type"])
			require.Equal(t, "slice-1", view["parentId"])
			require.Equal(t, "RoomsView", view["label"])

			fields := view["fields"].([]any)
			require.Len(t, fields, 2)
			f0 := fields[0].(map[string]any)
			require.Equal(t, "roomId", f0["name"])
			require.Equal(t, "string", f0["type"])
			require.Equal(t, "required", f0["modifier"])
			f1 := fields[1].(map[string]any)
			require.Equal(t, "status", f1["name"])
			require.Equal(t, "string", f1["type"])
			require.Equal(t, "optional", f1["modifier"])

			subs := view["subscribes"].([]any)
			require.Len(t, subs, 2)
			require.Equal(t, "RoomReserved", subs[0])
			require.Equal(t, "GuestCheckedOut", subs[1])
		})

		t.Run("automation node with trigger_event/command/target_context", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "Notify",
										Automations: []*ast.Automation{
											{
												Name:          "OrderNotifier",
												TriggerEvent:  "OrderPlaced",
												Command:       "SendNotification",
												TargetContext: "Notifications",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			// context, aggregate, slice, automation
			require.Len(t, nodes, 4)

			auto := nodes[3].(map[string]any)
			require.Equal(t, "automation", auto["type"])
			require.Equal(t, "slice-1", auto["parentId"])
			require.Equal(t, "OrderNotifier", auto["label"])
			require.Equal(t, "OrderPlaced", auto["trigger_event"])
			require.Equal(t, "SendNotification", auto["command"])
			require.Equal(t, "Notifications", auto["target_context"])
		})

		t.Run("translation node with external_system/reads/command and nested event", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "Import",
										Translations: []*ast.Translation{
											{
												Name:           "BookingImport",
												ExternalSystem: "Booking.com API",
												Reads:          "WebhookView",
												Command:        "ImportBooking",
												Event: &ast.Event{
													Name: "BookingImported",
													Fields: []*ast.Field{
														{Name: "bookingId", Type: "string", Modifier: "required"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			// context, aggregate, slice, translation_nested_event, translation
			require.Len(t, nodes, 5)

			trans := nodes[4].(map[string]any)
			require.Equal(t, "translation", trans["type"])
			require.Equal(t, "slice-1", trans["parentId"])
			require.Equal(t, "BookingImport", trans["label"])
			require.Equal(t, "Booking.com API", trans["external_system"])
			require.Equal(t, "WebhookView", trans["reads"])
			require.Equal(t, "ImportBooking", trans["command"])

			nestedEvent := trans["event"].(map[string]any)
			require.Equal(t, "BookingImported", nestedEvent["name"])
			fields := nestedEvent["fields"].([]any)
			require.Len(t, fields, 1)
			require.Equal(t, "bookingId", fields[0].(map[string]any)["name"])
		})

		t.Run("new node types have correct parentId chains", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Trigger: &ast.Trigger{
											Kind: "UI", Name: "Form", Actor: "User", Reads: "V",
										},
										Views: []*ast.View{
											{Name: "RoomsView"},
										},
										Automations: []*ast.Automation{
											{Name: "Auto"},
										},
										Translations: []*ast.Translation{
											{Name: "Import"},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			// context, aggregate, slice, trigger, view, automation, translation
			require.Len(t, nodes, 7)

			// trigger should reference slice-1
			trigger := findNodeByType(nodes, "trigger")
			require.Equal(t, "slice-1", trigger["parentId"])

			// view should reference slice-1
			view := findNodeByType(nodes, "view")
			require.Equal(t, "slice-1", view["parentId"])

			// automation should reference slice-1
			auto := findNodeByType(nodes, "automation")
			require.Equal(t, "slice-1", auto["parentId"])

			// translation should reference slice-1
			trans := findNodeByType(nodes, "translation")
			require.Equal(t, "slice-1", trans["parentId"])
		})

		t.Run("edges are created from flows", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{Name: "MakeReservation"},
										},
										Events: []*ast.Event{
											{Name: "ReservationMade"},
										},
										Flows: []*ast.Flow{
											{
												CommandName: "MakeReservation",
												EventName:   "ReservationMade",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			require.Len(t, edges, 1)

			edge := edges[0].(map[string]any)
			require.Equal(t, "command-1", edge["source"])
			require.Equal(t, "event-1", edge["target"])
			require.Equal(t, "flow", edge["type"])
		})

		t.Run("trigger_command edges from trigger to each command in same slice", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Trigger: &ast.Trigger{
											Kind: "UI", Name: "FormSubmit",
										},
										Commands: []*ast.Command{
											{Name: "MakeReservation"},
											{Name: "NotifyGuest"},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			require.Len(t, edges, 2)

			// Collect edges by target for verification
			triggerCmdEdges := make([]map[string]any, 0, 2)
			for _, e := range edges {
				edge := e.(map[string]any)
				require.Equal(t, "trigger_command", edge["type"])
				require.Equal(t, "trigger-1", edge["source"])
				triggerCmdEdges = append(triggerCmdEdges, edge)
			}

			targets := make([]string, len(triggerCmdEdges))
			for i, e := range triggerCmdEdges {
				targets[i] = e["target"].(string)
			}
			require.Contains(t, targets, "command-1")
			require.Contains(t, targets, "command-2")
		})

		t.Run("subscription edges from event to subscribing view", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Events: []*ast.Event{
											{Name: "ReservationMade"},
										},
										Views: []*ast.View{
											{
												Name:       "RoomsView",
												Subscribes: []string{"ReservationMade"},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			require.Len(t, edges, 1)

			edge := edges[0].(map[string]any)
			require.Equal(t, "event-1", edge["source"])
			require.Equal(t, "view-1", edge["target"])
			require.Equal(t, "subscription", edge["type"])
		})

		t.Run("automation_trigger edges from event to automation", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Events: []*ast.Event{
											{Name: "OrderPlaced"},
										},
										Automations: []*ast.Automation{
											{
												Name:         "OrderNotifier",
												TriggerEvent: "OrderPlaced",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			require.Len(t, edges, 1)

			edge := edges[0].(map[string]any)
			require.Equal(t, "event-1", edge["source"])
			require.Equal(t, "auto-1", edge["target"])
			require.Equal(t, "automation_trigger", edge["type"])
		})

		t.Run("automation_command edges from automation to command", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{Name: "SendNotification"},
										},
										Automations: []*ast.Automation{
											{
												Name:    "OrderNotifier",
												Command: "SendNotification",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			require.Len(t, edges, 1)

			edge := edges[0].(map[string]any)
			require.Equal(t, "auto-1", edge["source"])
			require.Equal(t, "command-1", edge["target"])
			require.Equal(t, "automation_command", edge["type"])
		})

		t.Run("reads, translation_command and implicit flow edges from translation", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "Import",
										Views: []*ast.View{
											{Name: "WebhookView"},
										},
										Commands: []*ast.Command{
											{Name: "ImportBooking"},
										},
										// No standalone Events — the translation's nested event creates the event node
										Translations: []*ast.Translation{
											{
												Name:           "BookingImport",
												ExternalSystem: "Booking.com API",
												Reads:          "WebhookView",
												Command:        "ImportBooking",
												Event: &ast.Event{
													Name: "BookingImported",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			require.Len(t, edges, 3)

			var readsEdge, cmdEdge, flowEdge map[string]any
			for _, e := range edges {
				edge := e.(map[string]any)
				switch edge["type"] {
				case "reads":
					readsEdge = edge
				case "translation_command":
					cmdEdge = edge
				case "flow":
					flowEdge = edge
				}
			}
			require.NotNil(t, readsEdge, "should have a reads edge")
			require.Equal(t, "view-1", readsEdge["source"])
			require.Equal(t, "trans-1", readsEdge["target"])

			require.NotNil(t, cmdEdge, "should have a translation_command edge")
			require.Equal(t, "trans-1", cmdEdge["source"])
			require.Equal(t, "command-1", cmdEdge["target"])

			require.NotNil(t, flowEdge, "should have an implicit flow edge for command→event")
			require.Equal(t, "command-1", flowEdge["source"])
			// Nested event node created by the translation (event-1 in Pass 1 order)
			require.Equal(t, "event-1", flowEdge["target"])
		})

		t.Run("cross-slice subscription edge resolves across boundaries", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Order",
								Slices: []*ast.Slice{
									{
										Name: "Place Order",
										Events: []*ast.Event{
											{Name: "OrderPlaced"},
										},
									},
								},
							},
						},
					},
					{
						Name: "Notifications",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Notification",
								Slices: []*ast.Slice{
									{
										Name: "Show Notifications",
										Views: []*ast.View{
											{
												Name:       "NotificationView",
												Subscribes: []string{"OrderPlaced"},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			require.Len(t, edges, 1)

			edge := edges[0].(map[string]any)
			require.Equal(t, "event-1", edge["source"])
			require.Equal(t, "view-1", edge["target"])
			require.Equal(t, "subscription", edge["type"])
		})

		t.Run("cross-slice automation_command resolves across boundaries", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Order",
								Slices: []*ast.Slice{
									{
										Name: "Process Order",
										Automations: []*ast.Automation{
											{
												Name:    "OrderNotifier",
												Command: "SendNotification",
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "Notifications",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Notification",
								Slices: []*ast.Slice{
									{
										Name: "Send",
										Commands: []*ast.Command{
											{Name: "SendNotification"},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			require.Len(t, edges, 1)

			edge := edges[0].(map[string]any)
			require.Equal(t, "auto-1", edge["source"])
			require.Equal(t, "command-1", edge["target"])
			require.Equal(t, "automation_command", edge["type"])
		})

		t.Run("unresolved name references silently skipped", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{Name: "ExistingCmd"},
										},
										Events: []*ast.Event{
											{Name: "ExistingEvent"},
										},
										Flows: []*ast.Flow{
											{CommandName: "ExistingCmd", EventName: "ExistingEvent"},
										},
										Views: []*ast.View{
											{
												Name:       "MyView",
												Subscribes: []string{"NonExistentEvent"},
											},
										},
										Automations: []*ast.Automation{
											{
												Name:         "Auto",
												TriggerEvent: "NonExistentEvent",
												Command:      "NonExistentCmd",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			// Only the flow edge should be emitted (command→event)
			// Subscription references non-existent event → silently skipped
			// Automation trigger references non-existent event → silently skipped
			// Automation command references non-existent command → silently skipped
			edges := doc["edges"].([]any)
			require.Len(t, edges, 1)

			edge := edges[0].(map[string]any)
			require.Equal(t, "command-1", edge["source"])
			require.Equal(t, "event-1", edge["target"])
			require.Equal(t, "flow", edge["type"])
		})

		t.Run("two-pass resolution produces all edge types in complex model", func(t *testing.T) {
			model := &ast.Model{
				Name: "Full",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Trigger: &ast.Trigger{
											Kind: "UI", Name: "Form", Actor: "User", Reads: "V",
										},
										Commands: []*ast.Command{
											{Name: "Cmd"},
										},
										Events: []*ast.Event{
											{Name: "Evt"},
										},
										Views: []*ast.View{
											{Name: "MyView", Subscribes: []string{"Evt"}},
										},
										Automations: []*ast.Automation{
											{Name: "Auto", TriggerEvent: "Evt", Command: "Cmd"},
										},
										Flows: []*ast.Flow{
											{CommandName: "Cmd", EventName: "Evt"},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			edges := doc["edges"].([]any)
			// Expected: flow, trigger_command, subscription, automation_trigger, automation_command
			require.Len(t, edges, 5)

			edgeTypes := make(map[string]bool)
			for _, e := range edges {
				edge := e.(map[string]any)
				edgeTypes[edge["type"].(string)] = true
			}
			require.True(t, edgeTypes["flow"], "flow edge should exist")
			require.True(t, edgeTypes["trigger_command"], "trigger_command edge should exist")
			require.True(t, edgeTypes["subscription"], "subscription edge should exist")
			require.True(t, edgeTypes["automation_trigger"], "automation_trigger edge should exist")
			require.True(t, edgeTypes["automation_command"], "automation_command edge should exist")
		})

		t.Run("command and event nodes include fields array", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{
												Name: "MakeReservation",
												Fields: []*ast.Field{
													{Name: "guestId", Type: "string", Modifier: "required"},
												},
											},
										},
										Events: []*ast.Event{
											{
												Name: "ReservationMade",
												Fields: []*ast.Field{
													{Name: "roomId", Type: "string"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)

			var command, event map[string]any
			for _, n := range nodes {
				node := n.(map[string]any)
				switch node["type"] {
				case "command":
					command = node
				case "event":
					event = node
				}
			}
			require.NotNil(t, command, "command node should exist")
			require.NotNil(t, event, "event node should exist")

			cmdFields := command["fields"].([]any)
			require.Len(t, cmdFields, 1)
			f0 := cmdFields[0].(map[string]any)
			require.Equal(t, "guestId", f0["name"])
			require.Equal(t, "string", f0["type"])
			require.Equal(t, "required", f0["modifier"])

			evtFields := event["fields"].([]any)
			require.Len(t, evtFields, 1)
			f1 := evtFields[0].(map[string]any)
			require.Equal(t, "roomId", f1["name"])
			require.Equal(t, "string", f1["type"])
			_, hasMod := f1["modifier"]
			require.False(t, hasMod, "empty modifier should be omitted")
		})

		t.Run("nil model returns null JSON", func(t *testing.T) {
			raw, err := export.ExportDiagramJSON(nil)
			require.NoError(t, err)
			require.Equal(t, "null", string(raw))
		})

		t.Run("model with empty slices produces correct nodes and edges", func(t *testing.T) {
			model := &ast.Model{
				Name: "Empty",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name:   "Agg",
								Slices: []*ast.Slice{},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			require.Len(t, nodes, 2) // context and aggregate only

			edges := doc["edges"].([]any)
			require.Empty(t, edges)
		})

		t.Run("includes position for actor node", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Actors: []*ast.Actor{
					{Name: "Guest", NamePos: ast.Position{Filename: "test.cue", Line: 2, Column: 3}},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			require.Len(t, nodes, 1)
			node := nodes[0].(map[string]any)
			require.Equal(t, "actor", node["type"])

			pos := node["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(2), pos["line"])
			require.Equal(t, float64(3), pos["column"])
		})

		t.Run("includes position for context node", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name:    "Ctx",
						NamePos: ast.Position{Filename: "test.cue", Line: 5, Column: 1},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			require.Len(t, nodes, 1)
			node := nodes[0].(map[string]any)
			require.Equal(t, "context", node["type"])

			pos := node["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(5), pos["line"])
			require.Equal(t, float64(1), pos["column"])
		})

		t.Run("includes position for aggregate node", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name:    "Agg",
								NamePos: ast.Position{Filename: "test.cue", Line: 3, Column: 3},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			require.Len(t, nodes, 2)
			agg := nodes[1].(map[string]any)
			require.Equal(t, "aggregate", agg["type"])

			pos := agg["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(3), pos["line"])
			require.Equal(t, float64(3), pos["column"])
		})

		t.Run("includes position for slice node", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name:    "S",
										NamePos: ast.Position{Filename: "test.cue", Line: 4, Column: 5},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			// context, aggregate, slice
			require.Len(t, nodes, 3)
			slice := nodes[2].(map[string]any)
			require.Equal(t, "slice", slice["type"])

			pos := slice["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(4), pos["line"])
			require.Equal(t, float64(5), pos["column"])
		})

		t.Run("includes position for command node", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{
												Name:    "Cmd",
												NamePos: ast.Position{Filename: "test.cue", Line: 5, Column: 7},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			// context, aggregate, slice, command
			require.Len(t, nodes, 4)
			cmd := nodes[3].(map[string]any)
			require.Equal(t, "command", cmd["type"])

			pos := cmd["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(5), pos["line"])
			require.Equal(t, float64(7), pos["column"])
		})

		t.Run("includes position for event node", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Events: []*ast.Event{
											{
												Name:    "Evt",
												NamePos: ast.Position{Filename: "test.cue", Line: 6, Column: 7},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			nodes := doc["nodes"].([]any)
			// context, aggregate, slice, event
			require.Len(t, nodes, 4)
			evt := nodes[3].(map[string]any)
			require.Equal(t, "event", evt["type"])

			pos := evt["position"].(map[string]any)
			require.Equal(t, "test.cue", pos["filename"])
			require.Equal(t, float64(6), pos["line"])
			require.Equal(t, float64(7), pos["column"])
		})

		t.Run("zero-value position is omitted from diagram nodes", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Actors: []*ast.Actor{
					{Name: "Guest"},
				},
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Agg",
								Slices: []*ast.Slice{
									{
										Name: "S",
										Commands: []*ast.Command{
											{Name: "Cmd"},
										},
										Events: []*ast.Event{
											{Name: "Evt"},
										},
									},
								},
							},
						},
					},
				},
			}

			raw, err := export.ExportDiagramJSON(model)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			for _, n := range doc["nodes"].([]any) {
				node := n.(map[string]any)
				_, hasPos := node["position"]
				require.False(t, hasPos, "node %s (%s) should not have position", node["id"], node["type"])
			}
		})

		t.Run("a described model still produces a document free of prose", func(t *testing.T) {
			raw, err := export.ExportDiagramJSON(buildFullModel())
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, json.Unmarshal(raw, &doc))

			require.Empty(t, descriptionsAnywhere(doc),
				"the diagram document is nodes and edges; descriptions belong to the model export")
		})

		t.Run("fields named after keywords reach the node field lists and nothing else", func(t *testing.T) {
			keywordNamed := diagramDocOf(t, test.KeywordFieldSearchCatalogModel(t))
			ordinaryNamed := diagramDocOf(t, test.WithOrdinaryFieldNames(test.KeywordFieldSearchCatalogModel(t)))

			require.Equal(t, keywordFieldsByOwner, diagramFieldsByLabel(keywordNamed))

			translation := findNodeByType(keywordNamed["nodes"].([]any), "translation")
			nestedEvent := translation["event"].(map[string]any)
			require.Equal(t, keywordFieldsByOwner["VendorSearchImported"],
				fieldSpecs(nestedEvent["fields"].([]any)))

			require.NotEqual(t, diagramFieldsByLabel(keywordNamed), diagramFieldsByLabel(ordinaryNamed),
				"the twin has to be named differently, or the comparison below says nothing")
			require.Equal(t, withoutFieldLists(ordinaryNamed), withoutFieldLists(keywordNamed),
				"a field name belongs to the field list it was declared in; no id, label, edge or metadata is drawn from it")
		})

		t.Run("a model declaring invariants still produces a document free of them", func(t *testing.T) {
			model := test.InvariantLibraryLendingModel(t)

			require.ElementsMatch(t, libraryLendingInvariantText(), invariantTextAnywhere(modelDocOf(t, model)),
				"the search has to find invariants where they do belong, or finding none below says nothing")
			require.Empty(t, invariantTextAnywhere(diagramDocOf(t, model)),
				"the diagram document is nodes and edges; an invariant is neither")
		})
	})

	t.Run("diagram json with diagnostics", func(t *testing.T) {
		t.Run("empty diagnostics produces empty array with diagram", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			raw, err := export.ExportDiagramJSONDiagnostics(model, nil)
			require.NoError(t, err)
			require.True(t, json.Valid(raw))

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			diags, ok := doc["diagnostics"].([]any)
			require.True(t, ok, "diagnostics should be present")
			require.Empty(t, diags, "nil diagnostics should produce empty array")

			diagram, ok := doc["diagram"].(map[string]any)
			require.True(t, ok, "diagram should be present")
			require.Equal(t, "Test", diagram["model_name"])
		})

		t.Run("single warning diagnostic with rule name", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			diags := []*diagnostic.Entry{
				{
					Filename: "test.cue",
					Line:     10,
					Column:   5,
					Message:  "unused actor",
					Severity: diagnostic.Warning,
					RuleName: "unused-actor",
				},
			}

			raw, err := export.ExportDiagramJSONDiagnostics(model, diags)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			d := doc["diagnostics"].([]any)[0].(map[string]any)
			require.Equal(t, "test.cue", d["file"])
			require.Equal(t, float64(10), d["line"])
			require.Equal(t, float64(5), d["column"])
			require.Equal(t, "unused actor", d["message"])
			require.Equal(t, "warning", d["severity"])
			require.Equal(t, "unused-actor", d["rule_name"])
		})

		t.Run("multiple diagnostics with different severities", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			diags := []*diagnostic.Entry{
				{
					Filename: "test.cue",
					Line:     5,
					Column:   1,
					Message:  "syntax error",
					Severity: diagnostic.Error,
				},
				{
					Filename: "test.cue",
					Line:     10,
					Column:   3,
					Message:  "unused field",
					Severity: diagnostic.Warning,
					RuleName: "unused-field",
				},
			}

			raw, err := export.ExportDiagramJSONDiagnostics(model, diags)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			diagList := doc["diagnostics"].([]any)
			require.Len(t, diagList, 2)

			d0 := diagList[0].(map[string]any)
			require.Equal(t, "syntax error", d0["message"])
			require.Equal(t, "error", d0["severity"])
			_, hasRule0 := d0["rule_name"]
			require.False(t, hasRule0, "diagnostic without rule_name should omit the field")

			d1 := diagList[1].(map[string]any)
			require.Equal(t, "unused field", d1["message"])
			require.Equal(t, "warning", d1["severity"])
			require.Equal(t, "unused-field", d1["rule_name"])
		})

		t.Run("nil model produces diagram null", func(t *testing.T) {
			raw, err := export.ExportDiagramJSONDiagnostics(nil, nil)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			diags, ok := doc["diagnostics"].([]any)
			require.True(t, ok)
			require.Empty(t, diags)

			require.Nil(t, doc["diagram"], "nil model should produce null diagram")
		})

		t.Run("diagnostic without rule_name omits rule_name field", func(t *testing.T) {
			model := &ast.Model{Name: "Test"}
			diags := []*diagnostic.Entry{
				{
					Filename: "test.cue",
					Line:     1,
					Column:   1,
					Message:  "error message",
					Severity: diagnostic.Error,
				},
			}

			raw, err := export.ExportDiagramJSONDiagnostics(model, diags)
			require.NoError(t, err)

			var doc map[string]any
			err = json.Unmarshal(raw, &doc)
			require.NoError(t, err)

			d := doc["diagnostics"].([]any)[0].(map[string]any)
			require.Equal(t, "error message", d["message"])
			require.Equal(t, "error", d["severity"])
			_, hasRule := d["rule_name"]
			require.False(t, hasRule, "empty rule_name should be omitted")
		})
	})

	t.Run("cue", func(t *testing.T) {
		t.Run("exports minimal model with name only", func(t *testing.T) {
			model := &ast.Model{Name: "TestModel"}

			raw, err := export.ExportCUE(model)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, `name: "TestModel"`)
			require.NotContains(t, output, "actors")
			require.NotContains(t, output, "contexts")
		})

		t.Run("handles nil model without panicking", func(t *testing.T) {
			raw, err := export.ExportCUE(nil)
			require.NoError(t, err)
			require.Empty(t, string(raw))
		})

		t.Run("field modifiers are omitted when unset", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name: "S",
							Commands: []*ast.Command{{
								Name: "Cmd",
								Fields: []*ast.Field{
									{Name: "name", Type: "string"},
									{Name: "age", Type: "int", Modifier: "optional"},
								},
							}},
						}},
					}},
				}},
			}

			raw, err := export.ExportCUE(model)
			require.NoError(t, err)

			require.NotContains(t, string(raw), `modifier: ""`)
			require.Contains(t, string(raw), `modifier: "optional"`)
		})

		t.Run("descriptions are omitted when a model describes nothing", func(t *testing.T) {
			raw, err := export.ExportCUE(test.HotelReservationModel(t))
			require.NoError(t, err)

			require.NotContains(t, string(raw), "description",
				"an undescribed model must not carry the key, not even empty-valued")
		})

		// The exported CUE is only useful if the cue tool can read it back, so the
		// structural assertions below decode it with the real compiler instead of
		// matching text fragments, which pass wherever a fragment happens to sit.
		t.Run("decoded structure", func(t *testing.T) {
			t.Run("carries actors in declaration order", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Name:   "App",
					Actors: []*ast.Actor{{Name: "Guest"}, {Name: "Admin"}},
				}

				doc := exportedCUEDoc(t, cueBin, model)

				require.Equal(t, "App", doc["name"])
				require.Equal(t, []any{
					map[string]any{"name": "Guest"},
					map[string]any{"name": "Admin"},
				}, doc["actors"])
			})

			t.Run("nests a command with its typed fields inside slice, aggregate and context", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Name: "Hotel",
					Contexts: []*ast.Context{{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{{
							Name: "Reservation",
							Slices: []*ast.Slice{{
								Name: "Make Reservation",
								Commands: []*ast.Command{{
									Name: "MakeReservation",
									Fields: []*ast.Field{
										{Name: "guestId", Type: "string", Modifier: "required"},
										{Name: "roomType", Type: "string", Modifier: "required"},
									},
								}},
							}},
						}},
					}},
				}

				slice := firstSliceOf(t, exportedCUEDoc(t, cueBin, model))

				require.Equal(t, "Make Reservation", slice["name"])
				require.Equal(t, []any{map[string]any{
					"name": "MakeReservation",
					"fields": []any{
						map[string]any{"name": "guestId", "type": "string", "modifier": "required"},
						map[string]any{"name": "roomType", "type": "string", "modifier": "required"},
					},
				}}, slice["commands"])
			})

			t.Run("nests a trigger under its slice", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Name: "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Slices: []*ast.Slice{{
								Name: "My Slice",
								Trigger: &ast.Trigger{
									Kind:  "UI",
									Name:  "Form",
									Actor: "Guest",
									Reads: "MyView",
								},
							}},
						}},
					}},
				}

				slice := firstSliceOf(t, exportedCUEDoc(t, cueBin, model))

				require.Equal(t, map[string]any{
					"kind":  "UI",
					"name":  "Form",
					"actor": "Guest",
					"reads": "MyView",
				}, slice["trigger"])
			})

			t.Run("keeps an external event's source and provider name", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Name: "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Slices: []*ast.Slice{{
								Name: "Receive",
								Events: []*ast.Event{{
									Name:         "PaymentReceived",
									Source:       "external",
									ExternalName: "Stripe",
									Fields: []*ast.Field{
										{Name: "paymentId", Type: "string", Modifier: "required"},
									},
								}},
							}},
						}},
					}},
				}

				slice := firstSliceOf(t, exportedCUEDoc(t, cueBin, model))

				require.Equal(t, []any{map[string]any{
					"name":          "PaymentReceived",
					"source":        "external",
					"external_name": "Stripe",
					"fields": []any{
						map[string]any{"name": "paymentId", "type": "string", "modifier": "required"},
					},
				}}, slice["events"])
			})

			t.Run("keeps a view's fields and subscriptions", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Name: "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Slices: []*ast.Slice{{
								Name: "S",
								Views: []*ast.View{{
									Name: "RoomsView",
									Fields: []*ast.Field{
										{Name: "roomId", Type: "string", Modifier: "required"},
										{Name: "status", Type: "string", Modifier: "optional"},
									},
									Subscribes: []string{"RoomReserved", "GuestCheckedOut"},
								}},
							}},
						}},
					}},
				}

				slice := firstSliceOf(t, exportedCUEDoc(t, cueBin, model))

				require.Equal(t, []any{map[string]any{
					"name": "RoomsView",
					"fields": []any{
						map[string]any{"name": "roomId", "type": "string", "modifier": "required"},
						map[string]any{"name": "status", "type": "string", "modifier": "optional"},
					},
					"subscribes": []any{"RoomReserved", "GuestCheckedOut"},
				}}, slice["views"])
			})

			t.Run("keeps an automation's trigger, command and target context", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Name: "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Slices: []*ast.Slice{{
								Name: "Notify",
								Automations: []*ast.Automation{{
									Name:          "OrderNotifier",
									TriggerEvent:  "OrderPlaced",
									Command:       "SendNotification",
									TargetContext: "Notifications",
								}},
							}},
						}},
					}},
				}

				slice := firstSliceOf(t, exportedCUEDoc(t, cueBin, model))

				require.Equal(t, []any{map[string]any{
					"name":           "OrderNotifier",
					"trigger_event":  "OrderPlaced",
					"command":        "SendNotification",
					"target_context": "Notifications",
				}}, slice["automations"])
			})

			t.Run("nests a translation's event inside the translation", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Name: "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Slices: []*ast.Slice{{
								Name: "Import",
								Translations: []*ast.Translation{{
									Name:           "BookingImport",
									ExternalSystem: "Booking.com API",
									Reads:          "WebhookView",
									Command:        "ImportBooking",
									Event: &ast.Event{
										Name: "BookingImported",
										Fields: []*ast.Field{
											{Name: "bookingId", Type: "string", Modifier: "required"},
										},
									},
								}},
							}},
						}},
					}},
				}

				slice := firstSliceOf(t, exportedCUEDoc(t, cueBin, model))

				require.Equal(t, []any{map[string]any{
					"name":            "BookingImport",
					"external_system": "Booking.com API",
					"reads":           "WebhookView",
					"command":         "ImportBooking",
					"event": map[string]any{
						"name": "BookingImported",
						"fields": []any{
							map[string]any{"name": "bookingId", "type": "string", "modifier": "required"},
						},
					},
				}}, slice["translations"])
			})

			t.Run("keeps a flow's command and event names", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Name: "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Slices: []*ast.Slice{{
								Name: "S",
								Flows: []*ast.Flow{{
									CommandName: "MakeReservation",
									EventName:   "ReservationMade",
								}},
							}},
						}},
					}},
				}

				slice := firstSliceOf(t, exportedCUEDoc(t, cueBin, model))

				require.Equal(t, []any{map[string]any{
					"command_name": "MakeReservation",
					"event_name":   "ReservationMade",
				}}, slice["flows"])
			})

			t.Run("keeps a description's exact text", func(t *testing.T) {
				cueBin := lookupCue(t)

				doc := exportedCUEDoc(t, cueBin, awkwardlyDescribedModel())

				event := firstSliceOf(t, doc)["events"].([]any)[0].(map[string]any)
				require.Equal(t, awkwardDescription, doc["description"])
				require.Equal(t, awkwardDescription, event["description"])
			})

			t.Run("keeps comments attached to the element they document", func(t *testing.T) {
				cueBin := lookupCue(t)
				model := &ast.Model{
					Comments: []*ast.Comment{{Text: "# System model"}},
					Name:     "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Invariants: []*ast.Invariant{{
								Comments:  []*ast.Comment{{Text: "# Invariant comment"}},
								Name:      "OneThingPerAgg",
								Statement: "An aggregate holds one thing",
							}},
							Slices: []*ast.Slice{{
								Name:     "S",
								Comments: []*ast.Comment{{Text: "# Slice comment"}},
								Commands: []*ast.Command{{
									Comments: []*ast.Comment{{Text: "# Cmd comment"}},
									Name:     "DoThing",
								}},
							}},
						}},
					}},
				}

				doc := exportedCUEDoc(t, cueBin, model)
				aggregate := firstOf(t, firstOf(t, doc, "contexts"), "aggregates")
				slice := firstSliceOf(t, doc)

				require.Equal(t, []any{map[string]any{"text": "# System model"}}, doc["comments"])
				require.Equal(t, []any{map[string]any{"text": "# Invariant comment"}},
					firstOf(t, aggregate, "invariants")["comments"])
				require.Equal(t, []any{map[string]any{"text": "# Slice comment"}}, slice["comments"])
				require.Equal(t, []any{map[string]any{"text": "# Cmd comment"}},
					slice["commands"].([]any)[0].(map[string]any)["comments"])
			})
		})

		t.Run("output conforms to the schema's Model definition", func(t *testing.T) {
			requireConformsToSchema(t, lookupCue(t), buildFullModel())
		})

		t.Run("a model whose fields are named after keywords conforms to the schema's Model definition", func(t *testing.T) {
			requireConformsToSchema(t, lookupCue(t), test.KeywordFieldSearchCatalogModel(t))
		})

		t.Run("a model declaring invariants conforms to the schema's Model definition", func(t *testing.T) {
			requireConformsToSchema(t, lookupCue(t), test.InvariantLibraryLendingModel(t))
		})

		t.Run("CUE and JSON exports describe the same model", func(t *testing.T) {
			requireBothFormatsAgree(t, lookupCue(t), buildFullModel())
		})

		t.Run("CUE and JSON exports agree on fields named after keywords", func(t *testing.T) {
			cueBin := lookupCue(t)

			requireBothFormatsAgree(t, cueBin, test.KeywordFieldSearchCatalogModel(t))

			doc := exportedCUEDoc(t, cueBin, test.KeywordFieldSearchCatalogModel(t))
			require.Equal(t, keywordFieldsByOwner, fieldsByOwner(doc))
		})

		t.Run("CUE and JSON exports agree on the invariants a model declares", func(t *testing.T) {
			requireBothFormatsAgree(t, lookupCue(t), test.InvariantLibraryLendingModel(t))
		})
	})
}

// findNodeByType finds a diagram node by its type field in a nodes array.
func findNodeByType(nodes []any, typ string) map[string]any {
	for _, n := range nodes {
		node := n.(map[string]any)
		if node["type"] == typ {
			return node
		}
	}
	return nil
}

// lookupCue returns the cue binary to shell out to, skipping the test when the
// tool is not installed. The subtests that use it check generated CUE against
// the real schema, which needs the actual compiler rather than a stand-in.
func lookupCue(t *testing.T) string {
	t.Helper()
	cueBin, err := exec.LookPath("cue")
	if err != nil {
		t.Skip("cue not installed; skipping schema conformance check")
	}
	return cueBin
}

// exportedCUEDoc exports model as CUE and decodes it with the cue tool, so the
// assertions run against the structure a consumer would read rather than the
// text layout.
func exportedCUEDoc(t *testing.T, cueBin string, model *ast.Model) map[string]any {
	t.Helper()

	raw, err := export.ExportCUE(model)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(cueExportJSON(t, cueBin, raw), &doc))

	return doc
}

// cueExportJSON converts exported CUE into JSON using the cue tool.
func cueExportJSON(t *testing.T, cueBin string, cueSource []byte) []byte {
	t.Helper()

	path := filepath.Join(t.TempDir(), "exported.cue")
	require.NoError(t, os.WriteFile(path, cueSource, 0644))

	out, err := exec.Command(cueBin, "export", path, "--out", "json").CombinedOutput()
	require.NoError(t, err, "cue export failed: %s\nCUE input:\n%s", out, cueSource)

	return out
}

func firstOf(t *testing.T, parent map[string]any, field string) map[string]any {
	t.Helper()

	items, ok := parent[field].([]any)
	require.True(t, ok && len(items) > 0, "no %s in %v", field, parent)

	return items[0].(map[string]any)
}

func firstSliceOf(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()

	context := firstOf(t, doc, "contexts")
	aggregate := firstOf(t, context, "aggregates")

	return firstOf(t, aggregate, "slices")
}

// buildFullModel returns a model with all node types populated, each carrying a
// distinct description so a description filed under the wrong construct shows up
// as a mismatch rather than passing unnoticed.
func buildFullModel() *ast.Model {
	return &ast.Model{
		Name:        "Full",
		Description: "How the whole system earns its keep",
		Actors: []*ast.Actor{
			{Name: "User", Description: "The person the system serves"},
		},
		Contexts: []*ast.Context{
			{
				Name:        "Ctx",
				Description: "The language boundary the model lives inside",
				Aggregates: []*ast.Aggregate{
					{
						Name:        "Agg",
						Description: "The thing decisions are made about",
						Slices: []*ast.Slice{
							{
								Name:        "S",
								Description: "One decision and everything it needs",
								Trigger: &ast.Trigger{
									Kind: "UI", Name: "Form", Actor: "User", Reads: "V",
									Description: "Where the decision starts",
								},
								Commands: []*ast.Command{
									{
										Name:        "Cmd",
										Description: "The request the actor makes",
										Fields:      []*ast.Field{{Name: "id", Type: "string"}},
									},
								},
								Events: []*ast.Event{
									{
										Name:         "Evt",
										Description:  "The fact the decision leaves behind",
										Source:       "external",
										ExternalName: "Provider",
									},
								},
								Views: []*ast.View{
									{
										Name:        "MyView",
										Description: "What the actor reads before deciding",
										Subscribes:  []string{"Evt"},
									},
								},
								Automations: []*ast.Automation{
									{
										Name:         "Auto",
										Description:  "The decision nobody has to make by hand",
										TriggerEvent: "Evt",
										Command:      "DoIt",
									},
								},
								Translations: []*ast.Translation{
									{
										Name:           "Trans",
										Description:    "How a partner's words become ours",
										ExternalSystem: "Partner API",
										Reads:          "MyView",
										Command:        "Cmd",
										Event: &ast.Event{
											Name:        "PartnerEvt",
											Description: "The fact a partner reported",
										},
									},
								},
								Flows: []*ast.Flow{
									{CommandName: "Cmd", EventName: "Evt"},
								},
							},
						},
					},
				},
			},
		},
	}
}

// awkwardDescription mixes the characters both formats have to escape with text
// outside ASCII.
const awkwardDescription = `Guest said "hold it" \ paid 50% — café`

func awkwardlyDescribedModel() *ast.Model {
	return &ast.Model{
		Name:        "Awkward",
		Description: awkwardDescription,
		Contexts: []*ast.Context{{
			Name: "Ctx",
			Aggregates: []*ast.Aggregate{{
				Name: "Agg",
				Slices: []*ast.Slice{{
					Name: "S",
					Events: []*ast.Event{{
						Name:        "Evt",
						Description: awkwardDescription,
					}},
				}},
			}},
		}},
	}
}

// keywordFieldsByOwner transcribes what test.KeywordFieldSearchCatalog declares,
// so an export is read back against the source an author wrote rather than
// against another export of it.
var keywordFieldsByOwner = map[string][]map[string]any{
	"DefineSavedSearch": {
		{"name": "model", "type": "string", "modifier": "required"},
		{"name": "source", "type": "string", "modifier": "required"},
		{"name": "where", "type": "string", "modifier": "required"},
		{"name": "and", "type": "string"},
		{"name": "not", "type": "string"},
		{"name": "fields", "type": "string", "modifier": "required"},
		{"name": "description", "type": "string", "modifier": "optional"},
	},
	"SavedSearchDefined": {
		{"name": "searchId", "type": "string", "modifier": "required"},
		{"name": "model", "type": "string", "modifier": "required"},
		{"name": "source", "type": "string", "modifier": "required"},
		{"name": "where", "type": "string", "modifier": "required"},
		{"name": "events", "type": "string", "modifier": "required"},
		{"name": "tag", "type": "string", "modifier": "required"},
		{"name": "emod", "type": "string"},
		{"name": "description", "type": "string", "modifier": "required"},
		{"name": "definedAt", "type": "date", "modifier": "required"},
	},
	"SavedSearchesView": {
		{"name": "searchId", "type": "string", "modifier": "required"},
		{"name": "description", "type": "string", "modifier": "required"},
		{"name": "tag", "type": "string", "modifier": "required"},
		{"name": "model", "type": "string"},
		{"name": "where", "type": "string", "modifier": "required"},
		{"name": "matches", "type": "int", "modifier": "required"},
	},
	"ShareSavedSearch": {
		{"name": "searchId", "type": "string", "modifier": "required"},
		{"name": "tag", "type": "string", "modifier": "required"},
	},
	"ImportVendorSearch": {
		{"name": "source", "type": "string", "modifier": "required"},
	},
	"VendorSearchImported": {
		{"name": "vendorSearchId", "type": "string", "modifier": "required"},
		{"name": "source", "type": "string", "modifier": "required"},
		{"name": "emod", "type": "string", "modifier": "required"},
		{"name": "where", "type": "string", "modifier": "required"},
		{"name": "tag", "type": "string"},
		{"name": "model", "type": "string", "modifier": "required"},
	},
}

// libraryLendingInvariants transcribes what test.InvariantLibraryLending
// declares, so an export is read back against the source an author wrote rather
// than against another export of it. Only the two scopes that declare an
// invariant appear: the plain context above the aggregate declares none, and
// must therefore inherit none.
var libraryLendingInvariants = map[string][]map[string]any{
	"Loan": {
		{"name": "OneCopyPerLoan", "statement": "A loan covers exactly one copy of one title"},
		{"name": "FiveCopiesPerMember", "statement": "A member holds at most five copies at one time"},
	},
	"Reading Room": {
		{"name": "OneReaderPerDesk", "statement": "A desk seats at most one reader at any moment"},
		{"name": "OneDeskPerReader", "statement": "A reader holds at most one desk for the length of a session"},
		{"name": "DeskFreeAtClosing", "statement": "No desk stays claimed past the closing hour"},
	},
}

func libraryLendingInvariantText() []string {
	var text []string
	for _, invariants := range libraryLendingInvariants {
		for _, invariant := range invariants {
			text = append(text, invariant["name"].(string), invariant["statement"].(string))
		}
	}
	return text
}

func invariantsByOwner(doc map[string]any) map[string][]map[string]any {
	return listsKeyedBy(doc, "invariants", "name", func(invariant map[string]any) map[string]any {
		return invariant
	})
}

func invariantTextAnywhere(doc map[string]any) []string {
	declared := make(map[string]bool)
	for _, text := range libraryLendingInvariantText() {
		declared[text] = true
	}

	var found []string
	for _, text := range stringsAnywhere(doc) {
		if declared[text] {
			found = append(found, text)
		}
	}
	return found
}

// stringsAnywhere returns every string a decoded document spells, key as well as
// value, so a search over it cannot miss text the exporter filed under a name
// the search did not think to look for.
func stringsAnywhere(node any) []string {
	var found []string
	switch n := node.(type) {
	case map[string]any:
		for key, child := range n {
			found = append(found, key)
			found = append(found, stringsAnywhere(child)...)
		}
	case []any:
		for _, child := range n {
			found = append(found, stringsAnywhere(child)...)
		}
	case string:
		found = append(found, n)
	}
	return found
}

// fieldSpec drops the positions off an exported field, leaving the name, type
// and modifier an author wrote.
func fieldSpec(field map[string]any) map[string]any {
	spec := map[string]any{"name": field["name"], "type": field["type"]}
	if modifier, ok := field["modifier"]; ok {
		spec["modifier"] = modifier
	}
	return spec
}

func fieldSpecs(fields []any) []map[string]any {
	out := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		out = append(out, fieldSpec(field.(map[string]any)))
	}
	return out
}

func fieldsByOwner(doc map[string]any) map[string][]map[string]any {
	return listsKeyedBy(doc, "fields", "name", fieldSpec)
}

func diagramFieldsByLabel(doc map[string]any) map[string][]map[string]any {
	return listsKeyedBy(doc, "fields", "label", fieldSpec)
}

// listsKeyedBy collects every list a decoded document holds under listKey, filed
// under the ownerKey of the object declaring it. Searching the whole document
// rather than the paths the list is expected on lets one that surfaces
// somewhere new show up as an unexpected entry.
func listsKeyedBy(doc map[string]any, listKey, ownerKey string, entryOf func(map[string]any) map[string]any) map[string][]map[string]any {
	byOwner := make(map[string][]map[string]any)
	eachObject(doc, func(object map[string]any) {
		list, declaresList := object[listKey].([]any)
		owner, named := object[ownerKey].(string)
		if !declaresList || !named {
			return
		}
		entries := make([]map[string]any, 0, len(list))
		for _, entry := range list {
			entries = append(entries, entryOf(entry.(map[string]any)))
		}
		byOwner[owner] = entries
	})
	return byOwner
}

func eachObject(node any, visit func(map[string]any)) {
	switch n := node.(type) {
	case map[string]any:
		visit(n)
		for _, child := range n {
			eachObject(child, visit)
		}
	case []any:
		for _, child := range n {
			eachObject(child, visit)
		}
	}
}

func withoutFieldLists(node any) any {
	return withoutKeys(node, func(key string) bool { return key == "fields" })
}

func withoutPositions(node any) any {
	return withoutKeys(node, func(key string) bool {
		return key == "position" || strings.HasSuffix(key, "_position")
	})
}

func withoutKeys(node any, drop func(key string) bool) any {
	switch n := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(n))
		for key, child := range n {
			if drop(key) {
				continue
			}
			out[key] = withoutKeys(child, drop)
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i, child := range n {
			out[i] = withoutKeys(child, drop)
		}
		return out
	}
	return node
}

func diagramDocOf(t *testing.T, model *ast.Model) map[string]any {
	t.Helper()

	raw, err := export.ExportDiagramJSON(model)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	return doc
}

func modelDocOf(t *testing.T, model *ast.Model) map[string]any {
	t.Helper()

	raw, err := export.ExportJSON(model)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	return doc
}

func requireConformsToSchema(t *testing.T, cueBin string, model *ast.Model) {
	t.Helper()

	cueOutput, err := export.ExportCUE(model)
	require.NoError(t, err)

	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.cue")
	schemaData, err := os.ReadFile("../cue/schema.cue")
	require.NoError(t, err, "failed to read schema.cue — test must run from internal/export directory")
	require.NoError(t, os.WriteFile(schemaPath, schemaData, 0644))

	// The export has to be handed to cue as data for -d to unify it with
	// #Model; vetting two CUE files only checks that both parse.
	modelPath := filepath.Join(dir, "model.json")
	require.NoError(t, os.WriteFile(modelPath, cueExportJSON(t, cueBin, cueOutput), 0644))

	output, err := exec.Command(cueBin, "vet", "-d", "#Model", schemaPath, modelPath).CombinedOutput()
	require.NoError(t, err, "cue vet failed: %s\nCUE export:\n%s", output, cueOutput)
}

// requireBothFormatsAgree decodes a model's two exports and requires that they
// describe the same model. Source positions are left out of the comparison
// because schema.cue has no place for them, so only the JSON export carries
// them; everything else has to match key for key.
func requireBothFormatsAgree(t *testing.T, cueBin string, model *ast.Model) {
	t.Helper()

	jsonRaw, err := export.ExportJSON(model)
	require.NoError(t, err)

	var fromJSON map[string]any
	require.NoError(t, json.Unmarshal(jsonRaw, &fromJSON))

	cueRaw, err := export.ExportCUE(model)
	require.NoError(t, err)

	var fromCUE map[string]any
	require.NoError(t, json.Unmarshal(cueExportJSON(t, cueBin, cueRaw), &fromCUE))

	require.Equal(t, withoutPositions(fromJSON), fromCUE)
}

func descriptionsByConstruct(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()

	context := firstOf(t, doc, "contexts")
	aggregate := firstOf(t, context, "aggregates")
	slice := firstOf(t, aggregate, "slices")
	translation := firstOf(t, slice, "translations")

	return map[string]any{
		"model":             doc["description"],
		"actor":             firstOf(t, doc, "actors")["description"],
		"context":           context["description"],
		"aggregate":         aggregate["description"],
		"slice":             slice["description"],
		"trigger":           slice["trigger"].(map[string]any)["description"],
		"command":           firstOf(t, slice, "commands")["description"],
		"event":             firstOf(t, slice, "events")["description"],
		"view":              firstOf(t, slice, "views")["description"],
		"automation":        firstOf(t, slice, "automations")["description"],
		"translation":       translation["description"],
		"translation event": translation["event"].(map[string]any)["description"],
	}
}

func descriptionsAnywhere(node any) []any {
	var found []any
	eachObject(node, func(object map[string]any) {
		if description, ok := object["description"]; ok {
			found = append(found, description)
		}
	})
	return found
}
