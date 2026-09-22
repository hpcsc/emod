//go:build unit

package arrange_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/arrange"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, source string) *ast.Model {
	t.Helper()
	model, diagnostics := oracle.Parse(source, "test.emod")
	require.Empty(t, diagnostics)
	return model
}

func sliceNames(m *ast.Model) []string {
	var names []string
	for _, ctx := range m.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, s := range agg.Slices {
				names = append(names, s.Name)
			}
		}
		for _, s := range ctx.Slices {
			names = append(names, s.Name)
		}
	}
	return names
}

func TestArrange(t *testing.T) {
	t.Run("moving views", func(t *testing.T) {
		t.Run("a view trailing the events it projects moves ahead of the slice reading it", func(t *testing.T) {
			// BorrowView projects CopyBorrowed and is read by "Return a Copy".
			// Declared last it is read from behind; moved between the two slices
			// it trails the event it projects and leads the trigger reading it,
			// so neither reference points backward.
			model := parse(t, `emod = 1

model "Library" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }

      event "CopyBorrowed" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
      FLOW
    }

    slice "Return a Copy" {
      trigger "Return Desk" {
        actor = Member
        reads = BorrowView
      }

      command "ReturnCopy" {
        fields {
          copyId = required(string)
        }
      }

      event "CopyReturned" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    ReturnCopy -> CopyReturned
      FLOW
    }

    slice "Borrowed Copies" {
      view "BorrowView" {
        subscribes = [CopyBorrowed]

        fields {
          copyId = required(string)
        }
      }
    }
  }
}
`)

			report := arrange.Model(model)

			require.Equal(t, []string{"Borrow a Copy", "Borrowed Copies", "Return a Copy"}, sliceNames(model))
			require.Equal(t, 1, report.Moved)
			require.Equal(t, 1, report.BackwardBefore)
			require.Equal(t, 0, report.BackwardAfter)
			require.Empty(t, report.Backward)
		})

		t.Run("a view read by nothing settles after the events it projects", func(t *testing.T) {
			model := parse(t, `emod = 1

model "Library" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Export Loans" {
      view "LoanExportView" {
        subscribes = [CopyBorrowed]

        fields {
          copyId = required(string)
        }
      }
    }

    slice "Borrow a Copy" {
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }

      event "CopyBorrowed" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
      FLOW
    }
  }
}
`)

			report := arrange.Model(model)

			require.Equal(t, []string{"Borrow a Copy", "Export Loans"}, sliceNames(model))
			require.Equal(t, 0, report.BackwardAfter)
		})

		t.Run("a slice declaring a view beside a command stays where its author put it", func(t *testing.T) {
			// Moving "Shelve and Report" between the other two would leave every
			// reference pointing forward, so the arrangement is available and
			// declined: the slice also writes, which makes it a step of the
			// process rather than a projection, and steps do not move.
			model := parse(t, `emod = 1

model "Library" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Open the Shelf" {
      command "OpenShelf" {
        fields {
          shelfId = required(string)
        }
      }

      event "ShelfOpened" {
        fields {
          shelfId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    OpenShelf -> ShelfOpened
      FLOW
    }

    slice "Borrow a Copy" {
      trigger "Borrow Desk" {
        actor = Member
        reads = ShelfView
      }

      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }

      event "CopyBorrowed" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
      FLOW
    }

    slice "Shelve and Report" {
      command "ShelveCopy" {
        fields {
          copyId = required(string)
        }
      }

      view "ShelfView" {
        subscribes = [ShelfOpened]

        fields {
          shelfId = required(string)
        }
      }
    }
  }
}
`)

			report := arrange.Model(model)

			require.Equal(t, []string{"Open the Shelf", "Borrow a Copy", "Shelve and Report"}, sliceNames(model))
			require.Equal(t, 0, report.Moved)
			require.Equal(t, 1, report.BackwardAfter)
		})
	})

	t.Run("leaving the process alone", func(t *testing.T) {
		t.Run("process slices keep their authored order even when reversing them would read forward", func(t *testing.T) {
			// Two slices produce CopyBorrowed, so whichever declares the event
			// the other one points back at it. Both write, so neither is free to
			// move and the reference stays — the shape a shared event always
			// takes, not something an ordering can undo.
			model := parse(t, `emod = 1

model "Library" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }

      event "CopyBorrowed" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
      FLOW
    }

    slice "Renew a Copy" {
      command "RenewCopy" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    RenewCopy -> CopyBorrowed
      FLOW
    }
  }
}
`)

			report := arrange.Model(model)

			require.Equal(t, []string{"Borrow a Copy", "Renew a Copy"}, sliceNames(model))
			require.Equal(t, 0, report.Moved)
			require.Equal(t, 1, report.BackwardAfter)
			require.Len(t, report.Backward, 1)
			require.Equal(t, arrange.KindFlow, report.Backward[0].Kind)
			require.Equal(t, "RenewCopy -> CopyBorrowed", report.Backward[0].Label)
		})

		t.Run("a reference into another context is left out, being unfixable by order", func(t *testing.T) {
			model := parse(t, `emod = 1

model "Library" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }

      event "CopyBorrowed" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
      FLOW
    }
  }
}

context "Notifications" {
  aggregate "Notice" {
    slice "Announce the Borrow" {
      command "SendNotice" {
        fields {
          copyId = required(string)
        }
      }

      automation "AnnounceOnBorrow" {
        on      = CopyBorrowed
        command = SendNotice
      }
    }
  }
}
`)

			report := arrange.Model(model)

			require.Equal(t, 0, report.BackwardBefore)
			require.Equal(t, 0, report.BackwardAfter)
			require.Empty(t, report.Backward)
		})
	})

	t.Run("stability", func(t *testing.T) {
		t.Run("arranging an arranged model changes nothing", func(t *testing.T) {
			source := `emod = 1

model "Library" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }

      event "CopyBorrowed" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    BorrowCopy -> CopyBorrowed
      FLOW
    }

    slice "Borrowed Copies" {
      view "BorrowView" {
        subscribes = [CopyBorrowed]

        fields {
          copyId = required(string)
        }
      }
    }

    slice "Return a Copy" {
      trigger "Return Desk" {
        actor = Member
        reads = BorrowView
      }

      command "ReturnCopy" {
        fields {
          copyId = required(string)
        }
      }

      event "CopyReturned" {
        fields {
          copyId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    ReturnCopy -> CopyReturned
      FLOW
    }
  }
}
`
			first := parse(t, source)
			firstReport := arrange.Model(first)
			require.Equal(t, 0, firstReport.Moved)
			require.False(t, firstReport.Changed())

			second := parse(t, source)
			arrange.Model(second)
			require.Equal(t, sliceNames(first), sliceNames(second))
		})

		t.Run("a model with no slices to weigh is left alone", func(t *testing.T) {
			model := parse(t, `emod = 1

model "Library" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command "BorrowCopy" {
        fields {
          copyId = required(string)
        }
      }
    }
  }
}
`)

			report := arrange.Model(model)

			require.Equal(t, []string{"Borrow a Copy"}, sliceNames(model))
			require.False(t, report.Changed())
		})
	})
}
