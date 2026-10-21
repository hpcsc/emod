//go:build unit

package importer_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/importer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/stretchr/testify/require"
)

const lendingModel = `emod = 1

# The lending desk of one library
model "Library Lending" {
}

actor "Member" {
}

context "Lending" {
  aggregate "Loan" {
    invariants {
      OneLoanPerCopy = "A copy is on at most one loan"
    }

    # A member takes a copy off the shelf
    slice "Borrow Copy" {
      # The desk terminal the librarian types into
      trigger "Lending Desk" {
        actor = Member
        reads = AvailableCopiesView
      }

      # Ask the library to hand a copy over
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }

      # A copy left the building
      event "CopyBorrowed" {
        fields {
          loanId = required(string)
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
        command -> rejected: BorrowCopy -> OneLoanPerCopy
      FLOW

      spec "lends a copy on the shelf" {
        when = BorrowCopy({ copyId = "C-1" })
        then = [CopyBorrowed({ copyId = "C-1" })]
      }
    }

    # Every copy still on the shelf
    slice "Available Copies" {
      # What the desk shows a member
      view "AvailableCopiesView" {
        subscribes = [CopyBorrowed]

        fields {
          copyId = required(string)
        }
      }
    }

    slice "Recall Overdue Copy" {
      # Pull a copy back from a member who kept it too long
      command "RecallCopy" {
        fields {
          loanId = required(string)
        }
      }

      # Chases a copy nobody brought back
      automation "RecallOverdueCopy" {
        on      = CopyBorrowed
        reads   = AvailableCopiesView
        command = RecallCopy
      }
    }
  }

  slice "Import Partner Loan" {
    command "ImportPartnerLoan" {
      fields {
        externalRef = required(string)
      }
    }

    # Restates a partner branch's notice in the library's own language
    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
    }
  }
}
`

// wiringModel declares one of every block kind with no arrow drawn between most
// of them, and two flows and two subscriptions that share an end.
const wiringModel = `emod = 1

model "Library Lending" {
}

context "Lending" {
  slice "Borrow Copy" {
    trigger "Lending Desk" {
    }

    command "BorrowCopy" {
    }

    command "RenewLoan" {
    }

    event "CopyBorrowed" {
    }

    event "LoanRenewed" {
    }

    view "AvailableCopiesView" {
    }

    view "MemberLoansView" {
      subscribes = [CopyBorrowed, LoanRenewed]
    }

    automation "RemindMember" {
    }

    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"

      event "PartnerLoanImported" {
      }
    }

    flow = <<-FLOW
      command -> event:    BorrowCopy -> CopyBorrowed
      command -> event:    BorrowCopy -> LoanRenewed
      command -> event:    RenewLoan -> LoanRenewed
    FLOW
  }
}
`

// everyKindTwiceModel declares two of every construct a node is drawn for, and
// names, in a view's subscription, a translation's reads and command and a
// nested event's source, constructs no node is drawn for.
const everyKindTwiceModel = `emod = 1

model "Library Lending" {
}

actor "Member" {
}

actor "Librarian" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Borrow Copy" {
      command "BorrowCopy" {
      }

      command "RenewLoan" {
      }

      event "CopyBorrowed" {
      }

      event "LoanRenewed" {
      }

      view "AvailableCopiesView" {
        subscribes = [CopyBorrowed, ExternalLoanReported]
      }

      view "MemberLoansView" {
      }

      automation "RemindMember" {
        on = CopyBorrowed
      }

      automation "RecallOverdueCopy" {
        on = LoanRenewed
      }

      translation "PartnerLoanImport" {
        external_system = "Partner Branch API"
        reads           = PartnerLedgerView
        command         = RecordPartnerLoan

        event "PartnerLoanImported" {
          source = external("Partner Branch")
        }
      }

      translation "PartnerReturnImport" {
        external_system = "Partner Branch API"

        event "PartnerReturnImported" {
        }
      }
    }
  }

  aggregate "Copy" {
    slice "Shelve Copy" {
      command "ShelveCopy" {
      }
    }
  }
}
`

const repointModel = `emod = 1

model "Library Lending" {
}

context "Lending" {
  slice "Borrow Copy" {
    command "BorrowCopy" {
    }

    command "RenewLoan" {
    }

    event "CopyBorrowed" {
    }

    event "LoanOpened" {
    }

    event "CopyReserved" {
    }

    event "CopyLost" {
    }

    view "MemberLoansView" {
      subscribes = [CopyBorrowed, LoanOpened]
    }

    # How a loan comes about at the desk
    flow = <<-FLOW
      command -> event:    BorrowCopy -> CopyBorrowed
      command -> event:    RenewLoan -> LoanOpened
      command -> event:    BorrowCopy -> LoanOpened
      command -> event:    BorrowCopy -> CopyLost
    FLOW
  }

  slice "Return Copy" {
    command "ReturnCopy" {
    }

    event "CopyReturned" {
    }

    event "LoanClosed" {
    }

    event "CopyShelved" {
    }

    view "LoanHistoryView" {
      subscribes = [CopyReturned, LoanClosed]
    }

    flow = <<-FLOW
      command -> event:    ReturnCopy -> CopyReturned
      command -> event:    ReturnCopy -> LoanClosed
    FLOW
  }
}
`

// viewerDiagram is the diagram document the viewer holds for a model, edited the
// way the viewer's gestures edit it. Nodes and edges stay maps so every key the
// exporter wrote travels back untouched.
type viewerDiagram struct {
	t        *testing.T
	Name     string           `json:"model_name"`
	Nodes    []map[string]any `json:"nodes"`
	Edges    []map[string]any `json:"edges"`
	newNodes int
}

func drawnDiagram(t *testing.T, model *ast.Model) *viewerDiagram {
	t.Helper()
	d := &viewerDiagram{t: t}
	require.NoError(t, json.Unmarshal([]byte(exportedDiagram(t, model)), d))
	return d
}

func (d *viewerDiagram) node(label string) map[string]any {
	d.t.Helper()
	var found []map[string]any
	for _, n := range d.Nodes {
		if n["label"] == label {
			found = append(found, n)
		}
	}
	require.Len(d.t, found, 1, "the diagram has to hold one node labelled %s", label)
	return found[0]
}

func (d *viewerDiagram) idOf(label string) string {
	d.t.Helper()
	return d.node(label)["id"].(string)
}

// add appends a node the way the context menu does: an id the exporter never
// gives, a label, a parent and nothing else.
func (d *viewerDiagram) add(typ, label, parentLabel string) {
	d.t.Helper()
	d.newNodes++
	d.Nodes = append(d.Nodes, map[string]any{
		"id":       fmt.Sprintf("_%s_%d", typ, d.newNodes),
		"type":     typ,
		"label":    label,
		"parentId": d.idOf(parentLabel),
	})
}

// remove deletes a node and every edge that touches it, as the viewer's delete
// does.
func (d *viewerDiagram) remove(label string) {
	d.t.Helper()
	id := d.idOf(label)
	var nodes []map[string]any
	for _, n := range d.Nodes {
		if n["id"] != id {
			nodes = append(nodes, n)
		}
	}
	d.Nodes = nodes
	var edges []map[string]any
	for _, e := range d.Edges {
		if e["source"] != id && e["target"] != id {
			edges = append(edges, e)
		}
	}
	d.Edges = edges
}

func (d *viewerDiagram) set(label, key string, value any) {
	d.t.Helper()
	d.node(label)[key] = value
}

func (d *viewerDiagram) rename(label, to string) {
	d.t.Helper()
	d.node(label)["label"] = to
}

func (d *viewerDiagram) setFields(label string, fields ...map[string]any) {
	d.t.Helper()
	d.node(label)["fields"] = fields
}

func (d *viewerDiagram) moveTo(label, sliceLabel string) {
	d.t.Helper()
	d.node(label)["parentId"] = d.idOf(sliceLabel)
}

// swap exchanges where two nodes sit in the document, which is how the viewer
// reorders slices.
func (d *viewerDiagram) swap(a, b string) {
	d.t.Helper()
	ia, ib := -1, -1
	for i, n := range d.Nodes {
		switch n["label"] {
		case a:
			ia = i
		case b:
			ib = i
		}
	}
	require.NotEqual(d.t, -1, ia)
	require.NotEqual(d.t, -1, ib)
	d.Nodes[ia], d.Nodes[ib] = d.Nodes[ib], d.Nodes[ia]
}

