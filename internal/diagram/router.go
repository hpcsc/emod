package diagram

import (
	"slices"
	"sort"
)

// stretchClearance is how far a stretch of an arrow keeps from a box it does not
// join.
const stretchClearance = 3

type yRange struct {
	top, bottom int
}

// overlaps reports whether r and o share more than one y, so that a range that
// only touches another at an end does not overlap it.
func (r yRange) overlaps(o yRange) bool {
	return r.top < o.bottom && o.top < r.bottom
}

// meets reports whether r and o share any y, an end included.
func (r yRange) meets(o yRange) bool {
	return r.top <= o.bottom && o.top <= r.bottom
}

// stretch is a part of an arrow that runs up or down at x, over ys.
type stretch struct {
	x  int
	ys yRange
}

// keepsClearOf reports whether s keeps stretchClearance from box. A stretch that
// ends on the side of a box only touches it, which is how an arrow keeps clear of
// the boxes it joins.
func (s stretch) keepsClearOf(box svgBox) bool {
	return s.x < box.x-stretchClearance || s.x > box.x+box.w+stretchClearance ||
		!s.ys.overlaps(yRangeOf(box.y, box.y+box.h))
}

// keepsApartFrom reports whether s keeps arrowSpacing from o. Two stretches at
// one x that only meet end to end would draw as one line, so they do not keep
// apart.
func (s stretch) keepsApartFrom(o stretch) bool {
	return abs(s.x-o.x) >= arrowSpacing || !s.ys.meets(o.ys)
}

// room is the x from left to right that a stretch can take, with best the x it
// is best placed at: across the side of a box, across the x two boxes share, or
// beside a slice.
type room struct {
	best, left, right int
}

// shared is the x r and o share, best at its middle.
func (r room) shared(o room) room {
	both := room{left: max(r.left, o.left), right: min(r.right, o.right)}
	both.best = (both.left + both.right) / 2

	return both
}

// router takes the x of each stretch up or down of the arrows of one picture, so
// that every stretch keeps clear of the boxes and keeps arrowSpacing from every
// stretch taken before it.
type router struct {
	lanes     lanePositions
	obstacles []svgBox
	rooms     []room
	taken     []stretch
}

// takeStraightRoute takes the route of arrow a straight up or down, at the free x
// nearest the middle of the x its two boxes share. It reports false when no
// such x is free, and for an arrow that carries text, which a leg carries.
func (r *router) takeStraightRoute(a boxArrow) (route, bool) {
	if a.label != "" {
		return route{}, false
	}
	from, to := a.sides()
	x, found := r.takeX(a.from.across().shared(a.to.across()), yRangeOf(from.y, to.y))

	return route{from: point{x, from.y}, to: point{x, to.y}}, found
}

// takeBentRoute takes the route of arrow a with its ends nearest the x best
// gives them: a leg in the gap under the lane of an arrow inside one lane, a leg
// in the gap between two lanes next to each other, or a leg in each of the two
// gaps an arrow crosses a lane between, joined beside a slice.
func (r *router) takeBentRoute(a boxArrow, best bestXs) route {
	from, to := a.bentSides()
	rt := route{from: point{y: from.y}, to: point{y: to.y}}
	rt.from.x = r.takeXOrBest(from.room(best.from), r.reachIntoGap(from))
	rt.to.x = r.takeXOrBest(to.room(best.to), r.reachIntoGap(to))

	upper, lower := min(a.from.lane, a.to.lane), max(a.from.lane, a.to.lane)
	if lower-upper <= 1 {
		rt.legs = []leg{{fromX: rt.from.x, toX: rt.to.x, gap: upper}}
		return rt
	}

	near, far := upper, lower-1
	if !a.goesDown() {
		near, far = far, near
	}
	x := r.takeXBesideSlice(rt.to.x, yRangeOf(r.lanes.gapTop(upper), r.lanes.top(lower)))
	rt.legs = []leg{{fromX: rt.from.x, toX: x, gap: near}, {fromX: x, toX: rt.to.x, gap: far}}

	return rt
}

// reachIntoGap is the y a stretch from side runs over: from the side out to the
// far edge of the gap beside the side's lane.
func (r *router) reachIntoGap(side boxSide) yRange {
	if side.isBottom() {
		return yRangeOf(side.y, r.lanes.top(side.box.lane+1))
	}

	return yRangeOf(r.lanes.gapTop(side.box.lane-1), side.y)
}

// takeX takes the x of in nearest in.best where a stretch over ys is free. It
// reports false when no x of in is free.
func (r *router) takeX(in room, ys yRange) (int, bool) {
	for step := 0; in.best-step >= in.left || in.best+step <= in.right; step++ {
		for _, x := range []int{in.best - step, in.best + step} {
			if s := (stretch{x, ys}); x >= in.left && x <= in.right && r.isFree(s) {
				r.taken = append(r.taken, s)
				return x, true
			}
		}
	}

	return 0, false
}

