//go:build unit

package parser_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestParseHCL(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		t.Run("a declared version is carried on the model", func(t *testing.T) {
			model, diags := parser.Parse("emod = 1\n", "test.emod")

			require.Empty(t, diags)
			require.Equal(t, 1, model.Version)
			require.True(t, model.VersionDeclared)
		})

		t.Run("a file without a header is read as version 1", func(t *testing.T) {
			model, diags := parser.Parse(`model "Hotel" {}`, "test.emod")

			require.Empty(t, diags)
			require.Equal(t, 1, model.Version)
			require.False(t, model.VersionDeclared)
		})

		t.Run("an unsupported version is reported on the header line", func(t *testing.T) {
			_, diags := parser.Parse("emod = 2\n", "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, 1, diags[0].Line)
			require.Equal(t, "unsupported version 2: this tool supports emod version 1", diags[0].Message)
		})

		for name, version := range map[string]string{
			"a version still being typed":             "-",
			"a quoted version":                        `"1"`,
			"a version that is null":                  "true ? null : 1",
			"a version whose expression has an error": "[1, foo][0]",
		} {
			t.Run(name+" is reported as not a whole number", func(t *testing.T) {
				_, diags := parser.Parse("emod = "+version+"\n", "test.emod")

				require.Contains(t, messagesOf(diags), "expected a whole number after emod")
			})
		}

		t.Run("an unsupported version is reported alone", func(t *testing.T) {
			_, diags := parser.Parse("emod = 2\n\nprocess \"Nope\" {}\n", "test.emod")

			require.Len(t, diags, 1)
			require.Contains(t, diags[0].Message, "unsupported version 2")
		})
	})

	t.Run("model", func(t *testing.T) {
		for name, value := range map[string]string{
			"still being typed":             `"a ${}"`,
			"that is a number":              "1",
			"that is a null string":         `true ? null : "a"`,
			"whose expression has an error": `["a", foo][0]`,
		} {
			t.Run("a description "+name+" is reported and not read", func(t *testing.T) {
				model, diags := parser.Parse("model \"M\" {\n  description = "+value+"\n}\n", "test.emod")

				require.Contains(t, messagesOf(diags), "expected a quoted string after description")
				require.Empty(t, model.Description)
			})
		}

		t.Run("the name is the label and the description is an attribute", func(t *testing.T) {
			source := `
model "Hotel Reservation" {
  description = "How the hotel takes a stay"
}
`
			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			require.Equal(t, "Hotel Reservation", model.Name)
			require.Equal(t, "How the hotel takes a stay", model.Description)
		})
	})

	t.Run("screen", func(t *testing.T) {
		t.Run("is read with its name, its description, its leading comments and its braces", func(t *testing.T) {
			source := "# where a guest pays\nscreen \"CheckoutPage\" {\n  description = \"Where a guest reviews the cart\"\n}\n"

			model, diags := parser.Parse(source, "shop.emod")

			require.Empty(t, diags)
			require.Len(t, model.Screens, 1)
			test.RequireEqual(t, &ast.Screen{
				Comments:       []*ast.Comment{{Text: "# where a guest pays", Position: ast.Position{Filename: "shop.emod", Line: 1, Column: 1}}},
				Name:           "CheckoutPage",
				NamePos:        ast.Position{Filename: "shop.emod", Line: 2, Column: 8},
				Description:    "Where a guest reviews the cart",
				DescriptionPos: ast.Position{Filename: "shop.emod", Line: 3, Column: 17},
				OpenPos:        ast.Position{Filename: "shop.emod", Line: 2, Column: 23},
				ClosePos:       ast.Position{Filename: "shop.emod", Line: 4, Column: 1},
			}, model.Screens[0])
		})

		t.Run("an empty body gives a screen with no description and no diagnostic", func(t *testing.T) {
			model, diags := parser.Parse(`screen "CheckoutPage" {}`, "shop.emod")

			require.Empty(t, diags)
			require.Equal(t, "CheckoutPage", model.Screens[0].Name)
			require.Empty(t, model.Screens[0].Description)
		})

		t.Run("a screen with no label gives one diagnostic", func(t *testing.T) {
			model, diags := parser.Parse("screen {}\n", "shop.emod")

			require.Equal(t, []string{"screen takes 1 name(s), not 0"}, messagesOf(diags))
			require.Len(t, model.Screens, 1)
			require.Equal(t, 1, model.Screens[0].OpenPos.Line, "the screen stays in the model, with its braces, for the editor")
		})

		t.Run("an unknown attribute and a nested block each give one diagnostic that names the screen", func(t *testing.T) {
			_, diags := parser.Parse("screen \"CheckoutPage\" {\n  image = \"./checkout.png\"\n  fields {}\n}\n", "shop.emod")

			require.Equal(t, []string{"unexpected image in screen", `unexpected "fields" block in screen`}, messagesOf(diags))
		})
	})

	t.Run("shows", func(t *testing.T) {
		appearancesOf := func(t *testing.T, body string) ([]*ast.Appearance, []*diagnostic.Entry) {
			t.Helper()
			source := "context \"Shop\" {\n  slice \"Apply Discount\" {\n" + body + "  }\n}\n"
			model, diags := parser.Parse(source, "shop.emod")
			return model.Contexts[0].Slices[0].Appearances, diags
		}

		t.Run("reads its screen, each reads entry and each command entry at its own position, in source order", func(t *testing.T) {
			appearances, diags := appearancesOf(t, "    shows {\n      screen  = CheckoutPage\n      reads   = [CartView, DiscountView]\n      command = [ApplyDiscount, RemoveDiscount]\n    }\n")

			require.Empty(t, diags)
			require.Len(t, appearances, 1)
			at := func(line, column int) ast.Position {
				return ast.Position{Filename: "shop.emod", Line: line, Column: column}
			}
			test.RequireEqual(t, &ast.Appearance{
				Screen:      "CheckoutPage",
				ScreenPos:   at(4, 17),
				Reads:       []string{"CartView", "DiscountView"},
				ReadsPos:    []ast.Position{at(5, 18), at(5, 28)},
				Commands:    []string{"ApplyDiscount", "RemoveDiscount"},
				CommandsPos: []ast.Position{at(6, 18), at(6, 33)},
				OpenPos:     at(3, 11),
				ClosePos:    at(7, 5),
			}, appearances[0])
		})

		t.Run("a comment lands on the appearance written under it", func(t *testing.T) {
			appearances, diags := appearancesOf(t, "    # what the guest sees\n    shows {\n      screen = CheckoutPage\n    }\n")

			require.Empty(t, diags)
			require.Len(t, appearances[0].Comments, 1)
			require.Equal(t, "# what the guest sees", appearances[0].Comments[0].Text)
		})

		t.Run("a bare reads and a bare command are each a list of one", func(t *testing.T) {
			appearances, diags := appearancesOf(t, "    shows {\n      screen  = CheckoutPage\n      reads   = CartView\n      command = ApplyDiscount\n    }\n")

			require.Empty(t, diags)
			require.Equal(t, []string{"CartView"}, appearances[0].Reads)
			require.Equal(t, []string{"ApplyDiscount"}, appearances[0].Commands)
		})

		t.Run("a shows block with no reads and no command parses", func(t *testing.T) {
			appearances, diags := appearancesOf(t, "    shows {\n      screen = CheckoutPage\n    }\n")

			require.Empty(t, diags)
			require.Empty(t, appearances[0].Reads)
			require.Empty(t, appearances[0].Commands)
		})

		t.Run("a slice holds each of three shows blocks in source order, beside a trigger", func(t *testing.T) {
			appearances, diags := appearancesOf(t, "    trigger \"Checkout Page\" {\n    }\n    shows {\n      screen = CheckoutPage\n    }\n    shows {\n      screen = OrderPage\n    }\n    shows {\n      screen = ReceiptPage\n    }\n")

			require.Empty(t, diags)
			var screens []string
			for _, appearance := range appearances {
				screens = append(screens, appearance.Screen)
			}
			require.Equal(t, []string{"CheckoutPage", "OrderPage", "ReceiptPage"}, screens)
		})

		t.Run("a shows block with no screen gives one diagnostic, at the shows keyword", func(t *testing.T) {
			_, diags := appearancesOf(t, "    shows {\n      reads = CartView\n    }\n")

			require.Len(t, diags, 1)
			require.Equal(t, 3, diags[0].Line)
			require.Equal(t, 5, diags[0].Column)
			require.Contains(t, diags[0].Message, "shows names no screen")
		})

		t.Run("a shows block with a label gives one diagnostic", func(t *testing.T) {
			appearances, diags := appearancesOf(t, "    shows \"Checkout\" {\n      screen = CheckoutPage\n    }\n")

			require.Equal(t, []string{"shows takes 0 name(s), not 1"}, messagesOf(diags))
			require.Len(t, appearances, 1)
			require.Equal(t, 3, appearances[0].OpenPos.Line, "the appearance stays in the slice, with its braces, for the editor")
		})

		t.Run("an unknown attribute and a nested block each give one diagnostic that names the shows block", func(t *testing.T) {
			_, diags := appearancesOf(t, "    shows {\n      screen = CheckoutPage\n      actor  = Guest\n      fields {}\n    }\n")

			require.Equal(t, []string{"unexpected actor in shows", `unexpected "fields" block in shows`}, messagesOf(diags))
		})

		t.Run("a field and a command named screen or shows parse with no diagnostic", func(t *testing.T) {
			_, diags := appearancesOf(t, "    command \"screen\" {\n      fields {\n        screen = string\n        shows  = string\n      }\n    }\n    command \"shows\" {\n    }\n")

			require.Empty(t, diags)
		})
	})

	t.Run("retired actor", func(t *testing.T) {
		t.Run("an actor block is reported once at its keyword, and the context after it is still read", func(t *testing.T) {
			source := "actor \"Guest\" {\n  description = \"Someone booking a room\"\n}\n\ncontext \"Reservations\" {\n}\n"

			model, diags := parser.Parse(source, "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, 1, diags[0].Line)
			require.Equal(t, 1, diags[0].Column)
			require.Regexp(t, `\bactor\b`, diags[0].Message)
			require.Contains(t, diags[0].Message, "retired")
			require.Contains(t, diags[0].Message, "delete this actor block")
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "Reservations", model.Contexts[0].Name)
			require.Equal(t, 6, model.Contexts[0].ClosePos.Line)
		})

		t.Run("an actor attribute in a trigger is reported once at the attribute, and the reads after it is still read", func(t *testing.T) {
			source := "context \"Shop\" {\n  slice \"Apply Discount\" {\n    trigger \"Checkout Page\" {\n      actor = Guest\n      reads = CartView\n    }\n  }\n}\n"

			model, diags := parser.Parse(source, "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, 4, diags[0].Line)
			require.Equal(t, 7, diags[0].Column)
			require.Regexp(t, `\bactor\b`, diags[0].Message)
			require.Contains(t, diags[0].Message, "retired")
			require.Contains(t, diags[0].Message, "delete this actor attribute")
			trigger := model.Contexts[0].Slices[0].Trigger
			require.Equal(t, "Checkout Page", trigger.Name)
			require.Equal(t, []string{"CartView"}, trigger.Reads)
		})

		t.Run("a field named actor and a construct named actor still parse with no diagnostic", func(t *testing.T) {
			source := "context \"Shop\" {\n  slice \"Assign\" {\n    command \"actor\" {\n      fields {\n        actor = string\n      }\n    }\n  }\n}\n"

			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			command := model.Contexts[0].Slices[0].Commands[0]
			require.Equal(t, "actor", command.Name)
			require.Equal(t, "actor", command.Fields[0].Name)
		})
	})

	t.Run("context", func(t *testing.T) {
		t.Run("the mode is read as a bare word", func(t *testing.T) {
			model, diags := parser.Parse(`context "Reading Room" { mode = dcb }`, "test.emod")

			require.Empty(t, diags)
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "dcb", model.Contexts[0].Mode)
		})

		t.Run("invariants keep the order they are written in", func(t *testing.T) {
			source := `
context "Lending" {
  aggregate "Loan" {
    invariants {
      OneCopyPerLoan      = "A loan covers exactly one copy"
      FiveCopiesPerMember = "A member holds at most five copies"
    }
  }
}
`
			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			invariants := model.Contexts[0].Aggregates[0].Invariants
			require.Len(t, invariants, 2)
			require.Equal(t, "OneCopyPerLoan", invariants[0].Name)
			require.Equal(t, "A loan covers exactly one copy", invariants[0].Statement)
			require.Equal(t, "FiveCopiesPerMember", invariants[1].Name)
		})
	})

	t.Run("fields", func(t *testing.T) {
		t.Run("a bare type carries no modifier", func(t *testing.T) {
			fields := fieldsOf(t, `roomId = string`)

			require.Len(t, fields, 1)
			require.Equal(t, "roomId", fields[0].Name)
			require.Equal(t, "string", fields[0].Type)
			require.Empty(t, fields[0].Modifier)
		})

		t.Run("a call carries the modifier it names", func(t *testing.T) {
			fields := fieldsOf(t, "roomId = required(string)\n  accessible = optional(bool)")

			require.Len(t, fields, 2)
			require.Equal(t, "required", fields[0].Modifier)
			require.Equal(t, "optional", fields[1].Modifier)
			require.Equal(t, "bool", fields[1].Type)
		})

		t.Run("a keyword is usable as a field name", func(t *testing.T) {
			fields := fieldsOf(t, "description = string\n  type = string\n  source = string")

			require.Len(t, fields, 3)
			require.Equal(t, "description", fields[0].Name)
			require.Equal(t, "source", fields[2].Name)
		})

		t.Run("fields keep the order they are written in", func(t *testing.T) {
			fields := fieldsOf(t, "zulu = string\n  alpha = string\n  mike = string")

			require.Equal(t, []string{"zulu", "alpha", "mike"},
				[]string{fields[0].Name, fields[1].Name, fields[2].Name})
		})
	})

	t.Run("event", func(t *testing.T) {
		for name, value := range map[string]string{
			"still being typed":             `"a ${}"`,
			"that is a number":              "1",
			"that is a null string":         `true ? null : "a"`,
			"whose expression has an error": `["a", foo][0]`,
		} {
			t.Run("an external name "+name+" is reported and not read", func(t *testing.T) {
				source := "context \"C\" {\n  slice \"S\" {\n    event \"E\" {\n      source = external(" + value + ")\n    }\n  }\n}\n"
				model, diags := parser.Parse(source, "test.emod")

				require.Contains(t, messagesOf(diags), "expected a quoted string after source external in event")
				require.Empty(t, model.Contexts[0].Slices[0].Events[0].ExternalName)
			})
		}

		t.Run("the wire type and the external source are read", func(t *testing.T) {
			source := `
context "Reservations" {
  slice "Reserve" {
    event "RoomReserved" {
      type   = "com.hotel.room-reserved"
      source = external("Partner Booking")
    }
  }
}
`
			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			event := model.Contexts[0].Slices[0].Events[0]
			require.Equal(t, "com.hotel.room-reserved", event.WireType)
			require.Equal(t, "external", event.Source)
			require.Equal(t, "Partner Booking", event.ExternalName)
		})
	})

	t.Run("automation", func(t *testing.T) {
		t.Run("the activation, the delay and the target context are read", func(t *testing.T) {
			source := `
context "Reservations" {
  slice "Confirm" {
    automation "ConfirmationEmailReactor" {
      on      = RoomReserved
      after   = "15m"
      reads   = PendingConfirmationsView
      command = SendConfirmationEmail

      target {
        context = Notifications
      }
    }
  }
}
`
			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			automation := model.Contexts[0].Slices[0].Automations[0]
			require.Equal(t, "RoomReserved", automation.OnEvent)
			require.Equal(t, "15m", automation.After)
			require.Equal(t, "PendingConfirmationsView", automation.Reads)
			require.Equal(t, "SendConfirmationEmail", automation.Command)
			require.Equal(t, "Notifications", automation.TargetContext)
		})
	})

	t.Run("trigger", func(t *testing.T) {
		readsOf := func(t *testing.T, reads string) (*ast.Trigger, []*diagnostic.Entry) {
			t.Helper()
			source := "context \"Shop\" {\n  slice \"Apply Discount\" {\n    trigger \"Checkout Page\" {\n      reads = " + reads + "\n    }\n  }\n}\n"
			model, diags := parser.Parse(source, "shop.emod")
			return model.Contexts[0].Slices[0].Trigger, diags
		}

		t.Run("a list of views is read in the order it is written, each at its own entry", func(t *testing.T) {
			trigger, diags := readsOf(t, "[CartView, DiscountView]")

			require.Empty(t, diags)
			require.Equal(t, []string{"CartView", "DiscountView"}, trigger.Reads)
			require.Equal(t, []ast.Position{
				{Filename: "shop.emod", Line: 4, Column: 16},
				{Filename: "shop.emod", Line: 4, Column: 26},
			}, trigger.ReadsPos)
		})

		t.Run("one bare name is read as a list of one, at the name", func(t *testing.T) {
			trigger, diags := readsOf(t, "CartView")

			require.Empty(t, diags)
			require.Equal(t, []string{"CartView"}, trigger.Reads)
			require.Equal(t, []ast.Position{{Filename: "shop.emod", Line: 4, Column: 15}}, trigger.ReadsPos)
		})

		t.Run("an entry that is not a name is reported at that entry, and the entries around it are still read", func(t *testing.T) {
			trigger, diags := readsOf(t, `[CartView, "DiscountView", LoyaltyView]`)

			require.Len(t, diags, 1)
			require.Equal(t, "expected a name", diags[0].Message)
			require.Equal(t, 4, diags[0].Line)
			require.Equal(t, 26, diags[0].Column)
			require.Equal(t, "CartView", trigger.Reads[0])
			require.Equal(t, "LoyaltyView", trigger.Reads[2])
		})

		t.Run("an empty list reads no view", func(t *testing.T) {
			trigger, diags := readsOf(t, "[]")

			require.Empty(t, diags)
			require.Empty(t, trigger.Reads)
		})
	})

	t.Run("flow", func(t *testing.T) {
		t.Run("an event entry and a rejection entry are told apart", func(t *testing.T) {
			source := `
context "Reservations" {
  slice "Reserve" {
    flow = <<-FLOW
      command -> event:    ReserveRoom -> RoomReserved
      command -> rejected: ReserveRoom -> RoomNotDoubleBooked
    FLOW
  }
}
`
			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			slice := model.Contexts[0].Slices[0]
			require.Len(t, slice.Flows, 1)
			require.Equal(t, "ReserveRoom", slice.Flows[0].CommandName)
			require.Equal(t, "RoomReserved", slice.Flows[0].EventName)
			require.Len(t, slice.Rejections, 1)
			require.Equal(t, "RoomNotDoubleBooked", slice.Rejections[0].InvariantName)
		})

		t.Run("an entry is reported at the line it is written on", func(t *testing.T) {
			source := "context \"C\" {\n  slice \"S\" {\n    flow = <<-FLOW\n      command -> event: A -> B\n    FLOW\n  }\n}\n"

			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			require.Equal(t, 4, model.Contexts[0].Slices[0].Flows[0].CommandPos.Line)
		})
	})

	t.Run("spec", func(t *testing.T) {
		for name, value := range map[string]string{
			"null":                          "null",
			"that is a null string":         `true ? null : "a"`,
			"whose expression has an error": `["a", foo][0]`,
		} {
			t.Run("a payload value "+name+" is reported", func(t *testing.T) {
				source := "context \"C\" {\n  slice \"S\" {\n    spec \"s\" {\n      when = Do({ id = " + value + " })\n    }\n  }\n}\n"

				_, diags := parser.Parse(source, "test.emod")

				require.Contains(t, messagesOf(diags), "expected a string, a number or a boolean")
			})
		}

		for name, value := range map[string]string{
			"a string":  `"b ${}"`,
			"a boolean": `!"${}"`,
		} {
			t.Run("a payload value that is "+name+" still being typed after an earlier one is reported, not a panic", func(t *testing.T) {
				source := "model \"M\" {\n  description = \"a ${}\"\n}\n\ncontext \"C\" {\n  aggregate \"A\" {\n    slice \"S\" {\n      spec \"s\" {\n        when = Do({ id = " + value + " })\n      }\n    }\n  }\n}\n"

				_, diags := parser.Parse(source, "test.emod")

				require.Contains(t, messagesOf(diags), "expected a string, a number or a boolean")
			})
		}

		t.Run("a list of events is the success outcome", func(t *testing.T) {
			spec := specOf(t, "given = [CopyReturned]\n      when  = BorrowCopy\n      then  = [CopyBorrowed, LoanOpened]")

			require.Len(t, spec.Given, 1)
			require.Equal(t, "CopyReturned", spec.Given[0].Name)
			require.Equal(t, "BorrowCopy", spec.When.Name)
			then, ok := spec.Then.(*ast.ThenEvents)
			require.True(t, ok)
			require.Len(t, then.Events, 2)
		})

		t.Run("a call names the outcome a rejection, a view or a command", func(t *testing.T) {
			rejected, ok := specOf(t, "then = rejected(OneCopyPerLoan)").Then.(*ast.ThenRejected)
			require.True(t, ok)
			require.Equal(t, "OneCopyPerLoan", rejected.InvariantName)

			view, ok := specOf(t, "then = view(MemberLoansView)").Then.(*ast.ThenView)
			require.True(t, ok)
			require.Equal(t, "MemberLoansView", view.ViewName)

			command, ok := specOf(t, "then = command(RecallCopy)").Then.(*ast.ThenCommand)
			require.True(t, ok)
			require.Equal(t, "RecallCopy", command.CommandName)
		})

		t.Run("a payload carries each literal as it is written", func(t *testing.T) {
			spec := specOf(t, `when = BorrowCopy({ copyId = "C-93204", lateFee = 12.50, renewals = 4, expedited = true })`)

			payload := spec.When.Payload
			require.Len(t, payload, 4)
			require.Equal(t, "C-93204", payload[0].Value)
			require.Equal(t, ast.StringLiteral, payload[0].Kind)
			require.Equal(t, "12.50", payload[1].Value)
			require.Equal(t, ast.DecimalLiteral, payload[1].Kind)
			require.Equal(t, "4", payload[2].Value)
			require.Equal(t, ast.IntegerLiteral, payload[2].Kind)
			require.Equal(t, "true", payload[3].Value)
			require.Equal(t, ast.BooleanLiteral, payload[3].Kind)
		})
	})

	t.Run("dynamic consistency boundaries", func(t *testing.T) {
		t.Run("tags name the field each key reads", func(t *testing.T) {
			source := `
context "Reading Room" {
  mode = dcb
  slice "Claim Desk" {
    event "DeskClaimed" {
      tags {
        desk   = deskId
        reader = memberId
      }
    }
  }
}
`
			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			tags := model.Contexts[0].Slices[0].Events[0].Tags
			require.Len(t, tags, 2)
			require.Equal(t, "desk", tags[0].Key)
			require.Equal(t, "deskId", tags[0].FieldRef)
		})

		t.Run("a predicate keeps the grouping HCL parsed", func(t *testing.T) {
			source := `
context "Reading Room" {
  mode = dcb
  slice "Claim Desk" {
    command "ClaimDesk" {
      decides_on {
        events = [DeskClaimed]
        where  = tag(desk, deskId) && !tag(reader, memberId)
      }
    }
  }
}
`
			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			clause := model.Contexts[0].Slices[0].Commands[0].DecidesOn
			require.Equal(t, []string{"DeskClaimed"}, clause.Events)
			logical, ok := clause.Predicate.(*ast.LogicalExpr)
			require.True(t, ok)
			require.Equal(t, "and", logical.Operator)
			left, ok := logical.Left.(*ast.TagPredicate)
			require.True(t, ok)
			require.Equal(t, "desk", left.Field)
			require.Equal(t, "deskId", left.Value)
			_, ok = logical.Right.(*ast.NotExpr)
			require.True(t, ok)
		})
	})

	t.Run("comments", func(t *testing.T) {
		t.Run("a comment lands on the construct written under it", func(t *testing.T) {
			source := "# the whole system\nmodel \"Hotel\" {}\n\n# the bookings\ncontext \"Reservations\" {}\n"

			model, diags := parser.Parse(source, "test.emod")

			require.Empty(t, diags)
			require.Len(t, model.Comments, 1)
			require.Equal(t, "# the whole system", model.Comments[0].Text)
			require.Len(t, model.Contexts[0].Comments, 1)
			require.Equal(t, "# the bookings", model.Contexts[0].Comments[0].Text)
		})
	})

	t.Run("faults", func(t *testing.T) {
		t.Run("an unknown top-level block names the ones that are allowed", func(t *testing.T) {
			_, diags := parser.Parse(`process "Nope" {}`, "test.emod")

			require.Len(t, diags, 1)
			require.Contains(t, diags[0].Message, `unexpected "process" block`)
			require.Contains(t, diags[0].Message, "expected one of: model, screen, context")
			require.NotRegexp(t, `\bactor\b`, diags[0].Message)
		})

		t.Run("an unknown attribute is reported where it is written", func(t *testing.T) {
			_, diags := parser.Parse("model \"Hotel\" {\n  colour = \"blue\"\n}\n", "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, 2, diags[0].Line)
			require.Equal(t, "unexpected colour in model", diags[0].Message)
		})

		t.Run("text that is not HCL is reported and loses nothing after it", func(t *testing.T) {
			_, diags := parser.Parse("model \"Hotel\" {\n", "test.emod")

			require.NotEmpty(t, diags)
		})
	})
}

func fieldsOf(t *testing.T, fields string) []*ast.Field {
	t.Helper()
	source := `
context "C" {
  slice "S" {
    command "Cmd" {
      fields {
        ` + fields + `
      }
    }
  }
}
`
	model, diags := parser.Parse(source, "test.emod")
	require.Empty(t, diags)
	return model.Contexts[0].Slices[0].Commands[0].Fields
}

func specOf(t *testing.T, body string) *ast.Spec {
	t.Helper()
	source := `
context "C" {
  slice "S" {
    spec "a scenario" {
      ` + body + `
    }
  }
}
`
	model, diags := parser.Parse(source, "test.emod")
	require.Empty(t, diags)
	return model.Contexts[0].Slices[0].Specs[0]
}

func messagesOf(diags []*diagnostic.Entry) []string {
	messages := make([]string, 0, len(diags))
	for _, d := range diags {
		messages = append(messages, d.Message)
	}
	return messages
}
