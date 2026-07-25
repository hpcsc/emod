import { describe, it, expect, vi, beforeEach } from 'vitest';

const applyViewport = vi.fn();
vi.mock('../static/interaction.js', () => ({ Interaction: { applyViewport: () => applyViewport() } }));
vi.mock('../static/bus.js', () => ({ bus: { on: vi.fn(), emit: vi.fn() } }));

const { Minimap } = await import('../static/minimap.js');
const { MINIMAP_W, MINIMAP_H, MINIMAP_PAD } = await import('../static/config.js');

// The diagram is 800x400 in a 180x120 minimap, so the fit is width-bound:
// scale = (180 - 4) / 800 = 0.22, leaving the projected diagram 88px tall and
// centred vertically at y = (120 - 88) / 2 = 16.
const DIAGRAM_W = 800;
const DIAGRAM_H = 400;
const SCALE = (MINIMAP_W - MINIMAP_PAD * 2) / DIAGRAM_W;

function createStore({ hidden = false } = {}) {
  document.body.innerHTML = `
    <div id="minimap"${hidden ? ' class="hidden"' : ''}>
      <div id="minimap-handle"></div>
      <svg id="minimap-svg"></svg>
      <button id="minimap-close"></button>
    </div>
    <button id="minimap-toggle"></button>
  `;

  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('viewBox', '0 0 ' + DIAGRAM_W + ' ' + DIAGRAM_H);
  Object.defineProperty(svg, 'clientWidth', { value: 600 });
  Object.defineProperty(svg, 'clientHeight', { value: 300 });
  document.body.appendChild(svg);

  const minimapSvg = document.getElementById('minimap-svg');
  minimapSvg.getBoundingClientRect = () => ({ left: 0, top: 0, width: MINIMAP_W, height: MINIMAP_H });

  return {
    viewport: { offsetX: 0, offsetY: 0, zoomScale: 1 },
    dom: {
      svg: svg,
      minimap: document.getElementById('minimap'),
      minimapSvg: minimapSvg,
      minimapToggle: document.getElementById('minimap-toggle'),
    },
  };
}

beforeEach(() => {
  applyViewport.mockReset();
});

describe('Minimap.toggleMinimap', () => {
  it('shows a hidden minimap and marks the toggle button active', () => {
    const store = createStore({ hidden: true });

    Minimap.toggleMinimap(store);

    expect(store.dom.minimap.classList.contains('hidden')).toBe(false);
    expect(store.dom.minimapToggle.classList.contains('active')).toBe(true);
  });

  it('hides a visible minimap and clears the active state', () => {
    const store = createStore();

    Minimap.toggleMinimap(store);

    expect(store.dom.minimap.classList.contains('hidden')).toBe(true);
    expect(store.dom.minimapToggle.classList.contains('active')).toBe(false);
  });

  it('hides the minimap when told to, even if already hidden', () => {
    const store = createStore({ hidden: true });

    Minimap.toggleMinimap(store, false);

    expect(store.dom.minimap.classList.contains('hidden')).toBe(true);
  });
});

describe('Minimap.updateMinimap', () => {
  it('draws the diagram bounds scaled to fit the minimap', () => {
    const store = createStore();

    Minimap.updateMinimap(store);

    const bg = store.dom.minimapSvg.querySelector('.minimap-bg');
    expect(Number(bg.getAttribute('width'))).toBeCloseTo(DIAGRAM_W * SCALE);
    expect(Number(bg.getAttribute('height'))).toBeCloseTo(DIAGRAM_H * SCALE);
  });

  it('draws the viewport rectangle sized to the visible fraction of the diagram', () => {
    const store = createStore();

    Minimap.updateMinimap(store);

    const vp = store.dom.minimapSvg.querySelector('.minimap-viewport');
    expect(Number(vp.getAttribute('width'))).toBeCloseTo(600 * SCALE);
    expect(Number(vp.getAttribute('height'))).toBeCloseTo(300 * SCALE);
  });

  it('shrinks the viewport rectangle as the diagram is zoomed in', () => {
    const store = createStore();
    store.viewport.zoomScale = 2;

    Minimap.updateMinimap(store);

    const vp = store.dom.minimapSvg.querySelector('.minimap-viewport');
    expect(Number(vp.getAttribute('width'))).toBeCloseTo(300 * SCALE);
  });

  it('draws nothing while the minimap is hidden', () => {
    const store = createStore({ hidden: true });

    Minimap.updateMinimap(store);

    expect(store.dom.minimapSvg.innerHTML).toBe('');
  });

  it('clears the minimap when the diagram has no dimensions', () => {
    const store = createStore();
    store.dom.svg.setAttribute('viewBox', '0 0 0 0');
    store.dom.minimapSvg.innerHTML = '<rect class="minimap-bg"/>';

    Minimap.updateMinimap(store);

    expect(store.dom.minimapSvg.innerHTML).toBe('');
  });
});