// takeXOrBest takes the x of in nearest in.best where a stretch over ys is free,
// or in.best itself when no x of in is free.
func (r *router) takeXOrBest(in room, ys yRange) int {
	if x, found := r.takeX(in, ys); found {
		return x
	}
	r.taken = append(r.taken, stretch{in.best, ys})

	return in.best
}

// takeXBesideSlice takes the x, in the room beside a slice nearest x, where a
// stretch over ys runs up or down between the two legs of an arrow, or the best
// x of that room when no room has a free x.
func (r *router) takeXBesideSlice(x int, ys yRange) int {
	rooms := slices.Clone(r.rooms)
	sort.SliceStable(rooms, func(a, b int) bool { return abs(rooms[a].best-x) < abs(rooms[b].best-x) })
	for _, in := range rooms {
		if taken, found := r.takeX(in, ys); found {
			return taken
		}
	}
	r.taken = append(r.taken, stretch{rooms[0].best, ys})

	return rooms[0].best
}

// isFree reports whether s keeps clear of every box and apart from every stretch
// taken before it.
func (r *router) isFree(s stretch) bool {
	for _, o := range r.obstacles {
		if !s.keepsClearOf(o) {
			return false
		}
	}
	for _, t := range r.taken {
		if !s.keepsApartFrom(t) {
			return false
		}
	}

	return true
}

// boxArrow is an arrow the picture draws from one box to another.
type boxArrow struct {
	from, to svgBox
	dashed   bool
	label    string
}

func (a boxArrow) goesDown() bool {
	return a.from.y < a.to.y
}

// sides are the side of its from box that a leaves and the side of its to box
// that it enters: the two sides that face each other.
func (a boxArrow) sides() (from, to boxSide) {
	if a.goesDown() {
		return boxSide{a.from, a.from.y + a.from.h}, boxSide{a.to, a.to.y}
	}

	return boxSide{a.from, a.from.y}, boxSide{a.to, a.to.y + a.to.h}
}

// bentSides are the sides a that does not run straight leaves and enters. An
// arrow between two rows of one lane has no gap between its two boxes, so it
// leaves and enters both through their bottom sides, with its leg in the gap
// under the lane.
func (a boxArrow) bentSides() (from, to boxSide) {
	if a.from.lane == a.to.lane {
		return boxSide{a.from, a.from.y + a.from.h}, boxSide{a.to, a.to.y + a.to.h}
	}

	return a.sides()
}

type point struct {
	x, y int
}

// boxSide is the top or the bottom side of box: the one that lies at y.
type boxSide struct {
	box svgBox
	y   int
}

func (s boxSide) isBottom() bool {
	return s.y > s.box.y
}

// room is the x across s that a stretch can take, with best the x it is best
// placed at.
func (s boxSide) room(best int) room {
	across := s.box.across()
	across.best = best

	return across
}

// route is the way an arrow runs from one box to another: out of its from box at
// from, across each of its legs in turn, and into its to box at to. A route with
// no leg runs straight up or down.
type route struct {
	from, to point
	legs     []leg
}

func (r route) points() []point {
	points := []point{r.from}
	for _, l := range r.legs {
		points = append(points, point{l.fromX, l.y}, point{l.toX, l.y})
	}

	return append(points, r.to)
}

// leg is a part of a route that runs across at y, from fromX to toX, on a track
// of the gap between two lanes it runs in, which gives its y.
type leg struct {
	fromX, toX, y int
	gap, track    int
}

func (l leg) span() (left, right int) {
	return min(l.fromX, l.toX), max(l.fromX, l.toX)
}

// bestXs are the x the two ends of one bent arrow are best placed at: the end at
// its from box and the end at its to box. The router moves an end off its best x
// when another stretch holds it.
type bestXs struct {
	from, to int
}

func yRangeOf(a, b int) yRange {
	return yRange{min(a, b), max(a, b)}
}

// routesFor routes every arrow of a picture whose boxes are obstacles, with its
// slices in columns and its lanes where lanes says. Every arrow that can run
// straight up or down takes its x first, so that the ends of the other arrows
// make room for it. Each leg runs on a track of its gap.
func routesFor(arrows []boxArrow, obstacles []svgBox, columns []sliceColumn, lanes lanePositions) []route {
	r := &router{lanes: lanes, obstacles: obstacles, rooms: roomsBesideSlices(columns, obstacles)}

	routes := make([]route, len(arrows))
	isStraight := make([]bool, len(arrows))
	for i, a := range arrows {
		routes[i], isStraight[i] = r.takeStraightRoute(a)
	}

	best := spreadEnds(arrows, routes, isStraight)
	for i, a := range arrows {
		if !isStraight[i] {
			routes[i] = r.takeBentRoute(a, best[i])
		}
	}

	putLegsOnTracks(routes)
	for _, l := range legsOf(routes) {
		l.y = lanes.trackY(l.gap, l.track)
	}

	return routes
}

