// The settings form: the algorithm, the board's edge length, the seed, and
// a panel of each algorithm's parameters, built from their descriptions.
import { el } from "./dom.js";

const form = document.getElementById("controls");
const algorithmSelect = document.getElementById("algorithm");
const seedInput = document.getElementById("seed");
const edgeInput = document.getElementById("edge");
const paramsEl = document.getElementById("params");
const resetButton = document.getElementById("reset");

// Parameter panels by algorithm ID: { alg, element, inputs: Map(name → input) }.
const panels = new Map();

// onChange is called whenever the settings change, to generate a board.
let onChange = () => {};

function paramInput(alg, param) {
  let input;
  if (param.type === "bool") {
    input = el("input", { type: "checkbox" });
  } else if (param.type === "choice") {
    input = el("select", {}, ...param.options.map((o) => el("option", { value: o.value }, o.label)));
  } else if (param.type === "float") {
    // A text box, as number boxes don't take fractions like 1/2. The server
    // checks the value.
    input = el("input", {
      type: "text",
      autocomplete: "off",
      spellcheck: "false",
      title: "A number like 0.5, or a fraction like 1/2",
      required: "",
    });
  } else {
    input = el("input", { type: "number", min: param.min, step: 1, required: "" });
    if (param.max != null) {
      input.max = param.max;
    }
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
// from initial (anything with has and get, like URLSearchParams) where present and defaults otherwise.
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
    updateDependents(panel);
    onChange();
  });
  updateDependents(panel);
  return panel;
}

// holds reports whether a condition is true: the name of a bool parameter,
// optionally negated with "!", or name=value for a choice parameter. An
// empty condition always holds.
function holds(panel, condition) {
  if (!condition) {
    return true;
  }
  const [name, value] = condition.split("=");
  if (value !== undefined) {
    return panel.inputs.get(name).value === value;
  }
  return panel.inputs.get(condition.replace(/^!/, "")).checked !== condition.startsWith("!");
}

// updateDependents switches off parameters whose onlyIf condition doesn't
// hold, and locks parameters that are following another to that one's
// value. A choice picks between sets of parameters, so the sets it doesn't
// pick are hidden, along with any group left empty; a parameter a bool
// switches off is greyed out. Parameters only follow earlier ones, so one
// pass in order is enough.
function updateDependents(panel) {
  const params = new Map(panel.alg.params.map((p) => [p.name, p]));
  for (const param of panel.alg.params) {
    const input = panel.inputs.get(param.name);
    const inactive = param.onlyIf ? !holds(panel, param.onlyIf) : false;
    const hidden = inactive && param.onlyIf.includes("=");
    const follow = param.follows?.find((f) => holds(panel, f.when));
    if (follow) {
      const value = panel.inputs.get(follow.param).value;
      input.value = follow.inverse
        ? params.get(follow.param).options.find((o) => o.value === value)?.inverse ?? ""
        : value;
    }
    input.disabled = inactive || Boolean(follow);
    input.classList.toggle("derived", Boolean(follow));
    const row = input.closest(".param");
    row.classList.toggle("inactive", inactive && !hidden);
    row.hidden = hidden;
  }
  for (const fieldset of panel.element.querySelectorAll("fieldset")) {
    fieldset.hidden = [...fieldset.querySelectorAll(".param")].every((row) => row.hidden);
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

export function query() {
  const { alg, inputs } = currentPanel();
  const params = new URLSearchParams({ algorithm: alg.id, edge: edgeInput.value });
  for (const param of alg.params) {
    // The server works out parameters that are following another itself.
    const input = inputs.get(param.name);
    if (!input.classList.contains("derived")) {
      params.set(param.name, getValue(input, param));
    }
  }
  const seed = seedInput.value.trim();
  if (seed !== "") {
    params.set("seed", seed);
  }
  return params;
}

// setupForm builds a panel for each of algorithms, taking the algorithm,
// edge length, seed and parameters from initial (URLSearchParams, from the
// address bar) where it has them, and calls change whenever they change.
export function setupForm(algorithms, initial, change) {
  onChange = change;
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
  if (initial.has("edge")) {
    edgeInput.value = initial.get("edge");
  }
  showCurrentPanel();

  edgeInput.addEventListener("change", () => onChange());
  algorithmSelect.addEventListener("change", () => {
    showCurrentPanel();
    onChange();
  });
  resetButton.addEventListener("click", () => {
    const old = currentPanel();
    const panel = buildPanel(old.alg, null);
    old.element.replaceWith(panel.element);
    panels.set(old.alg.id, panel);
    onChange();
  });
  form.addEventListener("submit", (event) => {
    event.preventDefault();
    onChange();
  });
}

// applySettings sets the form up as settings (parsed settings JSON) say,
// and generates the board. Parameters left out take their defaults, as do
// the edge length and seed. It returns warnings about parameters it
// ignored, and throws, changing nothing, if the settings make no sense.
export function applySettings(settings) {
  if (typeof settings !== "object" || settings === null || Array.isArray(settings)) {
    throw new Error("expected a JSON object");
  }
  const old = panels.get(settings.algorithm);
  if (!old) {
    throw new Error(`algorithm must be one of: ${[...panels.keys()].join(", ")}`);
  }
  const params = settings.params ?? {};
  if (typeof params !== "object" || params === null || Array.isArray(params)) {
    throw new Error("params must be an object");
  }

  const warnings = [];
  const known = new Set(old.alg.params.map((p) => p.name));
  const unknown = Object.keys(params).filter((name) => !known.has(name));
  if (unknown.length) {
    warnings.push(`Ignored parameters ${old.alg.name} doesn't have: ${unknown.join(", ")}.`);
  }

  const initial = new Map(
    Object.entries(params).filter(([name]) => known.has(name)).map(([name, value]) => [name, String(value)]),
  );
  const panel = buildPanel(old.alg, initial);
  old.element.replaceWith(panel.element);
  panels.set(old.alg.id, panel);
  algorithmSelect.value = old.alg.id;
  showCurrentPanel();
  edgeInput.value = settings.edge == null ? edgeInput.defaultValue : String(settings.edge);
  seedInput.value = settings.seed == null ? "" : String(settings.seed);
  onChange();
  return warnings;
}
