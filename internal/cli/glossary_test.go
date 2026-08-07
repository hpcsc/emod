//go:build unit

package cli_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
	urfave "github.com/urfave/cli/v2"
)

// The app terminates the process on an exit-coded error, which would take the
// test binary with it, so the handler is replaced with one that lets Run return.
func runCommandLine(t *testing.T, args ...string) error {
	t.Helper()
	app := cli.NewApp()
	app.ExitErrHandler = func(*urfave.Context, error) {}
	return cli.RunApp(app, args)
}

func TestGlossary(t *testing.T) {
	t.Run("markdown", func(t *testing.T) {
		t.Run("names every term of a model that describes none of them, each with an empty definition", func(t *testing.T) {
			path := writeTemp(t, "undescribed.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunGlossary(path, "markdown"))
			})

			require.Equal(t, `# Hotel Reservation

## Reservations

### Reservation

### Commands

#### MakeReservation

#### ConfirmReservation

#### ImportBooking

### Events

#### ReservationMade

#### BookingImported

### Views

#### ReservationsView

### Actors

#### Guest
`, output)
		})

		t.Run("pairs every term with the description the model declares for it", func(t *testing.T) {
			path := writeTemp(t, "described.emod", describedEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunGlossary(path, "markdown"))
			})

			require.Equal(t, `# Hotel Reservation

How the hotel takes, confirms and imports room bookings

## Reservations

Everything the hotel knows about a stay before the guest arrives

### Reservation

One guest holding one room over one date range

### Commands

#### MakeReservation

Ask the hotel to hold a room for a date range, 10% deposit taken up front

#### ConfirmReservation

Turn a held room into a confirmed stay

#### ImportBooking

Record a booking taken by a partner site

### Events

#### ReservationMade

A room is held for a guest

#### BookingImported

A partner site reported a booking

### Views

#### ReservationsView

Every reservation with the stage it has reached

### Actors

#### Guest

A person booking a room, not necessarily the one staying in it
`, output)
		})

		t.Run("lists the terms of a dcb context whose slices hang off the context itself", func(t *testing.T) {
			path := writeTemp(t, "dcb.emod", singleTagDCBEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunGlossary(path, "markdown"))
			})

			require.Equal(t, `# Orders

## Fulfillment

### Commands

#### PlaceOrder

#### AuthorizePayment

### Events

#### OrderPlaced

#### PaymentAuthorized
`, output)
		})

		t.Run("lists every invariant beneath the aggregate or the context that declares it", func(t *testing.T) {
			path := writeTemp(t, "invariants.emod", invariantEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunGlossary(path, "markdown"))
			})

			require.Equal(t, `# Library Lending

## Lending

### Loan

#### Invariants

##### OneCopyPerLoan

A loan covers exactly one copy of one title

##### FiveCopiesPerMember

A member holds at most five copies at one time

### Commands

#### BorrowCopy

### Events

#### CopyBorrowed

### Views

#### MemberLoansView

### Actors

#### Member

## Reading Room

### Invariants

#### OneReaderPerDesk

A desk seats at most one reader at any moment

#### OneDeskPerReader

A reader holds at most one desk for the length of a session

#### DeskFreeAtClosing

No desk stays claimed past the closing hour

### Commands

#### ClaimDesk

#### ReleaseDesk

### Events

#### DeskClaimed

#### DeskReleased
`, output)
		})
	})

	t.Run("json", func(t *testing.T) {
		t.Run("pairs every term with the description the model declares for it, ordered as the markdown lists them", func(t *testing.T) {
			path := writeTemp(t, "described.emod", describedEmod)

			var err error
			stdout := captureStdout(t, func() {
				err = runCommandLine(t, "emod", "glossary", path, "-f", "json")
			})
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(stdout), &doc))

			require.Equal(t, map[string]any{
				"name":        "Hotel Reservation",
				"description": "How the hotel takes, confirms and imports room bookings",
				"contexts": []any{map[string]any{
					"name":        "Reservations",
					"description": "Everything the hotel knows about a stay before the guest arrives",
					"aggregates": []any{
						map[string]any{"name": "Reservation", "description": "One guest holding one room over one date range"},
					},
					"commands": []any{
						map[string]any{"name": "MakeReservation", "description": "Ask the hotel to hold a room for a date range, 10% deposit taken up front"},
						map[string]any{"name": "ConfirmReservation", "description": "Turn a held room into a confirmed stay"},
						map[string]any{"name": "ImportBooking", "description": "Record a booking taken by a partner site"},
					},
					"events": []any{
						map[string]any{"name": "ReservationMade", "description": "A room is held for a guest"},
						map[string]any{"name": "BookingImported", "description": "A partner site reported a booking"},
					},
					"views": []any{
						map[string]any{"name": "ReservationsView", "description": "Every reservation with the stage it has reached"},
					},
					"actors": []any{
						map[string]any{"name": "Guest", "description": "A person booking a room, not necessarily the one staying in it"},
					},
				}},
			}, doc)
		})

		t.Run("keeps a description key on every term of a model that describes none of them", func(t *testing.T) {
			path := writeTemp(t, "undescribed.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunGlossary(path, "json"))
			})

			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &doc))

			require.Equal(t, map[string]any{
				"name":        "Hotel Reservation",
				"description": "",
				"contexts": []any{map[string]any{
					"name":        "Reservations",
					"description": "",
					"aggregates": []any{
						map[string]any{"name": "Reservation", "description": ""},
					},
					"commands": []any{
						map[string]any{"name": "MakeReservation", "description": ""},
						map[string]any{"name": "ConfirmReservation", "description": ""},
						map[string]any{"name": "ImportBooking", "description": ""},
					},
					"events": []any{
						map[string]any{"name": "ReservationMade", "description": ""},
						map[string]any{"name": "BookingImported", "description": ""},
					},
					"views": []any{
						map[string]any{"name": "ReservationsView", "description": ""},
					},
					"actors": []any{
						map[string]any{"name": "Guest", "description": ""},
					},
				}},
			}, doc)
		})

		t.Run("carries every invariant under the aggregate or the context that declares it", func(t *testing.T) {
			path := writeTemp(t, "invariants.emod", invariantEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunGlossary(path, "json"))
			})

			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &doc))

			require.Equal(t, map[string]any{
				"name":        "Library Lending",
				"description": "",
				"contexts": []any{
					map[string]any{
						"name":        "Lending",
						"description": "",
						"aggregates": []any{map[string]any{
							"name":        "Loan",
							"description": "",
							"invariants": []any{
								map[string]any{"name": "OneCopyPerLoan", "description": "A loan covers exactly one copy of one title"},
								map[string]any{"name": "FiveCopiesPerMember", "description": "A member holds at most five copies at one time"},
							},
						}},
						"commands": []any{
							map[string]any{"name": "BorrowCopy", "description": ""},
						},
						"events": []any{
							map[string]any{"name": "CopyBorrowed", "description": ""},
						},
						"views": []any{
							map[string]any{"name": "MemberLoansView", "description": ""},
						},
						"actors": []any{
							map[string]any{"name": "Member", "description": ""},
						},
					},
					map[string]any{
						"name":        "Reading Room",
						"description": "",
						"invariants": []any{
							map[string]any{"name": "OneReaderPerDesk", "description": "A desk seats at most one reader at any moment"},
							map[string]any{"name": "OneDeskPerReader", "description": "A reader holds at most one desk for the length of a session"},
							map[string]any{"name": "DeskFreeAtClosing", "description": "No desk stays claimed past the closing hour"},
						},
						"commands": []any{
							map[string]any{"name": "ClaimDesk", "description": ""},
							map[string]any{"name": "ReleaseDesk", "description": ""},
						},
						"events": []any{
							map[string]any{"name": "DeskClaimed", "description": ""},
							map[string]any{"name": "DeskReleased", "description": ""},
						},
					},
				},
			}, doc)
		})
	})

	t.Run("rejected input", func(t *testing.T) {
		t.Run("missing file argument names the cause callers branch on", func(t *testing.T) {
			err := cli.RunGlossary("", "markdown")

			require.ErrorIs(t, err, cli.ErrMissingFileArgument)
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("unreadable path is reported with the path the user supplied", func(t *testing.T) {
			missing := "/tmp/nonexistent-emod-glossary-file-abc123.emod"

			err := cli.RunGlossary(missing, "markdown")

			require.Error(t, err)
			require.Contains(t, err.Error(), missing)
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("unparseable file reports the rejected token and the expected keywords instead of a glossary", func(t *testing.T) {
			path := writeTemp(t, "invalid.emod", invalidEmod)

			var err error
			output := captureStdout(t, func() {
				err = cli.RunGlossary(path, "markdown")
			})

			require.Empty(t, output)
			require.Error(t, err)
			require.Contains(t, err.Error(), `"foobar"`)
			require.Contains(t, err.Error(), "model")
			require.Contains(t, err.Error(), "actor")
			require.Contains(t, err.Error(), "context")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("a format the command does not render is rejected by name", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			err := cli.RunGlossary(path, "xml")

			require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
			require.Contains(t, err.Error(), "markdown")
			require.Contains(t, err.Error(), "json")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})
	})

	t.Run("command line", func(t *testing.T) {
		t.Run("rejects a short-form format written after the file argument", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			var err error
			var stdout string
			stderr := captureStderr(t, func() {
				stdout = captureStdout(t, func() {
					err = runCommandLine(t, "emod", "glossary", path, "-f", "xml")
				})
			})

			require.Error(t, err)
			var exitErr urfave.ExitCoder
			require.True(t, errors.As(err, &exitErr))
			require.Equal(t, 1, exitErr.ExitCode())
			require.Contains(t, stderr, "xml")
			require.Contains(t, stderr, "markdown")
			require.NotContains(t, stdout, "Hotel Reservation")
		})

		t.Run("renders when the long-form format is written before the file argument", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			var err error
			stdout := captureStdout(t, func() {
				err = runCommandLine(t, "emod", "glossary", "--format", "markdown", path)
			})

			require.NoError(t, err)
			require.Contains(t, stdout, "Hotel Reservation")
			require.Contains(t, stdout, "Reservations")
		})

		t.Run("lists glossary among the commands the help offers", func(t *testing.T) {
			var err error
			stdout := captureStdout(t, func() {
				err = runCommandLine(t, "emod", "--help")
			})

			require.NoError(t, err)
			require.Contains(t, stdout, "glossary")
		})
	})
}
