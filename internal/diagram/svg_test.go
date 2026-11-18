//go:build unit

package diagram_test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestExportSVG(t *testing.T) {
	t.Run("empty model returns valid SVG with svg root and no diagram content", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `<svg xmlns="http://www.w3.org/2000/svg"`)
		requireValidXML(t, output)
		require.NotContains(t, output, "Wireframes")
	})

	t.Run("names the lane a person enters through for the wireframes it holds", func(t *testing.T) {
		model := minimalModel("Test", "Slice1")
		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		requireValidXML(t, output)
		require.Equal(t,
			[]string{"Wireframes", "Commands / Views", "Events", "External Systems"},
			svgLaneLabels(t, output),
			"only the lane holding what a person touches is renamed; the lanes below it keep their names")
		require.NotContains(t, output, "UI / Triggers")
	})

	t.Run("labels a trigger with its name only", func(t *testing.T) {
		model := singleSliceModel("Test", "S", &ast.Trigger{Name: "SubmitForm"})

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, ">SubmitForm</text>")
	})

	t.Run("renders command, event and view labels", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			command("MakeReservation"), event("ReservationMade"), view("AvailableRooms"))

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "MakeReservation")
		require.Contains(t, output, "ReservationMade")
		require.Contains(t, output, "AvailableRooms")
	})

	t.Run("connects trigger to command", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S",
						Trigger:  &ast.Trigger{Name: "Click"},
						Commands: []*ast.Command{{Name: "DoAction"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Click")
		require.Contains(t, output, "DoAction")
		require.Equal(t, 1, arrowCount(output), "trigger to command is one arrow")
	})

	t.Run("two triggers that share a name each get the arrows of their own slice", func(t *testing.T) {
		model := &ast.Model{
			Name: "Shop",
			Contexts: []*ast.Context{{
				Name: "Shop",
				Aggregates: []*ast.Aggregate{{
					Name: "Cart",
					Slices: []*ast.Slice{
						{
							Name:     "Apply Discount",
							Trigger:  &ast.Trigger{Name: "Checkout Page", Reads: []string{"CartView"}},
							Views:    []*ast.View{{Name: "CartView"}},
							Commands: []*ast.Command{{Name: "ApplyDiscount"}},
						},
						{
							Name:     "Place Order",
							Trigger:  &ast.Trigger{Name: "Checkout Page", Reads: []string{"OrderView"}},
							Views:    []*ast.View{{Name: "OrderView"}},
							Commands: []*ast.Command{{Name: "PlaceOrder"}},
						},
					},
				}},
			}},
		}

		joinedAt := make(map[int][]string)
		for _, end := range svgArrowEnds(t, svgOf(t, model)) {
			for i, box := range end {
				if box.label == "Checkout Page" {
					joinedAt[box.rect.x] = append(joinedAt[box.rect.x], end[1-i].label)
				}
			}
		}

		triggers := slices.Sorted(maps.Keys(joinedAt))
		require.Len(t, triggers, 2, "the arrows have to touch both trigger boxes")
		require.ElementsMatch(t, []string{"CartView", "ApplyDiscount"}, joinedAt[triggers[0]], "the trigger of the first slice")
		require.ElementsMatch(t, []string{"OrderView", "PlaceOrder"}, joinedAt[triggers[1]], "the trigger of the second slice")
	})

	t.Run("screens", func(t *testing.T) {
		// screenFixtureSlices names the slice of test.ScreenLibraryLending that
		// draws each box below the Wireframes lane.
		screenFixtureSlices := map[string]string{
			"BorrowCopy":      "Borrow Copy",
			"CopyBorrowed":    "Borrow Copy",
			"MemberLoansView": "Review Member Loans",
			"ReturnCopy":      "Return Copy",
			"RenewLoan":       "Return Copy",
			"CopyReturned":    "Return Copy",
			"LoanRenewed":     "Return Copy",
			"RemindMember":    "Chase Overdue Copy",
			"RecallCopy":      "Chase Overdue Copy",
			"MemberReminded":  "Chase Overdue Copy",
			"CopyRecalled":    "Chase Overdue Copy",
			"MemberLoansView\nfrom Review Member Loans":     "Chase Overdue Copy",
			"DeskOccupancyView\nfrom Browse Desk Occupancy": "Chase Overdue Copy",
			"CopyBorrowed\nfrom Borrow Copy":                "Chase Overdue Copy",
			"ClaimDesk":                                     "Claim Desk",
			"DeskClaimed":                                   "Claim Desk",
			"DeskOccupancyView":                             "Browse Desk Occupancy",
		}
		screenNames := []string{"LendingDesk", "OverdueReport", "DeskKiosk"}

		t.Run("draws one box for each shows block, under the header bar of a screen, in the Wireframes lane and in the column of its slice", func(t *testing.T) {
			shapes := svgShapes(t, svgOf(t, test.ScreenLibraryLendingModel(t)))
			lane := firstShapeLabelled(t, shapes, "Wireframes")

			var drawn []string
			for i, shape := range shapes {
				if !slices.Contains(screenNames, shape.label) {
					continue
				}
				header := shapes[i-1]
				drawn = append(drawn, fmt.Sprintf("%s in %s, in the Wireframes lane: %t, under a header bar: %t",
					shape.label, columnsUnder(shapes, shape.rect, screenFixtureSlices),
					shape.rect.within(lane.rect), header.label == "" && header.rect.within(shape.rect)))
			}

			require.Equal(t, []string{
				"LendingDesk in Borrow Copy, in the Wireframes lane: true, under a header bar: true",
				"LendingDesk in Return Copy, in the Wireframes lane: true, under a header bar: true",
				"OverdueReport in Chase Overdue Copy, in the Wireframes lane: true, under a header bar: true",
				"DeskKiosk in Chase Overdue Copy, in the Wireframes lane: true, under a header bar: true",
				"DeskKiosk in Claim Desk, in the Wireframes lane: true, under a header bar: true",
			}, drawn)
		})

		t.Run("two shows blocks in one slice draw two boxes that do not overlap, and the slice's commands and events keep their lanes", func(t *testing.T) {
			shapes := svgShapes(t, svgOf(t, test.ScreenLibraryLendingModel(t)))
			overdue := firstShapeLabelled(t, shapes, "OverdueReport")
			kiosk := firstShapeLabelled(t, shapes, "DeskKiosk")

			lanes := make(map[string]string)
			for _, name := range []string{"RemindMember", "RecallCopy", "MemberReminded", "CopyRecalled"} {
				box := firstShapeLabelled(t, shapes, name)
				for _, lane := range []string{"Wireframes", "Commands / Views", "Events"} {
					if box.rect.within(firstShapeLabelled(t, shapes, lane).rect) {
						lanes[name] = lane
					}
				}
			}

			require.False(t, overdue.rect.overlaps(kiosk.rect), "OverdueReport at %+v overlaps DeskKiosk at %+v", overdue.rect, kiosk.rect)
			require.Equal(t, map[string]string{
				"RemindMember":   "Commands / Views",
				"RecallCopy":     "Commands / Views",
				"MemberReminded": "Events",
				"CopyRecalled":   "Events",
			}, lanes)
		})

		t.Run("two shows blocks in a slice of two commands draw their boxes above the commands, as wide as them", func(t *testing.T) {
			model := singleSliceModel("Lending", "Chase Overdue Copy", command("RemindMember"), command("RecallCopy"))
			model.Contexts[0].Aggregates[0].Slices[0].Appearances = []*ast.Appearance{
				{Screen: "OverdueReport", Commands: []string{"RecallCopy"}},
				{Screen: "DeskKiosk"},
			}
			shapes := svgShapes(t, svgOf(t, model))
			across := func(labels ...string) []string {
				var spans []string
				for _, label := range labels {
					rect := firstShapeLabelled(t, shapes, label).rect
					spans = append(spans, fmt.Sprintf("from x %d, %d wide", rect.x, rect.w))
				}
				return spans
			}

			require.Equal(t, across("RemindMember", "RecallCopy"), across("OverdueReport", "DeskKiosk"))
		})

		t.Run("draws a screen box and its header bar painted as a trigger box and its header bar", func(t *testing.T) {
			model := &ast.Model{
				Name: "Shop",
				Contexts: []*ast.Context{{
					Name: "Shop",
					Aggregates: []*ast.Aggregate{{
						Name: "Cart",
						Slices: []*ast.Slice{
							{Name: "Apply Discount", Trigger: &ast.Trigger{Name: "Checkout Page"}},
							{Name: "Place Order", Appearances: []*ast.Appearance{{Screen: "CheckoutScreen"}}},
						},
					}},
				}},
			}
			boxes := svgBoxes(t, svgOf(t, model))
			paintOf := func(label string) []diagramBox {
				i := slices.IndexFunc(boxes, func(box diagramBox) bool { return box.label == label })
				require.Positive(t, i, "the picture draws %s after its header bar", label)
				return unplaced([]diagramBox{boxes[i-1], {attributes: boxes[i].attributes}})
			}

			require.Equal(t, paintOf("Checkout Page"), paintOf("CheckoutScreen"))
		})

		t.Run("writes the name of a screen at the centre of its box, at the size of a trigger's name", func(t *testing.T) {
			model := &ast.Model{
				Name: "Shop",
				Contexts: []*ast.Context{{
					Name: "Shop",
					Aggregates: []*ast.Aggregate{{
						Name: "Cart",
						Slices: []*ast.Slice{
							{Name: "Apply Discount", Trigger: &ast.Trigger{Name: "Checkout Page"}},
							{Name: "Place Order", Appearances: []*ast.Appearance{{Screen: "OrderScreen"}, {Screen: "CheckoutScreen"}}},
						},
					}},
				}},
			}
			shapes := svgShapes(t, svgOf(t, model))
			trigger := firstShapeLabelled(t, shapes, "Checkout Page")

			var written []string
			for _, label := range []string{"OrderScreen", "CheckoutScreen"} {
				screen := firstShapeLabelled(t, shapes, label)
				written = append(written, fmt.Sprintf("%s at the centre: %t, at size %s", label, screen.labelAt == screen.rect.centre(), screen.labelSize))
			}

			require.Equal(t, []string{
				"OrderScreen at the centre: true, at size " + trigger.labelSize,
				"CheckoutScreen at the centre: true, at size " + trigger.labelSize,
			}, written)
		})

		t.Run("gives each screen box the description of its screen's declaration", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library",
				Screens: []*ast.Screen{
					{Name: "LendingDesk", Description: "The terminal a librarian lends copies at"},
					{Name: "OverdueReport", Description: "The list of loans past their due date"},
				},
				Contexts: []*ast.Context{{
					Name: "Lending",
					Aggregates: []*ast.Aggregate{{
						Name: "Loan",
						Slices: []*ast.Slice{
							{Name: "Chase Overdue Copy", Appearances: []*ast.Appearance{{Screen: "OverdueReport"}}},
							{Name: "Borrow Copy", Appearances: []*ast.Appearance{{Screen: "LendingDesk"}}},
						},
					}},
				}},
			}
			shapes := svgShapes(t, svgOf(t, model))

			require.Equal(t, []string{
				"The list of loans past their due date",
				"The terminal a librarian lends copies at",
			}, []string{firstShapeLabelled(t, shapes, "OverdueReport").tooltip, firstShapeLabelled(t, shapes, "LendingDesk").tooltip})
		})

		t.Run("each reads entry draws one arrow into the box of its own shows block, from the view or its far copy, and each command entry one arrow out of it to the command that it names", func(t *testing.T) {
			output := svgOf(t, test.ScreenLibraryLendingModel(t))
			shapes := svgShapes(t, output)
			named := func(box svgArrowBox) string {
				if slices.Contains(screenNames, box.label) {
					return box.label + " in " + columnsUnder(shapes, box.rect, screenFixtureSlices)
				}
				return strings.ReplaceAll(lineText(box.label), "\n", " ")
			}

			var joined []string
			for _, end := range svgArrowEnds(t, output) {
				if slices.Contains(screenNames, end[0].label) || slices.Contains(screenNames, end[1].label) {
					joined = append(joined, named(end[0])+" -> "+named(end[1]))
				}
			}

			require.ElementsMatch(t, []string{
				"MemberLoansView -> LendingDesk in Borrow Copy",
				"LendingDesk in Borrow Copy -> BorrowCopy",
				"LendingDesk in Return Copy -> ReturnCopy",
				"LendingDesk in Return Copy -> RenewLoan",
				"MemberLoansView from Review Member Loans -> OverdueReport in Chase Overdue Copy",
				"DeskOccupancyView from Browse Desk Occupancy -> OverdueReport in Chase Overdue Copy",
				"OverdueReport in Chase Overdue Copy -> RecallCopy",
				"DeskOccupancyView from Browse Desk Occupancy -> DeskKiosk in Chase Overdue Copy",
				"DeskOccupancyView -> DeskKiosk in Claim Desk",
				"DeskKiosk in Claim Desk -> ClaimDesk",
			}, joined)
		})

		t.Run("a command entry joins the command of its own slice, and no command that it does not name, when another slice declares a command of that name", func(t *testing.T) {
			model := &ast.Model{
				Name: "Shop",
				Contexts: []*ast.Context{{
					Name: "Shop",
					Aggregates: []*ast.Aggregate{{
						Name: "Cart",
						Slices: []*ast.Slice{
							{
								Name:        "Apply Discount",
								Appearances: []*ast.Appearance{{Screen: "CheckoutPage", Commands: []string{"ApplyDiscount"}}},
								Commands:    []*ast.Command{{Name: "ApplyDiscount"}, {Name: "ClearCart"}, {Name: "RemoveDiscount"}},
								Automations: []*ast.Automation{{Name: "ExpireDiscount", Schedule: "0 0 * * *", Command: "RemoveDiscount"}},
							},
							{Name: "Reapply Discount", Commands: []*ast.Command{{Name: "ApplyDiscount"}}},
						},
					}},
				}},
			}
			output := svgOf(t, model)
			discounts := slices.DeleteFunc(svgShapes(t, output), func(shape svgShape) bool { return shape.label != "ApplyDiscount" })
			require.Len(t, discounts, 2, "each slice draws its own ApplyDiscount")
			ownSlice := slices.MinFunc(discounts, func(a, b svgShape) int { return a.rect.x - b.rect.x }).rect

			var joined []string
			for _, end := range svgArrowEnds(t, output) {
				if end[0].label == "CheckoutPage" {
					joined = append(joined, fmt.Sprintf("%s, in the slice of the screen: %t", end[1].label, end[1].rect == ownSlice))
				}
			}

			require.Equal(t, []string{"ApplyDiscount, in the slice of the screen: true"}, joined)
		})

		t.Run("a command entry naming a command that a slice two or more slices away declares ends at a copy of that command in the slice of the screen", func(t *testing.T) {
			model := &ast.Model{
				Name: "Orders",
				Contexts: []*ast.Context{{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{{
						Name: "Order",
						Slices: []*ast.Slice{
							{Name: "Cancel Order", Appearances: []*ast.Appearance{{Screen: "OrderPage", Commands: []string{"RefundOrder"}}}},
							{Name: "Track Order", Views: []*ast.View{{Name: "OrdersView"}}},
							{Name: "Refund Order", Commands: []*ast.Command{{Name: "RefundOrder"}}},
						},
					}},
				}},
			}
			output := svgOf(t, model)
			screen := firstShapeLabelled(t, svgShapes(t, output), "OrderPage")

			var joined []string
			for _, end := range svgArrowEnds(t, output) {
				if end[0].label == "OrderPage" {
					below := end[1].rect.x < screen.rect.x+screen.rect.w && screen.rect.x < end[1].rect.x+end[1].rect.w
					joined = append(joined, fmt.Sprintf("%s, below the screen: %t", strings.ReplaceAll(lineText(end[1].label), "\n", " "), below))
				}
			}

			require.Equal(t, []string{"RefundOrder in Refund Order, below the screen: true"}, joined)
		})
	})

	t.Run("connects each trigger to the command of its own slice when two slices declare a command of one name", func(t *testing.T) {
		model := &ast.Model{
			Name: "Shop",
			Contexts: []*ast.Context{{
				Name: "Shop",
				Aggregates: []*ast.Aggregate{{
					Name: "Cart",
					Slices: []*ast.Slice{
						{Name: "Apply Discount", Trigger: &ast.Trigger{Name: "Checkout Page"}, Commands: []*ast.Command{{Name: "SubmitCart"}}},
						{Name: "Place Order", Trigger: &ast.Trigger{Name: "Checkout Page"}, Commands: []*ast.Command{{Name: "SubmitCart"}}},
					},
				}},
			}},
		}

		ends := svgArrowEnds(t, svgOf(t, model))

		require.Len(t, ends, 2)
		require.NotEqual(t, ends[0][0].rect, ends[1][0].rect, "the two arrows start at the two trigger boxes")
		for _, end := range ends {
			trigger, command := end[0].rect, end[1].rect
			require.Equal(t, "SubmitCart", end[1].label)
			apart := trigger.centre()[0] - command.centre()[0]
			require.Less(t, max(apart, -apart), trigger.w,
				"the arrow from the trigger box at x %d ends at the command under it, not at the command of the other slice", trigger.x)
		}
	})

	t.Run("connects command to event via flow", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S",
						Commands: []*ast.Command{{Name: "Reserve"}},
						Events:   []*ast.Event{{Name: "Reserved"}},
						Flows:    []*ast.Flow{{CommandName: "Reserve", EventName: "Reserved"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Reserve")
		require.Contains(t, output, "Reserved")
		require.Equal(t, 1, arrowCount(output), "command to event is one arrow")
	})

	t.Run("connects event to view via subscribes", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:   "S",
						Events: []*ast.Event{{Name: "OrderPlaced"}},
						Views:  []*ast.View{{Name: "OrderList", Subscribes: []string{"OrderPlaced"}}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "OrderList")
		require.Equal(t, 1, arrowCount(output), "event to view is one arrow")
	})

	t.Run("draws the arrows into every view of a slice, not only its first", func(t *testing.T) {
		model := slicesModel(&ast.Slice{
			Name:   "Watch Orders",
			Events: []*ast.Event{{Name: "OrderPlaced"}},
			Views: []*ast.View{
				{Name: "OrderList", Subscribes: []string{"OrderPlaced"}},
				{Name: "OrderTotals", Subscribes: []string{"OrderPlaced"}},
			},
		})

		require.Equal(t, []string{"OrderPlaced", "OrderPlaced"},
			append(sourcesReaching(t, svgOf(t, model), "OrderList"), sourcesReaching(t, svgOf(t, model), "OrderTotals")...))
	})

	t.Run("draws the arrows from the event of every translation of a slice, not only its first", func(t *testing.T) {
		model := slicesModel(
			&ast.Slice{Name: "Import", Translations: []*ast.Translation{
				{Name: "CardFeed", ExternalSystem: "Card API", Event: &ast.Event{Name: "CardPaid"}},
				{Name: "BankFeed", ExternalSystem: "Bank API", Event: &ast.Event{Name: "BankPaid"}},
			}},
			&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "PaymentsView", Subscribes: []string{"CardPaid", "BankPaid"}}}},
		)

		require.Equal(t, []string{"CardPaid", "BankPaid"}, linesOf(sourcesReaching(t, svgOf(t, model), "PaymentsView")))
	})

	t.Run("draws the arrow into a view from the first slice that declares its event, which the copy rule measures from, when two slices declare it", func(t *testing.T) {
		output := svgOf(t, slicesModel(
			&ast.Slice{Name: "Book Direct", Events: []*ast.Event{{Name: "RoomBooked"}}},
			&ast.Slice{Name: "List Rooms", Views: []*ast.View{{Name: "RoomsView", Subscribes: []string{"RoomBooked"}}}},
			&ast.Slice{Name: "Between"},
			&ast.Slice{Name: "Book Partner", Events: []*ast.Event{{Name: "RoomBooked"}}},
		))
		ends := svgArrowBoxes(t, output)
		var booked []boxRect
		for _, box := range svgBoxes(t, output) {
			if lineText(box.label) == "RoomBooked" {
				booked = append(booked, box.rect)
			}
		}
		require.Len(t, booked, 2, "the two slices draw an event each, and the slice next to the first draws no copy")

		arrows := svgDrawnArrows(t, output)
		require.Len(t, arrows, 1)
		require.Equal(t, min(booked[0].x, booked[1].x), arrowBoxMet(t, ends, arrows[0].route[0]).rect.x,
			"the arrow starts at the event of Book Direct, the leftmost of the two")
	})

	t.Run("draws the arrow into a view from the event its own slice declares, though an earlier slice declares the event too", func(t *testing.T) {
		output := svgOf(t, slicesModel(
			&ast.Slice{Name: "Declares Own", Events: []*ast.Event{{Name: "Own"}}},
			&ast.Slice{Name: "Declares Far", Events: []*ast.Event{{Name: "Far"}}},
			&ast.Slice{Name: "Between"},
			&ast.Slice{
				Name:   "Watch",
				Events: []*ast.Event{{Name: "Own"}},
				Views:  []*ast.View{{Name: "WatchView", Subscribes: []string{"Own", "Far"}}},
			},
		))
		ends := svgArrowBoxes(t, output)
		var own []boxRect
		for _, box := range svgBoxes(t, output) {
			if lineText(box.label) == "Own" {
				own = append(own, box.rect)
			}
		}
		require.Len(t, own, 2, "the first and the last slice each draw an event Own, and Watch draws no copy of it")

		var starts []int
		for _, arrow := range svgDrawnArrows(t, output) {
			if from := arrowBoxMet(t, ends, arrow.route[0]); lineText(from.label) == "Own" {
				starts = append(starts, from.rect.x)
			}
		}
		require.Equal(t, []int{max(own[0].x, own[1].x)}, starts, "the arrow starts at the event of Watch, the rightmost of the two")
	})

	t.Run("draws the arrow into a view from the event its own slice declares, though an earlier slice of the same name declares the event too", func(t *testing.T) {
		output := svgOf(t, &ast.Model{
			Name: "Orders",
			Contexts: []*ast.Context{
				{Name: "Ordering", Aggregates: []*ast.Aggregate{{Name: "Order", Slices: []*ast.Slice{
					{Name: "Cancel", Events: []*ast.Event{{Name: "OrderCancelled"}}},
					{Name: "Ship"},
					{Name: "Deliver"},
				}}}},
				{Name: "Refunds", Aggregates: []*ast.Aggregate{{Name: "Refund", Slices: []*ast.Slice{{
					Name:   "Cancel",
					Events: []*ast.Event{{Name: "OrderCancelled"}},
					Views:  []*ast.View{{Name: "CancelView", Subscribes: []string{"OrderCancelled"}}},
				}}}}},
			},
		})
		ends := svgArrowBoxes(t, output)
		var cancelled []boxRect
		for _, box := range svgBoxes(t, output) {
			if lineText(box.label) == "OrderCancelled" {
				cancelled = append(cancelled, box.rect)
			}
		}
		require.Len(t, cancelled, 2, "each slice named Cancel draws an event OrderCancelled, and neither draws a copy of it")

		var starts []int
		for _, arrow := range svgDrawnArrows(t, output) {
			if from := arrowBoxMet(t, ends, arrow.route[0]); lineText(from.label) == "OrderCancelled" {
				starts = append(starts, from.rect.x)
			}
		}
		require.Equal(t, []int{max(cancelled[0].x, cancelled[1].x)}, starts,
			"the arrow starts at the event of the Cancel slice of Refunds, the rightmost of the two")
	})

	t.Run("draws no arrow into a view from an event no slice declares", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:   "S",
						Events: []*ast.Event{{Name: "OrderPlaced"}},
						Views:  []*ast.View{{Name: "OrderList", Subscribes: []string{"OrderPlaced", "OrderLost"}}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		require.Equal(t, 1, arrowCount(string(raw)), "only the declared event has a box to draw an arrow from")
	})

	t.Run("draws no arrow into an automation from an event no slice declares", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:   "S",
						Events: []*ast.Event{{Name: "LoanOverdue"}},
						Automations: []*ast.Automation{
							{Name: "Remind", OnEvent: "LoanOverdue"},
							{Name: "Chase", OnEvent: "LoanLost"},
						},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		require.Equal(t, 1, arrowCount(string(raw)), "only the declared event has a box to draw an arrow from")
	})

	t.Run("a translation that nests an event with no name leaves the other events of its slice their full width", func(t *testing.T) {
		withNested := func(nested *ast.Event) *ast.Model {
			return &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name:         "S",
							Events:       []*ast.Event{{Name: "OrderPaid"}},
							Translations: []*ast.Translation{{Name: "PaymentImport", ExternalSystem: "Bank", Event: nested}},
						}},
					}},
				}},
			}
		}
		widthOfPaid := func(model *ast.Model) int {
			raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
			require.NoError(t, err)
			return boxNamed(t, svgBoxes(t, string(raw)), "OrderPaid").rect.w
		}

		require.Equal(t, widthOfPaid(withNested(nil)), widthOfPaid(withNested(&ast.Event{Name: ""})))
	})

	t.Run("draws no arrow from an automation to a command no slice declares", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S",
						Commands: []*ast.Command{{Name: "NotifyMember"}},
						Automations: []*ast.Automation{
							{Name: "Announce", Command: "NotifyMember"},
							{Name: "Chase", Command: "ChaseMember"},
						},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		require.Equal(t, 1, arrowCount(string(raw)), "only the declared command has a box to draw an arrow to")
	})

	t.Run("draws no arrow to the event a translation nests from a command no slice declares", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Translations: []*ast.Translation{{
							Name: "PartnerFeed", ExternalSystem: "Partner API", Command: "ImportBooking", Event: &ast.Event{Name: "BookingImported"},
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		require.Equal(t, 1, arrowCount(string(raw)), "only the external system and the reactor have boxes to draw an arrow between")
	})

	t.Run("writes the name of a trigger at the centre of its box, at the size a command's name is written", func(t *testing.T) {
		shapes := svgShapes(t, svgOf(t, singleSliceModel("Test", "S", &ast.Trigger{Name: "SubmitForm"}, command("SubmitOrder"))))

		trigger := slices.IndexFunc(shapes, func(shape svgShape) bool { return lineText(shape.label) == "SubmitForm" })
		command := slices.IndexFunc(shapes, func(shape svgShape) bool { return lineText(shape.label) == "SubmitOrder" })
		require.GreaterOrEqual(t, trigger, 0, "the picture has to draw the trigger SubmitForm")
		require.GreaterOrEqual(t, command, 0, "the picture has to draw the command SubmitOrder")
		require.Equal(t, shapes[trigger].rect.centre(), shapes[trigger].labelAt)
		require.Equal(t, shapes[command].labelSize, shapes[trigger].labelSize)
	})

	t.Run("writes the label of each copy at the centre of its box, at the size an event's label is written", func(t *testing.T) {
		raw, err := diagram.ExportSVG(test.TriggerReadsLibraryLendingModel(t), diagram.StyleAuto)
		require.NoError(t, err)

		shapes := svgShapes(t, string(raw))
		copies := svgCopyShapes(shapes)
		require.Len(t, copies, 10, "the fixture draws four copies of events and six of views and commands")

		var centres, labelled [][2]int
		var sizes []string
		for _, copied := range copies {
			centres = append(centres, copied.rect.centre())
			labelled = append(labelled, copied.labelAt)
			sizes = append(sizes, copied.labelSize)
		}
		event := slices.IndexFunc(shapes, func(shape svgShape) bool { return lineText(shape.label) == "CopyBorrowed" })
		require.GreaterOrEqual(t, event, 0, "the fixture has to draw the event CopyBorrowed")

		require.Equal(t, centres, labelled)
		require.Equal(t, slices.Repeat([]string{shapes[event].labelSize}, len(sizes)), sizes)
	})

	t.Run("draws each copy as tall as the box of the element it copies, with the same rounded corners", func(t *testing.T) {
		raw, err := diagram.ExportSVG(test.TriggerReadsLibraryLendingModel(t), diagram.StyleAuto)
		require.NoError(t, err)
		output := string(raw)
		boxes := svgBoxes(t, output)
		corner := regexp.MustCompile(`\brx=\d+`)

		var originals, copied []string
		for _, c := range svgCopyShapes(svgShapes(t, output)) {
			name := cardLines(c.label)[0]
			original := boxNamed(t, boxes, name)
			originals = append(originals, fmt.Sprintf("%s: %d tall, %s", name, original.rect.h, corner.FindString(original.attributes)))
			copied = append(copied, fmt.Sprintf("%s: %d tall, %s", name, c.rect.h, corner.FindString(c.attributes)))
		}

		require.Len(t, copied, 10, "the fixture draws four copies of events and six of views and commands")
		require.Equal(t, originals, copied)
	})

	t.Run("arrow ends", func(t *testing.T) {
		models := func(t *testing.T) map[string]*ast.Model {
			return map[string]*ast.Model{
				"trigger reads":     test.TriggerReadsLibraryLendingModel(t),
				"screens":           test.ScreenLibraryLendingModel(t),
				"automation delays": test.AutomationDelayLibraryLendingModel(t),
				"subscribed events": test.SubscribedEventsHotelStaysModel(t),
				"rejections":        test.RejectionLibraryLendingModel(t),
				"a far translation": farTranslationModel(),
			}
		}

		t.Run("draws no arrow across the inside of either box it joins", func(t *testing.T) {
			for name, model := range models(t) {
				output := svgOf(t, model)
				boxes := svgArrowBoxes(t, output)

				var crossings []string
				for _, arrow := range svgDrawnArrows(t, output) {
					joined := []svgArrowBox{
						arrowBoxMet(t, boxes, arrow.route[0]),
						arrowBoxMet(t, boxes, arrow.route[len(arrow.route)-1]),
					}
					for i := 1; i < len(arrow.route); i++ {
						for _, box := range joined {
							if box.rect.crossedBy(arrow.route[i-1], arrow.route[i]) {
								crossings = append(crossings, lineText(joined[0].label)+" -> "+lineText(joined[1].label)+" crosses "+lineText(box.label))
							}
						}
					}
				}
				require.Empty(t, crossings, name)
			}
		})

		t.Run("meets each box it joins on a side, never at a corner", func(t *testing.T) {
			for name, model := range models(t) {
				output := svgOf(t, model)
				boxes := svgArrowBoxes(t, output)

				var corners []string
				for _, arrow := range svgDrawnArrows(t, output) {
					for _, end := range [][2]int{arrow.route[0], arrow.route[len(arrow.route)-1]} {
						box := arrowBoxMet(t, boxes, end).rect
						if (end[0] == box.x || end[0] == box.x+box.w) && (end[1] == box.y || end[1] == box.y+box.h) {
							corners = append(corners, fmt.Sprintf("%v", end))
						}
					}
				}
				require.Empty(t, corners, name)
			}
		})

		t.Run("ends an arrow from below an automation under the name written under its gear, and crosses no line of it", func(t *testing.T) {
			output := svgOf(t, test.AutomationDelayLibraryLendingModel(t))
			arrows := svgDrawnArrows(t, output)
			lanes := svgLaneRects(t, output)

			var endingUnderName int
			var crossings []string
			for _, gear := range svgBoxes(t, output) {
				if !svgDrawnAsGear(gear) {
					continue
				}
				name := cardLines(gear.label)[0]
				label := svgGearLabel(t, output, name)
				lines := boxRect{x: gear.rect.x, y: label.top, w: gear.rect.w, h: label.bottom - label.top}
				for _, arrow := range arrows {
					fromEvents := drawnLaneOf(lanes, arrow.route[0][1]) == svgEventsLane
					if end := arrow.route[len(arrow.route)-1]; fromEvents && end[1] == label.bottom && lines.meets(end) {
						endingUnderName++
					}
					for i := 1; i < len(arrow.route); i++ {
						if lines.crossedBy(arrow.route[i-1], arrow.route[i]) {
							crossings = append(crossings, "an arrow crosses the name under "+name)
						}
					}
				}
			}

			require.Equal(t, 4, endingUnderName, "the fixture activates four automations on an event drawn below them")
			require.Empty(t, crossings)
		})

	})

	t.Run("the text an arrow carries", func(t *testing.T) {
		t.Run("writes the text an arrow carries on that arrow's horizontal leg, between its two bends", func(t *testing.T) {
			var labelled int
			var offLeg []string
			for name, model := range labelledModels(t) {
				for _, arrow := range svgDrawnArrows(t, svgOf(t, model)) {
					if arrow.labelAt == nil {
						continue
					}
					labelled++
					at := *arrow.labelAt
					onLeg := false
					for i := 1; i < len(arrow.route); i++ {
						from, to := arrow.route[i-1], arrow.route[i]
						if from[1] == to[1] && at[1] == from[1] && at[0] > min(from[0], to[0]) && at[0] < max(from[0], to[0]) {
							onLeg = true
						}
					}
					if !onLeg {
						offLeg = append(offLeg, fmt.Sprintf("%s: the text at %v is off its leg %v", name, at, arrow.route))
					}
				}
			}

			require.Positive(t, labelled, "the models have to delay at least one automation")
			require.Empty(t, offLeg)
		})

		t.Run("writes the text an arrow carries clear of every other arrow, across the width of the text, where its leg has room for it", func(t *testing.T) {
			// 4 px a character is less than any glyph of the 10 px the SVG writes
			// an arrow's text at, so the band is narrower than the text.
			const pxPerCharacter = 4
			var labelled int
			var crossed []string
			for name, model := range map[string]*ast.Model{
				"two delayed automations":                               test.AutomationDelayLibraryLendingModel(t),
				"a delayed arrow over commands that flow straight down": delayOverStraightArrowsModel(),
			} {
				output := svgOf(t, model)
				arrows := svgDrawnArrows(t, output)
				texts := svgArrowLabels(t, output)
				for i, arrow := range arrows {
					if arrow.labelAt == nil {
						continue
					}
					labelled++
					at, half := *arrow.labelAt, pxPerCharacter*utf8.RuneCountInString(texts[i])/2
					for j, other := range arrows {
						if j != i && (routePasses(other.route, [2]int{at[0] - half, at[1]}) ||
							routePasses(other.route, at) || routePasses(other.route, [2]int{at[0] + half, at[1]}) ||
							routeCrossesY(other.route, at[0]-half, at[0]+half, at[1])) {
							crossed = append(crossed, fmt.Sprintf("%s: another arrow runs through the text at %v", name, at))
						}
					}
				}
			}

			require.Equal(t, 3, labelled, "the models delay three automations")
			require.Empty(t, crossed)
		})
	})

	t.Run("arrows clear of other boxes and of each other", func(t *testing.T) {
		t.Run("draws no arrow across a box it does not join, nor across the name under a gear", func(t *testing.T) {
			var arrowsChecked int
			var crossings []string
			for name, model := range routeModels(t) {
				output := svgOf(t, model)
				ends := svgArrowBoxes(t, output)
				obstacles := svgObstacles(t, output)

				for _, arrow := range svgDrawnArrows(t, output) {
					arrowsChecked++
					from := arrowBoxMet(t, ends, arrow.route[0])
					to := arrowBoxMet(t, ends, arrow.route[len(arrow.route)-1])
					for _, obstacle := range obstacles {
						if obstacle.rect.overlaps(from.rect) || obstacle.rect.overlaps(to.rect) {
							continue
						}
						for i := 1; i < len(arrow.route); i++ {
							if obstacle.rect.crossedBy(arrow.route[i-1], arrow.route[i]) {
								crossings = append(crossings, fmt.Sprintf("%s: %q -> %q crosses %q",
									name, lineText(from.label), lineText(to.label), lineText(obstacle.label)))
								break
							}
						}
					}
				}
			}

			require.Positive(t, arrowsChecked)
			require.Empty(t, crossings)
		})

		t.Run("keeps each stretch up or down far enough from a stretch of another arrow beside it for a reader to tell the two apart", func(t *testing.T) {
			// Less than the 6 px between two legs in one gap, so that the test
			// does not restate that spacing.
			const apart = 5
			var pairs int
			var near []string
			for name, model := range routeModels(t) {
				arrows := svgDrawnArrows(t, svgOf(t, model))
				for i := range arrows {
					for j := i + 1; j < len(arrows); j++ {
						a, b := arrows[i].route, arrows[j].route
						for k := 1; k < len(a); k++ {
							for l := 1; l < len(b); l++ {
								if a[k-1][0] != a[k][0] || b[l-1][0] != b[l][0] ||
									min(max(a[k-1][1], a[k][1]), max(b[l-1][1], b[l][1])) <= max(min(a[k-1][1], a[k][1]), min(b[l-1][1], b[l][1])) {
									continue
								}
								pairs++
								if a[k][0]-b[l][0] < apart && b[l][0]-a[k][0] < apart {
									near = append(near, fmt.Sprintf("%s: %v and %v", name, a, b))
								}
							}
						}
					}
				}
			}

			require.Positive(t, pairs, "the models have to hold two arrows that run up or down beside each other")
			require.Empty(t, near)
		})
	})

	t.Run("the way an arrow runs", func(t *testing.T) {
		t.Run("draws every arrow in lines that run across or up and down", func(t *testing.T) {
			var lines int
			var slanted []string
			for name, model := range routeModels(t) {
				for _, arrow := range svgDrawnArrows(t, svgOf(t, model)) {
					for i := 1; i < len(arrow.route); i++ {
						lines++
						if a, b := arrow.route[i-1], arrow.route[i]; a[0] != b[0] && a[1] != b[1] {
							slanted = append(slanted, fmt.Sprintf("%s: from %v to %v", name, a, b))
						}
					}
				}
			}

			require.Positive(t, lines)
			require.Empty(t, slanted)
		})

		t.Run("runs a lone arrow from a box to the box under it straight down the middle of both", func(t *testing.T) {
			output := svgOf(t, slicesModel(&ast.Slice{
				Name:     "Open",
				Commands: []*ast.Command{{Name: "OpenAccount"}},
				Events:   []*ast.Event{{Name: "AccountOpened"}},
				Flows:    []*ast.Flow{{CommandName: "OpenAccount", EventName: "AccountOpened"}},
			}))
			boxes := svgBoxes(t, output)
			command, event := boxNamed(t, boxes, "OpenAccount").rect, boxNamed(t, boxes, "AccountOpened").rect
			require.Equal(t, command.x+command.w/2, event.x+event.w/2, "the fixture draws the event straight under the command")

			middle := command.x + command.w/2
			require.Equal(t, []drawnArrow{{route: [][2]int{{middle, command.y + command.h}, {middle, event.y}}}}, svgDrawnArrows(t, output))
		})

		t.Run("spreads the ends on each side of a box evenly between the arrows that run straight from it, in the order of the boxes at their far ends", func(t *testing.T) {
			output := svgOf(t, slicesModel(
				&ast.Slice{
					Name:    "Open",
					Trigger: &ast.Trigger{Name: "Account Screen", Reads: []string{"AccountsView"}},
					Events:  []*ast.Event{{Name: "AccountOpened"}, {Name: "AccountRenamed"}},
				},
				&ast.Slice{
					Name:   "List",
					Events: []*ast.Event{{Name: "AccountListed"}},
					Views:  []*ast.View{{Name: "AccountsView", Subscribes: []string{"AccountRenamed", "AccountClosed", "AccountOpened", "AccountListed"}}},
				},
				&ast.Slice{Name: "Close", Events: []*ast.Event{{Name: "AccountClosed"}}},
			))
			view := boxNamed(t, svgBoxes(t, output), "AccountsView").rect
			ends := svgArrowBoxes(t, output)
			middle, right := view.x+view.w/2, view.x+view.w

			atView := make(map[string][2]int)
			for _, arrow := range svgDrawnArrows(t, output) {
				from, to := arrow.route[0], arrow.route[len(arrow.route)-1]
				if lineText(arrowBoxMet(t, ends, from).label) == "AccountsView" {
					atView[lineText(arrowBoxMet(t, ends, to).label)] = from
				} else {
					atView[lineText(arrowBoxMet(t, ends, from).label)] = to
				}
			}

			require.Equal(t, map[string][2]int{
				"AccountOpened":  {view.x + (middle-view.x)/3, view.y + view.h},
				"AccountRenamed": {view.x + 2*(middle-view.x)/3, view.y + view.h},
				"AccountListed":  {middle, view.y + view.h},
				"AccountClosed":  {middle + (right-middle)/2, view.y + view.h},
				"Account Screen": {middle, view.y},
			}, atView, "the event under the view runs straight up its middle, and the others spread on each side of it")
		})

		t.Run("runs the stretch that joins the two legs of an arrow in the room beside its slice on the side of the box it ends at", func(t *testing.T) {
			output := svgOf(t, slicesModel(
				&ast.Slice{Name: "Import", Commands: []*ast.Command{{Name: "ImportBooking"}}},
				&ast.Slice{Name: "Sync", Translations: []*ast.Translation{
					{Name: "PartnerSync", ExternalSystem: "Partner API", Event: &ast.Event{Name: "PartnerBooked"}},
					{Name: "BankSync", ExternalSystem: "Bank API", Event: &ast.Event{Name: "BankPaid"}},
				}},
			))
			boxes := svgBoxes(t, output)
			command := boxNamed(t, boxes, "ImportBooking").rect
			leftEvent, rightEvent := boxNamed(t, boxes, "PartnerBooked").rect, boxNamed(t, boxes, "BankPaid").rect
			betweenSlices := leftEvent.x - (command.x + command.w)
			ends := svgArrowBoxes(t, output)

			rooms := make(map[string]string)
			for _, arrow := range svgDrawnArrows(t, output) {
				require.Len(t, arrow.route, 6, "an arrow from an external system up to its reactor crosses the Events lane")
				x := arrow.route[2][0]
				switch {
				case x > command.x+command.w && x < leftEvent.x:
					rooms[lineText(arrowBoxMet(t, ends, arrow.route[5]).label)] = "between the two slices"
				case x > rightEvent.x+rightEvent.w && x < rightEvent.x+rightEvent.w+betweenSlices:
					rooms[lineText(arrowBoxMet(t, ends, arrow.route[5]).label)] = "after the last slice"
				}
			}

			require.Equal(t, map[string]string{"⚙ PartnerSync": "between the two slices", "⚙ BankSync": "after the last slice"}, rooms)
		})

		t.Run("runs the stretch that joins two legs of an arrow between two slices, where no box of the picture sits", func(t *testing.T) {
			var stretches int
			var onBoxes []string
			for name, model := range routeModels(t) {
				output := svgOf(t, model)
				lanes := svgLaneRects(t, output)
				obstacles := svgObstacles(t, output)
				for _, arrow := range svgDrawnArrows(t, output) {
					for i := 1; i < len(arrow.route); i++ {
						a, b := arrow.route[i-1], arrow.route[i]
						if a[0] != b[0] || drawnGapOf(lanes, a[1]) < 0 || drawnGapOf(lanes, b[1]) < 0 || a[1] == b[1] {
							continue
						}
						stretches++
						for _, box := range obstacles {
							if a[0] >= box.rect.x && a[0] <= box.rect.x+box.rect.w {
								onBoxes = append(onBoxes, fmt.Sprintf("%s: x %d lies across %q", name, a[0], lineText(box.label)))
							}
						}
					}
				}
			}

			require.Positive(t, stretches, "the models have to hold an arrow that crosses a whole lane")
			require.Empty(t, onBoxes)
		})
	})

	t.Run("translation reactors", func(t *testing.T) {
		t.Run("draws a translation reactor as wide as its name needs, centred under the slice's command", func(t *testing.T) {
			output := svgOf(t, slicesModel(&ast.Slice{
				Name:         "Sync",
				Commands:     []*ast.Command{{Name: "ImportBooking"}},
				Translations: []*ast.Translation{{Name: "PartnerSync", ExternalSystem: "Partner API", Command: "ImportBooking"}},
			}))
			boxes := svgBoxes(t, output)
			command, reactor := boxNamed(t, boxes, "ImportBooking").rect, boxNamed(t, boxes, "⚙ PartnerSync").rect

			// 5 px a character is less than any glyph of the 10 px the reactor's
			// name is written at, so the box is at least that wide.
			const pxPerCharacter = 5
			require.Equal(t, command.x+command.w/2, reactor.x+reactor.w/2)
			require.Greater(t, reactor.w, pxPerCharacter*utf8.RuneCountInString("⚙ PartnerSync"))
			require.Less(t, reactor.w, command.w, "the reactor leaves room beside it for an arrow from the command down to the Events lane")
		})

		t.Run("draws the arrow from an external system up to the reactor of a translation that has the name of that system", func(t *testing.T) {
			output := svgOf(t, slicesModel(&ast.Slice{
				Name:     "Import Payment",
				Commands: []*ast.Command{{Name: "ImportPayment"}},
				Translations: []*ast.Translation{{
					Name: "Stripe", ExternalSystem: "Stripe", Command: "ImportPayment", Event: &ast.Event{Name: "PaymentImported"},
				}},
			}))
			ends := svgArrowBoxes(t, output)

			var joined [][2]string
			for _, arrow := range svgDrawnArrows(t, output) {
				from, to := arrowBoxMet(t, ends, arrow.route[0]), arrowBoxMet(t, ends, arrow.route[len(arrow.route)-1])
				joined = append(joined, [2]string{lineText(from.label), lineText(to.label)})
			}
			require.Contains(t, joined, [2]string{"Stripe", "⚙ Stripe"})
		})
	})

	t.Run("legs between lanes", func(t *testing.T) {
		t.Run("runs the leg of an arrow that joins two lanes in a gap between those two lanes, and the leg of an arrow inside one lane in the gap under it", func(t *testing.T) {
			var inGaps, underOneLane int
			for name, model := range routeModels(t) {
				output := svgOf(t, model)
				lanes := svgLaneRects(t, output)

				var misplaced []string
				for _, leg := range svgLegs(t, output, lanes) {
					gap := drawnGapOf(lanes, leg.y)
					if leg.upperLane == leg.lowerLane {
						underOneLane++
					}
					if leg.upperLane == leg.lowerLane && gap != leg.upperLane ||
						leg.upperLane != leg.lowerLane && (gap < leg.upperLane || gap >= leg.lowerLane) {
						misplaced = append(misplaced, fmt.Sprintf("a leg at y %d joins lanes %d and %d", leg.y, leg.upperLane, leg.lowerLane))
						continue
					}
					inGaps++
				}

				require.Empty(t, misplaced, name)
			}
			require.Positive(t, underOneLane, "the models have to hold an arrow inside one lane that does not run straight")

			require.Positive(t, inGaps, "the models have to hold arrows whose legs run between lanes")
		})

		t.Run("gives two legs that overlap across in one gap different y", func(t *testing.T) {
			var overlapping int
			for name, model := range routeModels(t) {
				output := svgOf(t, model)
				lanes := svgLaneRects(t, output)
				legs := svgLegs(t, output, lanes)

				var shared []string
				for i := range legs {
					for j := i + 1; j < len(legs); j++ {
						a, b := legs[i], legs[j]
						if drawnGapOf(lanes, a.y) != drawnGapOf(lanes, b.y) || a.left > b.right || b.left > a.right {
							continue
						}
						overlapping++
						if a.y == b.y {
							shared = append(shared, fmt.Sprintf("two legs share y %d across x %d to %d", a.y, max(a.left, b.left), min(a.right, b.right)))
						}
					}
				}
				require.Empty(t, shared, name)
			}

			require.Positive(t, overlapping, "the models have to hold legs that overlap across, or there is nothing to keep apart")
		})

		t.Run("grows a gap that holds more legs than it fits, and moves the lanes below down with their boxes and the spec cards", func(t *testing.T) {
			raw, err := diagram.ExportSVG(fanInModel(), diagram.StyleAuto, diagram.WithSpecs())
			require.NoError(t, err)
			output := string(raw)
			boxes := svgBoxes(t, output)
			lanes := svgLaneRects(t, output)

			gapAbove := lanes[1].y - (lanes[0].y + lanes[0].h)
			gapWithLegs := lanes[2].y - (lanes[1].y + lanes[1].h)
			require.Greater(t, gapWithLegs, gapAbove, "the gap the eight legs run in grows past the gap that holds none")

			plain, err := diagram.ExportSVG(fanInModel(), diagram.StyleAuto)
			require.NoError(t, err)
			_, canvasH := svgCanvas(t, string(plain))
			plainLanes := svgLaneRects(t, string(plain))
			require.Greater(t, canvasH, plainLanes[3].y+plainLanes[3].h, "the picture is tall enough for its lowest lane")
			require.Equal(t, fanInEvents, labelsWithin(boxes, lanes[2]), "every event stays in the Events lane")
			require.Contains(t, labelsWithin(boxes, lanes[1]), "BalanceView")
			card := svgSpecCardRects(t, output)
			require.Len(t, card, 1)
			require.Greater(t, card[0].y, lanes[3].y+lanes[3].h, "the spec card stays under the lowest lane")
		})

		t.Run("puts the legs of one gap on no more tracks than the most legs that overlap at one x", func(t *testing.T) {
			mostOverlapping := make(map[string]int)
			tracksUsed := make(map[string]int)
			for name, model := range routeModels(t) {
				output := svgOf(t, model)
				lanes := svgLaneRects(t, output)
				inGap := make(map[int][]drawnLeg)
				for _, leg := range svgLegs(t, output, lanes) {
					inGap[drawnGapOf(lanes, leg.y)] = append(inGap[drawnGapOf(lanes, leg.y)], leg)
				}
				for gap, legs := range inGap {
					key := fmt.Sprintf("%s, gap %d", name, gap)
					ys := make(map[int]bool)
					for _, leg := range legs {
						ys[leg.y] = true
						var overlapping int
						for _, other := range legs {
							if other.left <= leg.left && other.right >= leg.left {
								overlapping++
							}
						}
						mostOverlapping[key] = max(mostOverlapping[key], overlapping)
					}
					tracksUsed[key] = len(ys)
				}
			}

			require.NotEmpty(t, tracksUsed)
			require.Equal(t, mostOverlapping, tracksUsed)
		})

		t.Run("gives an arrow between two lanes next to each other one leg at most", func(t *testing.T) {
			var arrowsChecked int
			var bent []string
			for name, model := range routeModels(t) {
				output := svgOf(t, model)
				lanes := svgLaneRects(t, output)
				ends := svgArrowBoxes(t, output)
				for _, arrow := range svgDrawnArrows(t, output) {
					from := drawnLaneOf(lanes, arrowBoxMet(t, ends, arrow.route[0]).rect.y)
					to := drawnLaneOf(lanes, arrowBoxMet(t, ends, arrow.route[len(arrow.route)-1]).rect.y)
					if from-to != 1 && to-from != 1 {
						continue
					}
					arrowsChecked++
					var legs int
					for i := 1; i < len(arrow.route); i++ {
						if arrow.route[i-1][1] == arrow.route[i][1] {
							legs++
						}
					}
					if legs > 1 {
						bent = append(bent, fmt.Sprintf("%s: %v", name, arrow.route))
					}
				}
			}

			require.Positive(t, arrowsChecked)
			require.Empty(t, bent)
		})
	})

	t.Run("renders automation as the gear with its name under it", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Automations: []*ast.Automation{{
							Name:    "Notifier",
							OnEvent: "OrderPlaced",
							Command: "SendEmail",
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		require.True(t, svgDrawnAsGear(boxLabelled(t, svgBoxes(t, string(raw)), "Notifier")))
	})

	t.Run("writes the lines under a gear one below the other, clear of each other", func(t *testing.T) {
		output := svgOf(t, test.AutomationDelayLibraryLendingModel(t))
		label := svgGearLabel(t, output, "SweepOverdueLoans")

		var size int
		for _, shape := range svgShapes(t, output) {
			if strings.Contains(shape.attributes, "href=#gear") && cardLines(shape.label)[0] == "SweepOverdueLoans" {
				parsed, err := strconv.Atoi(shape.labelSize)
				require.NoError(t, err)
				size = parsed
			}
		}

		require.Len(t, label.lines, 2, "a scheduled automation writes its cadence under its name")
		require.Positive(t, size)
		require.GreaterOrEqual(t, label.bottom-label.top, 2*size, "two lines of this size need at least twice its height")
	})

	t.Run("writes the name and the cadence of an automation centred under its gear", func(t *testing.T) {
		output := svgOf(t, sweepingModel(&ast.Automation{Name: "SweepOverdueCopies", Schedule: "0 9 * * *", Command: "RecallCopy"}))

		gear := boxLabelled(t, svgBoxes(t, output), "SweepOverdueCopies").rect
		label := svgGearLabel(t, output, "SweepOverdueCopies")

		require.Equal(t, []string{"SweepOverdueCopies", "⏱ 0 9 * * *"}, label.lines)
		require.Equal(t, "middle", label.anchor)
		require.Equal(t, gear.x+gear.w/2, label.x)
		require.GreaterOrEqual(t, label.top, gear.y+gear.h, "the name starts under the gear, not over it")
	})

	t.Run("keeps the name and the cadence of an automation inside the command and view lane", func(t *testing.T) {
		output := svgOf(t, sweepingModel(&ast.Automation{Name: "SweepOverdueCopies", Schedule: "0 9 * * *", Command: "RecallCopy"}))

		lane := boxLabelled(t, svgBoxes(t, output), "Commands / Views").rect
		label := svgGearLabel(t, output, "SweepOverdueCopies")

		require.Len(t, label.lines, 2, "the label has to carry the cadence, or the room it takes is one line short")
		require.Less(t, label.bottom, lane.y+lane.h)
	})

	t.Run("connects event to automation to command", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:        "S",
						Events:      []*ast.Event{{Name: "OrderPlaced"}},
						Commands:    []*ast.Command{{Name: "SendEmail"}},
						Automations: []*ast.Automation{{Name: "Notifier", OnEvent: "OrderPlaced", Command: "SendEmail"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "SendEmail")
		require.Contains(t, output, "Notifier")
		require.Equal(t, 2, arrowCount(output), "event to automation to command is two arrows")
	})

	t.Run("renders external system as gray dashed rounded rect", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Translations: []*ast.Translation{{
							Name:           "Import",
							ExternalSystem: "Stripe",
							Command:        "Charge",
							Event:          &ast.Event{Name: "Charged"},
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Stripe")
		require.Contains(t, output, `stroke-dasharray`)
	})

	t.Run("connects external system through reactor to command and event", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S",
						Commands: []*ast.Command{{Name: "Charge"}},
						Events:   []*ast.Event{{Name: "Charged"}},
						Translations: []*ast.Translation{{
							Name:           "Payment",
							ExternalSystem: "Stripe",
							Command:        "Charge",
							Event:          &ast.Event{Name: "Charged"},
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Charge")
		require.Contains(t, output, "Charged")
		require.Contains(t, output, "Stripe")
		// external system -> reactor, reactor -> command, command -> event
		require.Equal(t, 3, arrowCount(output))
	})

	t.Run("event with external source includes source label", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			eventWithSource("PaymentReceived", "external", "Stripe"))

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "PaymentReceived")
		require.Contains(t, output, "Stripe")
	})

	t.Run("complete model with all element types produces well-formed diagram", func(t *testing.T) {
		model := fullModel()

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Wireframes")
		require.Contains(t, output, "Commands / Views")
		require.Contains(t, output, "Events")
		require.Equal(t, 11, arrowCount(output), "every flow, subscription, automation and translation edge is drawn")
	})

	t.Run("descriptions", func(t *testing.T) {
		t.Run("hovering a shape shows the description of the construct it was drawn for", func(t *testing.T) {
			raw, err := diagram.ExportSVG(describedModel(), diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			requireValidXML(t, output)
			requireEveryDescriptionShown(t, output, svgTooltipOf)
		})

		t.Run("describing a model leaves the picture itself untouched", func(t *testing.T) {
			described, err := diagram.ExportSVG(describedModel(), diagram.StyleAuto)
			require.NoError(t, err)
			plain, err := diagram.ExportSVG(withoutDescriptions(describedModel()), diagram.StyleAuto)
			require.NoError(t, err)

			require.Equal(t, svgPicture(t, string(plain)), svgPicture(t, string(described)),
				"prose must not add, move or repaint a shape, nor disturb the arrows between shapes")
			require.NotContains(t, string(plain), "<title",
				"a model that describes nothing is written exactly as it was before shapes could be titled")
		})

		t.Run("a description written with markup characters reads back as written", func(t *testing.T) {
			prose := `Rooms held < 24h & marked "urgent"`
			model := describedModel()
			model.Contexts[0].Description = prose
			model.Contexts[0].Aggregates[0].Slices[0].Commands[0].Description = prose

			raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			requireValidXML(t, output)
			require.Equal(t, prose, svgTooltipOf(t, output, "Bookings"))
			require.Equal(t, prose, svgTooltipOf(t, output, "HoldRoom"))
		})
	})

	t.Run("rejection badges", func(t *testing.T) {
		t.Run("draws one badge per rejection edge, labelled with the invariant it names", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))

			var badged []string
			for _, box := range svgBoxes(t, output) {
				if strings.Contains(box.attributes, fillRejectionHex) {
					badged = append(badged, box.label)
				}
			}

			var expected []string
			for _, edge := range test.RejectionLibraryLendingRejections {
				expected = append(expected, edge.InvariantName)
			}
			require.Equal(t, expected, badged)
		})

		t.Run("a badge carries the invariant's statement as the title a browser shows on hover", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))

			// Both scopes an invariant can be declared in are named: the context
			// arm alone leaves the aggregate branch of the scope lookup free to
			// be deleted with the suite still green.
			require.Equal(t, "A desk seats at most one reader at any moment",
				svgTooltipOf(t, output, "OneReaderPerDesk"))

			var aggregateScoped []string
			for _, shape := range svgShapes(t, output) {
				if shape.label == "OneCopyPerLoan" {
					aggregateScoped = append(aggregateScoped, shape.tooltip)
				}
			}
			require.Equal(t, []string{
				"A loan covers exactly one copy of one title",
				"A loan covers exactly one copy of one title",
			}, aggregateScoped, "an aggregate-scoped invariant's statement reaches its badge too")
		})

		t.Run("a statement written with markup characters reads back as written", func(t *testing.T) {
			prose := `Held < 24h & marked "urgent"`
			model := test.RejectionLibraryLendingModel(t)
			model.Contexts[1].Invariants[0].Statement = prose

			output := svgOf(t, model)

			requireValidXML(t, output)
			require.Equal(t, prose, svgTooltipOf(t, output, "OneReaderPerDesk"))
		})

		t.Run("a dashed arrow runs from the rejected command to that badge, painted unlike a flow arrow", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))

			connections := svgConnections(t, output)
			var rejection, flow diagramConnection
			for _, c := range connections {
				if c.source == "ClaimDesk" && c.target == "OneReaderPerDesk" {
					rejection = c
				}
				if c.source == "ClaimDesk" && c.target == "DeskClaimed" {
					flow = c
				}
			}

			require.NotEmpty(t, flow.paint, "the flow arrow this is compared against must be in the same render")
			require.NotEmpty(t, rejection.paint, "no dashed arrow reaches the badge")
			require.NotEqual(t, flow.paint, rejection.paint,
				"a rejection must not be drawn like the flow beside it")
			require.Contains(t, rejection.paint, "stroke-dasharray")
		})

		t.Run("two slices rejecting one invariant each get their own badge and their own arrow", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))

			var badges []boxRect
			for _, box := range svgBoxes(t, output) {
				if box.label == "OneCopyPerLoan" {
					badges = append(badges, box.rect)
				}
			}
			require.Len(t, badges, 2, "each of the two slices draws its own badge")

			// Comparing the boxes the ends meet rather than their labels is the
			// whole point: both arrows resolve to the label "OneCopyPerLoan" even
			// when they end at the same box, so a label comparison cannot see the
			// collapse.
			require.ElementsMatch(t, badges, badgesReached(dashedArrowEndpoints(t, output), badges),
				"each slice's dashed arrow ends at the badge in its own column; filing a badge under the invariant's name alone points both at whichever slice was drawn last")

			require.Equal(t, []string{"BorrowCopy", "ReturnCopy"}, sourcesReaching(t, output, "OneCopyPerLoan"))
		})

		t.Run("one slice rejecting two invariants sends each arrow to its own badge", func(t *testing.T) {
			output := svgOf(t, twoRejectionsOneSliceModel())

			rectOf := func(label string) boxRect {
				for _, box := range svgBoxes(t, output) {
					if box.label == label {
						return box.rect
					}
				}
				require.FailNowf(t, "no box drawn", "%q", label)
				return boxRect{}
			}
			badges := []boxRect{rectOf("OneCopyPerLoan"), rectOf("FiveCopiesPerMember")}

			// Pairing the nth edge with badges[i][0] instead of badges[i][n]
			// orphans the second badge and stacks both arrows on the first.
			require.ElementsMatch(t, badges, badgesReached(dashedArrowEndpoints(t, output), badges))
		})

		t.Run("a badge adds one shape to its slice's event row and moves no label the twin drew", func(t *testing.T) {
			stated := test.RejectionLibraryLendingModel(t)
			unstated := test.WithoutRejections(stated)
			require.Empty(t, test.DeclaredRejections(unstated))
			require.Equal(t, test.RejectionLibraryLendingRejections, test.DeclaredRejections(stated))

			statedShapes := svgShapes(t, svgOf(t, stated))
			unstatedShapes := svgShapes(t, svgOf(t, unstated))

			require.Len(t, statedShapes, len(unstatedShapes)+len(test.RejectionLibraryLendingRejections),
				"a badge is one rect followed by exactly one text, so it adds one shape and no more")

			var statedLabels, unstatedLabels []string
			for _, shape := range statedShapes {
				if !strings.Contains(shape.attributes, fillRejectionHex) {
					statedLabels = append(statedLabels, shape.label)
				}
			}
			for _, shape := range unstatedShapes {
				unstatedLabels = append(unstatedLabels, shape.label)
			}
			require.Equal(t, unstatedLabels, statedLabels,
				"a badge emitting a stray text element overwrites the label of the box before it")
		})

		t.Run("badges overlap nothing and sit in the lane their slice's events are drawn in", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))
			boxes := svgBoxes(t, output)

			rects := drawnElementRects(t, output)
			require.Empty(t, boxesDrawnOver(rects, slices.Sorted(maps.Keys(rects))))

			eventLane := boxLabelled(t, boxes, "Events").rect
			within := labelsWithin(boxes, eventLane)
			for _, edge := range test.RejectionLibraryLendingRejections {
				require.Contains(t, within, edge.InvariantName)
			}
		})

		t.Run("a badge is narrower than a lane, so the diagram still names four lanes and keeps its viewBox", func(t *testing.T) {
			stated := test.RejectionLibraryLendingModel(t)
			statedOut := svgOf(t, stated)
			unstatedOut := svgOf(t, test.WithoutRejections(stated))

			requireValidXML(t, statedOut)
			require.Equal(t, svgLaneLabels(t, unstatedOut), svgLaneLabels(t, statedOut))
			require.Equal(t, svgViewBox(t, unstatedOut), svgViewBox(t, statedOut),
				"a badge takes a place in a row that already exists rather than growing the canvas")
		})
	})

	t.Run("spec cards", func(t *testing.T) {
		t.Run("draws a card under each slice stating a scenario and none under a slice stating none", func(t *testing.T) {
			output := svgWithSpecs(t, test.SlicePatternLibraryLendingModel(t))

			cards := svgSpecCards(t, output)
			require.Len(t, cards, 7,
				"seven of the fixture's eight slices state a scenario; Desk Occupancy states none")

			column := boxNamed(t, svgBoxes(t, output), "DeskOccupancyView").rect.centre()[0]
			for _, card := range svgSpecCardRects(t, output) {
				require.False(t, card.x <= column && column < card.x+card.w,
					"a card is drawn in the column of the one slice stating no scenario")
			}
		})

		t.Run("the cards name every scenario the model states, in declaration order", func(t *testing.T) {
			output := svgWithSpecs(t, test.SlicePatternLibraryLendingModel(t))

			var drawn []string
			for _, card := range svgSpecCards(t, output) {
				drawn = append(drawn, specNamesOn(card)...)
			}

			require.Equal(t, test.SlicePatternLibraryLendingSpecNames, drawn,
				"the cards name the scenarios of both slice homes, in the order they are written, whether or not a name had to be wrapped")
		})

		t.Run("an events outcome names the events, under the given and when it follows", func(t *testing.T) {
			output := svgWithSpecs(t, test.SlicePatternLibraryLendingModel(t))

			require.Equal(t, []string{
				`"borrows a copy the member before returned"`,
				"given [CopyBorrowed, CopyReturned]",
				"when BorrowCopy",
				"then [CopyBorrowed]",
			}, svgSpecCardBlock(t, output, "borrows a copy the member before returned"))
		})

		t.Run("a rejected outcome names the invariant", func(t *testing.T) {
			output := svgWithSpecs(t, test.SlicePatternLibraryLendingModel(t))

			require.Equal(t, []string{
				`"refuses a copy already on loan"`,
				"when BorrowCopy",
				"then rejected OneCopyPerLoan",
			}, svgSpecCardBlock(t, output, "refuses a copy already on loan"))
		})

		t.Run("a view outcome names the view", func(t *testing.T) {
			output := svgWithSpecs(t, test.SlicePatternLibraryLendingModel(t))

			require.Equal(t, []string{
				`"lists the loans a member holds"`,
				"then view MemberLoansView",
			}, svgSpecCardBlock(t, output, "lists the loans a member holds"))
		})

		t.Run("a command outcome names the command", func(t *testing.T) {
			output := svgWithSpecs(t, test.SlicePatternLibraryLendingModel(t))

			require.Equal(t, []string{
				`"reminds a member when a copy becomes due"`,
				"when CopyBorrowed",
				"then command RemindMember",
			}, svgSpecCardBlock(t, output, "reminds a member when a copy becomes due"))
		})

		t.Run("a rejected outcome states the invariant's name and nowhere the prose it stands for", func(t *testing.T) {
			model := test.SlicePatternLibraryLendingModel(t)
			output := svgWithSpecs(t, model)

			statements := invariantStatementsOf(model)
			require.NotEmpty(t, statements,
				"the fixture has to declare an invariant with prose, or the absence below is satisfied by there being none")
			require.Contains(t, output, "then rejected OneCopyPerLoan",
				"the rejection has to be drawn, or the absence below is satisfied by drawing no card at all")

			for _, statement := range statements {
				require.NotContains(t, output, statement,
					"a card names an invariant; its prose is what a rejection edge carries on hover")
			}
		})

		t.Run("a scenario omitting given and one writing an empty given state the same lines", func(t *testing.T) {
			output := svgWithSpecs(t, twoGivenSpellingsModel())

			cards := svgSpecCards(t, output)
			require.Len(t, cards, 2)
			require.Equal(t, cards[0], cards[1],
				"emod fmt writes no given line for either spelling, so the card and the formatted source agree")
			require.NotContains(t, cards[0], "given")
		})

		t.Run("a scenario stating no when keeps its given and its then", func(t *testing.T) {
			model := singleSpecModel("Sweep Overdue Loans", &ast.Spec{
				Name:  "recalls copies that are overdue",
				Given: []*ast.SpecElement{{Name: "CopyBorrowed"}},
				Then:  &ast.ThenCommand{CommandName: "RecallCopy"},
			})

			require.Equal(t, []string{
				`"recalls copies that are overdue"`,
				"given [CopyBorrowed]",
				"then command RecallCopy",
			}, svgSpecCardBlock(t, svgWithSpecs(t, model), "recalls copies that are overdue"))
		})

		t.Run("a scenario naming one unbroken word wider than a card is cut, not drawn past it", func(t *testing.T) {
			// A name of ordinary words breaks at a space; this one has nowhere
			// to break, which is the only path that cuts mid-word.
			name := "refusesAReservationWhenTheRoomIsAlreadyBookedForEveryRequestedNight"
			model := singleSpecModel("Reserve Room", &ast.Spec{
				Name: name,
				When: &ast.SpecElement{Name: "ReserveRoom"},
				Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "RoomReserved"}}},
			})

			cards := svgSpecCards(t, svgWithSpecs(t, model))
			require.Len(t, cards, 1)

			lines := strings.Split(cards[0], "\n")
			require.Greater(t, len(lines), 3, "the word has to have been cut, or this asserts nothing")
			for _, line := range lines {
				require.LessOrEqual(t, utf8.RuneCountInString(line), 44,
					"an unbreakable word wider than the card is drawn outside it unless it is cut")
			}
			require.Equal(t, name, strings.ReplaceAll(strings.Trim(strings.Join(lines[:len(lines)-2], ""), `"`), " ", ""),
				"the cut pieces have to still spell the name the author wrote")
		})

		t.Run("a card adds one shape and leaves every shape drawn without it exactly where it was", func(t *testing.T) {
			model := test.SlicePatternLibraryLendingModel(t)
			plain := svgOf(t, model)
			featured := svgWithSpecs(t, model)

			plainShapes := svgShapes(t, plain)
			featuredShapes := svgShapes(t, featured)
			cards := svgSpecCards(t, featured)
			require.NotEmpty(t, cards, "the featured render has to draw a card, or the counts below agree trivially")

			require.Len(t, featuredShapes, len(plainShapes)+len(cards)+1,
				"a card is one rect, and the band holding them is one shape more")
			require.Equal(t, plainShapes, featuredShapes[:len(plainShapes)],
				"a text emitted before a card's own rect relabels the box drawn last, and a band that reflows moves one")

			// svgShapes counts rects and lets a later text overwrite an earlier
			// one, so it cannot see a card that draws its text twice — which is
			// two labels painted over each other in the picture a reader opens.
			require.Equal(t, strings.Count(plain, "<text")+len(cards)+1, strings.Count(featured, "<text"),
				"a card draws one text element however many lines it states, and the band's own label is one more")
		})

		t.Run("the arrows are the ones drawn without the option", func(t *testing.T) {
			model := test.SlicePatternLibraryLendingModel(t)

			plain := svgConnections(t, svgOf(t, model))
			require.NotEmpty(t, plain, "the model has to draw an arrow, or the comparison below says nothing")

			require.Equal(t, plain, svgConnections(t, svgWithSpecs(t, model)),
				"a card is drawn into no arrow and captures no arrow's endpoint")
		})

		t.Run("no two boxes the featured render draws overlap, cards included", func(t *testing.T) {
			output := svgWithSpecs(t, test.SlicePatternLibraryLendingModel(t))
			require.NotEmpty(t, svgSpecCards(t, output),
				"the render has to draw a card for the cards to be in the comparison")

			rects := drawnElementRects(t, output)
			require.Empty(t, boxesDrawnOver(rects, slices.Sorted(maps.Keys(rects))))
		})

		t.Run("the band names a fifth lane and the picture grows downwards only", func(t *testing.T) {
			model := test.SlicePatternLibraryLendingModel(t)
			plain := svgOf(t, model)
			featured := svgWithSpecs(t, model)

			requireValidXML(t, featured)
			require.Equal(t,
				append(svgLaneLabels(t, plain), "Specs"),
				svgLaneLabels(t, featured),
				"the band spans the picture like a lane, so it reads back as one alongside the four")

			plainW, plainH := svgCanvas(t, plain)
			featuredW, featuredH := svgCanvas(t, featured)
			require.Equal(t, plainW, featuredW, "a card sits inside its slice's column, so nothing widens")
			require.Greater(t, featuredH, plainH, "the band is drawn below the lowest lane, so the canvas grows")

			// Growing is not the claim; holding the band is. A canvas that
			// stops partway down the band satisfies "taller than before".
			for label, rect := range drawnElementRects(t, featured) {
				require.LessOrEqual(t, rect.x+rect.w, featuredW, "%s is drawn past the right edge of the canvas", label)
				require.LessOrEqual(t, rect.y+rect.h, featuredH, "%s is drawn below the bottom of the canvas", label)
			}
		})

		t.Run("a scenario named longer than a card fits is wrapped, not drawn past the card", func(t *testing.T) {
			name := "refuses a reservation when the room is already booked for every one of the requested nights"
			model := singleSpecModel("Reserve Room", &ast.Spec{
				Name: name,
				When: &ast.SpecElement{Name: "ReserveRoom"},
				Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "RoomReserved"}}},
			})

			output := svgWithSpecs(t, model)
			cards := svgSpecCards(t, output)
			require.Len(t, cards, 1)

			lines := strings.Split(cards[0], "\n")
			require.Greater(t, len(lines), 3,
				"the name has to have been broken across lines, or this asserts nothing about wrapping")
			for _, line := range lines {
				require.LessOrEqual(t, utf8.RuneCountInString(line), 44,
					"a line wider than the card is drawn outside it: svg never wraps")
			}
			require.Equal(t, name, strings.Trim(strings.Join(lines[:len(lines)-2], " "), `"`),
				"the wrapped lines have to still read as the name the author wrote")

			// The card must have been measured for the lines it actually holds.
			card := svgSpecCardRects(t, output)[0]
			_, canvasH := svgCanvas(t, output)
			require.LessOrEqual(t, card.y+card.h, canvasH)
			require.GreaterOrEqual(t, card.h, len(lines)*14, "the card is sized from the wrapped line count")
		})
	})
}

