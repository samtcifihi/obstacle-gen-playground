// Sharing what's shown: copying the board as an image, and copying,
// saving and loading the settings behind it as JSON.
import { el } from "./dom.js";
import { view } from "./board.js";
import { applySettings } from "./form.js";

const svg = document.getElementById("board");
const copyButton = document.getElementById("copy-image");
const copySettingsButton = document.getElementById("copy-settings");
const saveSettingsButton = document.getElementById("save-settings");
const loadDialog = document.getElementById("load-dialog");
const settingsText = document.getElementById("settings-text");
const settingsFile = document.getElementById("settings-file");
const loadMessage = document.getElementById("load-message");

// Paint properties copied onto the board image, as the stylesheet doesn't
// reach it.
const PAINT = ["fill", "stroke", "stroke-width", "stroke-linejoin", "opacity", "fill-opacity", "stroke-opacity"];

// boardImage renders the board as shown, heatmap and all, to a PNG at
// scale times the size of its viewBox, on the page's background. The board
// is copied before anything waits, so stepping on straight after doesn't
// change the image.
async function boardImage(scale = 2) {
  const copy = svg.cloneNode(true);
  const originals = svg.querySelectorAll("*");
  copy.querySelectorAll("*").forEach((e, i) => {
    const style = getComputedStyle(originals[i]);
    for (const name of PAINT) {
      // Computed references to the hatching are absolute, like
      // url("http://host/?query#blocked"), but the image is its own
      // document.
      e.style.setProperty(name, style.getPropertyValue(name).replace(/url\(["']?[^#)]*#([^"')]+)["']?\)/, "url(#$1)"));
    }
  });
  const { width, height } = svg.viewBox.baseVal;
  copy.setAttribute("width", width * scale);
  copy.setAttribute("height", height * scale);
  const background = getComputedStyle(document.body).backgroundColor;

  const source = new Blob([new XMLSerializer().serializeToString(copy)], { type: "image/svg+xml" });
  const url = URL.createObjectURL(source);
  try {
    const image = new Image();
    image.src = url;
    await image.decode();
    const canvas = el("canvas", { width: Math.round(width * scale), height: Math.round(height * scale) });
    const context = canvas.getContext("2d");
    context.fillStyle = background;
    context.fillRect(0, 0, canvas.width, canvas.height);
    context.drawImage(image, 0, 0, canvas.width, canvas.height);
    return await new Promise((resolve, reject) => {
      canvas.toBlob((png) => (png ? resolve(png) : reject(new Error("couldn't make the image"))), "image/png");
    });
  } finally {
    URL.revokeObjectURL(url);
  }
}

function imageName() {
  const at = view.frame < view.data.trace.steps.length ? `step-${view.frame + 1}` : "final";
  return `${view.data.settings.algorithm}-seed-${view.data.seed}-${at}.png`;
}

async function download(png, name) {
  const url = URL.createObjectURL(await png);
  el("a", { href: url, download: name }).click();
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

const flashTimers = new Map();
// flash shows text on button for a moment, as feedback.
function flash(button, text) {
  button.dataset.label ??= button.textContent;
  button.textContent = text;
  clearTimeout(flashTimers.get(button));
  flashTimers.set(button, setTimeout(() => {
    button.textContent = button.dataset.label;
  }, 2000));
}

// copyImage puts the board on the clipboard. Browsers only allow that on
// secure pages (localhost counts, but not plain HTTP from another machine),
// so failing that it downloads the image instead.
function copyImage() {
  if (!view.data) {
    return;
  }
  const png = boardImage();
  const name = imageName();
  // Clipboard writes must start straight away, while the click still counts
  // as the user's, so the item takes the image while it's still rendering.
  const copied = window.ClipboardItem && navigator.clipboard?.write
    ? navigator.clipboard.write([new ClipboardItem({ "image/png": png })])
    : Promise.reject(new Error("can't write images to the clipboard"));
  copied
    .then(() => flash(copyButton, "Copied"))
    .catch(async (err) => {
      console.warn("Couldn't copy the board, so downloading it instead:", err);
      await download(png, name);
      flash(copyButton, "Downloaded");
    })
    .catch((err) => {
      console.error("Couldn't copy or download the board:", err);
      flash(copyButton, "Failed");
    });
}

// settingsJSON returns the settings behind the board shown, as JSON: the
// algorithm, board, seed and parameters, and the commit of the code that
// made it. See README.md.
function settingsJSON() {
  return `${JSON.stringify(view.data.settings, null, 2)}\n`;
}

function settingsName() {
  const { algorithm, seed } = view.data.settings;
  return `${algorithm}-seed-${seed}.json`;
}

function copySettings() {
  if (!view.data) {
    return;
  }
  const text = settingsJSON();
  const name = settingsName();
  const copied = navigator.clipboard?.writeText
    ? navigator.clipboard.writeText(text)
    : Promise.reject(new Error("can't write to the clipboard"));
  copied
    .then(() => flash(copySettingsButton, "Copied"))
    .catch(async (err) => {
      console.warn("Couldn't copy the settings, so downloading them instead:", err);
      await download(new Blob([text], { type: "application/json" }), name);
      flash(copySettingsButton, "Downloaded");
    });
}

function saveSettings() {
  if (view.data) {
    download(new Blob([settingsJSON()], { type: "application/json" }), settingsName());
  }
}

function showLoadMessage(text, isError) {
  loadMessage.textContent = text;
  loadMessage.classList.toggle("error", isError);
}

function openLoadDialog() {
  showLoadMessage("", false);
  loadDialog.showModal();
  // Selected, so pasting replaces whatever was loaded last time.
  settingsText.select();
}

function loadSettings() {
  let settings;
  try {
    settings = JSON.parse(settingsText.value);
  } catch (err) {
    showLoadMessage(`That isn't valid JSON: ${err.message}`, true);
    return;
  }
  const current = view.data?.settings.commit;
  let warnings;
  try {
    warnings = applySettings(settings);
  } catch (err) {
    showLoadMessage(`Couldn't load these settings: ${err.message}`, true);
    return;
  }
  if (settings.commit && current && settings.commit !== current) {
    warnings.push(
      `These settings came from commit ${settings.commit.slice(0, 7)}, but this is ${current.slice(0, 7)}, ` +
        "so the algorithm may have changed since.",
    );
  }
  if (warnings.length) {
    showLoadMessage(`Applied. ${warnings.join(" ")}`, false);
  } else {
    loadDialog.close();
  }
}

// enableSharing enables the buttons that need a board to share.
export function enableSharing() {
  copyButton.disabled = false;
  copySettingsButton.disabled = false;
  saveSettingsButton.disabled = false;
}

// setupSharing sets up the sharing buttons and the load dialog.
export function setupSharing() {
  copyButton.addEventListener("click", copyImage);
  copySettingsButton.addEventListener("click", copySettings);
  saveSettingsButton.addEventListener("click", saveSettings);
  document.getElementById("load-settings").addEventListener("click", openLoadDialog);
  document.getElementById("apply-settings").addEventListener("click", loadSettings);
  document.getElementById("cancel-load").addEventListener("click", () => loadDialog.close());
  document.getElementById("open-settings").addEventListener("click", () => settingsFile.click());
  settingsFile.addEventListener("change", async () => {
    const [file] = settingsFile.files;
    if (file) {
      settingsText.value = await file.text();
      showLoadMessage(`Opened ${file.name}.`, false);
    }
    settingsFile.value = ""; // so opening the same file again still counts as a change
  });
}
