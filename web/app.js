// The page: loads the algorithms, sets up the form, the board and sharing,
// and generates a board whenever the settings change. See README.md.
import { el } from "./dom.js";
import { setupBoard, showBoard } from "./board.js";
import { query, setupForm } from "./form.js";
import { enableSharing, setupSharing } from "./share.js";

const statusEl = document.getElementById("status");
const seedInput = document.getElementById("seed");

function showStatus(data) {
  const total = data.board.cells.length;
  const reuse = el("button", { type: "button", class: "link" }, "reuse");
  reuse.addEventListener("click", () => {
    seedInput.value = data.seed;
  });
  statusEl.replaceChildren(
    `Placed ${data.placed} obstacle${data.placed === 1 ? "" : "s"} on ${total} hex${total === 1 ? "" : "es"} · seed ${data.seed} `,
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
    showBoard(data);
    showStatus(data);
    enableSharing();
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

  setupForm(algorithms, new URLSearchParams(location.search), generate);
  generate();
}

setupBoard();
setupSharing();
init();
