package diagram

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The gear an automation is drawn as in the lane pictures and the viewer. Its
// measures are those of the gear eventFlowDefs builds from a circle and eight
// rects, centred on the origin, so the two pictures draw one gear.
const (
	gearTeeth          = 8
	gearBodyRadius     = 8.4
	gearToothHalfWidth = 2.9
	gearTipRadius      = 14.2
	gearCornerRadius   = 1.3
	gearHoleRadius     = 3.4
	// gearExtent is half the side of the square the gear is drawn in.
	gearExtent = 15
)

// gearStep is one step of a gear contour: a move that opens it, a straight line,
// or an arc of the given radius.
type gearStep struct {
	move      bool
	radius    float64
	clockwise bool
	x, y      float64
}

func (s gearStep) arc() bool { return s.radius > 0 }

// gearContours traces the gear's outline clockwise and the hole in its hub the
// other way, so a fill under the nonzero rule leaves the hole open.
func gearContours() [][]gearStep {
	toothAngle := 2 * math.Pi / gearTeeth
	root := -math.Sqrt(gearBodyRadius*gearBodyRadius - gearToothHalfWidth*gearToothHalfWidth)
	shoulder := -(gearTipRadius - gearCornerRadius)
	tipHalfWidth := gearToothHalfWidth - gearCornerRadius

	var outline []gearStep
	for tooth := range gearTeeth {
		turn := func(x, y float64) (float64, float64) {
			angle := float64(tooth) * toothAngle
			return x*math.Cos(angle) - y*math.Sin(angle), x*math.Sin(angle) + y*math.Cos(angle)
		}
		at := func(step gearStep, x, y float64) gearStep {
			step.x, step.y = turn(x, y)
			return step
		}

		if tooth == 0 {
			outline = append(outline, at(gearStep{move: true}, -gearToothHalfWidth, root))
		} else {
			outline = append(outline, at(gearStep{radius: gearBodyRadius, clockwise: true}, -gearToothHalfWidth, root))
		}
		outline = append(outline,
			at(gearStep{}, -gearToothHalfWidth, shoulder),
			at(gearStep{radius: gearCornerRadius, clockwise: true}, -tipHalfWidth, -gearTipRadius),
			at(gearStep{}, tipHalfWidth, -gearTipRadius),
			at(gearStep{radius: gearCornerRadius, clockwise: true}, gearToothHalfWidth, shoulder),
			at(gearStep{}, gearToothHalfWidth, root),
		)
	}
	outline = append(outline, gearStep{radius: gearBodyRadius, clockwise: true, x: outline[0].x, y: outline[0].y})

	hole := []gearStep{
		{move: true, x: 0, y: -gearHoleRadius},
		{radius: gearHoleRadius, x: 0, y: gearHoleRadius},
		{radius: gearHoleRadius, x: 0, y: -gearHoleRadius},
	}

	return [][]gearStep{outline, hole}
}

// gearPathData is the gear as SVG path data, in a square from -gearExtent to
// gearExtent on both axes.
var gearPathData = func() string {
	var commands []string
	for _, contour := range gearContours() {
		for _, step := range contour {
			switch {
			case step.move:
				commands = append(commands, "M "+gearPoint(step.x, step.y))
			case step.arc():
				commands = append(commands, fmt.Sprintf("A %s %s 0 0 %d %s",
					gearNumber(step.radius), gearNumber(step.radius), gearSweep(step), gearPoint(step.x, step.y)))
			default:
				commands = append(commands, "L "+gearPoint(step.x, step.y))
			}
		}
		commands = append(commands, "Z")
	}

	return strings.Join(commands, " ")
}()

// drawioGearShape is the gear as a draw.io shape: a stencil, which draw.io reads
// only deflated and base64-encoded. The stencil fills with the cell's own
// fillColor, so a user can repaint the gear in draw.io.
var drawioGearShape = func() string {
	var stencil strings.Builder
	fmt.Fprintf(&stencil, `<shape name="gear" w="%d" h="%d" aspect="fixed" strokewidth="inherit"><connections/><foreground><path>`,
		2*gearExtent, 2*gearExtent)
	for _, contour := range gearContours() {
		for _, step := range contour {
			x, y := gearNumber(step.x+gearExtent), gearNumber(step.y+gearExtent)
			switch {
			case step.move:
				fmt.Fprintf(&stencil, `<move x="%s" y="%s"/>`, x, y)
			case step.arc():
				r := gearNumber(step.radius)
				fmt.Fprintf(&stencil, `<arc rx="%s" ry="%s" x-axis-rotation="0" large-arc-flag="0" sweep-flag="%d" x="%s" y="%s"/>`,
					r, r, gearSweep(step), x, y)
			default:
				fmt.Fprintf(&stencil, `<line x="%s" y="%s"/>`, x, y)
			}
		}
		stencil.WriteString("<close/>")
	}
	stencil.WriteString("</path><fillstroke/></foreground></shape>")

	var deflated bytes.Buffer
	writer, err := flate.NewWriter(&deflated, flate.BestCompression)
	if err != nil {
		panic(err)
	}
	if _, err := writer.Write([]byte(stencil.String())); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}

	return "stencil(" + base64.StdEncoding.EncodeToString(deflated.Bytes()) + ")"
}()

func gearSweep(step gearStep) int {
	if step.clockwise {
		return 1
	}
	return 0
}

func gearPoint(x, y float64) string {
	return gearNumber(x) + " " + gearNumber(y)
}

func gearNumber(v float64) string {
	text := strconv.FormatFloat(v, 'f', 2, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	if text == "-0" {
		return "0"
	}
	return text
}