func (d *viewerDiagram) connect(from, to, typ string) {
	d.t.Helper()
	d.Edges = append(d.Edges, map[string]any{"source": d.idOf(from), "target": d.idOf(to), "type": typ})
}

func (d *viewerDiagram) disconnect(from, to string) {
	d.t.Helper()
	source, target := d.idOf(from), d.idOf(to)
	var edges []map[string]any
	for _, e := range d.Edges {
		if e["source"] != source || e["target"] != target {
			edges = append(edges, e)
		}
	}
	require.Less(d.t, len(edges), len(d.Edges), "the diagram has to draw an arrow from %s to %s", from, to)
	d.Edges = edges
}

// repoint moves one end of the arrow from from to to, as dragging an arrow's
// handle does: newFrom and newTo name its ends afterwards.
func (d *viewerDiagram) repoint(from, to, newFrom, newTo string) {
	d.t.Helper()
	source, target := d.idOf(from), d.idOf(to)
	for _, e := range d.Edges {
		if e["source"] == source && e["target"] == target {
			e["source"], e["target"] = d.idOf(newFrom), d.idOf(newTo)
			return
		}
	}
	require.Failf(d.t, "no arrow", "the diagram has to draw an arrow from %s to %s", from, to)
}

func (d *viewerDiagram) document() []byte {
	d.t.Helper()
	document, err := json.Marshal(d)
	require.NoError(d.t, err)
	return document
}

func field(name, typ, modifier string) map[string]any {
	f := map[string]any{"name": name, "type": typ}
	if modifier != "" {
		f["modifier"] = modifier
	}
	return f
}

func requireFmtClean(t *testing.T, source, why string) {
	t.Helper()
	model, diagnostics := parser.Parse(source, "fmt-check.emod")
	require.Empty(t, diagnostics, why)
	require.Equal(t, source, formatter.Format(model), why)
}

// applied draws source, lets edit change the drawing as the viewer would, and
// answers the source the edited drawing applies back to.
func applied(t *testing.T, source string, edit func(d *viewerDiagram)) string {
	t.Helper()
	requireFmtClean(t, source, "the fixture has to be in the emod fmt layout, or the result says nothing about keeping it there")

	model := parseModel(t, source)
	d := drawnDiagram(t, model)
	edit(d)
	require.NoError(t, importer.ApplyDiagram(model, d.document()))

	result := formatter.Format(model)
	requireFmtClean(t, result, "an applied diagram edit has to leave text in the emod fmt layout that parses")
	return result
}

func replaced(t *testing.T, source, old, replacement string) string {
	t.Helper()
	require.Equal(t, 1, strings.Count(source, old), "the fixture has to hold %q exactly once", old)
	return strings.Replace(source, old, replacement, 1)
}

