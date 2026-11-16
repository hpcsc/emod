package diagram

import (
	"slices"
	"sort"
)

// legSpacing is how far apart two legs run in one gap between lanes.
const legSpacing = 6

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

// lanesFor returns lanes whose gaps hold the tracks of onTracks.
func lanesFor(onTracks map[int]gapTrack) lanePositions {
	var l lanePositions
	for _, t := range onTracks {
		l.tracks[t.gap] = max(l.tracks[t.gap], t.track+1)
	}

	return l
}

// top is the y that lane starts at.
func (l lanePositions) top(lane int) int {
	y := marginY
	for gap := range lane {
		y += laneHeight + l.gapHeight(gap)
	}

	return y
}

// gapHeight is how tall a gap is.
func (l lanePositions) gapHeight(gap int) int {
	return max(laneGap, (l.tracks[gap]+1)*legSpacing)
}

// trackY is the y a leg on track t runs at. The tracks of a gap are spread
// evenly down it.
func (l lanePositions) trackY(t gapTrack) int {
	gapTop := l.top(t.gap) + laneHeight

	return gapTop + (t.track+1)*l.gapHeight(t.gap)/(l.tracks[t.gap]+1)
}

// gapTrack is a track of one gap between two lanes. The gaps count from the top,
// so gap 0 lies under the first lane.
type gapTrack struct {
	gap, track int
}

// legTracks gives each arrow whose leg runs between two lanes a track of the gap
// under the upper of the two lanes, where no other leg in that gap overlaps its
// span. An arrow inside one lane, or one that runs straight up or down, gets no
// track.
func legTracks(arrows []boxArrow) map[int]gapTrack {
	spans := make([][2]int, len(arrows))
	inGap := make([][]int, len(laneNames)-1)
	for i, a := range arrows {
		left, right, ok := a.route().legSpan()
		if !ok || a.from.lane == a.to.lane {
			continue
		}
		gap := min(a.from.lane, a.to.lane)
		spans[i] = [2]int{left, right}
		inGap[gap] = append(inGap[gap], i)
	}

	onTracks := make(map[int]gapTrack)
	for gap, arrowsInGap := range inGap {
		sort.SliceStable(arrowsInGap, func(a, b int) bool {
			return spans[arrowsInGap[a]][0] < spans[arrowsInGap[b]][0]
		})
		var trackEnds []int
		for _, i := range arrowsInGap {
			left, right := spans[i][0], spans[i][1]
			track := slices.IndexFunc(trackEnds, func(end int) bool { return end < left })
			if track < 0 {
				track = len(trackEnds)
				trackEnds = append(trackEnds, right)
			}
			trackEnds[track] = right
			onTracks[i] = gapTrack{gap: gap, track: track}
		}
	}

	return onTracks
}