// invariantStatementsOf collects the prose every invariant of the model states,
// from both the aggregates and the contexts that declare one.
func invariantStatementsOf(model *ast.Model) []string {
	var statements []string
	for _, ctx := range model.Contexts {
		for _, inv := range ctx.Invariants {
			statements = append(statements, inv.Statement)
		}
		for _, agg := range ctx.Aggregates {
			for _, inv := range agg.Invariants {
				statements = append(statements, inv.Statement)
			}
		}
	}

	return statements
}

// twoGivenSpellingsModel writes one scenario twice, once omitting given and once
// writing it empty — the two spellings emod fmt collapses onto the same line-less
// output. The shared fixture states both, but within one slice, so it cannot ask
// whether the two cards agree.
func twoGivenSpellingsModel() *ast.Model {
	scenario := func(given []*ast.SpecElement) *ast.Spec {
		return &ast.Spec{
			Name:  "seats a reader at a free desk",
			Given: given,
			When:  &ast.SpecElement{Name: "ClaimDesk"},
			Then:  &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "DeskClaimed"}}},
		}
	}

	return &ast.Model{
		Name: "Reading Room",
		Contexts: []*ast.Context{{
			Name: "Reading Room",
			Aggregates: []*ast.Aggregate{{
				Name: "Desk",
				Slices: []*ast.Slice{
					{Name: "Claim Desk", Specs: []*ast.Spec{scenario(nil)}},
					{Name: "Claim Another Desk", Specs: []*ast.Spec{scenario([]*ast.SpecElement{})}},
				},
			}},
		}},
	}
}

