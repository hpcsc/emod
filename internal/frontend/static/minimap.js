import { MINIMAP_W, MINIMAP_H, MINIMAP_PAD } from './config.js';
import { Interaction } from './interaction.js';
import { bus } from './bus.js';

function getDiagramDims(store) {
  const svgEl = store.dom.svg;
  const vg = svgEl.querySelector("#viewport-group");
  if (vg) {
    try {
      const b = vg.getBBox();
      if (b.width > 0 && b.height > 0) return { width: b.width, height: b.height };
    } catch(e) {}
  }
  const vb = svgEl.getAttribute("viewBox");
  if (vb) {
    const parts = vb.split(/\s+/).map(Number);
    if (parts.length === 4 && parts[2] > 0 && parts[3] > 0) {
      return { width: parts[2], height: parts[3] };
    }
  }
  return null;
}

function projection(store) {
  const dims = getDiagramDims(store);
  if (!dims || dims.width <= 0 || dims.height <= 0) return null;

  const availableX = MINIMAP_W - MINIMAP_PAD * 2;
  const availableY = MINIMAP_H - MINIMAP_PAD * 2;
  const scale = Math.min(availableX / dims.width, availableY / dims.height);

  return {
    dims: dims,
    scale: scale,
    offX: (MINIMAP_W - dims.width * scale) / 2,
    offY: (MINIMAP_H - dims.height * scale) / 2,
  };
}

function updateMinimap(store) {
  const minimapEl = store.dom.minimap;
  const minimapSvgEl = store.dom.minimapSvg;
  if (!minimapEl || minimapEl.classList.contains("hidden")) return;
  if (!minimapSvgEl) return;

  const p = projection(store);
  if (!p) {
    minimapSvgEl.innerHTML = "";
    return;
  }

  const svgEl = store.dom.svg;
  const cw = svgEl.clientWidth;
  const ch = svgEl.clientHeight;

  let html = "";
  html += '<rect class="minimap-bg" x="' + p.offX + '" y="' + p.offY +
    '" width="' + (p.dims.width * p.scale) + '" height="' + (p.dims.height * p.scale) +
    '" fill="#e9ecef" stroke="#adb5bd" stroke-width="0.5" rx="2"/>';

  const vpX = (-store.viewport.offsetX / store.viewport.zoomScale) * p.scale + p.offX;
  const vpY = (-store.viewport.offsetY / store.viewport.zoomScale) * p.scale + p.offY;
  const vpW = (cw / store.viewport.zoomScale) * p.scale;
  const vpH = (ch / store.viewport.zoomScale) * p.scale;

  html += '<rect class="minimap-viewport" x="' + vpX + '" y="' + vpY +
    '" width="' + vpW + '" height="' + vpH +
    '" fill="rgba(52,152,219,0.15)" stroke="#3498db" stroke-width="1" rx="1"/>';

  minimapSvgEl.innerHTML = html;
}

function minimapNavigate(store, evt) {
  const minimapSvgEl = store.dom.minimapSvg;
  if (!minimapSvgEl) return;
  const mRect = minimapSvgEl.getBoundingClientRect();
  const clientX = evt.clientX !== undefined ? evt.clientX : evt.pageX;
  const clientY = evt.clientY !== undefined ? evt.clientY : evt.pageY;
  const clickX = clientX - mRect.left;
  const clickY = clientY - mRect.top;

  const p = projection(store);
  if (!p) return;

  const diagramX = (clickX - p.offX) / p.scale;
  const diagramY = (clickY - p.offY) / p.scale;

  const svgEl = store.dom.svg;
  const cw = svgEl.clientWidth;
  const ch = svgEl.clientHeight;

  store.viewport.offsetX = -diagramX * store.viewport.zoomScale + cw / 2;
  store.viewport.offsetY = -diagramY * store.viewport.zoomScale + ch / 2;

  Interaction.applyViewport(store);
}

