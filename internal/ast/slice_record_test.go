//go:build unit

package ast_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/ast"
)

func TestSliceRecord(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		t.Run("gives each slice its pattern and its key elements, in source order", func(t *testing.T) {
			model := parseModel(t, `emod = 1

model "Hotel" {
}

context "Stays" {
  aggregate "Room" {
    slice "Reserve Room" {
      trigger "Booking Screen" {
      }

      command "HoldRoom" {
        fields {
          roomId = required(string)
        }
      }

      command "ReserveRoom" {
        fields {
          roomId = required(string)
        }
      }

      event "RoomHeld" {
        fields {
          roomId = required(string)
        }
      }

      event "RoomReserved" {
        fields {
          roomId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    ReserveRoom -> RoomReserved
      FLOW
    }

    slice "Show Rooms" {
      view "AvailableRoomsView" {
        fields {
          roomId = required(string)
        }
      }
    }

    slice "Confirm Stay" {
      command "SendConfirmation" {
        fields {
          roomId = required(string)
        }
      }

      automation "Confirmer" {
        on      = RoomReserved
        command = SendConfirmation
      }
    }

    slice "Import Booking" {
      command "ImportBooking" {
        fields {
          roomId = required(string)
        }
      }

      translation "PartnerImport" {
        external_system = "Partner"
        command         = ImportBooking

        event "BookingImported" {
          fields {
            roomId = required(string)
          }
        }
      }
    }
  }
}
`)

			require.Equal(t, []ast.SliceRecord{
				{Name: "Reserve Room", Pattern: "command", Context: "Stays", KeyElements: "ReserveRoom, RoomReserved"},
				{Name: "Show Rooms", Pattern: "view", Context: "Stays", KeyElements: "AvailableRoomsView"},
				{Name: "Confirm Stay", Pattern: "automation", Context: "Stays", KeyElements: "RoomReserved, SendConfirmation"},
				{Name: "Import Booking", Pattern: "translation", Context: "Stays", KeyElements: "ImportBooking, BookingImported"},
			}, ast.SliceRecords(model))
		})

		t.Run("makes a slice a command slice only when it has a trigger, a command, and an event or a flow", func(t *testing.T) {
			model := parseModel(t, `emod = 1

model "Hotel" {
}

context "Stays" {
  aggregate "Room" {
    slice "With Event" {
      trigger "Screen" {
      }

      command "HoldRoom" {
        fields {
          roomId = required(string)
        }
      }

      event "RoomHeld" {
        fields {
          roomId = required(string)
        }
      }
    }

    slice "With Flow" {
      trigger "Screen" {
      }

      command "ReleaseRoom" {
        fields {
          roomId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    ReleaseRoom -> RoomReleased
      FLOW
    }

    slice "No Trigger" {
      command "CleanRoom" {
        fields {
          roomId = required(string)
        }
      }

      event "RoomCleaned" {
        fields {
          roomId = required(string)
        }
      }
    }

    slice "No Command" {
      trigger "Screen" {
      }

      event "RoomInspected" {
        fields {
          roomId = required(string)
        }
      }
    }

    slice "No Event Or Flow" {
      trigger "Screen" {
      }

      command "PaintRoom" {
        fields {
          roomId = required(string)
        }
      }
    }

    slice "Flow Without Trigger" {
      command "LockRoom" {
        fields {
          roomId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    LockRoom -> RoomLocked
      FLOW
    }

    slice "Flow Without Command" {
      trigger "Screen" {
      }

      flow = <<-FLOW
        command -> event:    UnlockRoom -> RoomUnlocked
      FLOW
    }
  }
}
`)

			require.Equal(t, []ast.SliceRecord{
				{Name: "With Event", Pattern: "command", Context: "Stays", KeyElements: "HoldRoom, RoomHeld"},
				{Name: "With Flow", Pattern: "command", Context: "Stays", KeyElements: "ReleaseRoom, RoomReleased"},
				{Name: "No Trigger", Pattern: "unknown", Context: "Stays", KeyElements: ""},
				{Name: "No Command", Pattern: "unknown", Context: "Stays", KeyElements: ""},
				{Name: "No Event Or Flow", Pattern: "unknown", Context: "Stays", KeyElements: ""},
				{Name: "Flow Without Trigger", Pattern: "unknown", Context: "Stays", KeyElements: ""},
				{Name: "Flow Without Command", Pattern: "unknown", Context: "Stays", KeyElements: ""},
			}, ast.SliceRecords(model))
		})

		t.Run("names only the part that has a name when a command slice or a translation lacks the other", func(t *testing.T) {
			model := parseModel(t, `emod = 1

model "Hotel" {
}

context "Stays" {
  aggregate "Room" {
    slice "Unnamed Event" {
      trigger "Screen" {
      }

      command "HoldRoom" {
        fields {
          roomId = required(string)
        }
      }

      event "" {
        fields {
          roomId = required(string)
        }
      }
    }

    slice "Unnamed Command" {
      trigger "Screen" {
      }

      command "" {
        fields {
          roomId = required(string)
        }
      }

      event "RoomHeld" {
        fields {
          roomId = required(string)
        }
      }
    }

    slice "Translation Without Event" {
      command "ImportBooking" {
        fields {
          roomId = required(string)
        }
      }

      translation "PartnerImport" {
        external_system = "Partner"
        command         = ImportBooking
      }
    }

    slice "Translation Without Command" {
      translation "PartnerListener" {
        external_system = "Partner"

        event "PartnerBooked" {
          fields {
            roomId = required(string)
          }
        }
      }
    }
  }
}
`)

			require.Equal(t, []ast.SliceRecord{
				{Name: "Unnamed Event", Pattern: "command", Context: "Stays", KeyElements: "HoldRoom"},
				{Name: "Unnamed Command", Pattern: "command", Context: "Stays", KeyElements: "RoomHeld"},
				{Name: "Translation Without Event", Pattern: "translation", Context: "Stays", KeyElements: "ImportBooking"},
				{Name: "Translation Without Command", Pattern: "translation", Context: "Stays", KeyElements: "PartnerBooked"},
			}, ast.SliceRecords(model))
		})

		t.Run("names the activation of an automation: its schedule, its event, or its event and its delay", func(t *testing.T) {
			model := parseModel(t, `emod = 1

model "Hotel" {
}

context "Stays" {
  aggregate "Room" {
    slice "Sweep Holds" {
      command "SweepHolds" {
        fields {
          roomId = required(string)
        }
      }

      automation "HoldSweeper" {
        every   = "15m"
        command = SweepHolds
      }
    }

    slice "Confirm Stay" {
      command "SendConfirmation" {
        fields {
          roomId = required(string)
        }
      }

      automation "Confirmer" {
        on      = RoomReserved
        command = SendConfirmation
      }
    }

    slice "Expire Hold" {
      command "ExpireHold" {
        fields {
          roomId = required(string)
        }
      }

      automation "HoldExpirer" {
        on      = RoomHeld
        after   = "24h"
        command = ExpireHold
      }
    }
  }
}
`)

			require.Equal(t, []ast.SliceRecord{
				{Name: "Sweep Holds", Pattern: "automation", Context: "Stays", KeyElements: `every "15m", SweepHolds`},
				{Name: "Confirm Stay", Pattern: "automation", Context: "Stays", KeyElements: "RoomReserved, SendConfirmation"},
				{Name: "Expire Hold", Pattern: "automation", Context: "Stays", KeyElements: `RoomHeld after "24h", ExpireHold`},
			}, ast.SliceRecords(model))
		})

		t.Run("gives an automation with a delay and no event an empty activation", func(t *testing.T) {
			model := parseModel(t, `emod = 1

model "Hotel" {
}

context "Stays" {
  aggregate "Room" {
    slice "Release Hold" {
      command "ReleaseHold" {
        fields {
          holdId = required(string)
        }
      }

      automation "HoldReleaser" {
        after   = "24h"
        command = ReleaseHold
      }
    }
  }
}
`)

			require.Equal(t, []ast.SliceRecord{
				{Name: "Release Hold", Pattern: "automation", Context: "Stays", KeyElements: ", ReleaseHold"},
			}, ast.SliceRecords(model))
		})

		t.Run("gives an empty list, not nil, for a model with no slice", func(t *testing.T) {
			model := parseModel(t, `emod = 1

model "Hotel" {
}
`)

			records := ast.SliceRecords(model)

			require.NotNil(t, records)
			require.Empty(t, records)
		})
	})
}
