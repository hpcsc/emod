package diagram

// arrowSpacing is how far apart two arrows run side by side: two legs in one gap
// between lanes, or two stretches up or down.
const arrowSpacing = 6

// The lanes of the picture, from the top.
const (
	wireframesLane = iota
	commandsAndViewsLane
	eventsLane
	externalSystemsLane
)

var laneNames = [...]string{
	wireframesLane:       "Wireframes",
	commandsAndViewsLane: "Commands / Views",
	eventsLane:           "Events",
	externalSystemsLane:  "External Systems",
}

// lanePositions is where the lanes of the picture sit. Each lane is laneHeight
// tall, and the gap under each lane but the last is tall enough for the tracks
// of legs it holds. The zero value holds no tracks.
type lanePositions struct {
	tracks [len(laneNames) - 1]int
}

// withTracksOf returns l with each gap tall enough also for the tracks the legs
// of routes run on.
func (l lanePositions) withTracksOf(routes []route) lanePositions {
	for _, lg := range legsOf(routes) {
		l.tracks[lg.gap] = max(l.tracks[lg.gap], lg.track+1)
	}

	return l
}

func (l lanePositions) top(lane int) int {
	y := marginY
	for gap := range lane {
		y += laneHeight + l.gapHeight(gap)
	}

	return y
}

func (l lanePositions) gapHeight(gap int) int {
	return max(laneGap, (l.tracks[gap]+1)*arrowSpacing)
}

// gapTop is the y gap starts at, under the lane of the same number.
func (l lanePositions) gapTop(gap int) int {
	return l.top(gap) + laneHeight
}

// trackY is the y a leg on track number track of gap runs at. The gaps count
// from the top, so gap 0 lies under the first lane, and the tracks of a gap are
// spread evenly down it.
func (l lanePositions) trackY(gap, track int) int {
	return l.gapTop(gap) + (track+1)*l.gapHeight(gap)/(l.tracks[gap]+1)
}
