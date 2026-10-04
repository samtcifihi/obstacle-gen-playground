"use strict";

const SVG_NS = "http://www.w3.org/2000/svg";
// Distance from a hex's centre to its corners, in SVG units.
const HEX_SIZE = 30;
const SQRT3 = Math.sqrt(3);

const form = document.getElementById("controls");
const algorithmSelect = document.getElementById("algorithm");
const seedInput = document.getElementById("seed");
const paramsEl = document.getElementById("params");
const resetButton = document.getElementById("reset");
const statusEl = document.getElementById("status");
const svg = document.getElementById("board");

// Parameter panels by algorithm ID: { alg, element, inputs: Map(name → input) }.
const panels = new Map();

function el(tag, attrs = {}, ...children) {
  const e = document.createElement(tag);
  for (const [name, value] of Object.entries(attrs)) {
    e.setAttribute(name, value);
  }
  e.append(...children);
  return e;
}

// Hexes are drawn pointy-top; see https://www.redblobgames.com/grids/hexagons/
function hexCenter({ q, r }) {
  return { x: HEX_SIZE * SQRT3 * (q + r / 2), y: HEX_SIZE * 1.5 * r };
}

function hexCorners({ x, y }) {
  const corners = [];
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i - Math.PI / 6;
    corners.push(`${x + HEX_SIZE * Math.cos(angle)},${y + HEX_SIZE * Math.sin(angle)}`);
  }
  return corners.join(" ");
}

function render(board) {
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  const hexes = board.cells.map((cell) => {
    const center = hexCenter(cell);
    minX = Math.min(minX, center.x);
    maxX = Math.max(maxX, center.x);
    minY = Math.min(minY, center.y);
    maxY = Math.max(maxY, center.y);

    const polygon = document.createElementNS(SVG_NS, "polygon");
    polygon.setAttribute("points", hexCorners(center));
    polygon.classList.add("hex");
    polygon.classList.toggle("obstacle", cell.obstacle);
    const title = document.createElementNS(SVG_NS, "title");
    title.textContent = `q ${cell.q}, r ${cell.r}${cell.obstacle ? " (obstacle)" : ""}`;
    polygon.append(title);
    return polygon;
  });
  svg.replaceChildren(...hexes);

  // Pad by half a hex's width/height plus room for the outline.
  const padX = (HEX_SIZE * SQRT3) / 2 + 2;
  const padY = HEX_SIZE + 2;
  svg.setAttribute(
    "viewBox",
    `${minX - padX} ${minY - padY} ${maxX - minX + 2 * padX} ${maxY - minY + 2 * padY}`,
  );
}

function paramInput(alg, param) {
  let input;
  if (param.type === "bool") {
    input = el("input", { type: "checkbox" });
  } else if (param.type === "choice") {
    input = el("select", {}, ...param.options.map((o) => el("option", { value: o.value }, o.label)));
  } else {
    input = el("input", {
      type: "number",
      min: param.min,
      step: param.type === "int" ? 1 : "any",
      required: "",
    });
  }
  input.id = `param-${alg.id}-${param.name}`;
  input.name = param.name;
  return input;
}

function setValue(input, param, value) {
  if (param.type === "bool") {
    input.checked = value === true || value === "true";
  } else {
    input.value = String(value);
  }
}

function getValue(input, param) {
  return param.type === "bool" ? String(input.checked) : input.value;
}

// buildPanel makes the form fields for alg's parameters, taking values
// from initial (URLSearchParams) where present and defaults otherwise.
function buildPanel(alg, initial) {
  const element = el("div", { class: "panel" }, el("p", { class: "description" }, alg.description));
  const inputs = new Map();
  const groups = new Map();
  for (const param of alg.params) {
    let fieldset = groups.get(param.group);
    if (!fieldset) {
      fieldset = el("fieldset");
      if (param.group) {
        fieldset.append(el("legend", {}, param.group));
      }
      groups.set(param.group, fieldset);
      element.append(fieldset);
    }
    const input = paramInput(alg, param);
    setValue(input, param, initial?.has(param.name) ? initial.get(param.name) : param.default);
    inputs.set(param.name, input);
    fieldset.append(
      el(
        "div",
        { class: "param" },
        el("label", { for: input.id }, el("code", {}, param.name)),
        input,
        el("p", { class: "hint" }, param.description),
      ),
    );
  }
  const panel = { alg, element, inputs };
  element.addEventListener("change", () => {
    updateActive(panel);
    generate();
  });
  updateActive(panel);
  return panel;
}

