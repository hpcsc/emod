//go:build unit

package export_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
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
		if _, err := exec.LookPath("cue"); err != nil {
			t.Skip("cue tool not available in PATH — using explicit path")
		}

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

		cueBin := "/Users/davidnguyen/.local/share/mise/installs/cue/0.14.2/cue"
		cmd := exec.Command(cueBin, "vet", schemaPath, exportPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("cue vet failed: %v\nOutput: %s\nCUE export:\n%s", err, output, string(cueOutput))
		}
	})

	t.Run("round-trip fidelity via cue export", func(t *testing.T) {
		if _, err := exec.LookPath("cue"); err != nil {
			t.Skip("cue tool not available in PATH — using explicit path")
		}

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

		cueBin := "/Users/davidnguyen/.local/share/mise/installs/cue/0.14.2/cue"
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