func legsOf(routes []route) []*leg {
	var legs []*leg
	for i := range routes {
		for j := range routes[i].legs {
			legs = append(legs, &routes[i].legs[j])
		}
	}

	return legs
}

// putLegsOnTracks puts each leg of routes on a track of its gap, where no other
// leg in the gap overlaps its span.
func putLegsOnTracks(routes []route) {
	inGap := make(map[int][]*leg)
	for _, l := range legsOf(routes) {
		inGap[l.gap] = append(inGap[l.gap], l)
	}

	for _, legs := range inGap {
		sort.SliceStable(legs, func(a, b int) bool {
			leftA, _ := legs[a].span()
			leftB, _ := legs[b].span()
			return leftA < leftB
		})
		var trackRights []int
		for _, l := range legs {
			left, right := l.span()
			track := slices.IndexFunc(trackRights, func(trackRight int) bool { return trackRight < left })
			if track < 0 {
				track = len(trackRights)
				trackRights = append(trackRights, right)
			}
			trackRights[track] = right
			l.track = track
		}
	}
}

// spreadEnds gives each end of each arrow the x it is best placed at. The ends
// that meet one side of one box keep the order of the middles of the boxes at
// their far ends, from the left, and an end of a straight arrow keeps its own x.
// Between two straight arrows from that side, the ends of the others spread
// evenly across the part of the side the two leave between them.
func spreadEnds(arrows []boxArrow, routes []route, isStraight []bool) []bestXs {
	type end struct {
		arrow int
		atTo  bool
	}
	onSides := make(map[boxSide][]end)
	order := make(map[end]int)
	for i, a := range arrows {
		from, to := a.bentSides()
		fromEnd, toEnd := end{i, false}, end{i, true}
		order[fromEnd], order[toEnd] = a.to.x+a.to.w/2, a.from.x+a.from.w/2
		if isStraight[i] {
			from, to = a.sides()
			order[fromEnd], order[toEnd] = routes[i].from.x, routes[i].from.x
		}
		onSides[from] = append(onSides[from], fromEnd)
		onSides[to] = append(onSides[to], toEnd)
	}

	best := make([]bestXs, len(arrows))
	spread := func(ends []end, left, right int) {
		for j, e := range ends {
			x := left + (j+1)*(right-left)/(len(ends)+1)
			if e.atTo {
				best[e.arrow].to = x
			} else {
				best[e.arrow].from = x
			}
		}
	}
	for side, ends := range onSides {
		sort.SliceStable(ends, func(a, b int) bool { return order[ends[a]] < order[ends[b]] })
		left, first := side.box.x, 0
		for j, e := range ends {
			if isStraight[e.arrow] {
				spread(ends[first:j], left, routes[e.arrow].from.x)
				left, first = routes[e.arrow].from.x, j+1
			}
		}
		spread(ends[first:], left, side.box.x+side.box.w)
	}

	return best
}

// roomsBesideSlices are the rooms where a stretch can run up or down beside a
// slice, clear of the boxes on either side: one in each space between two
// slices, best at its middle, and one after the last slice up to the end of the
// lanes. None lies before the first slice, where each lane has its name written,
// and a picture with no slice has none.
func roomsBesideSlices(columns []sliceColumn, obstacles []svgBox) []room {
	if len(columns) == 0 {
		return nil
	}
	boxesLeft := make([]int, len(columns))
	boxesRight := make([]int, len(columns))
	for i, c := range columns {
		boxesLeft[i], boxesRight[i] = c.x+c.w, c.x
		for _, o := range obstacles {
			if o.x < c.x+c.w && o.x+o.w > c.x {
				boxesLeft[i], boxesRight[i] = min(boxesLeft[i], o.x), max(boxesRight[i], o.x+o.w)
			}
		}
	}

	last := columns[len(columns)-1]
	rooms := []room{{
		best:  last.x + last.w + sliceGap/2,
		left:  boxesRight[len(columns)-1] + stretchClearance,
		right: last.x + last.w + lanesPastLastSlice - stretchClearance,
	}}
	for i := 1; i < len(columns); i++ {
		between := room{
			best:  (columns[i-1].x + columns[i-1].w + columns[i].x) / 2,
			left:  boxesRight[i-1] + stretchClearance,
			right: boxesLeft[i] - stretchClearance,
		}
		if between.left <= between.right {
			rooms = append(rooms, between)
		}
	}

	return rooms
}