// singleSpecModel puts one scenario on a slice of its own, for the shapes the
// shared fixture cannot state — a scenario naming a given and an outcome but no
// when among them.
func singleSpecModel(sliceName string, spec *ast.Spec) *ast.Model {
	return &ast.Model{
		Name: "Library Lending",
		Contexts: []*ast.Context{{
			Name: "Lending",
			Aggregates: []*ast.Aggregate{{
				Name:   "Loan",
				Slices: []*ast.Slice{{Name: sliceName, Specs: []*ast.Spec{spec}}},
			}},
		}},
	}
}

func svgWithSpecs(t *testing.T, model *ast.Model) string {
	t.Helper()

	raw, err := diagram.ExportSVG(model, diagram.StyleAuto, diagram.WithSpecs())
	require.NoError(t, err)

	return string(raw)
}

// fillSpecCardHex is stated here rather than read from the production constant,
// so a test compares the bytes a reader sees against a value written down
// independently of the code that emits them.
const fillSpecCardHex = "#fff2cc"

// svgSpecCards returns the text of each card the picture draws, in document
// order, one string per card with its lines as the writer joined them.
func svgSpecCards(t *testing.T, output string) []string {
	t.Helper()

	var cards []string
	for _, card := range specCardBoxesIn(t, svgBoxes(t, output)) {
		cards = append(cards, strings.TrimSpace(card.label))
	}

	return cards
}

