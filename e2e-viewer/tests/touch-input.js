// Playwright's page.touchscreen only taps, so multi-finger gestures go through
// the CDP input domain directly. Chromium only, which is why the touch spec
// runs as its own project.
//
// Every dispatch sends the full set of fingers currently down, each keeping the
// id it was pressed with. Renumbering them per call — the obvious shortcut —
// makes a release followed by a move look like one finger lifting and a
// different one landing, so the page sees a touchstart where a touchmove was
// intended.
export async function touchInput(page) {
  const client = await page.context().newCDPSession(page);

  let active = [];
  let nextId = 0;

  const dispatch = (type) =>
    client.send('Input.dispatchTouchEvent', {
      type,
      touchPoints: active.map((p) => ({ x: Math.round(p.x), y: Math.round(p.y), id: p.id })),
    });

  const api = {
    // press puts additional fingers down, keeping any already down.
    async press(points) {
      for (const p of points) active.push({ id: nextId++, x: p.x, y: p.y });
      await dispatch('touchStart');
    },

    // move repositions the fingers that are down, in the order they landed.
    async move(points) {
      points.forEach((p, i) => {
        if (active[i]) {
          active[i].x = p.x;
          active[i].y = p.y;
        }
      });
      await dispatch('touchMove');
    },

    // lift releases one finger by the order it landed, leaving the rest down.
    //
    // A touchEnd carries the points being released, not the ones still down —
    // sending the survivors instead lifts exactly the wrong finger, and every
    // later dispatch is then out of step with what the browser thinks is down.
    async lift(index = 0) {
      const [released] = active.splice(index, 1);
      if (!released) return;
      const remaining = active;
      active = [released];
      await dispatch('touchEnd');
      active = remaining;
    },

    // release lifts every remaining finger.
    async release() {
      if (active.length === 0) return;
      await dispatch('touchEnd');
      active = [];
    },

    // down reports where the fingers currently are, so a test can keep moving
    // whichever one survived a lift without hardcoding which that is.
    down: () => active.map((p) => ({ x: p.x, y: p.y })),

    // drag walks a single finger across the screen in steps, because a lone
    // jump would not exercise the incremental offset maths.
    async drag(from, to, steps = 5) {
      await api.press([from]);
      for (let i = 1; i <= steps; i++) {
        await api.move([{
          x: from.x + ((to.x - from.x) * i) / steps,
          y: from.y + ((to.y - from.y) * i) / steps,
        }]);
      }
      await api.release();
    },

    // pinch spreads two fingers from `from` apart to `to` apart, and the
    // viewer should end up scaled by exactly to/from.
    //
    // The extra move matters. A two-finger press reaches the page as two
    // events, one finger at a time, and the handler bails out of the second
    // because a gesture is already in progress — so it is still in pan mode
    // when the fingers start moving. The first two-finger touchmove is what
    // switches it to pinch and takes its baseline distance, which is why the
    // fingers must already be `from` apart by then.
    async pinch(centre, from, to) {
      await api.press(spread(centre, from * 0.9));
      await api.move(spread(centre, from));
      await api.move(spread(centre, to));
      await api.release();
    },
  };

  return api;
}

// spread returns two points centred on `centre`, `distance` apart horizontally.
export function spread(centre, distance) {
  return [
    { x: centre.x - distance / 2, y: centre.y },
    { x: centre.x + distance / 2, y: centre.y },
  ];
}
