// jsdom implements the SVG DOM but none of its geometry APIs, which the layout
// and interaction code calls. These shims fill that environment gap so tests can
// exercise the real modules; they are not stand-ins for project code.
export function installSVGGeometry() {
  const proto = globalThis.SVGElement.prototype;

  if (!proto.getComputedTextLength) {
    // Roughly 7px per character is enough for layout to produce distinct widths.
    proto.getComputedTextLength = function () {
      return (this.textContent || '').length * 7;
    };
  }

  if (!proto.getBBox) {
    proto.getBBox = function () {
      return { x: 0, y: 0, width: 100, height: 20 };
    };
  }

  if (!globalThis.SVGSVGElement.prototype.createSVGPoint) {
    globalThis.SVGSVGElement.prototype.createSVGPoint = function () {
      return { x: 0, y: 0, matrixTransform() { return { x: this.x, y: this.y }; } };
    };
  }

  if (!globalThis.SVGSVGElement.prototype.getScreenCTM) {
    globalThis.SVGSVGElement.prototype.getScreenCTM = function () {
      return {
        inverse() {
          return { a: 1, b: 0, c: 0, d: 1, e: 0, f: 0, multiply(p) { return p; } };
        },
      };
    };
  }
}

// jsdom has no SVG coordinate mapping, so diagram space and client space are
// made one and the same — a 10px mouse move is a 10px move in the diagram.
export function installIdentityMapping(svg) {
  svg.createSVGPoint = function () {
    return { x: 0, y: 0, matrixTransform() { return { x: this.x, y: this.y }; } };
  };
  svg.getScreenCTM = function () {
    return { inverse() { return { multiply(p) { return p; } }; } };
  };
}