// svgSpecCardRects returns where each card was drawn, in the order svgSpecCards
// returns their text.
func svgSpecCardRects(t *testing.T, output string) []boxRect {
	t.Helper()

	var rects []boxRect
	for _, card := range specCardBoxesIn(t, svgBoxes(t, output)) {
		rects = append(rects, card.rect)
	}

	return rects
}

// svgSpecCardBlock returns the lines a card states for the scenario named name:
// its quoted name and everything under it, up to the quoted name opening the
// next scenario.
func svgSpecCardBlock(t *testing.T, output, name string) []string {
	t.Helper()

	quoted := `"` + name + `"`
	for _, card := range svgSpecCards(t, output) {
		lines := strings.Split(card, "\n")
		for i, line := range lines {
			if line != quoted {
				continue
			}
			block := []string{line}
			for _, next := range lines[i+1:] {
				if strings.HasPrefix(next, `"`) {
					break
				}
				block = append(block, next)
			}
			return block
		}
	}

	require.FailNowf(t, "no card states the scenario", "%q", name)
	return nil
}

// svgCanvas is the width and height the diagram's viewBox declares.
func svgCanvas(t *testing.T, output string) (int, int) {
	t.Helper()

	bounds := strings.Fields(svgViewBox(t, output))
	require.Len(t, bounds, 4, "a viewBox states an origin and a size")

	w, err := strconv.Atoi(bounds[2])
	require.NoError(t, err, "a viewBox states a whole-number width")
	h, err := strconv.Atoi(bounds[3])
	require.NoError(t, err, "a viewBox states a whole-number height")

	return w, h
}