// updateActive disables parameters whose onlyIf condition doesn't hold.
function updateActive(panel) {
  for (const param of panel.alg.params) {
    if (!param.onlyIf) {
      continue;
    }
    const negated = param.onlyIf.startsWith("!");
    const flag = panel.inputs.get(param.onlyIf.replace(/^!/, "")).checked;
    const input = panel.inputs.get(param.name);
    input.disabled = flag === negated;
    input.closest(".param").classList.toggle("inactive", input.disabled);
  }
}

function currentPanel() {
  return panels.get(algorithmSelect.value);
}

function showCurrentPanel() {
  for (const [id, panel] of panels) {
    panel.element.hidden = id !== algorithmSelect.value;
  }
}

function query() {
  const { alg, inputs } = currentPanel();
  const params = new URLSearchParams({ algorithm: alg.id });
  for (const param of alg.params) {
    params.set(param.name, getValue(inputs.get(param.name), param));
  }
  const seed = seedInput.value.trim();
  if (seed !== "") {
    params.set("seed", seed);
  }
  return params;
}

function showStatus(data) {
  const total = data.board.cells.length;
  const reuse = el("button", { type: "button", class: "link" }, "reuse");
  reuse.addEventListener("click", () => {
    seedInput.value = data.seed;
  });
  statusEl.replaceChildren(
    `Placed ${data.placed} obstacle${data.placed === 1 ? "" : "s"} on ${total} hexes · seed ${data.seed} `,
    reuse,
  );
  statusEl.classList.remove("error");
}

let latestRequest = 0;

async function generate() {
  const params = query();
  const request = ++latestRequest;
  try {
    const res = await fetch(`/api/generate?${params}`);
    if (!res.ok) {
      throw new Error((await res.text()).trim());
    }
    const data = await res.json();
    // Ignore responses that arrive after a newer request was made.
    if (request !== latestRequest) {
      return;
    }
    render(data.board);
    showStatus(data);
    // Keep the settings in the address bar, so reloading or sharing the
    // page keeps them.
    history.replaceState(null, "", `?${params}`);
  } catch (err) {
    if (request === latestRequest) {
      statusEl.textContent = `Couldn't generate a board: ${err.message}`;
      statusEl.classList.add("error");
    }
  }
}

async function init() {
  let algorithms;
  try {
    const res = await fetch("/api/algorithms");
    if (!res.ok) {
      throw new Error((await res.text()).trim());
    }
    algorithms = await res.json();
  } catch (err) {
    statusEl.textContent = `Couldn't load the algorithms: ${err.message}`;
    statusEl.classList.add("error");
    return;
  }

  const initial = new URLSearchParams(location.search);
  const selected = algorithms.some((a) => a.id === initial.get("algorithm"))
    ? initial.get("algorithm")
    : algorithms[0].id;
  for (const alg of algorithms) {
    algorithmSelect.append(el("option", { value: alg.id }, alg.name));
    const panel = buildPanel(alg, alg.id === selected ? initial : null);
    panels.set(alg.id, panel);
    paramsEl.append(panel.element);
  }
  algorithmSelect.value = selected;
  seedInput.value = initial.get("seed") ?? "";
  showCurrentPanel();
  generate();
}

algorithmSelect.addEventListener("change", () => {
  showCurrentPanel();
  generate();
});

resetButton.addEventListener("click", () => {
  const old = currentPanel();
  const panel = buildPanel(old.alg, null);
  old.element.replaceWith(panel.element);
  panels.set(old.alg.id, panel);
  generate();
});

form.addEventListener("submit", (event) => {
  event.preventDefault();
  generate();
});

init();
