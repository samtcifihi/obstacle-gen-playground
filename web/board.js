// The board: drawing it, colouring each step's heatmap, stepping through
// the placements and describing cells.
import { el, svgEl } from "./dom.js";

// Distance from a hex's centre to its corners, in SVG units.
const HEX_SIZE = 30;
const SQRT3 = Math.sqrt(3);

// Sequential blue ramp for the heatmap, from low to high. Dark mode uses
// its own steps so low values recede towards the dark background.
const HEAT_RAMP = {
  light: ["#cde2fb", "#b7d3f6", "#9ec5f4", "#86b6ef", "#6da7ec", "#5598e7", "#3987e5", "#2a78d6", "#256abf", "#1c5cab", "#184f95"],
  dark: ["#104281", "#184f95", "#1c5cab", "#256abf", "#2a78d6", "#3987e5", "#5598e7", "#6da7ec", "#86b6ef"],
};
const darkMode = matchMedia("(prefers-color-scheme: dark)");

const svg = document.getElementById("board");
const frameInput = document.getElementById("frame");
const frameLabel = document.getElementById("frame-label");
const metricSelect = document.getElementById("metric");
const legendEl = document.getElementById("legend");
const cellInfo = document.getElementById("cell-info");

// The latest generated board and what's shown of it. Frame f < steps shows
// the board before obstacle f+1 was placed, with a heatmap of that choice;
// frame steps shows the final board.
export const view = {
  data: null,
  hexes: new Map(), // "q,r" → polygon
  markers: null, // outlines around the next obstacles
  frame: 0,
  metric: null, // name of the metric the heatmap shows
  hovered: null, // "q,r" of the cell under the pointer
};

const CELL_INFO_HINT = "Hover over a cell to see its values.";

const hexKey = ({ q, r }) => `${q},${r}`;

// Hexes are drawn pointy-top; see https://www.redblobgames.com/grids/hexagons/
function hexCenter({ q, r }) {
  return { x: HEX_SIZE * SQRT3 * (q + r / 2), y: HEX_SIZE * 1.5 * r };
}

function hexCorners({ x, y }, size = HEX_SIZE) {
  const corners = [];
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i - Math.PI / 6;
    corners.push(`${x + size * Math.cos(angle)},${y + size * Math.sin(angle)}`);
  }
  return corners.join(" ");
}

// drawBoard creates the board's hexes; showFrame colours them.
function drawBoard(board) {
  view.hexes.clear();
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  const polygons = board.cells.map((cell) => {
    const center = hexCenter(cell);
    minX = Math.min(minX, center.x);
    maxX = Math.max(maxX, center.x);
    minY = Math.min(minY, center.y);
    maxY = Math.max(maxY, center.y);
    const polygon = svgEl("polygon", { points: hexCorners(center), class: "hex" });
    polygon.dataset.key = hexKey(cell);
    view.hexes.set(polygon.dataset.key, polygon);
    return polygon;
  });
  const hatch = svgEl(
    "pattern",
    { id: "blocked", width: 6, height: 6, patternUnits: "userSpaceOnUse", patternTransform: "rotate(45)" },
    svgEl("rect", { width: 6, height: 6, class: "hatch-bg" }),
    svgEl("line", { x1: 1, y1: 0, x2: 1, y2: 6, class: "hatch-line" }),
  );
  if (!view.hexes.has(view.hovered)) {
    view.hovered = null; // the board shrank
  }
  view.markers = svgEl("g");
  svg.replaceChildren(svgEl("defs", {}, hatch), ...polygons, view.markers);

  // Pad by half a hex's width/height plus room for the outline.
  const padX = (HEX_SIZE * SQRT3) / 2 + 3;
  const padY = HEX_SIZE + 3;
  svg.setAttribute(
    "viewBox",
    `${minX - padX} ${minY - padY} ${maxX - minX + 2 * padX} ${maxY - minY + 2 * padY}`,
  );
}

function heatColor(t) {
  const ramp = darkMode.matches ? HEAT_RAMP.dark : HEAT_RAMP.light;
  const pos = Math.min(Math.max(t, 0), 1) * (ramp.length - 1);
  const i = Math.min(Math.floor(pos), ramp.length - 2);
  const mix = (a, b) => Math.round(a + (b - a) * (pos - i));
  const [r1, g1, b1] = rgb(ramp[i]);
  const [r2, g2, b2] = rgb(ramp[i + 1]);
  return `rgb(${mix(r1, r2)}, ${mix(g1, g2)}, ${mix(b1, b2)})`;
}