// fillRejectionHex is stated here rather than read from the production
// constants, so a test compares the bytes a reader sees against a value written
// down independently of the code that emits it.
const fillRejectionHex = "#f8cecc"

// twoRejectionsOneSliceModel gives a single slice two rejection edges, the shape
// that tells a per-edge badge index apart from one that always reaches the
// slice's first badge. The shared fixture states one edge per slice, so it
// cannot ask this.
func twoRejectionsOneSliceModel() *ast.Model {
	return &ast.Model{
		Name: "Library Lending",
		Contexts: []*ast.Context{{
			Name: "Lending",
			Aggregates: []*ast.Aggregate{{
				Name: "Loan",
				Invariants: []*ast.Invariant{
					{Name: "OneCopyPerLoan", Statement: "A loan covers exactly one copy"},
					{Name: "FiveCopiesPerMember", Statement: "A member holds at most five copies"},
				},
				Slices: []*ast.Slice{{
					Name:     "Borrow Copy",
					Commands: []*ast.Command{{Name: "BorrowCopy"}},
					Events:   []*ast.Event{{Name: "CopyBorrowed"}},
					Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
					Rejections: []*ast.Rejection{
						{CommandName: "BorrowCopy", InvariantName: "OneCopyPerLoan"},
						{CommandName: "BorrowCopy", InvariantName: "FiveCopiesPerMember"},
					},
				}},
			}},
		}},
	}
}