function toggleMinimap(store, show) {
  const minimapEl = store.dom.minimap;
  const minimapToggleEl = store.dom.minimapToggle;
  if (show === undefined) {
    minimapEl.classList.toggle("hidden");
  } else if (show) {
    minimapEl.classList.remove("hidden");
  } else {
    minimapEl.classList.add("hidden");
  }
  const isHidden = minimapEl.classList.contains("hidden");
  minimapToggleEl.classList.toggle("active", !isHidden);
  if (!isHidden) updateMinimap(store);
}

const IGNORED_TARGETS = "#minimap-close, #minimap-toggle, #minimap-handle, #visibility-panel, #visibility-toggle, #visibility-tree";

function readTranslate(el) {
  const mat = el.style.transform;
  if (mat && mat.startsWith("translate(")) {
    const parts = mat.match(/translate\(([\d.-]+)px,\s*([\d.-]+)px\)/);
    if (parts) return { x: parseFloat(parts[1]), y: parseFloat(parts[2]) };
  }
  return { x: 0, y: 0 };
}

function initMinimap(store) {
  const minimapEl = store.dom.minimap;
  const minimapHandle = document.getElementById("minimap-handle");
  const minimapClose = document.getElementById("minimap-close");

  // Dragging the handle repositions the minimap itself; dragging the body pans
  // the diagram. Both run off the same pointer stream, so they are tracked
  // separately rather than as one mode.
  let repositioning = false;
  let navigating = false;

  function beginReposition(point) {
    repositioning = true;
    const t = readTranslate(minimapEl);
    minimapEl.dataset.dragOffX = point.clientX - t.x;
    minimapEl.dataset.dragOffY = point.clientY - t.y;
  }

  function moveReposition(point) {
    const tx = point.clientX - parseFloat(minimapEl.dataset.dragOffX);
    const ty = point.clientY - parseFloat(minimapEl.dataset.dragOffY);
    minimapEl.style.transform = "translate(" + tx + "px, " + ty + "px)";
  }

  function endDrag() {
    repositioning = false;
    navigating = false;
  }

  bus.on('viewport:changed', function({ store: s }) {
    updateMinimap(s);
  });

  store.dom.minimapToggle.addEventListener("click", function() {
    toggleMinimap(store);
  });

  minimapClose.addEventListener("click", function(evt) {
    evt.stopPropagation();
    toggleMinimap(store, false);
  });

  minimapHandle.addEventListener("mousedown", function(evt) {
    evt.preventDefault();
    evt.stopPropagation();
    beginReposition(evt);
  });

  minimapEl.addEventListener("mousedown", function(evt) {
    if (evt.target.closest(IGNORED_TARGETS)) return;
    evt.preventDefault();
    evt.stopPropagation();
    minimapNavigate(store, evt);
    navigating = true;
  });

  document.addEventListener("mousemove", function(evt) {
    if (repositioning) {
      moveReposition(evt);
      return;
    }
    if (navigating) {
      minimapNavigate(store, evt);
      evt.preventDefault();
    }
  });

  document.addEventListener("mouseup", endDrag);

  minimapHandle.addEventListener("touchstart", function(evt) {
    evt.preventDefault();
    evt.stopPropagation();
    beginReposition(evt.touches[0]);
  }, { passive: false });

  minimapEl.addEventListener("touchstart", function(evt) {
    if (evt.target.closest(IGNORED_TARGETS)) return;
    evt.preventDefault();
    evt.stopPropagation();
    minimapNavigate(store, evt.touches[0]);
    navigating = true;
  }, { passive: false });

  minimapEl.addEventListener("touchmove", function(evt) {
    if (repositioning) {
      moveReposition(evt.touches[0]);
      return;
    }
    if (navigating) {
      evt.preventDefault();
      minimapNavigate(store, evt.touches[0]);
    }
  }, { passive: false });

  minimapEl.addEventListener("touchend", endDrag);
}

export const Minimap = {
  updateMinimap,
  minimapNavigate,
  toggleMinimap,
  initMinimap,
};