function rgb(hex) {
  return [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16));
}

// formatValue shows a metric's value. Small values that aren't 0 shouldn't
// look like 0.
function formatValue(metric, value) {
  if (metric.percent) {
    return value > 0 && value < 0.0005 ? "<0.1%" : `${(value * 100).toFixed(1)}%`;
  }
  if (metric.integer) {
    return String(value);
  }
  return value !== 0 && Math.abs(value) < 0.001 ? value.toExponential(2) : value.toFixed(3);
}

// stepHexes returns the hexes a step put obstacles on: usually one, but
// some algorithms place a few at once.
const stepHexes = (step) => [step.placed, ...(step.also ?? [])];

// frameState works out what each cell is at the current frame.
function frameState() {
  const { board, trace } = view.data;
  const steps = trace.steps;
  const stepOf = new Map(); // "q,r" → index of the step that placed it
  const number = new Map(); // "q,r" → which obstacle it was, from 1
  let before = 0; // obstacles placed before this frame
  steps.forEach((s, i) => {
    for (const h of stepHexes(s)) {
      stepOf.set(hexKey(h), i);
      number.set(hexKey(h), number.size + 1);
    }
    if (i < view.frame) {
      before += stepHexes(s).length;
    }
  });
  // Obstacles placed at or after this frame aren't on the board yet.
  const obstacles = new Set(
    board.cells.filter((c) => c.obstacle && !(stepOf.get(hexKey(c)) >= view.frame)).map(hexKey),
  );
  const step = steps[view.frame];
  const candidates = new Map(step ? step.candidates.map((c) => [hexKey(c), c.values]) : []);
  return { number, before, obstacles, step, candidates };
}

function showFrame() {
  const { trace } = view.data;
  const total = view.data.placed;
  const { before, obstacles, step, candidates } = frameState();
  const metricIndex = trace.metrics.findIndex((m) => m.name === view.metric);
  const metric = trace.metrics[metricIndex];

  let lo = Infinity, hi = -Infinity;
  for (const values of candidates.values()) {
    lo = Math.min(lo, values[metricIndex]);
    hi = Math.max(hi, values[metricIndex]);
  }

  let blocked = false;
  for (const [key, polygon] of view.hexes) {
    const values = candidates.get(key);
    const isObstacle = obstacles.has(key);
    const isBlocked = Boolean(step) && !isObstacle && !values;
    blocked ||= isBlocked;
    polygon.classList.toggle("obstacle", isObstacle);
    polygon.classList.toggle("blocked", isBlocked);
    polygon.style.fill = values ? heatColor(hi > lo ? (values[metricIndex] - lo) / (hi - lo) : 0.5) : "";
  }

  view.markers.replaceChildren(
    ...(step ? stepHexes(step) : []).map((h) =>
      svgEl("polygon", { class: "next-marker", points: hexCorners(hexCenter(h), HEX_SIZE - 2) }),
    ),
  );

  frameInput.max = trace.steps.length;
  frameInput.value = view.frame;
  if (step) {
    const n = stepHexes(step).length;
    frameLabel.textContent = n === 1
      ? `Choosing obstacle ${before + 1} of ${total}`
      : `Choosing obstacles ${before + 1}–${before + n} of ${total}`;
  } else {
    const note = trace.note ? ` · ${trace.note}` : "";
    frameLabel.textContent = `Final board · ${total} obstacle${total === 1 ? "" : "s"}${note}`;
  }
  metricSelect.disabled = !step;
  showLegend(step ? { metric, lo, hi, blocked } : null);
  cellInfo.textContent = view.hovered ? describeCell(view.hovered) : CELL_INFO_HINT;
}

function swatch(className) {
  return svgEl(
    "svg",
    { class: "swatch", viewBox: "-11 -11 22 22", "aria-hidden": "true" },
    svgEl("polygon", { points: hexCorners({ x: 0, y: 0 }, 9), class: className }),
  );
}

