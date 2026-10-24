package diagram

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The gear an automation is drawn as in the lane picture and the viewer. Its
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
