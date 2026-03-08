//go:build integration

package parser_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/hpcsc/emod/internal/validator"
	"github.com/stretchr/testify/require"
)

var ignorePositions = cmpopts.IgnoreTypes(ast.Position{})

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
		test.RequireEqual(t, &ast.Command{
			Name: "MakeReservation",
			Fields: []*ast.Field{
				{Name: "guestId", Type: "string", Modifier: "required"},
				{Name: "roomType", Type: "string", Modifier: "required"},
				{Name: "checkIn", Type: "date", Modifier: "required"},
				{Name: "checkOut", Type: "date", Modifier: "required"},
			},
		}, slice.Commands[0], ignorePositions)

		require.Len(t, slice.Events, 1)
		test.RequireEqual(t, &ast.Event{
			Name: "ReservationMade",
			Fields: []*ast.Field{
				{Name: "reservationId", Type: "string", Modifier: "required"},
				{Name: "guestId", Type: "string", Modifier: "required"},
				{Name: "roomType", Type: "string", Modifier: "required"},
				{Name: "checkIn", Type: "date", Modifier: "required"},
				{Name: "checkOut", Type: "date", Modifier: "required"},
				{Name: "status", Type: "string", Modifier: "required"},
			},
		}, slice.Events[0], ignorePositions)

		require.Len(t, slice.Flows, 1)
		test.RequireEqual(t, &ast.Flow{
			CommandName: "MakeReservation",
			EventName:   "ReservationMade",
		}, slice.Flows[0], ignorePositions)
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

		// Comment attachment assertions
		require.Len(t, model.Comments, 1)
		require.Equal(t, "# Hotel Reservation System — exercises all four slice patterns", model.Comments[0].Text)

		require.Len(t, model.Actors, 1)
		require.Equal(t, "Guest", model.Actors[0].Name)
		require.Len(t, model.Contexts, 2)

		ctx := model.Contexts[0]
		require.Equal(t, "Reservations", ctx.Name)
		require.Len(t, ctx.Aggregates, 1)

		agg := ctx.Aggregates[0]
		require.Equal(t, "Reservation", agg.Name)
		require.Len(t, agg.Slices, 5)

		require.Len(t, agg.Slices[0].Comments, 1)
		require.Equal(t, "# Slice 1: Command Pattern", agg.Slices[0].Comments[0].Text)

		require.Len(t, agg.Slices[1].Comments, 1)
		require.Equal(t, "# Slice 2: View Pattern", agg.Slices[1].Comments[0].Text)

		require.Len(t, agg.Slices[2].Comments, 1)
		require.Equal(t, "# Slice 3: Command Pattern — Check Out", agg.Slices[2].Comments[0].Text)

		require.Len(t, agg.Slices[3].Comments, 1)
		require.Equal(t, "# Slice 4: Automation Pattern", agg.Slices[3].Comments[0].Text)

		require.Len(t, agg.Slices[4].Comments, 1)
		require.Equal(t, "# Slice 5: Translation Pattern", agg.Slices[4].Comments[0].Text)

		// Slice 0: Command Pattern — "Reserve a Room"
		commandSlice := agg.Slices[0]
		require.Equal(t, "Reserve a Room", commandSlice.Name)

		require.NotNil(t, commandSlice.Trigger)
		test.RequireEqual(t, &ast.Trigger{
			Kind:  "UI",
			Name:  "Reservation Form",
			Actor: "Guest",
			Reads: "AvailableRoomsView",
		}, commandSlice.Trigger, ignorePositions)

		require.Len(t, commandSlice.Commands, 1)
		test.RequireEqual(t, &ast.Command{
			Name: "ReserveRoom",
			Fields: []*ast.Field{
				{Name: "roomId", Type: "string", Modifier: "required"},
				{Name: "guestName", Type: "string", Modifier: "required"},
				{Name: "checkIn", Type: "date", Modifier: "required"},
				{Name: "checkOut", Type: "date", Modifier: "required"},
			},
		}, commandSlice.Commands[0], ignorePositions)

		require.Len(t, commandSlice.Events, 1)
		test.RequireEqual(t, &ast.Event{
			Name: "RoomReserved",
			Fields: []*ast.Field{
				{Name: "reservationId", Type: "string", Modifier: "required"},
				{Name: "roomId", Type: "string", Modifier: "required"},
				{Name: "guestName", Type: "string", Modifier: "required"},
				{Name: "checkIn", Type: "date", Modifier: "required"},
				{Name: "checkOut", Type: "date", Modifier: "required"},
				{Name: "reservedAt", Type: "timestamp", Modifier: "required"},
			},
		}, commandSlice.Events[0], ignorePositions)

		require.Len(t, commandSlice.Flows, 1)
		test.RequireEqual(t, &ast.Flow{
			CommandName: "ReserveRoom",
			EventName:   "RoomReserved",
		}, commandSlice.Flows[0], ignorePositions)

		// Slice 1: View Pattern — "View Available Rooms"
		viewSlice := agg.Slices[1]
		require.Equal(t, "View Available Rooms", viewSlice.Name)

		require.Len(t, viewSlice.Views, 1)
		test.RequireEqual(t, &ast.View{
			Name: "AvailableRoomsView",
			Fields: []*ast.Field{
				{Name: "roomId", Type: "string", Modifier: "required"},
				{Name: "roomNumber", Type: "string", Modifier: "required"},
				{Name: "status", Type: "string", Modifier: "required"},
				{Name: "nextCheckIn", Type: "date", Modifier: "optional"},
			},
			Subscribes: []string{"RoomReserved", "GuestCheckedOut"},
		}, viewSlice.Views[0], ignorePositions)

		// Slice 2: Command Pattern — "Check Out Guest"
		checkOutSlice := agg.Slices[2]
		require.Equal(t, "Check Out Guest", checkOutSlice.Name)

		require.Len(t, checkOutSlice.Commands, 1)
		test.RequireEqual(t, &ast.Command{
			Name: "CheckOutGuest",
			Fields: []*ast.Field{
				{Name: "reservationId", Type: "string", Modifier: "required"},
			},
		}, checkOutSlice.Commands[0], ignorePositions)

		require.Len(t, checkOutSlice.Events, 1)
		test.RequireEqual(t, &ast.Event{
			Name: "GuestCheckedOut",
			Fields: []*ast.Field{
				{Name: "reservationId", Type: "string", Modifier: "required"},
				{Name: "roomId", Type: "string", Modifier: "required"},
				{Name: "checkedOutAt", Type: "timestamp", Modifier: "required"},
			},
		}, checkOutSlice.Events[0], ignorePositions)

		require.Len(t, checkOutSlice.Flows, 1)
		test.RequireEqual(t, &ast.Flow{
			CommandName: "CheckOutGuest",
			EventName:   "GuestCheckedOut",
		}, checkOutSlice.Flows[0], ignorePositions)

		// Slice 3: Automation Pattern — "Send Confirmation Email"
		automationSlice := agg.Slices[3]
		require.Equal(t, "Send Confirmation Email", automationSlice.Name)

		require.Len(t, automationSlice.Automations, 1)
		test.RequireEqual(t, &ast.Automation{
			Name:          "ConfirmationEmailReactor",
			TriggerEvent:  "RoomReserved",
			Command:       "SendConfirmationEmail",
			TargetContext: "Notifications",
		}, automationSlice.Automations[0], ignorePositions)

		// Slice 4: Translation Pattern — "Import External Booking"
		translationSlice := agg.Slices[4]
		require.Equal(t, "Import External Booking", translationSlice.Name)

		require.Len(t, translationSlice.Commands, 1)
		test.RequireEqual(t, &ast.Command{
			Name: "ImportExternalReservation",
			Fields: []*ast.Field{
				{Name: "externalRef", Type: "string", Modifier: "required"},
				{Name: "roomId", Type: "string", Modifier: "required"},
				{Name: "guestName", Type: "string", Modifier: "required"},
				{Name: "checkIn", Type: "date", Modifier: "required"},
				{Name: "checkOut", Type: "date", Modifier: "required"},
			},
		}, translationSlice.Commands[0], ignorePositions)

		require.Len(t, translationSlice.Translations, 1)
		test.RequireEqual(t, &ast.Translation{
			Name:           "BookingComImport",
			ExternalSystem: "Booking.com API",
			Reads:          "BookingComWebhookView",
			Command:        "ImportExternalReservation",
			Event: &ast.Event{
				Name: "ExternalReservationImported",
				Fields: []*ast.Field{
					{Name: "reservationId", Type: "string", Modifier: "required"},
					{Name: "externalRef", Type: "string", Modifier: "required"},
					{Name: "source", Type: "string", Modifier: "required"},
					{Name: "roomId", Type: "string", Modifier: "required"},
					{Name: "guestName", Type: "string", Modifier: "required"},
					{Name: "checkIn", Type: "date", Modifier: "required"},
					{Name: "checkOut", Type: "date", Modifier: "required"},
				},
			},
		}, translationSlice.Translations[0], ignorePositions)

		// Notifications context
		notifCtx := model.Contexts[1]
		require.Equal(t, "Notifications", notifCtx.Name)
		require.Len(t, notifCtx.Aggregates, 1)
		require.Equal(t, "Notification", notifCtx.Aggregates[0].Name)
		require.Len(t, notifCtx.Aggregates[0].Slices, 1)

		notifSlice := notifCtx.Aggregates[0].Slices[0]
		require.Equal(t, "Send Confirmation", notifSlice.Name)
		require.Len(t, notifSlice.Commands, 1)
		test.RequireEqual(t, &ast.Command{
			Name: "SendConfirmationEmail",
			Fields: []*ast.Field{
				{Name: "reservationId", Type: "string", Modifier: "required"},
				{Name: "guestName", Type: "string", Modifier: "required"},
				{Name: "email", Type: "string", Modifier: "required"},
			},
		}, notifSlice.Commands[0], ignorePositions)
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
		test.RequireEqual(t, &ast.Automation{
			Name:          "OrderNotifier",
			TriggerEvent:  "OrderPlaced",
			Command:       "SendNotification",
			TargetContext: "Notifications",
		}, automationSlice.Automations[0], ignorePositions)

		notificationsCtx := model.Contexts[1]
		require.Len(t, notificationsCtx.Aggregates, 1)
		notifSlice := notificationsCtx.Aggregates[0].Slices[0]

		require.Len(t, notifSlice.Events, 1)
		test.RequireEqual(t, &ast.Event{
			Name:         "NotificationReceived",
			Source:       "external",
			ExternalName: "Email Provider",
			Fields: []*ast.Field{
				{Name: "notificationId", Type: "string", Modifier: "required"},
				{Name: "receivedAt", Type: "timestamp", Modifier: "required"},
			},
		}, notifSlice.Events[0], ignorePositions)
	})
}
