// Every value from the files is set as text. The one exception is the document
// body, which the server has already sanitized.

const DONE_PREVIEW = 20;
const MOVED_MS = 2400; // Same duration as the `moved` animation in style.css.
const NARROW = window.matchMedia("(max-width: 720px)");

const el = Object.fromEntries(
  [...document.querySelectorAll("[data-el]")].map((node) => [node.dataset.el, node]),
);

const state = {
  board: null,
  boardText: "",
  columnOf: null,
  movedAt: new Map(),
  treeText: "",
  tree: null,
  docPath: null,
  docPathLoaded: null,
  docText: "",
  docTitle: "ork",
  lastDoc: null,
  showAllDone: false,
  closedDirs: new Set(["tickets/done"]),
  openedDirs: new Set(),
};

function h(tag, attrs = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (value === undefined || value === null || value === false) continue;
    if (key === "class") node.className = value;
    else if (key === "on") for (const [type, fn] of Object.entries(value)) node.addEventListener(type, fn);
    else node.setAttribute(key, value === true ? "" : value);
  }
  node.append(...children.flat().filter((child) => child !== null && child !== undefined));
  return node;
}

function docHref(path) {
  return "#/doc/" + path.split("/").map(encodeURIComponent).join("/");
}

function safeDecode(value) {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function parseRoute() {
  const raw = location.hash.slice(1) || "/board";
  if (raw === "/doc" || raw === "/doc/") return { view: "docs", path: null, anchor: null };
  if (raw.startsWith("/doc/")) {
    const [path, query = ""] = raw.slice("/doc/".length).split("?");
    const anchor = query.startsWith("a=") ? safeDecode(query.slice(2)) : null;
    return { view: "docs", path: safeDecode(path), anchor };
  }
  return { view: "board" };
}

async function fetchJSON(url) {
  const response = await fetch(url);
  if (!response.ok) {
    const error = new Error(response.statusText);
    error.status = response.status;
    throw error;
  }
  return response.text();
}

function setOffline(offline) {
  el.offline.hidden = !offline;
}

/* Top bar */

function renderSummary() {
  const columns = state.board.columns;
  const ids = (status) => columns.find((c) => c.status === status)?.tickets.map((t) => t.id) ?? [];
  const blocked = ids("blocked");
  const plans = ids("plan-review");
  const reviews = ids("in-review").length;
  const running = ids("planning").length + ids("in-progress").length;
  const parts = [];
  if (blocked.length) parts.push(h("span", { class: "waits" }, `⏸ bloqué : ${blocked.join(" ")}`));
  if (plans.length) parts.push(h("span", { class: "waits" }, `📋 plan : ${plans.join(" ")}`));
  if (reviews) parts.push(h("span", { class: "waits" }, `👀 ${reviews} à relire`));
  if (running) parts.push(h("span", {}, `⚙ ${running} en cours`));
  if (!blocked.length && !plans.length && !reviews && !running) parts.push(h("span", {}, "rien ne t'attend"));
  el.summary.replaceChildren(...parts);
}

/* Board */

function renderCard(ticket, moved, showStatus) {
  const tags = [];
  if (ticket.priority === "high") tags.push(h("span", { class: "tag high" }, "high"));
  if (ticket.review === "required") tags.push(h("span", { class: "tag required" }, "review: required"));
  if (ticket.waiting.length) tags.push(h("span", { class: "tag" }, `attend ${ticket.waiting.join(", ")}`));
  if (ticket.malformed) tags.push(h("span", { class: "tag malformed" }, "frontmatter illisible"));
  else if (showStatus) tags.push(h("span", { class: "tag malformed" }, ticket.status ? `status: ${ticket.status}` : "sans status"));
  const card = h(
    "a",
    { class: "card", href: docHref(ticket.path) },
    h(
      "div",
      { class: "card-meta" },
      h("span", { class: "card-id" }, ticket.id),
      ticket.updated ? h("span", { class: "card-date" }, ticket.updated) : null,
    ),
    h("p", { class: "card-title" }, ticket.title),
    h("div", { class: "card-tags" }, tags),
  );
  markMoved(card, ticket.path, moved);
  return card;
}

// The next write can rebuild the board mid-animation: the signal resumes where it
// was instead of disappearing.
function markMoved(card, path, moved) {
  const now = performance.now();
  if (moved) state.movedAt.set(path, now);
  const since = state.movedAt.get(path);
  if (since === undefined) return;
  const elapsed = now - since;
  if (elapsed >= MOVED_MS) {
    state.movedAt.delete(path);
    return;
  }
  card.classList.add("moved");
  card.style.animationDelay = `-${elapsed}ms`;
}

function renderColumn(column, previous) {
  const isDone = column.status === "done";
  const isOther = column.status === "other";
  const tickets = isDone && !state.showAllDone ? column.tickets.slice(0, DONE_PREVIEW) : column.tickets;
  const empty = column.tickets.length === 0;
  const classes = ["column", column.attention && "attention", empty && "empty", isDone && "done"];
  // A rail writes vertically, which would turn the icon on its side.
  const label = column.icon && !empty ? `${column.icon} ${column.status}` : column.status;
  const cards = tickets.map((ticket) => {
    const moved = previous !== null && previous.get(ticket.path) !== column.status;
    return renderCard(ticket, moved, isOther);
  });
  const hidden = column.tickets.length - tickets.length;
  const showAll = hidden > 0
    ? h("button", {
        class: "show-all",
        type: "button",
        on: { click: () => { state.showAllDone = true; renderBoard(); } },
      }, `Tout afficher (${hidden} de plus)`)
    : null;
  return h(
    "section",
    { class: classes.filter(Boolean).join(" "), "aria-label": `${column.status}, ${column.tickets.length} tickets`, "data-status": column.status },
    h(
      "h2",
      { class: "column-head" },
      h("span", { class: "mono" }, label),
      h("span", { class: "count" }, String(column.tickets.length)),
    ),
    h("div", { class: "column-body" }, cards, showAll),
  );
}

function renderBoard(previous = null) {
  const scrollX = el.board.scrollLeft;
  const scrolls = new Map(
    [...el.board.querySelectorAll(".column")].map((c) => [c.dataset.status, c.querySelector(".column-body").scrollTop]),
  );
  const focused = document.activeElement?.getAttribute?.("href");

  const total = state.board.columns.reduce((sum, column) => sum + column.tickets.length, 0);
  const columns = state.board.columns.map((column) => renderColumn(column, previous));
  if (total === 0) {
    el.board.replaceChildren(
      ...columns,
      h("p", { class: "board-empty" }, "Aucun ticket dans doc/tickets/. Envoie une remarque au chef, ou tape ", h("code", {}, "/ork:ticket"), "."),
    );
  } else {
    el.board.replaceChildren(...columns);
  }

  el.board.scrollLeft = scrollX;
  for (const column of el.board.querySelectorAll(".column")) {
    column.querySelector(".column-body").scrollTop = scrolls.get(column.dataset.status) ?? 0;
  }
  if (focused) el.board.querySelector(`a[href="${CSS.escape(focused)}"]`)?.focus();
}

async function loadBoard() {
  const text = await fetchJSON("/api/tickets");
  if (text === state.boardText) return;
  state.boardText = text;
  state.board = JSON.parse(text);
  const columnOf = new Map();
  for (const column of state.board.columns) {
    for (const ticket of column.tickets) columnOf.set(ticket.path, column.status);
  }
  renderBoard(state.columnOf);
  state.columnOf = columnOf;
  renderSummary();
}

/* File tree */

function normalize(value) {
  return value.normalize("NFD").replace(/\p{Diacritic}/gu, "").toLowerCase();
}

function isDirOpen(path) {
  if (state.openedDirs.has(path)) return true;
  return !state.closedDirs.has(path);
}

function renderTreeNode(node, filter, current) {
  if (!node.dir) {
    if (filter && !normalize(node.path).includes(filter)) return null;
    return h("a", {
      href: docHref(node.path),
      "aria-current": node.path === current ? "page" : null,
    }, node.name);
  }
  const children = (node.children ?? []).map((child) => renderTreeNode(child, filter, current)).filter(Boolean);
  if (filter && children.length === 0) return null;
  if (node.path === "") return children;
  const containsCurrent = current?.startsWith(node.path + "/");
  const details = h(
    "details",
    { open: filter || containsCurrent || isDirOpen(node.path) },
    h("summary", {}, node.name),
    h("div", {}, children),
  );
  if (!filter) {
    details.addEventListener("toggle", () => {
      if (details.open) {
        state.closedDirs.delete(node.path);
        state.openedDirs.add(node.path);
      } else {
        state.closedDirs.add(node.path);
        state.openedDirs.delete(node.path);
      }
    });
  }
  return details;
}

function renderTree() {
  if (!state.tree) return;
  const filter = normalize(el.filter.value.trim());
  const nodes = renderTreeNode(state.tree, filter, state.docPath) ?? [];
  if (nodes.length === 0) {
    el.tree.replaceChildren(h("p", { class: "tree-none" }, filter ? "Aucun fichier ne correspond." : "Aucun Markdown dans doc/."));
    return;
  }
  el.tree.replaceChildren(...nodes);
}

async function loadTree() {
  const text = await fetchJSON("/api/tree");
  if (text !== state.treeText) {
    state.treeText = text;
    state.tree = JSON.parse(text);
    renderTree();
  }
}

function treeHasFile(node, path) {
  if (!node) return false;
  if (!node.dir) return node.path === path;
  return (node.children ?? []).some((child) => treeHasFile(child, path));
}

/* Document */

const HIDDEN_PROPS = new Set(["id", "title"]);

function findTicket(id) {
  const wanted = id.toLowerCase();
  for (const column of state.board?.columns ?? []) {
    for (const ticket of column.tickets) {
      if (ticket.id.toLowerCase() === wanted) return ticket;
    }
  }
  return null;
}

function statusIcon(status) {
  return state.board?.columns.find((c) => c.status === status)?.icon;
}

function renderPropValue(field) {
  if (field.isList) {
    return (field.list ?? []).map((item) => {
      const target = field.key === "depends_on" ? findTicket(item) : null;
      return target
        ? h("a", { class: "tag", href: docHref(target.path) }, item)
        : h("span", { class: "tag" }, item);
    });
  }
  if (field.key === "status") {
    const icon = statusIcon(field.value);
    return h("span", { class: icon ? "mono waits" : "mono" }, icon ? `${icon} ${field.value}` : field.value);
  }
  const inDoc = field.value.match(/^doc\/(.+\.md)$/);
  if (inDoc) return h("a", { class: "mono", href: docHref(inDoc[1]) }, field.value);
  return h("span", { class: "mono" }, field.value);
}

// A screenshot cited by its path also shows under the paragraph that cites it.
function showCitedImages(prose) {
  const shown = new Set();
  for (const link of prose.querySelectorAll('a[href^="/raw/"]')) {
    const href = link.getAttribute("href");
    link.target = "_blank";
    link.rel = "noopener";
    if (shown.has(href) || !link.querySelector("code")) continue;
    shown.add(href);
    link.closest("p, li")?.append(
      h("a", { class: "capture", href, target: "_blank", rel: "noopener" }, h("img", { src: href, alt: "" })),
    );
  }
}

function renderDoc(doc) {
  const title = doc.fields.find((f) => f.key === "title")?.value;
  const id = doc.fields.find((f) => f.key === "id")?.value;
  const props = doc.fields.filter((f) => !HIDDEN_PROPS.has(f.key) && (f.value || f.list?.length));
  const nodes = [];
  if (title) {
    nodes.push(h("h1", { class: "doc-title" }, id ? h("span", { class: "mono" }, id) : null, title));
  }
  if (props.length) {
    nodes.push(h("dl", { class: "props" }, props.flatMap((f) => [h("dt", {}, f.key), h("dd", {}, renderPropValue(f))])));
  }
  const prose = h("div", { class: "prose" });
  prose.innerHTML = doc.html;
  showCitedImages(prose);
  nodes.push(prose);
  el.doc.replaceChildren(...nodes);
  state.docTitle = `${title ?? doc.path.split("/").pop()} · ork`;
}

function scrollToAnchor(anchor) {
  const scroller = NARROW.matches ? document.scrollingElement : el.doc;
  if (!anchor) return;
  const target = el.doc.querySelector(`[id="${CSS.escape(anchor)}"]`);
  if (target) target.scrollIntoView({ block: "start" });
  else scroller.scrollTop = 0;
}

function showMissing(path) {
  el.doc.replaceChildren(
    h("p", { class: "doc-missing" }, h("code", {}, path), " n'existe pas ou plus. Choisis un autre fichier dans la liste."),
  );
  state.docTitle = "ork";
  state.docPathLoaded = null;
  state.lastDoc = null;
  el.navDocs.href = "#/doc";
}

async function loadDoc(path, anchor, isNavigation) {
  const url = "/api/doc?path=" + encodeURIComponent(path);
  let text;
  try {
    text = await fetchJSON(url);
  } catch (error) {
    if (error.status !== 404) throw error;
    // A ticket moves to done/ when it closes, possibly while it is on screen.
    if (path.startsWith("tickets/") && !path.startsWith("tickets/done/")) {
      const archived = "tickets/done/" + path.slice("tickets/".length);
      try {
        await fetchJSON("/api/doc?path=" + encodeURIComponent(archived));
        location.replace(docHref(archived));
        return;
      } catch {
        // Falls through to the message below.
      }
    }
    state.docText = "";
    showMissing(path);
    document.title = state.docTitle;
    return;
  }
  const changed = text !== state.docText || path !== state.docPathLoaded;
  if (changed) {
    const scroller = NARROW.matches ? document.scrollingElement : el.doc;
    const scroll = scroller.scrollTop;
    state.docText = text;
    state.docPathLoaded = path;
    renderDoc(JSON.parse(text));
    scroller.scrollTop = isNavigation ? 0 : scroll;
  }
  document.title = state.docTitle;
  if (isNavigation) scrollToAnchor(anchor);
}

/* Views */

function setNav(view) {
  el.navBoard.toggleAttribute("aria-current", false);
  el.navDocs.toggleAttribute("aria-current", false);
  (view === "board" ? el.navBoard : el.navDocs).setAttribute("aria-current", "page");
  el.navDocs.href = state.lastDoc ? docHref(state.lastDoc) : "#/doc";
  el.viewBoard.hidden = view !== "board";
  el.viewDocs.hidden = view !== "docs";
}

async function route(isNavigation = true) {
  const current = parseRoute();
  setNav(current.view);
  try {
    await loadBoard();
    if (current.view === "board") {
      document.title = "ork";
      state.docPath = null;
    } else {
      await loadTree();
      let path = current.path;
      if (!path) {
        path = state.lastDoc && treeHasFile(state.tree, state.lastDoc) ? state.lastDoc : null;
        if (!path && treeHasFile(state.tree, "ork.md")) path = "ork.md";
        if (path) {
          location.replace(docHref(path));
          return;
        }
        el.doc.replaceChildren(h("p", { class: "doc-missing" }, "Choisis un fichier dans la liste."));
      } else {
        const opened = path !== state.docPath;
        state.docPath = path;
        state.lastDoc = path;
        el.navDocs.href = docHref(path);
        await loadDoc(path, current.anchor, isNavigation);
        if (opened) {
          renderTree();
          if (NARROW.matches && isNavigation) el.treePanel.open = false;
          el.tree.querySelector('[aria-current="page"]')?.scrollIntoView({ block: "nearest" });
        }
      }
    }
    setOffline(false);
  } catch {
    setOffline(true);
  }
}

el.filter.addEventListener("input", renderTree);
// The panel has no toggle on wide screens: never leave it closed there.
NARROW.addEventListener("change", (event) => {
  el.treePanel.open = !event.matches;
});
window.addEventListener("hashchange", () => route(true));

function listen() {
  const events = new EventSource("/api/events");
  let lost = false;
  events.addEventListener("open", () => {
    setOffline(false);
    if (lost) route(false);
    lost = false;
  });
  events.addEventListener("changed", () => route(false));
  events.addEventListener("error", () => {
    lost = true;
    setOffline(true);
  });
}

async function start() {
  if (NARROW.matches) el.treePanel.open = false;
  try {
    const health = JSON.parse(await fetchJSON("/api/health"));
    el.project.textContent = health.repo.split("/").pop();
  } catch {
    setOffline(true);
  }
  await route(true);
  listen();
}

start();
