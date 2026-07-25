//go:build unit

package export_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/export"
	"github.com/stretchr/testify/require"
)

func TestExportJSON(t *testing.T) {
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

		s := getFirstSlice(doc)
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

		s := getFirstSlice(doc)
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

		s := getFirstSlice(doc)
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

		s := getFirstSlice(doc)
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

		s := getFirstSlice(doc)
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

		s := getFirstSlice(doc)
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

		s := getFirstSlice(doc)
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

		s := getFirstSlice(doc)
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

	require.NotNil(t, s["trigger"])
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
				Name:      "Ctx",
				NamePos:   ast.Position{Filename: "test.cue", Line: 5, Column: 1},
				OpenPos:   ast.Position{Filename: "test.cue", Line: 5, Column: 6},
				ClosePos:  ast.Position{Filename: "test.cue", Line: 10, Column: 1},
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

	s := getFirstSlice(doc)
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

	s := getFirstSlice(doc)
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

	s := getFirstSlice(doc)
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

	s := getFirstSlice(doc)
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

	s := getFirstSlice(doc)
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
									Kind:         "UI",
									KindPos:      ast.Position{Filename: "test.cue", Line: 3, Column: 3},
									Name:         "Form",
									NamePos:      ast.Position{Filename: "test.cue", Line: 3, Column: 7},
									Actor:        "Guest",
									ActorPos:     ast.Position{Filename: "test.cue", Line: 3, Column: 13},
									Reads:        "MyView",
									ReadsPos:     ast.Position{Filename: "test.cue", Line: 3, Column: 20},
									OpenPos:      ast.Position{Filename: "test.cue", Line: 3, Column: 28},
									ClosePos:     ast.Position{Filename: "test.cue", Line: 5, Column: 3},
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

	s := getFirstSlice(doc)
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

	s := getFirstSlice(doc)
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
										Name:            "Auto",
										NamePos:         ast.Position{Filename: "test.cue", Line: 5, Column: 5},
										TriggerEvent:    "Evt",
										TriggerEventPos: ast.Position{Filename: "test.cue", Line: 5, Column: 11},
										Command:         "DoIt",
										CommandPos:      ast.Position{Filename: "test.cue", Line: 5, Column: 16},
										TargetContext:   "Other",
										TargetContextPos: ast.Position{Filename: "test.cue", Line: 5, Column: 22},
										OpenPos:         ast.Position{Filename: "test.cue", Line: 5, Column: 28},
										ClosePos:        ast.Position{Filename: "test.cue", Line: 7, Column: 3},
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

	s := getFirstSlice(doc)
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
										Reads:    "V",
										ReadsPos: ast.Position{Filename: "test.cue", Line: 6, Column: 18},
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

	s := getFirstSlice(doc)
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
}

func TestExportJSONDiagnostics(t *testing.T) {
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
}

func TestExportDiagramJSON(t *testing.T) {
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
}

