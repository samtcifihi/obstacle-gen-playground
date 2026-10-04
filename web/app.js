"use strict";

const SVG_NS = "http://www.w3.org/2000/svg";
// Distance from a hex's centre to its corners, in SVG units.
const HEX_SIZE = 30;
const SQRT3 = Math.sqrt(3);

const form = document.getElementById("controls");
const statusEl = document.getElementById("status");
const svg = document.getElementById("board");

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

async function generate() {
  const params = new URLSearchParams({ n: form.elements.n.value });
  const seed = form.elements.seed.value.trim();
  if (seed !== "") {
    params.set("seed", seed);
  }

  try {
    const res = await fetch(`/api/generate?${params}`);
    if (!res.ok) {
      throw new Error((await res.text()).trim());
    }
    const data = await res.json();
    render(data.board);
    const total = data.board.cells.length;
    statusEl.textContent =
      `Placed ${data.placed} obstacle${data.placed === 1 ? "" : "s"} on ${total} hexes · seed ${data.seed}`;
    statusEl.classList.remove("error");
  } catch (err) {
    statusEl.textContent = `Couldn't generate a board: ${err.message}`;
    statusEl.classList.add("error");
  }
}

form.addEventListener("submit", (event) => {
  event.preventDefault();
  generate();
});

generate();
