//go:build integration

package parser_test

import (
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	t.Run("parses minimal.emod fixture with zero errors and fully populated AST", func(t *testing.T) {
		source, err := os.ReadFile("testdata/minimal.emod")
		require.NoError(t, err)

		tokens, lexDiags := lexer.Scan(string(source), "testdata/minimal.emod")
		require.Empty(t, lexDiags)

		p := parser.New(tokens, "testdata/minimal.emod")
		model, diags := p.Parse()

		require.Empty(t, diags)
		require.Equal(t, "Hotel Reservation", model.Name)
		require.Len(t, model.Actors, 1)
		require.Equal(t, "Guest", model.Actors[0].Name)
		require.Len(t, model.Contexts, 1)

		ctx := model.Contexts[0]
		require.Equal(t, "Reservations", ctx.Name)
		require.Len(t, ctx.Aggregates, 1)

		agg := ctx.Aggregates[0]
		require.Equal(t, "Reservation", agg.Name)
		require.Len(t, agg.Slices, 1)

		slice := agg.Slices[0]
		require.Equal(t, "Make Reservation", slice.Name)

		require.Len(t, slice.Commands, 1)
		cmd := slice.Commands[0]
		require.Equal(t, "MakeReservation", cmd.Name)
		require.Len(t, cmd.Fields, 4)
		require.Equal(t, "guestId", cmd.Fields[0].Name)
		require.Equal(t, "string", cmd.Fields[0].Type)
		require.Equal(t, "required", cmd.Fields[0].Modifier)

		require.Len(t, slice.Events, 1)
		evt := slice.Events[0]
		require.Equal(t, "ReservationMade", evt.Name)
		require.Len(t, evt.Fields, 6)

		require.Len(t, slice.Flows, 1)
		flow := slice.Flows[0]
		require.Equal(t, "MakeReservation", flow.CommandName)
		require.Equal(t, "ReservationMade", flow.EventName)
	})

	t.Run("parses invalid.emod fixture and returns errors with correct filenames and line numbers", func(t *testing.T) {
		source, err := os.ReadFile("testdata/invalid.emod")
		require.NoError(t, err)

		tokens, _ := lexer.Scan(string(source), "testdata/invalid.emod")

		p := parser.New(tokens, "testdata/invalid.emod")
		_, diags := p.Parse()

		require.NotEmpty(t, diags)
		for _, d := range diags {
			require.Equal(t, "testdata/invalid.emod", d.Filename)
			require.Greater(t, d.Line, 0)
			require.NotEmpty(t, d.Message)
		}

		// foobar on line 3 should produce an unrecognized keyword error
		require.Contains(t, diags[0].Message, `"foobar"`)
		require.Equal(t, 3, diags[0].Line)
	})

	t.Run("parses all_patterns.emod fixture with zero errors and all four patterns populated", func(t *testing.T) {
		source, err := os.ReadFile("testdata/all_patterns.emod")
		require.NoError(t, err)

		tokens, lexDiags := lexer.Scan(string(source), "testdata/all_patterns.emod")
		require.Empty(t, lexDiags)

		p := parser.New(tokens, "testdata/all_patterns.emod")
		model, diags := p.Parse()

		require.Empty(t, diags)
		require.Equal(t, "Hotel Reservation", model.Name)
		require.Len(t, model.Actors, 1)
		require.Equal(t, "Guest", model.Actors[0].Name)
		require.Len(t, model.Contexts, 1)

		ctx := model.Contexts[0]
		require.Equal(t, "Reservations", ctx.Name)
		require.Len(t, ctx.Aggregates, 1)

		agg := ctx.Aggregates[0]
		require.Equal(t, "Reservation", agg.Name)
		require.Len(t, agg.Slices, 4)

		// Slice 0: Command Pattern — "Reserve a Room"
		commandSlice := agg.Slices[0]
		require.Equal(t, "Reserve a Room", commandSlice.Name)

		require.NotNil(t, commandSlice.Trigger)
		require.Equal(t, "UI", commandSlice.Trigger.Kind)
		require.Equal(t, "Reservation Form", commandSlice.Trigger.Name)
		require.Equal(t, "Guest", commandSlice.Trigger.Actor)
		require.Equal(t, "AvailableRoomsView", commandSlice.Trigger.Reads)

		require.Len(t, commandSlice.Commands, 1)
		cmd := commandSlice.Commands[0]
		require.Equal(t, "ReserveRoom", cmd.Name)
		require.Len(t, cmd.Fields, 4)
		require.Equal(t, "roomId", cmd.Fields[0].Name)
		require.Equal(t, "string", cmd.Fields[0].Type)
		require.Equal(t, "required", cmd.Fields[0].Modifier)
		require.Equal(t, "guestName", cmd.Fields[1].Name)
		require.Equal(t, "checkIn", cmd.Fields[2].Name)
		require.Equal(t, "date", cmd.Fields[2].Type)
		require.Equal(t, "checkOut", cmd.Fields[3].Name)

		require.Len(t, commandSlice.Events, 1)
		evt := commandSlice.Events[0]
		require.Equal(t, "RoomReserved", evt.Name)
		require.Len(t, evt.Fields, 6)
		require.Equal(t, "reservationId", evt.Fields[0].Name)
		require.Equal(t, "reservedAt", evt.Fields[5].Name)
		require.Equal(t, "timestamp", evt.Fields[5].Type)

		require.Len(t, commandSlice.Flows, 1)
		require.Equal(t, "ReserveRoom", commandSlice.Flows[0].CommandName)
		require.Equal(t, "RoomReserved", commandSlice.Flows[0].EventName)

		// Slice 1: View Pattern — "View Available Rooms"
		viewSlice := agg.Slices[1]
		require.Equal(t, "View Available Rooms", viewSlice.Name)

		require.Len(t, viewSlice.Views, 1)
		view := viewSlice.Views[0]
		require.Equal(t, "AvailableRoomsView", view.Name)
		require.Len(t, view.Fields, 4)
		require.Equal(t, "roomId", view.Fields[0].Name)
		require.Equal(t, "string", view.Fields[0].Type)
		require.Equal(t, "required", view.Fields[0].Modifier)
		require.Equal(t, "nextCheckIn", view.Fields[3].Name)
		require.Equal(t, "date", view.Fields[3].Type)
		require.Equal(t, "optional", view.Fields[3].Modifier)

		require.Len(t, view.Subscribes, 2)
		require.Equal(t, "RoomReserved", view.Subscribes[0])
		require.Equal(t, "GuestCheckedOut", view.Subscribes[1])

		// Slice 2: Automation Pattern — "Send Confirmation Email"
		automationSlice := agg.Slices[2]
		require.Equal(t, "Send Confirmation Email", automationSlice.Name)

		require.Len(t, automationSlice.Automations, 1)
		auto := automationSlice.Automations[0]
		require.Equal(t, "ConfirmationEmailReactor", auto.Name)
		require.Equal(t, "RoomReserved", auto.TriggerEvent)
		require.Equal(t, "SendConfirmationEmail", auto.Command)
		require.Equal(t, "Notifications", auto.TargetContext)

		// Slice 3: Translation Pattern — "Import External Booking"
		translationSlice := agg.Slices[3]
		require.Equal(t, "Import External Booking", translationSlice.Name)

		require.Len(t, translationSlice.Translations, 1)
		trans := translationSlice.Translations[0]
		require.Equal(t, "BookingComImport", trans.Name)
		require.Equal(t, "Booking.com API", trans.ExternalSystem)
		require.Equal(t, "BookingComWebhookView", trans.Reads)
		require.Equal(t, "ImportExternalReservation", trans.Command)

		require.NotNil(t, trans.Event)
		require.Equal(t, "ExternalReservationImported", trans.Event.Name)
		require.Len(t, trans.Event.Fields, 7)
		require.Equal(t, "reservationId", trans.Event.Fields[0].Name)
		require.Equal(t, "string", trans.Event.Fields[0].Type)
		require.Equal(t, "required", trans.Event.Fields[0].Modifier)
		require.Equal(t, "checkOut", trans.Event.Fields[6].Name)
		require.Equal(t, "date", trans.Event.Fields[6].Type)
	})

	t.Run("parses and validates multi_context.emod fixture with cross-context automation and external source", func(t *testing.T) {
		source, err := os.ReadFile("testdata/multi_context.emod")
		require.NoError(t, err)

		tokens, lexDiags := lexer.Scan(string(source), "testdata/multi_context.emod")
		require.Empty(t, lexDiags)

		p := parser.New(tokens, "testdata/multi_context.emod")
		model, parseDiags := p.Parse()
		require.Empty(t, parseDiags)

		validationDiags := validator.Validate(model)
		require.Empty(t, validationDiags)

		require.Len(t, model.Contexts, 2)
		require.Equal(t, "Orders", model.Contexts[0].Name)
		require.Equal(t, "Notifications", model.Contexts[1].Name)

		ordersCtx := model.Contexts[0]
		require.Len(t, ordersCtx.Aggregates, 1)
		require.Len(t, ordersCtx.Aggregates[0].Slices, 2)
		automationSlice := ordersCtx.Aggregates[0].Slices[1]

		require.Len(t, automationSlice.Automations, 1)
		auto := automationSlice.Automations[0]
		require.Equal(t, "OrderNotifier", auto.Name)
		require.Equal(t, "OrderPlaced", auto.TriggerEvent)
		require.Equal(t, "SendNotification", auto.Command)
		require.Equal(t, "Notifications", auto.TargetContext)

		notificationsCtx := model.Contexts[1]
		require.Len(t, notificationsCtx.Aggregates, 1)
		notifSlice := notificationsCtx.Aggregates[0].Slices[0]

		require.Len(t, notifSlice.Events, 1)
		evt := notifSlice.Events[0]
		require.Equal(t, "NotificationReceived", evt.Name)
		require.Equal(t, "external", evt.Source)
		require.Equal(t, "Email Provider", evt.ExternalName)
		require.Len(t, evt.Fields, 2)
		require.Equal(t, "notificationId", evt.Fields[0].Name)
		require.Equal(t, "string", evt.Fields[0].Type)
		require.Equal(t, "required", evt.Fields[0].Modifier)
		require.Equal(t, "receivedAt", evt.Fields[1].Name)
		require.Equal(t, "timestamp", evt.Fields[1].Type)
		require.Equal(t, "required", evt.Fields[1].Modifier)
	})
}