func svgOf(t *testing.T, model *ast.Model) string {
	t.Helper()

	raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
	require.NoError(t, err)

	return string(raw)
}

var svgViewBoxAttribute = regexp.MustCompile(`viewBox="([^"]*)"`)

func svgViewBox(t *testing.T, output string) string {
	t.Helper()

	found := svgViewBoxAttribute.FindStringSubmatch(output)
	require.NotNil(t, found, "the diagram states no viewBox")

	return found[1]
}

// dashedArrowEndpoints returns the point each dashed arrow ends at, so a test can
// say which box an arrow reached rather than which label that box happens to
// carry — two boxes may carry the same one.
func dashedArrowEndpoints(t *testing.T, output string) [][2]int {
	t.Helper()

	var ends [][2]int
	for _, arrow := range svgArrows(output) {
		if !strings.Contains(arrow, "stroke-dasharray") {
			continue
		}
		path := svgPathData.FindStringSubmatch(arrow)
		require.NotNil(t, path, "an arrow carries no path: %s", arrow)

		points := svgPathPoint.FindAllStringSubmatch(path[1], -1)
		require.NotEmpty(t, points, "an arrow's path names no point: %s", arrow)

		ends = append(ends, svgPoint(t, points[len(points)-1]))
	}

	return ends
}

func sourcesReaching(t *testing.T, output, label string) []string {
	t.Helper()

	var sources []string
	for _, c := range svgConnections(t, output) {
		if c.target == label {
			sources = append(sources, c.source)
		}
	}

	return sources
}

// drawnElementRects keys every labelled box by its label and the position it was
// drawn at, so two slices drawing a badge for one invariant are two entries
// rather than one overwriting the other. The lanes are left out: a lane is drawn
// around the boxes it holds, so it overlaps all of them by design.
func drawnElementRects(t *testing.T, output string) map[string]boxRect {
	t.Helper()

	lanes := make(map[string]bool)
	for _, label := range svgLaneLabels(t, output) {
		lanes[label] = true
	}

	rects := make(map[string]boxRect)
	for i, box := range svgBoxes(t, output) {
		if box.label == "" || lanes[box.label] {
			continue
		}
		rects[fmt.Sprintf("%s#%d", box.label, i)] = box.rect
	}

	return rects
}

// --- svg helpers ---

func arrowCount(output string) int {
	return len(svgArrows(output))
}

// svgArrowLabels names, for every arrow svgArrows returns and in the same
// order, the text painted on it — "" for an arrow carrying none. The writer
// paints a label on the line immediately after the arrow it belongs to, so the
// two are read together rather than guessed at from where each landed.
func svgArrowLabels(t *testing.T, output string) []string {
	t.Helper()

	lines := strings.Split(output, "\n")

	var labels []string
	for i, line := range lines {
		if !strings.Contains(line, "marker-end") {
			continue
		}

		label := ""
		if i+1 < len(lines) && strings.Contains(lines[i+1], `class="`+edgeLabelClass+`"`) {
			label = svgTextContentOf(t, lines[i+1])
		}
		labels = append(labels, label)
	}

	return labels
}

// svgTextContentOf reads the text a <text> element shows, decoded, so an
// assertion names the duration an author wrote rather than its escaped form.
func svgTextContentOf(t *testing.T, element string) string {
	t.Helper()

	var text struct {
		Content string `xml:",chardata"`
	}
	require.NoError(t, xml.Unmarshal([]byte(element), &text))

	return text.Content
}

// svgArrows returns the arrows the diagram draws between its boxes.
func svgArrows(output string) []string {
	var arrows []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "marker-end") {
			arrows = append(arrows, line)
		}
	}

	return arrows
}