func TestExportDiagramJSONDiagnostics(t *testing.T) {
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

// getFirstSlice navigates the parsed JSON document to find the first slice
// in the first context's first aggregate.
func getFirstSlice(doc map[string]any) map[string]any {
	ctxs := doc["contexts"].([]any)
	c := ctxs[0].(map[string]any)
	aggs := c["aggregates"].([]any)
	agg := aggs[0].(map[string]any)
	slices := agg["slices"].([]any)
	return slices[0].(map[string]any)
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

func TestExportCUE(t *testing.T) {
	t.Run("exports minimal model with name only", func(t *testing.T) {
		model := &ast.Model{Name: "TestModel"}

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `name: "TestModel"`)
		require.NotContains(t, output, "actors")
		require.NotContains(t, output, "contexts")
	})

	t.Run("exports actors", func(t *testing.T) {
		model := &ast.Model{
			Name: "App",
			Actors: []*ast.Actor{
				{Name: "Guest"},
				{Name: "Admin"},
			},
		}

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `name: "Guest"`)
		require.Contains(t, output, `name: "Admin"`)
	})

	t.Run("exports context with aggregate slice and command with typed fields", func(t *testing.T) {
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `name: "Hotel"`)
		require.Contains(t, output, `name: "Reservations"`)
		require.Contains(t, output, `name: "Reservation"`)
		require.Contains(t, output, `name: "Make Reservation"`)
		require.Contains(t, output, `name: "MakeReservation"`)
		require.Contains(t, output, `name: "guestId"`)
		require.Contains(t, output, `type: "string"`)
		require.Contains(t, output, `modifier: "required"`)
	})

	t.Run("exports trigger block", func(t *testing.T) {
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `kind: "UI"`)
		require.Contains(t, output, `name: "Form"`)
		require.Contains(t, output, `actor: "Guest"`)
		require.Contains(t, output, `reads: "MyView"`)
	})

	t.Run("exports event with external source", func(t *testing.T) {
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `name: "PaymentReceived"`)
		require.Contains(t, output, `source: "external"`)
		require.Contains(t, output, `external_name: "Stripe"`)
	})

	t.Run("exports view with fields and subscribes", func(t *testing.T) {
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `name: "RoomsView"`)
		require.Contains(t, output, `name: "roomId"`)
		require.Contains(t, output, `modifier: "required"`)
		require.Contains(t, output, `modifier: "optional"`)
		require.Contains(t, output, `subscribes: ["RoomReserved", "GuestCheckedOut"]`)
	})

	t.Run("exports automation with target context", func(t *testing.T) {
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `name: "OrderNotifier"`)
		require.Contains(t, output, `trigger_event: "OrderPlaced"`)
		require.Contains(t, output, `command: "SendNotification"`)
		require.Contains(t, output, `target_context: "Notifications"`)
	})

	t.Run("exports translation with nested event", func(t *testing.T) {
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `name: "BookingImport"`)
		require.Contains(t, output, `external_system: "Booking.com API"`)
		require.Contains(t, output, `reads: "WebhookView"`)
		require.Contains(t, output, `command: "ImportBooking"`)
		require.Contains(t, output, `name: "BookingImported"`)
	})

	t.Run("exports flows", func(t *testing.T) {
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `command_name: "MakeReservation"`)
		require.Contains(t, output, `event_name: "ReservationMade"`)
	})

	t.Run("handles nil model without panicking", func(t *testing.T) {
		raw, err := export.ExportCUE(nil)
		require.NoError(t, err)
		require.Empty(t, string(raw))
	})

	t.Run("output passes cue vet against schema", func(t *testing.T) {
		cueBin := lookupCue(t)

		model := buildFullModel()

		cueOutput, err := export.ExportCUE(model)
		require.NoError(t, err)

		dir := t.TempDir()
		schemaPath := filepath.Join(dir, "schema.cue")
		exportPath := filepath.Join(dir, "exported.cue")

		// Read the schema from the cue package
		schemaData, err := os.ReadFile("../cue/schema.cue")
		require.NoError(t, err, "failed to read schema.cue — test must run from internal/export directory")
		err = os.WriteFile(schemaPath, schemaData, 0644)
		require.NoError(t, err)

		err = os.WriteFile(exportPath, cueOutput, 0644)
		require.NoError(t, err)

		cmd := exec.Command(cueBin, "vet", schemaPath, exportPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("cue vet failed: %v\nOutput: %s\nCUE export:\n%s", err, output, string(cueOutput))
		}
	})

	t.Run("round-trip fidelity via cue export", func(t *testing.T) {
		cueBin := lookupCue(t)

		model := buildFullModel()

		// Export as JSON
		jsonRaw, err := export.ExportJSON(model)
		require.NoError(t, err)

		var jsonDoc map[string]any
		err = json.Unmarshal(jsonRaw, &jsonDoc)
		require.NoError(t, err)

		// Export as CUE
		cueOutput, err := export.ExportCUE(model)
		require.NoError(t, err)

		// Convert CUE to JSON via cue export
		dir := t.TempDir()
		exportPath := filepath.Join(dir, "exported.cue")
		err = os.WriteFile(exportPath, cueOutput, 0644)
		require.NoError(t, err)

		// Also write schema in the same directory
		schemaData, err := os.ReadFile("../cue/schema.cue")
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "schema.cue"), schemaData, 0644)
		require.NoError(t, err)

		cmd := exec.Command(cueBin, "export", exportPath, "--out", "json")
		cueJSONRaw, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("cue export failed: %v\nOutput: %s\nCUE input:\n%s", err, cueJSONRaw, string(cueOutput))
		}

		var cueJSONDoc map[string]any
		err = json.Unmarshal(cueJSONRaw, &cueJSONDoc)
		require.NoError(t, err)

		// Compare — the cue export should produce equivalent JSON
		require.Equal(t, jsonDoc["name"], cueJSONDoc["name"])

		// Compare actors
		jsonActors := jsonDoc["actors"].([]any)
		cueActors := cueJSONDoc["actors"].([]any)
		require.Equal(t, len(jsonActors), len(cueActors))
		for i := range jsonActors {
			ja := jsonActors[i].(map[string]any)
			ca := cueActors[i].(map[string]any)
			require.Equal(t, ja["name"], ca["name"])
		}

		// Compare contexts (recursively)
		jsonCtxs := jsonDoc["contexts"].([]any)
		cueCtxs := cueJSONDoc["contexts"].([]any)
		require.Equal(t, len(jsonCtxs), len(cueCtxs))
		for i := range jsonCtxs {
			compareContexts(t, jsonCtxs[i].(map[string]any), cueCtxs[i].(map[string]any))
		}
	})

	t.Run("exports all node types in one model", func(t *testing.T) {
		model := buildFullModel()

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		// Verify all key fields are present
		require.Contains(t, output, `name: "Full"`)
		require.Contains(t, output, `name: "User"`)
		require.Contains(t, output, `kind: "UI"`)
		require.Contains(t, output, `name: "Cmd"`)
		require.Contains(t, output, `name: "Evt"`)
		require.Contains(t, output, `source: "external"`)
		require.Contains(t, output, `external_name: "Provider"`)
		require.Contains(t, output, `name: "MyView"`)
		require.Contains(t, output, `subscribes: ["Evt"]`)
		require.Contains(t, output, `command_name: "Cmd"`)
		require.Contains(t, output, `event_name: "Evt"`)
	})

	t.Run("field modifiers preserved in CUE output", func(t *testing.T) {
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		// Field without modifier omits field
		require.Contains(t, output, `name: "name"`)
		require.Contains(t, output, `type: "string"`)
		require.NotContains(t, output, `modifier: ""`)

		// Field with optional modifier includes it
		require.Contains(t, output, `modifier: "optional"`)
	})

	t.Run("comments are serialized as CUE structs", func(t *testing.T) {
		model := &ast.Model{
			Comments: []*ast.Comment{{Text: "# System model"}},
			Name:     "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
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

		raw, err := export.ExportCUE(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `text: "# System model"`)
		require.Contains(t, output, `text: "# Slice comment"`)
		require.Contains(t, output, `text: "# Cmd comment"`)
	})
}