function showLegend(heat) {
  if (!heat) {
    legendEl.replaceChildren(
      el("span", { class: "legend-item" }, swatch("hex obstacle"), "Obstacle"),
      el("span", { class: "legend-item" }, swatch("hex"), "Empty"),
    );
    return;
  }
  const { metric, lo, hi, blocked } = heat;
  const ramp = darkMode.matches ? HEAT_RAMP.dark : HEAT_RAMP.light;
  const bar = el("span", { class: "ramp-bar" });
  bar.style.background = `linear-gradient(to right, ${ramp.join(", ")})`;
  const items = [
    el(
      "span",
      { class: "legend-item ramp" },
      el("span", {}, formatValue(metric, lo)),
      bar,
      el("span", {}, formatValue(metric, hi)),
    ),
    el("span", { class: "legend-item" }, swatch("next-marker swatch-marker"), "Chosen"),
    el("span", { class: "legend-item" }, swatch("hex obstacle"), "Obstacle"),
  ];
  if (blocked) {
    items.push(el("span", { class: "legend-item" }, swatch("hex blocked"), "Can't be chosen"));
  }
  legendEl.replaceChildren(...items);
}

function describeCell(key) {
  const { trace } = view.data;
  const { number, obstacles, step, candidates } = frameState();
  const [q, r] = key.split(",");
  const where = `q ${q}, r ${r}`;
  if (obstacles.has(key)) {
    return number.has(key) ? `${where} · obstacle ${number.get(key)}` : `${where} · obstacle`;
  }
  if (!step) {
    return `${where} · empty`;
  }
  const values = candidates.get(key);
  if (!values) {
    return `${where} · can't be chosen`;
  }
  const parts = trace.metrics.map((m, i) => `${m.name} ${formatValue(m, values[i])}`);
  const chosen = stepHexes(step).some((h) => hexKey(h) === key) ? " · chosen" : "";
  return `${where}${chosen} · ${parts.join(" · ")}`;
}

function setFrame(frame) {
  if (!view.data) {
    return;
  }
  view.frame = Math.min(Math.max(frame, 0), view.data.trace.steps.length);
  showFrame();
}

// showBoard displays a newly generated board. If the previous board was
// being stepped through, it stays on the same step so changes are easy to
// compare.
export function showBoard(data) {
  const previous = view.data;
  const wasAtEnd = !previous || view.frame >= previous.trace.steps.length;
  view.data = data;

  const metrics = data.trace.metrics;
  if (!metrics.some((m) => m.name === view.metric)) {
    view.metric = metrics[0]?.name ?? null;
  }
  metricSelect.replaceChildren(
    ...metrics.map((m) => el("option", { value: m.name }, m.description)),
  );
  metricSelect.value = view.metric;

  drawBoard(data.board);
  setFrame(wasAtEnd ? data.trace.steps.length : view.frame);
}

// setupBoard sets up the stepper, the heatmap's metric, and describing the
// cell under the pointer.
export function setupBoard() {
  document.getElementById("first").addEventListener("click", () => setFrame(0));
  document.getElementById("prev").addEventListener("click", () => setFrame(view.frame - 1));
  document.getElementById("next").addEventListener("click", () => setFrame(view.frame + 1));
  document.getElementById("last").addEventListener("click", () => setFrame(Infinity));
  frameInput.addEventListener("input", () => setFrame(Number(frameInput.value)));
  document.addEventListener("keydown", (event) => {
    if (event.target.closest("input, select, textarea") || event.altKey || event.ctrlKey || event.metaKey) {
      return;
    }
    if (event.key === "ArrowLeft") {
      setFrame(view.frame - 1);
    } else if (event.key === "ArrowRight") {
      setFrame(view.frame + 1);
    } else {
      return;
    }
    event.preventDefault();
  });
  metricSelect.addEventListener("change", () => {
    view.metric = metricSelect.value;
    showFrame();
  });
  svg.addEventListener("pointerover", (event) => {
    const key = event.target.dataset?.key;
    if (key && view.data) {
      view.hovered = key;
      cellInfo.textContent = describeCell(key);
    }
  });
  svg.addEventListener("pointerleave", () => {
    view.hovered = null;
    cellInfo.textContent = CELL_INFO_HINT;
  });
  darkMode.addEventListener("change", () => {
    if (view.data) {
      showFrame();
    }
  });
}