type svgShape struct {
	attributes string
	rect       boxRect
	label      string
	tooltip    string
	// labelAt is the point the label is centred on, and labelSize its font size.
	labelAt   [2]int
	labelSize string
}

// svgShapes returns the diagram's boxes, and the gear an automation is drawn as,
// in document order, decoded through an XML parser so a test sees the text a
// reader sees rather than its escaped form.
// A browser shows only the title nested inside the box being hovered, so a title
// written anywhere else belongs to no box here either; labels are drawn as text
// siblings, so those attach to the nearest box before them.
func svgShapes(t *testing.T, output string) []svgShape {
	t.Helper()

	var (
		shapes      []svgShape
		inRect      bool
		inEdgeLabel bool
		text        strings.Builder
	)

	decoder := xml.NewDecoder(strings.NewReader(output))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err, "output must be well-formed XML")

		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "rect", "use":
				shapes = append(shapes, svgShape{attributes: svgAttributes(element), rect: svgRectOf(t, element)})
				inRect = true
			case "title":
				text.Reset()
			case "text":
				text.Reset()
				inEdgeLabel = svgAttributeValue(element, "class") == edgeLabelClass
				if len(shapes) > 0 && !inEdgeLabel {
					x, err := strconv.Atoi(svgAttributeValue(element, "x"))
					require.NoError(t, err, "a label's x must be a whole number")
					y, err := strconv.Atoi(svgAttributeValue(element, "y"))
					require.NoError(t, err, "a label's y must be a whole number")
					shapes[len(shapes)-1].labelAt = [2]int{x, y}
					shapes[len(shapes)-1].labelSize = svgAttributeValue(element, "font-size")
				}
			}
		case xml.CharData:
			text.Write(element)
		case xml.EndElement:
			switch element.Name.Local {
			case "rect", "use":
				inRect = false
			case "title":
				if inRect {
					shapes[len(shapes)-1].tooltip = text.String()
				}
			case "text":
				// An arrow's label is text the picture draws outside any box.
				// Binding it to the last rect seen would rename that box, and
				// every reader of a box label reads through this one.
				if len(shapes) > 0 && !inEdgeLabel {
					shapes[len(shapes)-1].label = text.String()
				}
				inEdgeLabel = false
			}
		}
	}

	return shapes
}

// svgCopyShapes selects the copies: the shapes whose second line names the slice
// that declares what they copy.
func svgCopyShapes(shapes []svgShape) []svgShape {
	var copies []svgShape
	for _, shape := range shapes {
		lines := cardLines(shape.label)
		if len(lines) == 2 && (strings.HasPrefix(lines[1], "from ") || strings.HasPrefix(lines[1], "in ")) {
			copies = append(copies, shape)
		}
	}

	return copies
}

func svgBoxes(t *testing.T, output string) []diagramBox {
	t.Helper()

	var boxes []diagramBox
	for _, shape := range svgShapes(t, output) {
		boxes = append(boxes, diagramBox{label: shape.label, attributes: shape.attributes, rect: shape.rect})
	}

	return boxes
}

// svgLabel is the text the SVG writes under a gear: where its lines are centred
// across, and how far down its first line starts and its last line ends.
type svgLabel struct {
	lines       []string
	anchor      string
	x           int
	top, bottom int
}

// svgGearLabel reads the text written after the gear whose label carries name,
// adding up the dy of each line to find where the lines are drawn.
func svgGearLabel(t *testing.T, output, name string) svgLabel {
	t.Helper()

	measure := func(element xml.StartElement, attribute string) int {
		value, err := strconv.Atoi(svgAttributeValue(element, attribute))
		require.NoError(t, err, "%s of %s must be a whole number", attribute, element.Name.Local)
		return value
	}

	var (
		afterGear bool
		inLabel   bool
		label     svgLabel
		y, size   int
		lineY     []int
		text      strings.Builder
	)

	decoder := xml.NewDecoder(strings.NewReader(output))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err, "output must be well-formed XML")

		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "use":
				afterGear = true
			case "rect":
				afterGear = false
			case "text":
				if afterGear {
					inLabel = true
					label = svgLabel{anchor: svgAttributeValue(element, "text-anchor"), x: measure(element, "x")}
					y, size, lineY = measure(element, "y"), measure(element, "font-size"), nil
					text.Reset()
				}
			case "tspan":
				if inLabel {
					at := y
					if len(lineY) > 0 {
						at = lineY[len(lineY)-1]
					}
					lineY = append(lineY, at+measure(element, "dy"))
					text.Reset()
				}
			}
		case xml.CharData:
			if inLabel {
				text.Write(element)
			}
		case xml.EndElement:
			switch element.Name.Local {
			case "tspan":
				label.lines = append(label.lines, text.String())
			case "text":
				if !inLabel {
					continue
				}
				if len(lineY) == 0 {
					label.lines, lineY = []string{text.String()}, []int{y}
				}
				if label.lines[0] == name {
					label.top, label.bottom = lineY[0]-size/2, lineY[len(lineY)-1]+size/2
					return label
				}
				inLabel, afterGear = false, false
			}
		}
	}

	require.Fail(t, "no text is written under a gear named "+name)
	return svgLabel{}
}

func svgDrawnAsGear(box diagramBox) bool {
	return strings.Contains(box.attributes, "href=#gear")
}

// svgLaneLabels names the swimlanes the diagram draws, in the order it draws
// them. A lane is the only shape drawn as a band across the whole picture, so
// the widest shapes are the lanes and nothing else is.
func svgLaneLabels(t *testing.T, output string) []string {
	t.Helper()

	shapes := svgShapes(t, output)

	var widest int
	for _, shape := range shapes {
		widest = max(widest, shape.rect.w)
	}

	var labels []string
	for _, shape := range shapes {
		if shape.rect.w == widest {
			labels = append(labels, shape.label)
		}
	}

	return labels
}

func svgRectOf(t *testing.T, element xml.StartElement) boxRect {
	t.Helper()

	measure := func(name string) int {
		for _, a := range element.Attr {
			if a.Name.Local != name {
				continue
			}
			value, err := strconv.Atoi(a.Value)
			require.NoError(t, err, "a box's %s must be a whole number", name)
			return value
		}
		require.Fail(t, "a box is drawn with no "+name)
		return 0
	}

	return boxRect{x: measure("x"), y: measure("y"), w: measure("width"), h: measure("height")}
}

var (
	svgPathData  = regexp.MustCompile(`\sd="([^"]*)"`)
	svgPathPoint = regexp.MustCompile(`(-?\d+),(-?\d+)`)
)

// svgArrowEnds answers the boxes at the start and at the end of each arrow, with
// where each one is drawn, for an arrow whose two ends share a label with
// another box.
func svgArrowEnds(t *testing.T, output string) [][2]svgArrowBox {
	t.Helper()

	boxes := svgArrowBoxes(t, output)
	var ends [][2]svgArrowBox
	for _, arrow := range svgArrows(output) {
		path := svgPathData.FindStringSubmatch(arrow)
		require.NotNil(t, path, "an arrow carries no path: %s", arrow)
		points := svgPathPoint.FindAllStringSubmatch(path[1], -1)
		require.NotEmpty(t, points, "an arrow's path names no point: %s", arrow)

		ends = append(ends, [2]svgArrowBox{
			arrowBoxMet(t, boxes, svgPoint(t, points[0])),
			arrowBoxMet(t, boxes, svgPoint(t, points[len(points)-1])),
		})
	}

	return ends
}

// firstShapeLabelled returns the first shape, in document order, whose label
// is label.
func firstShapeLabelled(t *testing.T, shapes []svgShape, label string) svgShape {
	t.Helper()

	i := slices.IndexFunc(shapes, func(shape svgShape) bool { return lineText(shape.label) == label })
	require.GreaterOrEqual(t, i, 0, "the picture draws no box labelled %q", label)

	return shapes[i]
}