func TestApplyDiagram(t *testing.T) {
	t.Run("an unedited diagram", func(t *testing.T) {
		t.Run("leaves every model in the emod fmt layout under examples and parser testdata as it was written", func(t *testing.T) {
			paths, err := filepath.Glob("../../examples/*.emod")
			require.NoError(t, err)
			testdata, err := filepath.Glob("../parser/testdata/*.emod")
			require.NoError(t, err)

			checked := map[string]int{}
			for _, path := range append(paths, testdata...) {
				source, err := os.ReadFile(path)
				require.NoError(t, err)
				model, diagnostics := parser.Parse(string(source), filepath.Base(path))
				if len(diagnostics) > 0 || formatter.Format(model) != string(source) {
					continue
				}

				require.NoError(t, importer.ApplyDiagram(model, []byte(exportedDiagram(t, model))), path)
				require.Equal(t, string(source), formatter.Format(model), path)
				checked[filepath.Dir(path)]++
			}

			require.Positive(t, checked["../../examples"], "examples has to hold a model in the emod fmt layout, or nothing above ran")
			require.Positive(t, checked["../parser/testdata"], "parser testdata has to hold a model in the emod fmt layout, or nothing above ran")
		})

		t.Run("leaves a model that declares two of every kind as it was written", func(t *testing.T) {
			require.Equal(t, everyKindTwiceModel, applied(t, everyKindTwiceModel, func(*viewerDiagram) {}))
		})
	})

	t.Run("what the diagram does not carry", func(t *testing.T) {
		t.Run("a command added to one slice keeps the model comment, invariants, specs, rejections and event types around it", func(t *testing.T) {
			source := readExample(t, "specs_hotel.emod")
			for _, kept := range []string{
				"# Hotel holds and reservations — the scenarios each slice must satisfy\n",
				"    invariants {\n",
				`      spec "reserves a free room" {`,
				"        command -> rejected: ReserveRoom -> RoomNotDoubleBooked\n",
				`        type = "com.hotel.reservations.room-reserved"`,
			} {
				require.Contains(t, source, kept, "the fixture has to carry this, or keeping it says nothing")
			}

			result := applied(t, source, func(d *viewerDiagram) {
				d.add("command", "ExtendHold", "Release Hold")
			})

			require.Equal(t, replaced(t, source, `      command "ReleaseHold" {
        fields {
          holdId = required(string)
        }
      }
`, `      command "ReleaseHold" {
        fields {
          holdId = required(string)
        }
      }

      command "ExtendHold" {
      }
`), result)
		})

		t.Run("an event added to one slice keeps the context mode, event tags and decides_on clauses around it", func(t *testing.T) {
			source := readExample(t, "dcb_model.emod")
			for _, kept := range []string{
				"# Order Processing — Dynamic Consistency Boundary model\n",
				"  mode = dcb\n",
				"      tags {\n",
				"      decides_on {\n",
			} {
				require.Contains(t, source, kept, "the fixture has to carry this, or keeping it says nothing")
			}

			result := applied(t, source, func(d *viewerDiagram) {
				d.add("event", "ShipmentDelayed", "Ship Order")
			})

			require.Equal(t, replaced(t, source, `        shipmentId = required(string)
        orderId    = required(string)
        customerId = required(string)
      }
    }
`, `        shipmentId = required(string)
        orderId    = required(string)
        customerId = required(string)
      }
    }

    event "ShipmentDelayed" {
    }
`), result)
		})

		t.Run("a slice renamed elsewhere keeps an automation's target context", func(t *testing.T) {
			source := readExample(t, "all_patterns.emod")
			require.Contains(t, source, "        target {\n", "the fixture has to carry a target, or keeping it says nothing")

			result := applied(t, source, func(d *viewerDiagram) {
				d.rename("Check Out Guest", "Settle Guest Bill")
			})

			require.Equal(t, replaced(t, source, `slice "Check Out Guest" {`, `slice "Settle Guest Bill" {`), result)
		})
	})

	t.Run("adding", func(t *testing.T) {
		t.Run("a command added to a slice is declared after the commands already there", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.add("command", "RenewLoan", "Recall Overdue Copy")
			})

			require.Equal(t, replaced(t, lendingModel, `          loanId = required(string)
        }
      }

      # Chases a copy nobody brought back
`, `          loanId = required(string)
        }
      }

      command "RenewLoan" {
      }

      # Chases a copy nobody brought back
`), result)
		})

		t.Run("an event added to a slice is declared after the events already there", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.add("event", "CopyReserved", "Borrow Copy")
			})

			require.Equal(t, replaced(t, lendingModel, `          copyId = required(string)
        }
      }

      flow = <<-FLOW
`, `          copyId = required(string)
        }
      }

      event "CopyReserved" {
      }

      flow = <<-FLOW
`), result)
		})

		t.Run("a slice added to an aggregate is declared after the aggregate's slices", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.add("slice", "Return Copy", "Loan")
			})

			require.Equal(t, replaced(t, lendingModel, `        command = RecallCopy
      }
    }
  }
`, `        command = RecallCopy
      }
    }

    slice "Return Copy" {
    }
  }
`), result)
		})

		t.Run("a command added to a slice that the same edit adds is declared in that slice", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.add("slice", "Return Copy", "Loan")
				d.add("command", "ReturnCopy", "Return Copy")
			})

			require.Equal(t, replaced(t, lendingModel, `        command = RecallCopy
      }
    }
  }
`, `        command = RecallCopy
      }
    }

    slice "Return Copy" {
      command "ReturnCopy" {
      }
    }
  }
`), result)
		})

		t.Run("a slice added to a context is declared after the context's own slices", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.add("slice", "Transfer Copy", "Lending")
			})

			require.Equal(t, replaced(t, lendingModel, `      }
    }
  }
}
`, `      }
    }
  }

  slice "Transfer Copy" {
  }
}
`), result)
		})

		t.Run("Add Flow declares the event and the flow from the slice's last command", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.add("event", "CopyRecalled", "Recall Overdue Copy")
				d.connect("RecallCopy", "CopyRecalled", "flow")
			})

			require.Equal(t, replaced(t, lendingModel, `          loanId = required(string)
        }
      }

      # Chases a copy nobody brought back
      automation "RecallOverdueCopy" {
        on      = CopyBorrowed
        reads   = AvailableCopiesView
        command = RecallCopy
      }
`, `          loanId = required(string)
        }
      }

      event "CopyRecalled" {
      }

      # Chases a copy nobody brought back
      automation "RecallOverdueCopy" {
        on      = CopyBorrowed
        reads   = AvailableCopiesView
        command = RecallCopy
      }

      flow = <<-FLOW
        command -> event:    RecallCopy -> CopyRecalled
      FLOW
`), result)
		})

		t.Run("Add Flow on a slice with no command declares the command as well", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.add("event", "new-event", "Available Copies")
				d.add("command", "new-command", "Available Copies")
				d.connect("new-command", "new-event", "flow")
			})

			require.Equal(t, replaced(t, lendingModel, `      # What the desk shows a member
      view "AvailableCopiesView" {
        subscribes = [CopyBorrowed]

        fields {
          copyId = required(string)
        }
      }
`, `      command "new-command" {
      }

      event "new-event" {
      }

      # What the desk shows a member
      view "AvailableCopiesView" {
        subscribes = [CopyBorrowed]

        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    new-command -> new-event
      FLOW
`), result)
		})
	})

	t.Run("adding what the viewer cannot add", func(t *testing.T) {
		for _, addition := range []struct {
			name     string
			edit     func(d *viewerDiagram)
			old, new string
		}{
			{"a trigger is declared with the actor and the view its node names", func(d *viewerDiagram) {
				d.add("trigger", "Returns Counter", "Available Copies")
				d.set("Returns Counter", "actor", "Member")
				d.set("Returns Counter", "reads", "AvailableCopiesView")
			}, "    slice \"Available Copies\" {\n", "    slice \"Available Copies\" {\n      trigger \"Returns Counter\" {\n        actor = Member\n        reads = AvailableCopiesView\n      }\n\n"},
			{"a view is declared with the events its node subscribes to", func(d *viewerDiagram) {
				d.add("view", "OverdueLoansView", "Recall Overdue Copy")
				d.set("OverdueLoansView", "subscribes", []string{"CopyBorrowed"})
			}, "      # Chases a copy nobody brought back\n", "      view \"OverdueLoansView\" {\n        subscribes = [CopyBorrowed]\n      }\n\n      # Chases a copy nobody brought back\n"},
			{"an automation is declared with the event and the command its node names", func(d *viewerDiagram) {
				d.add("automation", "RemindMember", "Import Partner Loan")
				d.set("RemindMember", "on_event", "CopyBorrowed")
				d.set("RemindMember", "command", "RecallCopy")
			}, "    # Restates a partner branch's notice in the library's own language\n", "    automation \"RemindMember\" {\n      on      = CopyBorrowed\n      command = RecallCopy\n    }\n\n    # Restates a partner branch's notice in the library's own language\n"},
			{"a translation is declared with its external system and the event its node nests", func(d *viewerDiagram) {
				d.add("translation", "CourierNoticeImport", "Recall Overdue Copy")
				d.set("CourierNoticeImport", "external_system", "Courier API")
				d.set("CourierNoticeImport", "event", map[string]any{"name": "CopyCollected"})
			}, "        command = RecallCopy\n      }\n", "        command = RecallCopy\n      }\n\n      translation \"CourierNoticeImport\" {\n        external_system = \"Courier API\"\n\n        event \"CopyCollected\" {\n        }\n      }\n"},
		} {
			t.Run(addition.name, func(t *testing.T) {
				result := applied(t, lendingModel, addition.edit)

				require.Equal(t, replaced(t, lendingModel, addition.old, addition.new), result)
			})
		}

		t.Run("a context is declared after the model's contexts", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.Nodes = append(d.Nodes, map[string]any{"id": "_context_1", "type": "context", "label": "Acquisitions", "parentId": nil})
			})

			require.Equal(t, lendingModel+"\ncontext \"Acquisitions\" {\n}\n", result)
		})

		t.Run("an aggregate is declared after its context's aggregates", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.add("aggregate", "Copy", "Lending")
			})

			require.Equal(t, replaced(t, lendingModel, "\n  slice \"Import Partner Loan\" {", "\n  aggregate \"Copy\" {\n  }\n\n  slice \"Import Partner Loan\" {"), result)
		})
	})

	t.Run("deleting", func(t *testing.T) {
		t.Run("deleting a command removes its declaration and its comments, and clears the automation that issued it", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("RecallCopy")
			})

			require.Equal(t, replaced(t, lendingModel, `    slice "Recall Overdue Copy" {
      # Pull a copy back from a member who kept it too long
      command "RecallCopy" {
        fields {
          loanId = required(string)
        }
      }

      # Chases a copy nobody brought back
      automation "RecallOverdueCopy" {
        on      = CopyBorrowed
        reads   = AvailableCopiesView
        command = RecallCopy
      }
`, `    slice "Recall Overdue Copy" {
      # Chases a copy nobody brought back
      automation "RecallOverdueCopy" {
        on    = CopyBorrowed
        reads = AvailableCopiesView
      }
`), result)
		})

		t.Run("deleting a command removes its flow line, and leaves the rejection and the spec that name it", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("BorrowCopy")
			})

			require.Equal(t, replaced(t, lendingModel, `      # Ask the library to hand a copy over
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }

      # A copy left the building
      event "CopyBorrowed" {
        fields {
          loanId = required(string)
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
        command -> rejected: BorrowCopy -> OneLoanPerCopy
      FLOW
`, `      # A copy left the building
      event "CopyBorrowed" {
        fields {
          loanId = required(string)
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> rejected: BorrowCopy -> OneLoanPerCopy
      FLOW
`), result)
		})

		t.Run("deleting an event clears the flow, the subscription and the activation drawn to it, and leaves the spec that names it", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("CopyBorrowed")
			})

			expected := replaced(t, lendingModel, `      # A copy left the building
      event "CopyBorrowed" {
        fields {
          loanId = required(string)
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
        command -> rejected: BorrowCopy -> OneLoanPerCopy
      FLOW
`, `      flow = <<-FLOW
        command -> rejected: BorrowCopy -> OneLoanPerCopy
      FLOW
`)
			expected = replaced(t, expected, `      view "AvailableCopiesView" {
        subscribes = [CopyBorrowed]

        fields {`, `      view "AvailableCopiesView" {
        fields {`)
			expected = replaced(t, expected, `        on      = CopyBorrowed
        reads   = AvailableCopiesView
        command = RecallCopy`, `        reads   = AvailableCopiesView
        command = RecallCopy`)
			require.Contains(t, expected, `then = [CopyBorrowed({ copyId = "C-1" })]`)
			require.Equal(t, expected, result)
		})

		t.Run("deleting a trigger removes its declaration and its comments", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("Lending Desk")
			})

			require.Equal(t, replaced(t, lendingModel, `      # The desk terminal the librarian types into
      trigger "Lending Desk" {
        actor = Member
        reads = AvailableCopiesView
      }

`, ``), result)
		})

		t.Run("deleting a view clears what the trigger, the automation and the translation read", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("AvailableCopiesView")
			})

			expected := replaced(t, lendingModel, `    slice "Available Copies" {
      # What the desk shows a member
      view "AvailableCopiesView" {
        subscribes = [CopyBorrowed]

        fields {
          copyId = required(string)
        }
      }
    }
`, `    slice "Available Copies" {
    }
`)
			expected = replaced(t, expected, `        actor = Member
        reads = AvailableCopiesView
`, `        actor = Member
`)
			expected = replaced(t, expected, `        on      = CopyBorrowed
        reads   = AvailableCopiesView
        command = RecallCopy`, `        on      = CopyBorrowed
        command = RecallCopy`)
			expected = replaced(t, expected, `      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan`, `      external_system = "Partner Branch API"
      command         = ImportPartnerLoan`)
			require.Equal(t, expected, result)
		})

		t.Run("deleting an automation removes its declaration and its comments", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("RecallOverdueCopy")
			})

			require.Equal(t, replaced(t, lendingModel, `

      # Chases a copy nobody brought back
      automation "RecallOverdueCopy" {
        on      = CopyBorrowed
        reads   = AvailableCopiesView
        command = RecallCopy
      }
`, `
`), result)
		})

		t.Run("deleting the event a translation nests removes it from the translation", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("PartnerLoanImported")
			})

			require.Equal(t, replaced(t, lendingModel, `      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
`, `      command         = ImportPartnerLoan
`), result)
		})

		t.Run("deleting a translation removes it with the event it nests", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("PartnerLoanImport")
			})

			require.Equal(t, replaced(t, lendingModel, `

    # Restates a partner branch's notice in the library's own language
    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
    }
`, `
`), result)
		})

		t.Run("deleting a translation clears the subscription and the activation drawn from the event it nests", func(t *testing.T) {
			source := replaced(t, lendingModel, "subscribes = [CopyBorrowed]", "subscribes = [CopyBorrowed, PartnerLoanImported]")
			source = replaced(t, source, "on      = CopyBorrowed", "on      = PartnerLoanImported")

			result := applied(t, source, func(d *viewerDiagram) {
				d.remove("PartnerLoanImport")
			})

			expected := replaced(t, lendingModel, "        on      = CopyBorrowed\n        reads   = AvailableCopiesView\n", "        reads   = AvailableCopiesView\n")
			expected = replaced(t, expected, `

    # Restates a partner branch's notice in the library's own language
    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
    }
`, `
`)
			require.Equal(t, expected, result)
		})

		t.Run("deleting a translation and drawing a subscription from the event it nested adds no subscription", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("PartnerLoanImport")
				d.connect("PartnerLoanImported", "AvailableCopiesView", "subscription")
			})

			require.Equal(t, replaced(t, lendingModel, `

    # Restates a partner branch's notice in the library's own language
    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
    }
`, "\n"), result)
		})

		t.Run("deleting a translation and drawing an activation from the event it nested sets no activation", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("PartnerLoanImport")
				d.disconnect("CopyBorrowed", "RecallOverdueCopy")
				d.connect("PartnerLoanImported", "RecallOverdueCopy", "automation_trigger")
			})

			expected := replaced(t, lendingModel, `

    # Restates a partner branch's notice in the library's own language
    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
    }
`, "\n")
			expected = replaced(t, expected, "        on      = CopyBorrowed\n        reads   = AvailableCopiesView\n        command = RecallCopy\n", "        reads   = AvailableCopiesView\n        command = RecallCopy\n")
			require.Equal(t, expected, result)
		})

		t.Run("deleting a translation and drawing a flow arrow to the event it nested adds no flow line", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("PartnerLoanImport")
				d.connect("ImportPartnerLoan", "PartnerLoanImported", "flow")
			})

			require.Equal(t, replaced(t, lendingModel, `

    # Restates a partner branch's notice in the library's own language
    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
    }
`, "\n"), result)
		})

		t.Run("deleting a translation and moving a subscription onto the event it nested removes the subscription", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.remove("PartnerLoanImport")
				d.repoint("CopyBorrowed", "AvailableCopiesView", "PartnerLoanImported", "AvailableCopiesView")
			})

			expected := replaced(t, lendingModel, `

    # Restates a partner branch's notice in the library's own language
    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
    }
`, "\n")
			expected = replaced(t, expected, "        subscribes = [CopyBorrowed]\n\n", "")
			require.Equal(t, expected, result)
		})
	})

	t.Run("renaming", func(t *testing.T) {
		for _, rename := range []struct {
			kind, label, to, header, renamed string
		}{
			{"context", "Lending", "Circulation", `context "Lending" {`, `context "Circulation" {`},
			{"aggregate", "Loan", "Lending Record", `aggregate "Loan" {`, `aggregate "Lending Record" {`},
			{"slice", "Borrow Copy", "Lend Copy", `slice "Borrow Copy" {`, `slice "Lend Copy" {`},
			{"trigger", "Lending Desk", "Front Desk", `trigger "Lending Desk" {`, `trigger "Front Desk" {`},
			{"automation", "RecallOverdueCopy", "ChaseOverdueCopy", `automation "RecallOverdueCopy" {`, `automation "ChaseOverdueCopy" {`},
			{"translation", "PartnerLoanImport", "PartnerNoticeImport", `translation "PartnerLoanImport" {`, `translation "PartnerNoticeImport" {`},
		} {
			t.Run("renaming a "+rename.kind+" renames its block and nothing else", func(t *testing.T) {
				result := applied(t, lendingModel, func(d *viewerDiagram) {
					d.rename(rename.label, rename.to)
				})

				require.Equal(t, replaced(t, lendingModel, rename.header, rename.renamed), result)
			})
		}

		t.Run("renaming a command renames its flow line, and leaves the rejection and the spec that name it", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.rename("BorrowCopy", "LendCopy")
			})

			expected := replaced(t, lendingModel, `command "BorrowCopy" {`, `command "LendCopy" {`)
			expected = replaced(t, expected, `command -> event:    BorrowCopy -> CopyBorrowed`, `command -> event:    LendCopy -> CopyBorrowed`)
			require.Contains(t, expected, `command -> rejected: BorrowCopy -> OneLoanPerCopy`)
			require.Contains(t, expected, `when = BorrowCopy({ copyId = "C-1" })`)
			require.Equal(t, expected, result)
		})

		t.Run("renaming a command renames it in the automation and the translation that issue it", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.rename("RecallCopy", "ReclaimCopy")
				d.rename("ImportPartnerLoan", "RecordPartnerLoan")
			})

			expected := replaced(t, lendingModel, `command "RecallCopy" {`, `command "ReclaimCopy" {`)
			expected = replaced(t, expected, `command = RecallCopy`, `command = ReclaimCopy`)
			expected = replaced(t, expected, `command "ImportPartnerLoan" {`, `command "RecordPartnerLoan" {`)
			expected = replaced(t, expected, `command         = ImportPartnerLoan`, `command         = RecordPartnerLoan`)
			require.Equal(t, expected, result)
		})

		t.Run("renaming an event renames its flow line, the view's subscription and the automation's activation, and leaves the spec", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.rename("CopyBorrowed", "CopyLent")
			})

			expected := replaced(t, lendingModel, `event "CopyBorrowed" {`, `event "CopyLent" {`)
			expected = replaced(t, expected, `BorrowCopy -> CopyBorrowed`, `BorrowCopy -> CopyLent`)
			expected = replaced(t, expected, `subscribes = [CopyBorrowed]`, `subscribes = [CopyLent]`)
			expected = replaced(t, expected, `on      = CopyBorrowed`, `on      = CopyLent`)
			require.Contains(t, expected, `then = [CopyBorrowed({ copyId = "C-1" })]`)
			require.Equal(t, expected, result)
		})

		t.Run("renaming one command to the old name of another renames each flow line once", func(t *testing.T) {
			result := applied(t, wiringModel, func(d *viewerDiagram) {
				d.rename("RenewLoan", "ExtendLoan")
				d.rename("BorrowCopy", "RenewLoan")
			})

			expected := replaced(t, wiringModel, `command "RenewLoan" {`, `command "ExtendLoan" {`)
			expected = replaced(t, expected, `command "BorrowCopy" {`, `command "RenewLoan" {`)
			expected = replaced(t, expected, `      command -> event:    BorrowCopy -> CopyBorrowed
      command -> event:    BorrowCopy -> LoanRenewed
      command -> event:    RenewLoan -> LoanRenewed
`, `      command -> event:    RenewLoan -> CopyBorrowed
      command -> event:    RenewLoan -> LoanRenewed
      command -> event:    ExtendLoan -> LoanRenewed
`)
			require.Equal(t, expected, result)
		})

		t.Run("renaming one event to the old name of another renames each flow line and subscription once", func(t *testing.T) {
			result := applied(t, wiringModel, func(d *viewerDiagram) {
				d.rename("LoanRenewed", "LoanExtended")
				d.rename("CopyBorrowed", "LoanRenewed")
			})

			expected := replaced(t, wiringModel, `event "LoanRenewed" {`, `event "LoanExtended" {`)
			expected = replaced(t, expected, `event "CopyBorrowed" {`, `event "LoanRenewed" {`)
			expected = replaced(t, expected, "subscribes = [CopyBorrowed, LoanRenewed]", "subscribes = [LoanRenewed, LoanExtended]")
			expected = replaced(t, expected, `      command -> event:    BorrowCopy -> CopyBorrowed
      command -> event:    BorrowCopy -> LoanRenewed
      command -> event:    RenewLoan -> LoanRenewed
`, `      command -> event:    BorrowCopy -> LoanRenewed
      command -> event:    BorrowCopy -> LoanExtended
      command -> event:    RenewLoan -> LoanExtended
`)
			require.Equal(t, expected, result)
		})

		t.Run("renaming an event renames every flow line and the subscription that name it", func(t *testing.T) {
			result := applied(t, wiringModel, func(d *viewerDiagram) {
				d.rename("LoanRenewed", "LoanExtended")
			})

			expected := replaced(t, wiringModel, `event "LoanRenewed" {`, `event "LoanExtended" {`)
			expected = replaced(t, expected, "subscribes = [CopyBorrowed, LoanRenewed]", "subscribes = [CopyBorrowed, LoanExtended]")
			expected = replaced(t, expected, "BorrowCopy -> LoanRenewed\n", "BorrowCopy -> LoanExtended\n")
			expected = replaced(t, expected, "RenewLoan -> LoanRenewed\n", "RenewLoan -> LoanExtended\n")
			require.Equal(t, expected, result)
		})

		t.Run("renaming a command whose flow is in a later slice renames that flow line, and leaves its specs", func(t *testing.T) {
			source := readExample(t, "specs_hotel.emod")

			result := applied(t, source, func(d *viewerDiagram) {
				d.rename("HoldRoom", "PlaceHold")
			})

			expected := replaced(t, source, `command "HoldRoom" {`, `command "PlaceHold" {`)
			expected = replaced(t, expected, "HoldRoom -> RoomHeld", "PlaceHold -> RoomHeld")
			require.Contains(t, expected, `when = HoldRoom({ roomId = "204", guestName = "Ada Lovelace" })`)
			require.Equal(t, expected, result)
		})

		t.Run("renaming an event second in a view's subscriptions renames it there", func(t *testing.T) {
			source := readExample(t, "specs_hotel.emod")

			result := applied(t, source, func(d *viewerDiagram) {
				d.rename("RoomReserved", "RoomBooked")
			})

			expected := replaced(t, source, `event "RoomReserved" {`, `event "RoomBooked" {`)
			expected = replaced(t, expected, "ReserveRoom -> RoomReserved", "ReserveRoom -> RoomBooked")
			expected = strings.ReplaceAll(expected, "subscribes  = [RoomHeld, RoomReserved, HoldReleased]", "subscribes  = [RoomHeld, RoomBooked, HoldReleased]")
			require.Equal(t, 2, strings.Count(expected, "[RoomHeld, RoomBooked, HoldReleased]"), "both views have to subscribe to it")
			require.Equal(t, expected, result)
		})

		t.Run("renaming a command that two identical flow lines name renames both lines", func(t *testing.T) {
			source := `emod = 1

model "Library Lending" {
}

context "Lending" {
  slice "Borrow Copy" {
    command "BorrowCopy" {
    }

    event "CopyBorrowed" {
    }

    flow = <<-FLOW
      command -> event:    BorrowCopy -> CopyBorrowed
      command -> event:    BorrowCopy -> CopyBorrowed
    FLOW
  }
}
`

			result := applied(t, source, func(d *viewerDiagram) {
				d.rename("BorrowCopy", "LendCopy")
			})

			expected := replaced(t, source, `command "BorrowCopy" {`, `command "LendCopy" {`)
			expected = replaced(t, expected, `      command -> event:    BorrowCopy -> CopyBorrowed
      command -> event:    BorrowCopy -> CopyBorrowed
`, `      command -> event:    LendCopy -> CopyBorrowed
      command -> event:    LendCopy -> CopyBorrowed
`)
			require.Equal(t, expected, result)
		})

		t.Run("renaming an event that a view subscribes to twice renames both entries", func(t *testing.T) {
			source := `emod = 1

model "Library Lending" {
}

context "Lending" {
  slice "Borrow Copy" {
    event "CopyBorrowed" {
    }

    view "MemberLoansView" {
      subscribes = [CopyBorrowed, CopyBorrowed]
    }
  }
}
`

			result := applied(t, source, func(d *viewerDiagram) {
				d.rename("CopyBorrowed", "CopyLent")
			})

			expected := replaced(t, source, `event "CopyBorrowed" {`, `event "CopyLent" {`)
			expected = replaced(t, expected, "subscribes = [CopyBorrowed, CopyBorrowed]", "subscribes = [CopyLent, CopyLent]")
			require.Equal(t, expected, result)
		})

		t.Run("renaming the event a translation nests renames it inside the translation and declares no other", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.rename("PartnerLoanImported", "PartnerLoanRecorded")
			})

			require.Equal(t, replaced(t, lendingModel, `event "PartnerLoanImported" {`, `event "PartnerLoanRecorded" {`), result)
		})

		t.Run("renaming a view renames it in what the trigger, the automation and the translation read", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.rename("AvailableCopiesView", "ShelfView")
			})

			expected := replaced(t, lendingModel, `view "AvailableCopiesView" {`, `view "ShelfView" {`)
			expected = replaced(t, expected, `        reads = AvailableCopiesView`, `        reads = ShelfView`)
			expected = replaced(t, expected, `        reads   = AvailableCopiesView`, `        reads   = ShelfView`)
			expected = replaced(t, expected, `      reads           = AvailableCopiesView`, `      reads           = ShelfView`)
			require.Equal(t, expected, result)
		})
	})

	t.Run("fields", func(t *testing.T) {
		for _, edit := range []struct {
			name    string
			fields  []map[string]any
			written string
		}{
			{"renaming a field renames it", []map[string]any{field("bookId", "string", "required")},
				"          bookId = required(string)\n"},
			{"retyping a field retypes it", []map[string]any{field("copyId", "int", "required")},
				"          copyId = required(int)\n"},
			{"setting a modifier writes it", []map[string]any{field("copyId", "string", "optional")},
				"          copyId = optional(string)\n"},
			{"clearing a modifier writes the bare type", []map[string]any{field("copyId", "string", "")},
				"          copyId = string\n"},
			{"adding a field declares it after the others", []map[string]any{field("copyId", "string", "required"), field("memberId", "string", "")},
				"          copyId   = required(string)\n          memberId = string\n"},
			{"a field with no name is left out", []map[string]any{field("copyId", "string", "required"), field("", "string", "")},
				"          copyId = required(string)\n"},
		} {
			t.Run(edit.name, func(t *testing.T) {
				result := applied(t, lendingModel, func(d *viewerDiagram) {
					d.setFields("BorrowCopy", edit.fields...)
				})

				require.Equal(t, replaced(t, lendingModel, `      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }`, `      command "BorrowCopy" {
        fields {
`+edit.written+`        }`), result)
			})
		}

		t.Run("changing the fields of the event a translation nests changes them inside the translation", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.setFields("PartnerLoanImported", field("externalRef", "string", "required"), field("branch", "string", "required"))
			})

			require.Equal(t, replaced(t, lendingModel, `          externalRef = required(string)
        }
      }
    }
  }
}`, `          externalRef = required(string)
          branch      = required(string)
        }
      }
    }
  }
}`), result)
		})

		t.Run("deleting a command's last field removes its fields block", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.setFields("BorrowCopy")
			})

			require.Equal(t, replaced(t, lendingModel, `      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }`, `      command "BorrowCopy" {
      }`), result)
		})

		t.Run("an event's fields follow its node's fields", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.setFields("CopyBorrowed", field("loanId", "string", "required"))
			})

			require.Equal(t, replaced(t, lendingModel, `        fields {
          loanId = required(string)
          copyId = required(string)
        }`, `        fields {
          loanId = required(string)
        }`), result)
		})
	})

	t.Run("arrows", func(t *testing.T) {
		t.Run("an arrow drawn from an event to a view subscribes the view to the event", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.connect("PartnerLoanImported", "AvailableCopiesView", "subscription")
			})

			require.Equal(t, replaced(t, lendingModel, `subscribes = [CopyBorrowed]`, `subscribes = [CopyBorrowed, PartnerLoanImported]`), result)
		})

		for _, connection := range []struct {
			name, from, to, typ, written, folded string
		}{
			{"a view's arrow to a trigger makes the trigger read the view", "AvailableCopiesView", "Lending Desk", "reads",
				"    trigger \"Lending Desk\" {\n    }", "    trigger \"Lending Desk\" {\n      reads = AvailableCopiesView\n    }"},
			{"a view's arrow to an automation makes the automation read the view", "AvailableCopiesView", "RemindMember", "reads",
				"    automation \"RemindMember\" {\n    }", "    automation \"RemindMember\" {\n      reads = AvailableCopiesView\n    }"},
			{"a view's arrow to a translation makes the translation read the view", "AvailableCopiesView", "PartnerLoanImport", "reads",
				"      external_system = \"Partner Branch API\"\n", "      external_system = \"Partner Branch API\"\n      reads           = AvailableCopiesView\n"},
			{"an event's arrow to an automation makes the automation activate on the event", "CopyBorrowed", "RemindMember", "automation_trigger",
				"    automation \"RemindMember\" {\n    }", "    automation \"RemindMember\" {\n      on = CopyBorrowed\n    }"},
			{"an automation's arrow to a command makes the automation issue the command", "RemindMember", "RenewLoan", "automation_command",
				"    automation \"RemindMember\" {\n    }", "    automation \"RemindMember\" {\n      command = RenewLoan\n    }"},
			{"a translation's arrow to a command makes the translation issue the command", "PartnerLoanImport", "RenewLoan", "translation_command",
				"      external_system = \"Partner Branch API\"\n", "      external_system = \"Partner Branch API\"\n      command         = RenewLoan\n"},
			{"an event's arrow to a view subscribes the view to the event", "CopyBorrowed", "AvailableCopiesView", "subscription",
				"    view \"AvailableCopiesView\" {\n    }", "    view \"AvailableCopiesView\" {\n      subscribes = [CopyBorrowed]\n    }"},
			{"a command's arrow to an event of its slice declares the flow", "RenewLoan", "CopyBorrowed", "flow",
				"      command -> event:    RenewLoan -> LoanRenewed\n", "      command -> event:    RenewLoan -> LoanRenewed\n      command -> event:    RenewLoan -> CopyBorrowed\n"},
			{"a command's arrow to the event a translation nests declares the flow", "RenewLoan", "PartnerLoanImported", "flow",
				"      command -> event:    RenewLoan -> LoanRenewed\n", "      command -> event:    RenewLoan -> LoanRenewed\n      command -> event:    RenewLoan -> PartnerLoanImported\n"},
		} {
			t.Run(connection.name, func(t *testing.T) {
				result := applied(t, wiringModel, func(d *viewerDiagram) {
					d.connect(connection.from, connection.to, connection.typ)
				})

				require.Equal(t, replaced(t, wiringModel, connection.written, connection.folded), result)
			})
		}

		t.Run("removing one of two flows that share a command and an event removes only its line", func(t *testing.T) {
			result := applied(t, wiringModel, func(d *viewerDiagram) {
				d.disconnect("BorrowCopy", "LoanRenewed")
			})

			require.Equal(t, replaced(t, wiringModel, "      command -> event:    BorrowCopy -> LoanRenewed\n", ""), result)
		})

		t.Run("removing the second of a view's subscriptions keeps the first", func(t *testing.T) {
			result := applied(t, wiringModel, func(d *viewerDiagram) {
				d.disconnect("LoanRenewed", "MemberLoansView")
			})

			require.Equal(t, replaced(t, wiringModel, "subscribes = [CopyBorrowed, LoanRenewed]", "subscribes = [CopyBorrowed]"), result)
		})

		t.Run("an arrow left behind by a deleted node is cleared with it", func(t *testing.T) {
			result := applied(t, wiringModel, func(d *viewerDiagram) {
				id := d.idOf("LoanRenewed")
				d.Nodes = slices.DeleteFunc(d.Nodes, func(n map[string]any) bool { return n["id"] == id })
			})

			expected := replaced(t, wiringModel, "    event \"LoanRenewed\" {\n    }\n\n", "")
			expected = replaced(t, expected, "subscribes = [CopyBorrowed, LoanRenewed]", "subscribes = [CopyBorrowed]")
			expected = replaced(t, expected, "      command -> event:    BorrowCopy -> LoanRenewed\n      command -> event:    RenewLoan -> LoanRenewed\n", "")
			require.Equal(t, expected, result)
		})

		t.Run("removing a flow arrow in a later slice removes its line", func(t *testing.T) {
			source := readExample(t, "specs_hotel.emod")

			result := applied(t, source, func(d *viewerDiagram) {
				d.disconnect("HoldRoom", "RoomHeld")
			})

			require.Equal(t, replaced(t, source, "      flow = <<-FLOW\n        command -> event:    HoldRoom -> RoomHeld\n      FLOW\n\n", ""), result)
		})

		t.Run("an arrow drawn between a command and an event of another slice changes nothing", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.connect("RecallCopy", "CopyBorrowed", "flow")
			})

			require.Equal(t, lendingModel, result)
		})

		t.Run("an arrow drawn from a trigger to a command changes nothing", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.connect("Lending Desk", "RecallCopy", "trigger_command")
			})

			require.Equal(t, lendingModel, result)
		})

		for _, removal := range []struct {
			name, from, to, written, cleared string
		}{
			{"removing a flow arrow removes its flow line", "BorrowCopy", "CopyBorrowed",
				"        command -> event:    BorrowCopy -> CopyBorrowed\n", ""},
			{"removing a subscription arrow unsubscribes the view", "CopyBorrowed", "AvailableCopiesView",
				"        subscribes = [CopyBorrowed]\n\n", ""},
			{"removing an activation arrow clears the automation's on", "CopyBorrowed", "RecallOverdueCopy",
				"        on      = CopyBorrowed\n        reads   = AvailableCopiesView\n        command = RecallCopy\n",
				"        reads   = AvailableCopiesView\n        command = RecallCopy\n"},
			{"removing an automation's command arrow clears its command", "RecallOverdueCopy", "RecallCopy",
				"        on      = CopyBorrowed\n        reads   = AvailableCopiesView\n        command = RecallCopy\n",
				"        on    = CopyBorrowed\n        reads = AvailableCopiesView\n"},
			{"removing a translation's command arrow clears its command", "PartnerLoanImport", "ImportPartnerLoan",
				"      reads           = AvailableCopiesView\n      command         = ImportPartnerLoan\n",
				"      reads           = AvailableCopiesView\n"},
			{"removing a trigger's reads arrow clears what it reads", "AvailableCopiesView", "Lending Desk",
				"        actor = Member\n        reads = AvailableCopiesView\n", "        actor = Member\n"},
		} {
			t.Run(removal.name, func(t *testing.T) {
				result := applied(t, lendingModel, func(d *viewerDiagram) {
					d.disconnect(removal.from, removal.to)
				})

				require.Equal(t, replaced(t, lendingModel, removal.written, removal.cleared), result)
			})
		}

		t.Run("removing the first flow arrow keeps the comment above the flow block on the lines that stay", func(t *testing.T) {
			source := replaced(t, lendingModel, "      flow = <<-FLOW\n", "      # How a loan comes about at the desk\n      flow = <<-FLOW\n")

			result := applied(t, source, func(d *viewerDiagram) {
				d.disconnect("BorrowCopy", "CopyBorrowed")
			})

			require.Equal(t, replaced(t, source, "        command -> event:    BorrowCopy -> CopyBorrowed\n", ""), result)
		})

		t.Run("removing the first of several flow arrows keeps the comment above the flow block on the next line", func(t *testing.T) {
			source := replaced(t, wiringModel, "    flow = <<-FLOW\n", "    # Every way a loan comes about\n    flow = <<-FLOW\n")

			result := applied(t, source, func(d *viewerDiagram) {
				d.disconnect("BorrowCopy", "CopyBorrowed")
			})

			require.Equal(t, replaced(t, source, "      command -> event:    BorrowCopy -> CopyBorrowed\n", ""), result)
		})

		t.Run("removing all but the last of several flow arrows keeps the comment above the flow block on the line that stays", func(t *testing.T) {
			source := replaced(t, wiringModel, "    flow = <<-FLOW\n", "    # Every way a loan comes about\n    flow = <<-FLOW\n")

			result := applied(t, source, func(d *viewerDiagram) {
				d.disconnect("BorrowCopy", "CopyBorrowed")
				d.disconnect("BorrowCopy", "LoanRenewed")
			})

			require.Equal(t, replaced(t, source, "      command -> event:    BorrowCopy -> CopyBorrowed\n      command -> event:    BorrowCopy -> LoanRenewed\n", ""), result)
		})

		t.Run("removing the first flow arrow of two slices in one edit keeps the comment above each flow block", func(t *testing.T) {
			source := `emod = 1

model "Library Lending" {
}

context "Lending" {
  slice "Borrow Copy" {
    command "BorrowCopy" {
    }

    event "CopyBorrowed" {
    }

    event "LoanOpened" {
    }

    # How a loan comes about at the desk
    flow = <<-FLOW
      command -> event:    BorrowCopy -> CopyBorrowed
      command -> event:    BorrowCopy -> LoanOpened
    FLOW
  }

  slice "Return Copy" {
    command "ReturnCopy" {
    }

    event "CopyReturned" {
    }

    event "LoanClosed" {
    }

    # How a loan ends at the desk
    flow = <<-FLOW
      command -> event:    ReturnCopy -> CopyReturned
      command -> event:    ReturnCopy -> LoanClosed
    FLOW
  }
}
`

			result := applied(t, source, func(d *viewerDiagram) {
				d.disconnect("BorrowCopy", "CopyBorrowed")
				d.disconnect("ReturnCopy", "CopyReturned")
			})

			expected := replaced(t, source, "      command -> event:    BorrowCopy -> CopyBorrowed\n", "")
			expected = replaced(t, expected, "      command -> event:    ReturnCopy -> CopyReturned\n", "")
			require.Equal(t, expected, result)
		})

		t.Run("moving the target end of the only flow arrow keeps the comment above the flow block", func(t *testing.T) {
			source := `emod = 1

model "Library Lending" {
}

context "Lending" {
  slice "Borrow Copy" {
    command "BorrowCopy" {
    }

    event "CopyBorrowed" {
    }

    event "CopyReserved" {
    }

    # How a loan comes about at the desk
    flow = <<-FLOW
      command -> event:    BorrowCopy -> CopyBorrowed
    FLOW
  }
}
`

			result := applied(t, source, func(d *viewerDiagram) {
				d.repoint("BorrowCopy", "CopyBorrowed", "BorrowCopy", "CopyReserved")
			})

			require.Equal(t, replaced(t, source, "BorrowCopy -> CopyBorrowed", "BorrowCopy -> CopyReserved"), result)
		})

		t.Run("removing an arrow no text of its own states changes nothing", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.disconnect("Lending Desk", "BorrowCopy")
				d.disconnect("ImportPartnerLoan", "PartnerLoanImported")
			})

			require.Equal(t, lendingModel, result)
		})

		t.Run("moving the source end of the first of two subscription arrows rewrites its entry where it stands", func(t *testing.T) {
			source := repointModel

			result := applied(t, source, func(d *viewerDiagram) {
				d.repoint("CopyBorrowed", "MemberLoansView", "CopyReserved", "MemberLoansView")
			})

			require.Equal(t, replaced(t, source, "subscribes = [CopyBorrowed, LoanOpened]", "subscribes = [CopyReserved, LoanOpened]"), result)
		})

		t.Run("moving the target end of the first of two flow arrows rewrites its line where it stands and the comment above the block", func(t *testing.T) {
			source := repointModel

			result := applied(t, source, func(d *viewerDiagram) {
				d.repoint("BorrowCopy", "CopyBorrowed", "BorrowCopy", "CopyReserved")
			})

			require.Equal(t, replaced(t, source, "BorrowCopy -> CopyBorrowed", "BorrowCopy -> CopyReserved"), result)
		})

		t.Run("moving the target end of a flow arrow onto an event of another slice removes its line and keeps the comment above the block", func(t *testing.T) {
			source := repointModel

			result := applied(t, source, func(d *viewerDiagram) {
				d.repoint("BorrowCopy", "CopyBorrowed", "BorrowCopy", "CopyReturned")
			})

			require.Equal(t, replaced(t, source, "      command -> event:    BorrowCopy -> CopyBorrowed\n", ""), result)
		})

		t.Run("moving the target end of the second flow arrow of a command rewrites its line where it stands", func(t *testing.T) {
			result := applied(t, repointModel, func(d *viewerDiagram) {
				d.repoint("BorrowCopy", "LoanOpened", "BorrowCopy", "CopyReserved")
			})

			require.Equal(t, replaced(t, repointModel, "BorrowCopy -> LoanOpened", "BorrowCopy -> CopyReserved"), result)
		})

		t.Run("moving the target end of a flow arrow in a later slice rewrites its line where it stands", func(t *testing.T) {
			result := applied(t, repointModel, func(d *viewerDiagram) {
				d.repoint("ReturnCopy", "CopyReturned", "ReturnCopy", "CopyShelved")
			})

			require.Equal(t, replaced(t, repointModel, "ReturnCopy -> CopyReturned", "ReturnCopy -> CopyShelved"), result)
		})

		t.Run("removing a subscription arrow of one view and moving the source end of one of a later view rewrites the moved entry where it stands", func(t *testing.T) {
			result := applied(t, repointModel, func(d *viewerDiagram) {
				d.disconnect("CopyBorrowed", "MemberLoansView")
				d.repoint("CopyReturned", "LoanHistoryView", "CopyShelved", "LoanHistoryView")
			})

			expected := replaced(t, repointModel, "subscribes = [CopyBorrowed, LoanOpened]", "subscribes = [LoanOpened]")
			expected = replaced(t, expected, "subscribes = [CopyReturned, LoanClosed]", "subscribes = [CopyShelved, LoanClosed]")
			require.Equal(t, expected, result)
		})

		t.Run("moving the ends of two arrows and removing a third in one edit writes each change", func(t *testing.T) {
			result := applied(t, repointModel, func(d *viewerDiagram) {
				d.repoint("CopyBorrowed", "MemberLoansView", "CopyReserved", "MemberLoansView")
				d.repoint("ReturnCopy", "CopyReturned", "ReturnCopy", "CopyShelved")
				d.disconnect("BorrowCopy", "LoanOpened")
			})

			expected := replaced(t, repointModel, "subscribes = [CopyBorrowed, LoanOpened]", "subscribes = [CopyReserved, LoanOpened]")
			expected = replaced(t, expected, "ReturnCopy -> CopyReturned", "ReturnCopy -> CopyShelved")
			expected = replaced(t, expected, "      command -> event:    BorrowCopy -> LoanOpened\n", "")
			require.Equal(t, expected, result)
		})

		t.Run("moving the source end of a subscription arrow and drawing another to the same view keeps both", func(t *testing.T) {
			result := applied(t, repointModel, func(d *viewerDiagram) {
				d.repoint("CopyBorrowed", "MemberLoansView", "CopyReserved", "MemberLoansView")
				d.connect("CopyLost", "MemberLoansView", "subscription")
			})

			require.Equal(t, replaced(t, repointModel, "subscribes = [CopyBorrowed, LoanOpened]", "subscribes = [CopyReserved, LoanOpened, CopyLost]"), result)
		})

		t.Run("moving the source end of an activation arrow activates the automation on the new event", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.repoint("CopyBorrowed", "RecallOverdueCopy", "PartnerLoanImported", "RecallOverdueCopy")
			})

			require.Equal(t, replaced(t, lendingModel, "on      = CopyBorrowed", "on      = PartnerLoanImported"), result)
		})

		t.Run("an edit elsewhere keeps the subscription and the activation that name the event a translation nests", func(t *testing.T) {
			source := replaced(t, lendingModel, "subscribes = [CopyBorrowed]", "subscribes = [CopyBorrowed, PartnerLoanImported]")
			source = replaced(t, source, "on      = CopyBorrowed", "on      = PartnerLoanImported")

			result := applied(t, source, func(d *viewerDiagram) {
				d.rename("Lending Desk", "Front Desk")
			})

			require.Equal(t, replaced(t, source, `trigger "Lending Desk" {`, `trigger "Front Desk" {`), result)
		})

		t.Run("moving the target end of the arrow from a translation's command to its event states the moved arrow as a flow line", func(t *testing.T) {
			source := replaced(t, lendingModel, "    # Restates a partner branch's notice", "    event \"PartnerLoanRecorded\" {\n    }\n\n    # Restates a partner branch's notice")

			result := applied(t, source, func(d *viewerDiagram) {
				d.repoint("ImportPartnerLoan", "PartnerLoanImported", "ImportPartnerLoan", "PartnerLoanRecorded")
			})

			require.Equal(t, replaced(t, source, "          externalRef = required(string)\n        }\n      }\n    }\n  }\n}\n",
				"          externalRef = required(string)\n        }\n      }\n    }\n\n    flow = <<-FLOW\n      command -> event:    ImportPartnerLoan -> PartnerLoanRecorded\n    FLOW\n  }\n}\n"), result)
		})

		t.Run("moving the source end of a subscription arrow subscribes the view to the new event", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.repoint("CopyBorrowed", "AvailableCopiesView", "PartnerLoanImported", "AvailableCopiesView")
			})

			require.Equal(t, replaced(t, lendingModel, `subscribes = [CopyBorrowed]`, `subscribes = [PartnerLoanImported]`), result)
		})

		t.Run("moving the target end of an automation's command arrow makes it issue the new command", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.repoint("RecallOverdueCopy", "RecallCopy", "RecallOverdueCopy", "BorrowCopy")
			})

			require.Equal(t, replaced(t, lendingModel, `command = RecallCopy`, `command = BorrowCopy`), result)
		})
	})

	t.Run("slice order", func(t *testing.T) {
		t.Run("two slices swapped in an aggregate are declared in the new order, each with its comments", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.swap("Borrow Copy", "Available Copies")
			})

			borrow := lendingModel[strings.Index(lendingModel, "    # A member takes a copy off the shelf\n"):strings.Index(lendingModel, "    # Every copy still on the shelf\n")]
			available := lendingModel[strings.Index(lendingModel, "    # Every copy still on the shelf\n"):strings.Index(lendingModel, `    slice "Recall Overdue Copy" {`)]
			require.Contains(t, borrow, `slice "Borrow Copy" {`)
			require.Contains(t, available, `slice "Available Copies" {`)

			require.Equal(t, replaced(t, lendingModel, borrow+available, available+borrow), result)
		})
	})

	t.Run("moving to another slice", func(t *testing.T) {
		t.Run("a command moved to another slice is declared there after the commands already there, with its comments", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.moveTo("RecallCopy", "Borrow Copy")
			})

			recall := `      # Pull a copy back from a member who kept it too long
      command "RecallCopy" {
        fields {
          loanId = required(string)
        }
      }
`
			expected := replaced(t, lendingModel, recall+"\n", "")
			expected = replaced(t, expected, `          copyId = required(string)
        }
      }

      # A copy left the building
`, `          copyId = required(string)
        }
      }

`+recall+`
      # A copy left the building
`)
			require.Equal(t, expected, result)
		})

		t.Run("the flow line, the rejection and the spec that name a moved command stay in the slice that states them", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.moveTo("BorrowCopy", "Recall Overdue Copy")
			})

			borrow := `      # Ask the library to hand a copy over
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }
`
			expected := replaced(t, lendingModel, borrow+"\n", "")
			expected = replaced(t, expected, `          loanId = required(string)
        }
      }

      # Chases a copy nobody brought back
`, `          loanId = required(string)
        }
      }

`+borrow+`
      # Chases a copy nobody brought back
`)
			require.Contains(t, expected, `      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
        command -> rejected: BorrowCopy -> OneLoanPerCopy
      FLOW

      spec "lends a copy on the shelf" {`)
			require.Equal(t, expected, result)
		})

		t.Run("a translation moved to another slice takes the event it nests with it", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.moveTo("PartnerLoanImport", "Available Copies")
			})

			translation := `    # Restates a partner branch's notice in the library's own language
    translation "PartnerLoanImport" {
      external_system = "Partner Branch API"
      reads           = AvailableCopiesView
      command         = ImportPartnerLoan

      # A partner branch reported a loan
      event "PartnerLoanImported" {
        fields {
          externalRef = required(string)
        }
      }
    }
`
			expected := replaced(t, lendingModel, "\n"+translation, "")
			expected = replaced(t, expected, `          copyId = required(string)
        }
      }
    }

    slice "Recall Overdue Copy" {`, `          copyId = required(string)
        }
      }

`+indented(translation)+`    }

    slice "Recall Overdue Copy" {`)
			require.Equal(t, expected, result)
		})

		t.Run("a trigger moved onto a slice that has one stays where it was", func(t *testing.T) {
			source := `emod = 1

model "M" {
}

context "C" {
  slice "Borrow Copy" {
    trigger "Lending Desk" {
    }
  }

  slice "Return Copy" {
    trigger "Returns Counter" {
    }
  }
}
`
			result := applied(t, source, func(d *viewerDiagram) {
				d.moveTo("Lending Desk", "Return Copy")
			})

			require.Equal(t, source, result)
		})

		t.Run("a block moved onto a node the diagram does not hold stays where it was", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.set("RecallCopy", "parentId", "slice-gone")
			})

			require.Equal(t, lendingModel, result)
		})

		t.Run("a block moved onto something that is not a slice stays where it was", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.moveTo("RecallCopy", "Loan")
			})

			require.Equal(t, lendingModel, result)
		})

		t.Run("a trigger moved onto a slice with no trigger is declared there", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.moveTo("Lending Desk", "Available Copies")
			})

			trigger := `      # The desk terminal the librarian types into
      trigger "Lending Desk" {
        actor = Member
        reads = AvailableCopiesView
      }
`
			expected := replaced(t, lendingModel, trigger+"\n", "")
			expected = replaced(t, expected, `    slice "Available Copies" {
`, `    slice "Available Copies" {
`+trigger+"\n")
			require.Equal(t, expected, result)
		})

		t.Run("the event a translation nests, moved apart from its translation, stays in it", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.moveTo("PartnerLoanImported", "Borrow Copy")
			})

			require.Equal(t, lendingModel, result)
		})
	})

	t.Run("display-only keys", func(t *testing.T) {
		t.Run("a node's position and the keys the importer does not read change nothing", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				for _, n := range d.Nodes {
					n["position"] = map[string]any{"filename": "elsewhere.emod", "line": 1, "column": 1}
					n["offset"] = map[string]any{"dx": 40, "dy": 12}
				}
			})

			require.Equal(t, lendingModel, result)
		})
	})

	t.Run("documentation", func(t *testing.T) {
		t.Run("the architecture page names both ways the importer works", func(t *testing.T) {
			page, err := os.ReadFile("../../docs/architecture.md")
			require.NoError(t, err)
			paragraph := strings.Split(strings.Split(string(page), "`internal/importer` is the inverse")[1], "## Frontends")[0]
			words := strings.Join(strings.Fields(paragraph), " ")

			require.Contains(t, words, "`ImportDiagram`")
			require.Contains(t, words, "`ApplyDiagram` applies a diagram edit onto the model the diagram was drawn from")
		})
	})

	t.Run("malformed input", func(t *testing.T) {
		t.Run("invalid diagram JSON is refused and leaves the model as it was", func(t *testing.T) {
			model := parseModel(t, lendingModel)

			err := importer.ApplyDiagram(model, []byte("{not json"))

			require.ErrorContains(t, err, "invalid diagram JSON")
			require.Equal(t, lendingModel, formatter.Format(model))
		})

		t.Run("null nodes, nodes with no id and null edges are skipped", func(t *testing.T) {
			result := applied(t, lendingModel, func(d *viewerDiagram) {
				d.Nodes = append(d.Nodes, nil, map[string]any{"type": "command", "label": "Stray", "parentId": d.idOf("Borrow Copy")})
				d.Edges = append(d.Edges, nil)
			})

			require.Equal(t, lendingModel, result)
		})

		t.Run("a model that was not parsed from source is refused, because its drawing has no positions to match", func(t *testing.T) {
			model := importFrom(t, lendingModel)
			before := formatter.Format(model)

			err := importer.ApplyDiagram(model, []byte(exportedDiagram(t, model)))

			require.ErrorContains(t, err, "carries no position")
			require.Equal(t, before, formatter.Format(model))
		})
	})
}

// indented answers text with every line that holds anything indented two more
// columns, which is how a context's block reads once it sits in an aggregate's
// slice.
func indented(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "  " + line
		}
	}
	return strings.Join(lines, "\n")
}

func readExample(t *testing.T, name string) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("../../examples", name))
	require.NoError(t, err)
	return string(source)
}
