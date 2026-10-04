// Helpers for building DOM elements.

const SVG_NS = "http://www.w3.org/2000/svg";

export function el(tag, attrs = {}, ...children) {
  const e = document.createElement(tag);
  for (const [name, value] of Object.entries(attrs)) {
    e.setAttribute(name, value);
  }
  e.append(...children);
  return e;
}

export function svgEl(tag, attrs = {}, ...children) {
  const e = document.createElementNS(SVG_NS, tag);
  for (const [name, value] of Object.entries(attrs)) {
    e.setAttribute(name, value);
  }
  e.append(...children);
  return e;
}
