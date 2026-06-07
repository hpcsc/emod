# emod

Event Modeling DSL & visualization CLI tool. Write `.emod` files to model event-driven architectures, validate against anti-patterns, and generate diagrams.

## Quick Start

### Install

```bash
go install github.com/hpcsc/emod/cmd/emod@latest
```

Or build from source:

```bash
git clone https://github.com/hpcsc/emod.git
cd emod
go build -o ./bin/emod ./cmd/emod
```

### Write a model

Create `reservation.emod`:

```emod
model "Hotel Reservation"

actor Guest

context Reservations {
  aggregate Reservation {
    slice "Reserve a Room" {
      trigger UI "Reservation Form" {
        actor Guest
        reads AvailableRoomsView
      }

      command ReserveRoom {
        fields {
          roomId    string required
          guestName string required
          checkIn   date   required
          checkOut  date   required
        }
      }

      event RoomReserved {
        fields {
          reservationId string required
          roomId        string required
          guestName     string required
          reservedAt    timestamp required
        }
      }

      flow {
        command -> event: ReserveRoom -> RoomReserved
      }
    }
  }
}
```

### Validate

```bash
emod validate reservation.emod
```

### Lint for anti-patterns

```bash
emod lint reservation.emod
```

### Format

```bash
emod fmt reservation.emod          # format in place
emod fmt --check reservation.emod  # check only (CI)
```

### Generate diagrams

```bash
emod diagram reservation.emod -f drawio   # draw.io XML (default)
emod diagram reservation.emod -f mermaid  # Mermaid markdown
emod diagram reservation.emod -f svg      # standalone SVG
emod diagram reservation.emod -f ascii    # terminal preview
```

### Export

```bash
emod export reservation.emod -f json  # JSON
emod export reservation.emod -f cue   # CUE schema
```

### List slices

```bash
emod slices reservation.emod
```

## Editor Setup

### VS Code — symlink (recommended)

Link the extension directory directly for auto-reload on edit:

```bash
ln -sf "$(pwd)/editors/vscode" ~/.vscode/extensions/emod
```

### VS Code — .vsix package

Build a `.vsix` from `editors/vscode` and install it:

```bash
npx @vscode/vsce package --cwd editors/vscode
code --install-extension emod-*.vsix
```

### JetBrains (GoLand / IntelliJ)

1. Open **Settings → Editor → TextMate Bundles**.
2. Click **+** and browse to `editors/vscode/`.
3. Apply the setting — `.emod` files will be highlighted.

Alternatively, symlink the bundle:

```bash
ln -sf "$(pwd)/editors/vscode" ~/.textmate/bundles/emod
```

## Development

```bash
go test -tags unit ./...  # unit tests
go test -tags unit -count=1 ./...  # bypass cache
```