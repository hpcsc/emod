//go:build unit

package diagram_test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
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

	t.Run("renders a trigger with its actor", func(t *testing.T) {
		model := singleSliceModel("Test", "S", &ast.Trigger{Name: "SubmitForm", Actor: "User"})

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "SubmitForm")
		require.Contains(t, output, "(User)")
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

	t.Run("renders automation with gear icon indicator", func(t *testing.T) {
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

		output := string(raw)
		require.Contains(t, output, "Notifier")
		require.Contains(t, output, "\u2699") // gear character
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
				if strings.Contains(box.appearance, fillRejectionHex) {
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

			var badgeCentres [][2]int
			for _, box := range svgBoxes(t, output) {
				if box.label == "OneCopyPerLoan" {
					badgeCentres = append(badgeCentres, box.rect.centre())
				}
			}
			require.Len(t, badgeCentres, 2, "each of the two slices draws its own badge")

			// Comparing endpoints rather than target labels is the whole point:
			// both arrows resolve to the label "OneCopyPerLoan" even when they
			// end at the same box, so a label comparison cannot see the collapse.
			var reached [][2]int
			for _, end := range dashedArrowEndpoints(t, output) {
				if slices.Contains(badgeCentres, end) {
					reached = append(reached, end)
				}
			}

			require.ElementsMatch(t, badgeCentres, reached,
				"each slice's dashed arrow ends at the badge in its own column; filing a badge under the invariant's name alone points both at whichever slice was drawn last")

			require.Equal(t, []string{"BorrowCopy", "ReturnCopy"}, sourcesReaching(t, output, "OneCopyPerLoan"))
		})

		t.Run("one slice rejecting two invariants sends each arrow to its own badge", func(t *testing.T) {
			output := svgOf(t, twoRejectionsOneSliceModel())

			centreOf := func(label string) [2]int {
				for _, box := range svgBoxes(t, output) {
					if box.label == label {
						return box.rect.centre()
					}
				}
				require.FailNowf(t, "no box drawn", "%q", label)
				return [2]int{}
			}

			// Pairing the nth edge with badges[i][0] instead of badges[i][n]
			// orphans the second badge and stacks both arrows on the first.
			require.ElementsMatch(t,
				[][2]int{centreOf("OneCopyPerLoan"), centreOf("FiveCopiesPerMember")},
				dashedArrowEndpoints(t, output))
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

			column := boxLabelled(t, svgBoxes(t, output), "DeskOccupancyView").rect.centre()[0]
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
					"a line wider than the card is drawn outside it: svg never wraps and draw.io wraps into a height the card was not measured for")
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

var svgFontSizeAttribute = regexp.MustCompile(`font-size="(\d+)"`)

// svgSpecCardFontSize is the size the SVG card paints its text at, read out of a
// render rather than restated, so the draw.io card can be required to state the
// same one — it is the size both formats' band heights are measured from.
func svgSpecCardFontSize(t *testing.T) string {
	t.Helper()

	lines := strings.Split(svgWithSpecs(t, test.SlicePatternLibraryLendingModel(t)), "\n")
	for i, line := range lines {
		if !strings.Contains(line, fillSpecCardHex) || i+1 >= len(lines) {
			continue
		}
		found := svgFontSizeAttribute.FindStringSubmatch(lines[i+1])
		require.NotNil(t, found, "the card's text states no font size: %s", lines[i+1])
		return found[1]
	}

	require.FailNow(t, "the render draws no card to read a font size from")
	return ""
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

// fillRejectionHex and fillEventHex are stated here rather than read from the
// production constants, so a test compares the bytes a reader sees against a
// value written down independently of the code that emits it.
const (
	fillRejectionHex   = "#f8cecc"
	strokeRejectionHex = "#b85450"
	fillEventHex       = "#ffe6cc"
)

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
}

// svgShapes returns the diagram's boxes in document order, decoded through an
// XML parser so a test sees the text a reader sees rather than its escaped form.
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
			case "rect":
				shapes = append(shapes, svgShape{attributes: svgAttributes(element), rect: svgRectOf(t, element)})
				inRect = true
			case "title":
				text.Reset()
			case "text":
				text.Reset()
				inEdgeLabel = svgAttributeValue(element, "class") == edgeLabelClass
			}
		case xml.CharData:
			text.Write(element)
		case xml.EndElement:
			switch element.Name.Local {
			case "rect":
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

func svgBoxes(t *testing.T, output string) []diagramBox {
	t.Helper()

	var boxes []diagramBox
	for _, shape := range svgShapes(t, output) {
		boxes = append(boxes, diagramBox{label: shape.label, appearance: shape.attributes, rect: shape.rect})
	}

	return boxes
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

// svgConnections returns the arrows the diagram draws, each named by the boxes
// its two ends meet and carrying how it is painted, so a test can say which
// boxes an arrow runs between instead of restating its coordinates.
func svgConnections(t *testing.T, output string) []diagramConnection {
	t.Helper()

	labelled := make(map[[2]int]string)
	for _, shape := range svgShapes(t, output) {
		labelled[shape.rect.centre()] = shape.label
	}

	boxAt := func(point [2]int) string {
		label, drawn := labelled[point]
		require.True(t, drawn, "an arrow meets %v, where the diagram draws no box", point)
		return label
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
		drawn = append(drawn, box.appearance+" "+box.label)
	}

	return append(drawn, svgArrows(output)...)
}
