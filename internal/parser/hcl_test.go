//go:build unit

package parser_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestParseHCL(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		t.Run("a declared version is carried on the model", func(t *testing.T) {
			model, diags := parser.ParseHCL("emod = 1\n", "test.emod")

			require.Empty(t, diags)
			require.Equal(t, 1, model.Version)
			require.True(t, model.VersionDeclared)
		})

		t.Run("a file without a header is read as version 1", func(t *testing.T) {
			model, diags := parser.ParseHCL(`model "Hotel" {}`, "test.emod")

			require.Empty(t, diags)
			require.Equal(t, 1, model.Version)
			require.False(t, model.VersionDeclared)
		})

		t.Run("an unsupported version is reported on the header line", func(t *testing.T) {
			_, diags := parser.ParseHCL("emod = 2\n", "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, 1, diags[0].Line)
			require.Equal(t, "unsupported version 2: this tool supports emod version 1", diags[0].Message)
		})

		t.Run("an unsupported version is reported alone", func(t *testing.T) {
			_, diags := parser.ParseHCL("emod = 2\n\nprocess \"Nope\" {}\n", "test.emod")

			require.Len(t, diags, 1)
			require.Contains(t, diags[0].Message, "unsupported version 2")
		})
	})

	t.Run("model and actor", func(t *testing.T) {
		t.Run("the name is the label and the description is an attribute", func(t *testing.T) {
			source := `
model "Hotel Reservation" {
  description = "How the hotel takes a stay"
}

actor "Guest" {
  description = "Someone booking a room"
}
`
			model, diags := parser.ParseHCL(source, "test.emod")

			require.Empty(t, diags)
			require.Equal(t, "Hotel Reservation", model.Name)
			require.Equal(t, "How the hotel takes a stay", model.Description)
			require.Len(t, model.Actors, 1)
			require.Equal(t, "Guest", model.Actors[0].Name)
			require.Equal(t, "Someone booking a room", model.Actors[0].Description)
		})
	})

	t.Run("context", func(t *testing.T) {
		t.Run("the mode is read as a bare word", func(t *testing.T) {
			model, diags := parser.ParseHCL(`context "Reading Room" { mode = dcb }`, "test.emod")

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
			model, diags := parser.ParseHCL(source, "test.emod")

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
			model, diags := parser.ParseHCL(source, "test.emod")

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
			model, diags := parser.ParseHCL(source, "test.emod")

			require.Empty(t, diags)
			automation := model.Contexts[0].Slices[0].Automations[0]
			require.Equal(t, "RoomReserved", automation.OnEvent)
			require.Equal(t, "15m", automation.After)
			require.Equal(t, "PendingConfirmationsView", automation.Reads)
			require.Equal(t, "SendConfirmationEmail", automation.Command)
			require.Equal(t, "Notifications", automation.TargetContext)
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
			model, diags := parser.ParseHCL(source, "test.emod")

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

			model, diags := parser.ParseHCL(source, "test.emod")

			require.Empty(t, diags)
			require.Equal(t, 4, model.Contexts[0].Slices[0].Flows[0].CommandPos.Line)
		})
	})

	t.Run("spec", func(t *testing.T) {
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
			model, diags := parser.ParseHCL(source, "test.emod")

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
			model, diags := parser.ParseHCL(source, "test.emod")

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
			source := "# the whole system\nmodel \"Hotel\" {}\n\n# who books\nactor \"Guest\" {}\n"

			model, diags := parser.ParseHCL(source, "test.emod")

			require.Empty(t, diags)
			require.Len(t, model.Comments, 1)
			require.Equal(t, "# the whole system", model.Comments[0].Text)
			require.Len(t, model.Actors[0].Comments, 1)
			require.Equal(t, "# who books", model.Actors[0].Comments[0].Text)
		})
	})

	t.Run("faults", func(t *testing.T) {
		t.Run("an unknown top-level block names the ones that are allowed", func(t *testing.T) {
			_, diags := parser.ParseHCL(`process "Nope" {}`, "test.emod")

			require.Len(t, diags, 1)
			require.Contains(t, diags[0].Message, `unexpected "process" block`)
			require.Contains(t, diags[0].Message, "model, actor, context")
		})

		t.Run("an unknown attribute is reported where it is written", func(t *testing.T) {
			_, diags := parser.ParseHCL("model \"Hotel\" {\n  colour = \"blue\"\n}\n", "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, 2, diags[0].Line)
			require.Equal(t, "unexpected colour in model", diags[0].Message)
		})

		t.Run("text that is not HCL is reported and loses nothing after it", func(t *testing.T) {
			_, diags := parser.ParseHCL("model \"Hotel\" {\n", "test.emod")

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
	model, diags := parser.ParseHCL(source, "test.emod")
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
	model, diags := parser.ParseHCL(source, "test.emod")
	require.Empty(t, diags)
	return model.Contexts[0].Slices[0].Specs[0]
}