describe('Minimap.minimapNavigate', () => {
  it('centres the viewport on the clicked point of the diagram', () => {
    const store = createStore();
    const offX = (MINIMAP_W - DIAGRAM_W * SCALE) / 2;
    const offY = (MINIMAP_H - DIAGRAM_H * SCALE) / 2;

    // Click the centre of the projected diagram — diagram point (400, 200).
    Minimap.minimapNavigate(store, {
      clientX: offX + (DIAGRAM_W / 2) * SCALE,
      clientY: offY + (DIAGRAM_H / 2) * SCALE,
    });

    expect(store.viewport.offsetX).toBeCloseTo(-400 + 300);
    expect(store.viewport.offsetY).toBeCloseTo(-200 + 150);
    expect(applyViewport).toHaveBeenCalled();
  });

  it('accepts touch points, which carry no clientX fallback of their own', () => {
    const store = createStore();

    Minimap.minimapNavigate(store, { pageX: 0, pageY: 0 });

    expect(applyViewport).toHaveBeenCalled();
  });

  it('leaves the viewport alone when the diagram has no dimensions', () => {
    const store = createStore();
    store.dom.svg.setAttribute('viewBox', '0 0 0 0');

    Minimap.minimapNavigate(store, { clientX: 50, clientY: 50 });

    expect(store.viewport.offsetX).toBe(0);
    expect(applyViewport).not.toHaveBeenCalled();
  });
});

describe('Minimap.initMinimap', () => {
  it('toggles the minimap when the toggle button is clicked', () => {
    const store = createStore();
    Minimap.initMinimap(store);

    store.dom.minimapToggle.click();

    expect(store.dom.minimap.classList.contains('hidden')).toBe(true);
  });

  it('hides the minimap when the close button is clicked', () => {
    const store = createStore();
    Minimap.initMinimap(store);

    document.getElementById('minimap-close').click();

    expect(store.dom.minimap.classList.contains('hidden')).toBe(true);
  });

  it('repositions the minimap when its handle is dragged', () => {
    const store = createStore();
    Minimap.initMinimap(store);

    document.getElementById('minimap-handle')
      .dispatchEvent(new MouseEvent('mousedown', { bubbles: true, clientX: 100, clientY: 50 }));
    document.dispatchEvent(new MouseEvent('mousemove', { clientX: 130, clientY: 90 }));

    expect(store.dom.minimap.style.transform).toBe('translate(30px, 40px)');
  });

  it('stops repositioning once the pointer is released', () => {
    const store = createStore();
    Minimap.initMinimap(store);

    document.getElementById('minimap-handle')
      .dispatchEvent(new MouseEvent('mousedown', { bubbles: true, clientX: 100, clientY: 50 }));
    document.dispatchEvent(new MouseEvent('mouseup', {}));
    document.dispatchEvent(new MouseEvent('mousemove', { clientX: 200, clientY: 200 }));

    expect(store.dom.minimap.style.transform).toBe('');
  });

  it('ignores clicks on the close button instead of navigating', () => {
    const store = createStore();
    Minimap.initMinimap(store);

    document.getElementById('minimap-close')
      .dispatchEvent(new MouseEvent('mousedown', { bubbles: true, clientX: 50, clientY: 50 }));

    expect(applyViewport).not.toHaveBeenCalled();
  });
});