// columnsUnder names the slices whose boxes, among those that slicesOf files
// under a slice, run below or above rect, in name order.
func columnsUnder(shapes []svgShape, rect boxRect, slicesOf map[string]string) string {
	var names []string
	for _, shape := range shapes {
		name, filed := slicesOf[lineText(shape.label)]
		if filed && shape.rect.x < rect.x+rect.w && rect.x < shape.rect.x+shape.rect.w && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	return strings.Join(names, ", ")
}

// svgConnections returns the arrows the diagram draws, each named by the boxes
// whose outline its two ends lie on and carrying how it is painted, so a test
// can say which boxes an arrow runs between instead of restating its
// coordinates.
func svgConnections(t *testing.T, output string) []diagramConnection {
	t.Helper()

	boxes := svgArrowBoxes(t, output)
	boxAt := func(point [2]int) string {
		return arrowBoxMet(t, boxes, point).label
	}

	labels := svgArrowLabels(t, output)

	var connections []diagramConnection
	for i, arrow := range svgArrows(output) {
		path := svgPathData.FindStringSubmatch(arrow)
		require.NotNil(t, path, "an arrow carries no path: %s", arrow)

		points := svgPathPoint.FindAllStringSubmatch(path[1], -1)
		require.NotEmpty(t, points, "an arrow's path names no point: %s", arrow)

		connections = append(connections, diagramConnection{
			source: boxAt(svgPoint(t, points[0])),
			target: boxAt(svgPoint(t, points[len(points)-1])),
			paint:  strings.TrimSpace(svgPathData.ReplaceAllString(arrow, "")),
			label:  labels[i],
		})
	}

	return connections
}

// svgArrowBox is the box an arrow meets a shape at: the shape itself, or for an
// automation its gear and the lines written under it.
type svgArrowBox struct {
	label string
	rect  boxRect
}

func svgArrowBoxes(t *testing.T, output string) []svgArrowBox {
	t.Helper()

	var boxes []svgArrowBox
	for _, shape := range svgShapes(t, output) {
		rect := shape.rect
		if strings.Contains(shape.attributes, "href=#gear") {
			rect.h = svgGearLabel(t, output, cardLines(shape.label)[0]).bottom - rect.y
		}
		boxes = append(boxes, svgArrowBox{label: shape.label, rect: rect})
	}

	return boxes
}

// arrowBoxMet returns the one box whose outline point lies on, and fails loudly
// on a point that lies on none or on more than one.
func arrowBoxMet(t *testing.T, boxes []svgArrowBox, point [2]int) svgArrowBox {
	t.Helper()

	var met []svgArrowBox
	for _, box := range boxes {
		if box.rect.meets(point) {
			met = append(met, box)
		}
	}
	require.Len(t, met, 1, "an arrow ends at %v, which has to lie on the outline of exactly one box", point)

	return met[0]
}

// badgesReached returns, for each end, the badge whose outline it lies on.
func badgesReached(ends [][2]int, badges []boxRect) []boxRect {
	var reached []boxRect
	for _, end := range ends {
		for _, badge := range badges {
			if badge.meets(end) {
				reached = append(reached, badge)
			}
		}
	}

	return reached
}

// drawnArrow is an arrow as the SVG draws it: the points of its route, and the
// point its text is centred on when it carries any.
type drawnArrow struct {
	route   [][2]int
	labelAt *[2]int
}

var svgTextAt = regexp.MustCompile(`\bx="(-?\d+)" y="(-?\d+)"`)

func svgDrawnArrows(t *testing.T, output string) []drawnArrow {
	t.Helper()

	lines := strings.Split(output, "\n")

	var arrows []drawnArrow
	for i, line := range lines {
		if !strings.Contains(line, "marker-end") {
			continue
		}
		path := svgPathData.FindStringSubmatch(line)
		require.NotNil(t, path, "an arrow carries no path: %s", line)

		var arrow drawnArrow
		for _, point := range svgPathPoint.FindAllStringSubmatch(path[1], -1) {
			arrow.route = append(arrow.route, svgPoint(t, point))
		}
		if i+1 < len(lines) && strings.Contains(lines[i+1], `class="`+edgeLabelClass+`"`) {
			at := svgPoint(t, svgTextAt.FindStringSubmatch(lines[i+1]))
			arrow.labelAt = &at
		}
		arrows = append(arrows, arrow)
	}

	return arrows
}

// routeCrossesY reports whether a segment of route that runs up or down crosses
// y between left and right.
func routeCrossesY(route [][2]int, left, right, y int) bool {
	for i := 1; i < len(route); i++ {
		a, b := route[i-1], route[i]
		if a[0] == b[0] && a[0] >= left && a[0] <= right && y >= min(a[1], b[1]) && y <= max(a[1], b[1]) {
			return true
		}
	}

	return false
}

// svgObstacles returns every box an arrow must keep clear of: every shape but the
// lanes, the band of spec cards and the bars over the contexts, and for an
// automation the box its gear and the lines under it take, about 5 px wide for a
// character of its widest line.
func svgObstacles(t *testing.T, output string) []svgArrowBox {
	t.Helper()

	lanes := svgLaneRects(t, output)
	var obstacles []svgArrowBox
	for _, shape := range svgShapes(t, output) {
		if slices.Contains(lanes, shape.rect) || lineText(shape.label) == "Specs" || shape.rect.y < lanes[0].y {
			continue
		}
		rect := shape.rect
		if strings.Contains(shape.attributes, "href=#gear") {
			label := svgGearLabel(t, output, cardLines(shape.label)[0])
			widest := 0
			for _, line := range label.lines {
				widest = max(widest, utf8.RuneCountInString(line))
			}
			half := max(rect.w/2, widest*5/2)
			rect = boxRect{x: label.x - half, y: rect.y, w: 2 * half, h: label.bottom - rect.y}
		}
		obstacles = append(obstacles, svgArrowBox{label: shape.label, rect: rect})
	}

	return obstacles
}

// routePasses reports whether p lies on one of the segments of route.
func routePasses(route [][2]int, p [2]int) bool {
	for i := 1; i < len(route); i++ {
		a, b := route[i-1], route[i]
		if p[0] >= min(a[0], b[0]) && p[0] <= max(a[0], b[0]) && p[1] >= min(a[1], b[1]) && p[1] <= max(a[1], b[1]) {
			return true
		}
	}

	return false
}

// svgLaneRects returns the four lanes, from the top.
func svgLaneRects(t *testing.T, output string) []boxRect {
	t.Helper()

	boxes := svgBoxes(t, output)
	var lanes []boxRect
	for _, name := range []string{"Wireframes", "Commands / Views", "Events", "External Systems"} {
		lanes = append(lanes, boxNamed(t, boxes, name).rect)
	}

	return lanes
}

// svgEventsLane is the place of the Events lane among the lanes, from the top.
const svgEventsLane = 2

// drawnLaneOf returns the lane a box starting at y sits in: the lowest lane that
// starts above it.
func drawnLaneOf(lanes []boxRect, y int) int {
	lane := 0
	for i, l := range lanes {
		if l.y <= y {
			lane = i
		}
	}

	return lane
}

// drawnGapOf returns the gap y lies in, gap i being the one under lane i, or -1
// when y lies in no gap.
func drawnGapOf(lanes []boxRect, y int) int {
	for i := 0; i+1 < len(lanes); i++ {
		if y > lanes[i].y+lanes[i].h && y < lanes[i+1].y {
			return i
		}
	}

	return -1
}

// drawnLeg is the horizontal leg of an arrow, with the lanes of the two boxes
// the arrow joins.
type drawnLeg struct {
	y, left, right       int
	upperLane, lowerLane int
}

func svgLegs(t *testing.T, output string, lanes []boxRect) []drawnLeg {
	t.Helper()

	boxes := svgArrowBoxes(t, output)
	var legs []drawnLeg
	for _, arrow := range svgDrawnArrows(t, output) {
		from := drawnLaneOf(lanes, arrowBoxMet(t, boxes, arrow.route[0]).rect.y)
		to := drawnLaneOf(lanes, arrowBoxMet(t, boxes, arrow.route[len(arrow.route)-1]).rect.y)
		for i := 1; i < len(arrow.route); i++ {
			a, b := arrow.route[i-1], arrow.route[i]
			if a[1] == b[1] && a[0] != b[0] {
				legs = append(legs, drawnLeg{
					y: a[1], left: min(a[0], b[0]), right: max(a[0], b[0]),
					upperLane: min(from, to), lowerLane: max(from, to),
				})
			}
		}
	}

	return legs
}

// routeModels are the models the routes of arrows are checked over: the models
// below, each built for one way an arrow can run, and every example that parses.
func routeModels(t *testing.T) map[string]*ast.Model {
	t.Helper()

	paths, err := filepath.Glob("../../examples/*.emod")
	require.NoError(t, err)

	models := map[string]*ast.Model{
		"forty slices":                 fortySliceModel(),
		"one view fed by eight events": fanInModel(),
		"a translation that records through the command of the slice before it":              nextSliceTranslationModel(),
		"an automation whose schedule is wider than its name":                                wideScheduleModel(),
		"two automations that read views of the slice before and issue each other's command": crossedAutomationsModel(),
		"two translations in one slice":                                                      twoTranslationsModel(),
		"two arrows that cross the Events lane beside each slice":                            sharedRoomTranslationsModel(),
		"a delayed arrow over commands that flow straight down":                              delayOverStraightArrowsModel(),
		"more arrows beside one slice than the room beside it holds":                         fullRoomModel(),
	}
	var examples int
	for _, path := range paths {
		source, err := os.ReadFile(path)
		require.NoError(t, err)
		if model, diagnostics := parser.Parse(string(source), path); len(diagnostics) == 0 {
			models[filepath.Base(path)] = model
			examples++
		}
	}
	require.Positive(t, examples, "the examples have to hold at least one model that parses")

	return models
}

// labelledModels are the leg models and a model that delays two automations,
// so that arrows carry text.
func labelledModels(t *testing.T) map[string]*ast.Model {
	t.Helper()

	models := routeModels(t)
	models["two delayed automations"] = test.AutomationDelayLibraryLendingModel(t)

	return models
}

// delayOverStraightArrowsModel delays an automation on the first of four events
// of the slice before it. The leg of that arrow runs over the arrows from the
// other three commands straight down to their events, and the middle of the
// leg is 12 px from one of them.
func delayOverStraightArrowsModel() *ast.Model {
	return slicesModel(
		&ast.Slice{
			Name:     "Lend",
			Commands: []*ast.Command{{Name: "BorrowCopy"}, {Name: "RenewCopy"}, {Name: "ReturnCopy"}, {Name: "LoseCopy"}},
			Events:   []*ast.Event{{Name: "CopyBorrowed"}, {Name: "CopyRenewed"}, {Name: "CopyReturned"}, {Name: "CopyLost"}},
			Flows: []*ast.Flow{
				{CommandName: "BorrowCopy", EventName: "CopyBorrowed"},
				{CommandName: "RenewCopy", EventName: "CopyRenewed"},
				{CommandName: "ReturnCopy", EventName: "CopyReturned"},
				{CommandName: "LoseCopy", EventName: "CopyLost"},
			},
		},
		&ast.Slice{Name: "Chase", Automations: []*ast.Automation{{Name: "ChaseLoan", OnEvent: "CopyBorrowed", After: "14 days"}}},
	)
}

// fortySliceModel lays out forty slices, each of which reads the view of the
// slice before it and subscribes to that slice's event beside its own, so arrows
// run between neighbouring slices in every gap between two lanes.
func fortySliceModel() *ast.Model {
	var slices []*ast.Slice
	for i := 1; i <= 40; i++ {
		s := &ast.Slice{
			Name:     fmt.Sprintf("Step %d", i),
			Trigger:  &ast.Trigger{Name: fmt.Sprintf("Screen %d", i)},
			Commands: []*ast.Command{{Name: fmt.Sprintf("Take%d", i)}},
			Events:   []*ast.Event{{Name: fmt.Sprintf("Taken%d", i)}},
			Views:    []*ast.View{{Name: fmt.Sprintf("Board%d", i), Subscribes: []string{fmt.Sprintf("Taken%d", i)}}},
			Flows:    []*ast.Flow{{CommandName: fmt.Sprintf("Take%d", i), EventName: fmt.Sprintf("Taken%d", i)}},
		}
		if i > 1 {
			s.Trigger.Reads = []string{fmt.Sprintf("Board%d", i-1)}
			s.Views[0].Subscribes = append(s.Views[0].Subscribes, fmt.Sprintf("Taken%d", i-1))
		}
		slices = append(slices, s)
	}

	return slicesModel(slices...)
}

// nextSliceTranslationModel has a translation that records its event through a
// command the slice before it declares, so the arrow from that command to the
// event runs between the two slices rather than straight down one.
func nextSliceTranslationModel() *ast.Model {
	return slicesModel(
		&ast.Slice{Name: "Import", Commands: []*ast.Command{{Name: "ImportBooking"}}},
		&ast.Slice{Name: "Sync", Translations: []*ast.Translation{{
			Name:           "PartnerSync",
			ExternalSystem: "Partner API",
			Command:        "ImportBooking",
			Event:          &ast.Event{Name: "BookingImported"},
		}}},
	)
}

// sharedRoomTranslationsModel has two slices, each with a translation that reads
// a view of its own slice. The arrow from the view down to the external system
// and the arrow from the external system up to the reactor both cross the
// Events lane, beside the same slice and over the same lanes, so the second of
// the two must find an x of its own.
func sharedRoomTranslationsModel() *ast.Model {
	return slicesModel(
		&ast.Slice{
			Name:  "Import",
			Views: []*ast.View{{Name: "PartnersView"}},
			Translations: []*ast.Translation{{
				Name: "PartnerSync", ExternalSystem: "Partner API", Reads: "PartnersView", Event: &ast.Event{Name: "PartnerBooked"},
			}},
		},
		&ast.Slice{
			Name:  "Pay",
			Views: []*ast.View{{Name: "BanksView"}},
			Translations: []*ast.Translation{{
				Name: "BankSync", ExternalSystem: "Bank API", Reads: "BanksView", Event: &ast.Event{Name: "BankPaid"},
			}},
		},
	)
}

// fullRoomModel has a slice whose eight translations each read the view of
// that slice. More arrows then cross the Events lane beside the slice than the
// room next to it holds, so some of them run in the next room.
func fullRoomModel() *ast.Model {
	var translations []*ast.Translation
	for i := 1; i <= 8; i++ {
		translations = append(translations, &ast.Translation{
			Name:           fmt.Sprintf("Sync%d", i),
			ExternalSystem: fmt.Sprintf("Partner %d", i),
			Reads:          "PartnersView",
			Event:          &ast.Event{Name: fmt.Sprintf("Synced%d", i)},
		})
	}

	return slicesModel(
		&ast.Slice{Name: "Sync", Views: []*ast.View{{Name: "PartnersView"}}, Translations: translations},
		&ast.Slice{Name: "Pay", Commands: []*ast.Command{{Name: "Pay"}}, Events: []*ast.Event{{Name: "Paid"}}},
	)
}

// twoTranslationsModel has one slice with two translations, so it draws two
// external systems in one row of the External Systems lane, and an arrow from
// each up to its reactor.
func twoTranslationsModel() *ast.Model {
	return slicesModel(&ast.Slice{Name: "Import", Translations: []*ast.Translation{
		{Name: "PartnerSync", ExternalSystem: "Partner API", Event: &ast.Event{Name: "PartnerBooked"}},
		{Name: "BankSync", ExternalSystem: "Bank API", Event: &ast.Event{Name: "BankPaid"}},
	}})
}

// crossedAutomationsModel has two automations that each read a view the slice
// before declares and issue the command the other one sits under. Their four
// arrows join two rows of one lane and none of them can run straight, so their
// legs overlap in the gap under that lane.
func crossedAutomationsModel() *ast.Model {
	return slicesModel(
		&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "AlphaTodoView"}, {Name: "BetaTodoView"}}},
		&ast.Slice{
			Name:     "Run",
			Commands: []*ast.Command{{Name: "Alpha"}, {Name: "Beta"}},
			Automations: []*ast.Automation{
				{Name: "First", Schedule: "0 2 * * *", Reads: "AlphaTodoView", Command: "Beta"},
				{Name: "Second", Schedule: "0 3 * * *", Reads: "BetaTodoView", Command: "Alpha"},
			},
		},
	)
}

// wideScheduleModel has an automation whose schedule line is wider than its
// name, under the command whose arrow runs down past it to the event.
func wideScheduleModel() *ast.Model {
	return slicesModel(&ast.Slice{
		Name:        "Sweep",
		Commands:    []*ast.Command{{Name: "SweepAccounts"}},
		Events:      []*ast.Event{{Name: "AccountsSwept"}},
		Flows:       []*ast.Flow{{CommandName: "SweepAccounts", EventName: "AccountsSwept"}},
		Automations: []*ast.Automation{{Name: "Nightly", Schedule: "0 2 * * MON,TUE,WED,THU,FRI", Command: "SweepAccounts"}},
	})
}

// fanInEvents are the events whose arrows all reach BalanceView in fanInModel.
var fanInEvents = []string{
	"Deposited", "Withdrawn", "Refunded", "ChargedBack", "FeeTaken", "InterestPaid", "Transferred", "Reversed",
}

// fanInModel has one slice whose view subscribes to the eight events beside it,
// so eight legs overlap in the gap above the Events lane, more than that gap
// holds; the slice states a scenario so that the picture draws a spec card. Its
// two commands push the view to the right third of its row, so that the legs
// from the events under the commands all run to the view.
func fanInModel() *ast.Model {
	var events []*ast.Event
	for _, name := range fanInEvents {
		events = append(events, &ast.Event{Name: name})
	}

	return slicesModel(&ast.Slice{
		Name:     "Keep Balance",
		Commands: []*ast.Command{{Name: "Deposit"}, {Name: "Withdraw"}},
		Events:   events,
		Views:    []*ast.View{{Name: "BalanceView", Subscribes: fanInEvents}},
		Specs:    []*ast.Spec{{Name: "keeps the balance", When: &ast.SpecElement{Name: "Deposited"}, Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "Deposited"}}}}},
	})
}

func svgPoint(t *testing.T, point []string) [2]int {
	t.Helper()

	x, err := strconv.Atoi(point[1])
	require.NoError(t, err, "an arrow's path names %q, which is no point", point[0])
	y, err := strconv.Atoi(point[2])
	require.NoError(t, err, "an arrow's path names %q, which is no point", point[0])

	return [2]int{x, y}
}

// edgeLabelClass marks the text an arrow carries, which the writer paints
// outside any box.
const edgeLabelClass = "edge-label"

func svgAttributeValue(element xml.StartElement, name string) string {
	for _, a := range element.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}

	return ""
}

func svgAttributes(element xml.StartElement) string {
	pairs := make([]string, 0, len(element.Attr))
	for _, a := range element.Attr {
		pairs = append(pairs, a.Name.Local+"="+a.Value)
	}
	return strings.Join(pairs, " ")
}

func svgTooltipOf(t *testing.T, output, label string) string {
	t.Helper()

	var tooltips []string
	for _, shape := range svgShapes(t, output) {
		if strings.Contains(shape.label, label) {
			tooltips = append(tooltips, shape.tooltip)
		}
	}
	require.Len(t, tooltips, 1, "expected one svg shape labelled %q", label)

	return tooltips[0]
}

// svgPicture returns everything the diagram draws — every box with the text on
// it, how it is painted and where it sits, then the arrows between them —
// leaving out what a box only says when hovered, so a described diagram can be
// compared with the one the same model draws without prose.
func svgPicture(t *testing.T, output string) []string {
	t.Helper()

	var drawn []string
	for _, box := range svgBoxes(t, output) {
		drawn = append(drawn, box.attributes+" "+box.label)
	}

	return append(drawn, svgArrows(output)...)
}
