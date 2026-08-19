// Readers for the route an arrow was actually drawn along, parsed from the path
// the renderer emitted rather than recomputed from the rule that placed it. A
// label positioned by a rule the arrow does not follow — or left behind by a
// drag — reads as away from the line here, which a bounding-box check cannot
// see. jsdom implements no SVG path geometry, so the sampling is done by hand.

function pointsOf(pathEl) {
  const numbers = pathEl.getAttribute('d').match(/-?\d+(?:\.\d+)?/g).map(Number);
  const points = [];
  for (let i = 0; i < numbers.length; i += 2) points.push({ x: numbers[i], y: numbers[i + 1] });
  return points;
}

function sampler(pathEl) {
  const p = pointsOf(pathEl);
  if (!pathEl.getAttribute('d').includes(' C ')) {
    return (t) => ({ x: p[0].x + (p[1].x - p[0].x) * t, y: p[0].y + (p[1].y - p[0].y) * t });
  }
  return (t) => {
    const u = 1 - t;
    const blend = (a, b, c, d) => u * u * u * a + 3 * u * u * t * b + 3 * u * t * t * c + t * t * t * d;
    return {
      x: blend(p[0].x, p[1].x, p[2].x, p[3].x),
      y: blend(p[0].y, p[1].y, p[2].y, p[3].y),
    };
  };
}

export function distanceToPath(pathEl, x, y) {
  const at = sampler(pathEl);
  let nearest = Infinity;
  for (let step = 0; step <= 1000; step++) {
    const p = at(step / 1000);
    nearest = Math.min(nearest, Math.hypot(p.x - x, p.y - y));
  }
  return nearest;
}

// The two points an arrow runs between.
export function pathEnds(pathEl) {
  const p = pointsOf(pathEl);
  return { start: p[0], end: p[p.length - 1] };
}