// buildFullModel returns a model with all node types populated.
func buildFullModel() *ast.Model {
	return &ast.Model{
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
}

// compareContexts recursively compares two context maps from JSON unmarshalling.
func compareContexts(t *testing.T, expected, actual map[string]any) {
	t.Helper()
	require.Equal(t, expected["name"], actual["name"])

	expAggs := expected["aggregates"].([]any)
	actAggs := actual["aggregates"].([]any)
	require.Equal(t, len(expAggs), len(actAggs))
	for i := range expAggs {
		compareAggregates(t, expAggs[i].(map[string]any), actAggs[i].(map[string]any))
	}
}

func compareAggregates(t *testing.T, expected, actual map[string]any) {
	t.Helper()
	require.Equal(t, expected["name"], actual["name"])

	expSlices := expected["slices"].([]any)
	actSlices := actual["slices"].([]any)
	require.Equal(t, len(expSlices), len(actSlices))
	for i := range expSlices {
		compareSlices(t, expSlices[i].(map[string]any), actSlices[i].(map[string]any))
	}
}

func compareSlices(t *testing.T, expected, actual map[string]any) {
	t.Helper()
	require.Equal(t, expected["name"], actual["name"])

	// Compare commands
	compareNameLists(t, expected["commands"], actual["commands"])
	// Compare events
	compareNameLists(t, expected["events"], actual["events"])
	// Compare views
	compareNameLists(t, expected["views"], actual["views"])
	// Compare automations
	compareNameLists(t, expected["automations"], actual["automations"])
}

func compareNameLists(t *testing.T, expected, actual any) {
	t.Helper()
	if expected == nil && actual == nil {
		return
	}
	expList := expected.([]any)
	actList := actual.([]any)
	require.Equal(t, len(expList), len(actList))
	for i := range expList {
		exp := expList[i].(map[string]any)
		act := actList[i].(map[string]any)
		require.Equal(t, exp["name"], act["name"])
	}
}
