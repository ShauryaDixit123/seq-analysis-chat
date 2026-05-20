const messagesEl = document.getElementById("messages");
const formEl = document.getElementById("chat-form");
const inputEl = document.getElementById("message-input");
const fileInputEl = document.getElementById("file-input");
const previewEl = document.getElementById("file-preview");
const processTypeEl = document.getElementById("process-type");
const sendBtn = formEl.querySelector(".send-btn");

/** Long timeout — DP on large FASTA files can take a while. */
const REQUEST_TIMEOUT_MS = 120_000;

let pendingFiles = [];

function formatTime(iso) {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatSize(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/**
 * fetch with AbortController timeout.
 * @param {string} url
 * @param {RequestInit} options
 * @param {number} [timeoutMs]
 */
async function fetchWithTimeout(
  url,
  options = {},
  timeoutMs = REQUEST_TIMEOUT_MS,
) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(url, { ...options, signal: controller.signal });
  } catch (err) {
    if (err.name === "AbortError") {
      throw new Error(
        `Request timed out after ${Math.round(timeoutMs / 1000)}s. Try smaller FASTA files.`,
      );
    }
    throw err;
  } finally {
    clearTimeout(timer);
  }
}

function messageSenderClass(sender) {
  if (sender === "assistant") return "assistant";
  return "you";
}

function sortMessages(messages) {
  return [...messages].sort(
    (a, b) => new Date(a.timestamp) - new Date(b.timestamp),
  );
}

/** In-memory store for [][][]int — too large for data-* attributes. */
const dpChartStore = new WeakMap();

/** True when value is [][]int (one DP table). */
function isMatrix2D(value) {
  if (!Array.isArray(value) || !value.length) return false;
  const row = value[0];
  if (!Array.isArray(row)) return false;
  return row.length === 0 || typeof row[0] === "number";
}

/** Normalize attachment payload to [][][]int. */
function normalizeDPData(raw) {
  if (!Array.isArray(raw) || !raw.length) return [];
  if (isMatrix2D(raw[0])) return raw;
  if (isMatrix2D(raw)) return [raw];
  return [];
}

function getDynamicData(att) {
  if (att.data != null) {
    const normalized = normalizeDPData(att.data);
    if (normalized.length) return normalized;
  }
  if (att.kind === "dynamic_result" && att.sequence) {
    try {
      const parsed = JSON.parse(att.sequence);
      const normalized = normalizeDPData(parsed);
      if (normalized.length) return normalized;
    } catch {
      /* legacy JSON in sequence field */
    }
  }
  return null;
}

function matrixToPoints(matrix, maxPoints = 10_000) {
  const points = [];
  if (!isMatrix2D(matrix)) return points;
  for (let row = 0; row < matrix.length; row++) {
    const cols = matrix[row];
    for (let col = 0; col < cols.length; col++) {
      const value = Number(cols[col]);
      if (!Number.isFinite(value)) continue;
      points.push({ x: col, y: row, v: value });
    }
  }
  if (points.length <= maxPoints) return points;
  const step = Math.ceil(points.length / maxPoints);
  return points.filter((_, i) => i % step === 0);
}

const plotlyLayoutBase = {
  paper_bgcolor: "#12151e",
  plot_bgcolor: "#1a1d27",
  font: { color: "#8b93a7", family: "system-ui, -apple-system, sans-serif" },
  margin: { l: 52, r: 40, t: 44, b: 48 },
  autosize: true,
  height: 280,
  xaxis: {
    title: { text: "Column (j)", font: { color: "#8b93a7" } },
    gridcolor: "#2a2f3d",
    zerolinecolor: "#2a2f3d",
    tickfont: { color: "#8b93a7" },
  },
  yaxis: {
    title: { text: "Row (i)", font: { color: "#8b93a7" } },
    gridcolor: "#2a2f3d",
    zerolinecolor: "#2a2f3d",
    tickfont: { color: "#8b93a7" },
  },
  showlegend: false,
};

const plotlyConfig = {
  responsive: true,
  displayModeBar: false,
};

/** Map score → marker diameter (scatter, not heatmap colorscale). */
function scoreToMarkerSize(values, minSize = 0.5, maxSize = 2) {
  if (!values.length) return [];
  const minV = Math.min(...values);
  const maxV = Math.max(...values);
  const span = maxV - minV || 1;
  return values.map((v) => minSize + ((v - minV) / span) * (maxSize - minSize));
}

function matrixToPlotlyTrace(matrix) {
  const points = matrixToPoints(matrix);
  if (!points.length) {
    return {
      type: "scatter",
      mode: "text",
      x: [0],
      y: [0],
      text: ["No DP table data"],
      textfont: { color: "#8b93a7", size: 13 },
    };
  }

  const values = points.map((p) => p.v);

  return {
    type: "scattergl",
    mode: "markers",
    name: "DP scores",
    x: points.map((p) => p.x),
    y: points.map((p) => p.y),
    marker: {
      color: "rgba(91, 141, 239, 0.75)",
      size: scoreToMarkerSize(values),
      sizemode: "diameter",
      sizeref: 1,
      line: { color: "rgba(232, 234, 239, 0.35)", width: 0.5 },
      opacity: 0.85,
    },
    text: points.map((p) => `row ${p.y}, col ${p.x}<br>score: ${p.v}`),
    hovertemplate: "%{text}<extra></extra>",
  };
}

function buildPlotlyLayout(plotEl, title) {
  return {
    paper_bgcolor: plotlyLayoutBase.paper_bgcolor,
    plot_bgcolor: plotlyLayoutBase.plot_bgcolor,
    font: plotlyLayoutBase.font,
    margin: plotlyLayoutBase.margin,
    autosize: true,
    height: plotlyLayoutBase.height,
    width: plotEl.clientWidth > 0 ? plotEl.clientWidth : 400,
    xaxis: { ...plotlyLayoutBase.xaxis },
    yaxis: { ...plotlyLayoutBase.yaxis },
    title: {
      text: title,
      font: { color: "#e8eaef", size: 13 },
      x: 0,
      xanchor: "left",
    },
  };
}

function plotMatrixWithPlotly(plotEl, matrix, title) {
  if (typeof Plotly === "undefined") {
    plotEl.textContent =
      "Plotly failed to load. Check your network connection.";
    return;
  }

  const trace = matrixToPlotlyTrace(matrix);
  const layout = buildPlotlyLayout(plotEl, title);

  const run = () => {
    const promise = plotEl._fullLayout
      ? Plotly.react(plotEl, [trace], layout, plotlyConfig)
      : Plotly.newPlot(plotEl, [trace], layout, plotlyConfig);

    Promise.resolve(promise)
      .then(() => Plotly.Plots.resize(plotEl))
      .catch((err) => {
        console.error("Plotly render failed:", err);
        plotEl.textContent = `Chart error: ${err.message || err}`;
      });
  };

  requestAnimationFrame(() => requestAnimationFrame(run));
}

function plotChartFromStore(plotEl) {
  const store = dpChartStore.get(plotEl);
  if (!store?.data3d?.length) {
    plotEl.textContent = "No chart data.";
    return;
  }
  const idx = store.matrixIndex ?? 0;
  const matrix = store.data3d[idx];
  if (!isMatrix2D(matrix)) {
    plotEl.textContent = "Invalid DP table shape (expected [][]int).";
    return;
  }
  plotMatrixWithPlotly(plotEl, matrix, store.title || "DP table");
}

/** Plot charts after their message node is in the document (Plotly needs layout). */
function scheduleCharts(messageRoot) {
  messageRoot.querySelectorAll(".dynamic-plotly-chart").forEach((plotEl) => {
    plotChartFromStore(plotEl);
  });
}

function refreshAllCharts() {
  document.querySelectorAll(".message").forEach((msg) => scheduleCharts(msg));
}

function renderDynamicChart(container, data3d, att) {
  const wrap = document.createElement("div");
  wrap.className = "dynamic-chart";

  const header = document.createElement("div");
  header.className = "dynamic-chart-header";

  const title = document.createElement("span");
  title.className = "dynamic-chart-title";
  title.textContent = att.name || "DP scatter plot";
  header.appendChild(title);

  if (att.process_type) {
    const badge = document.createElement("span");
    badge.className = "dynamic-chart-badge";
    badge.textContent = att.process_type.replaceAll("_", " ");
    header.appendChild(badge);
  }

  wrap.appendChild(header);

  let matrixIndex = 0;

  const plotEl = document.createElement("div");
  plotEl.className = "dynamic-plotly-chart";
  plotEl.id = `plot-${att.id || crypto.randomUUID()}`;
  plotEl.setAttribute("role", "img");
  plotEl.setAttribute(
    "aria-label",
    "Scatter plot of dynamic programming table values",
  );

  function chartTitle(index, total) {
    return total > 1 ? `Matrix ${index + 1} of ${total}` : "DP table";
  }

  if (data3d.length > 1) {
    const selectWrap = document.createElement("label");
    selectWrap.className = "dynamic-matrix-picker";
    selectWrap.textContent = "Comparison ";
    const select = document.createElement("select");
    data3d.forEach((_, i) => {
      const opt = document.createElement("option");
      opt.value = String(i);
      opt.textContent = `Pair ${i + 1}`;
      select.appendChild(opt);
    });
    selectWrap.appendChild(select);
    wrap.appendChild(selectWrap);

    select.addEventListener("change", () => {
      matrixIndex = Number(select.value);
      updatePlot();
    });
  }

  wrap.appendChild(plotEl);
  container.appendChild(wrap);

  function updatePlot() {
    const title = chartTitle(matrixIndex, data3d.length);
    dpChartStore.set(plotEl, { data3d, matrixIndex, title });
    if (plotEl.isConnected) {
      plotChartFromStore(plotEl);
    }
  }

  dpChartStore.set(plotEl, {
    data3d,
    matrixIndex: 0,
    title: chartTitle(0, data3d.length),
  });

  const ro = new ResizeObserver(() => {
    if (plotEl._fullLayout && typeof Plotly !== "undefined") {
      Plotly.Plots.resize(plotEl);
    }
  });
  ro.observe(plotEl);
}

function renderFileAttachment(ul, att) {
  const li = document.createElement("li");
  if (att.url) {
    const a = document.createElement("a");
    a.href = att.url;
    a.target = "_blank";
    a.rel = "noopener noreferrer";
    a.textContent = att.name || "Attachment";
    const size = document.createElement("span");
    size.className = "attachment-size";
    size.textContent = formatSize(att.size);
    a.appendChild(size);
    li.appendChild(a);
  } else {
    const span = document.createElement("span");
    span.className = "attachment-name";
    span.textContent = att.name || "FASTA file";
    li.appendChild(span);
    if (att.size) {
      const size = document.createElement("span");
      size.className = "attachment-size";
      size.textContent = formatSize(att.size);
      li.appendChild(size);
    }
  }
  ul.appendChild(li);
}

function renderMessage(msg) {
  const isAssistant = msg.sender === "assistant";
  const article = document.createElement("article");
  article.className = `message ${messageSenderClass(msg.sender)}`;
  article.dataset.id = msg.id;

  let hasChart = false;

  if (isAssistant) {
    const label = document.createElement("span");
    label.className = "message-label";
    label.textContent = "Assistant";
    article.appendChild(label);
  }

  const bubble = document.createElement("div");
  bubble.className = "message-bubble";

  if (msg.text) {
    const p = document.createElement("p");
    p.className = "message-text";
    p.textContent = msg.text;
    bubble.appendChild(p);
  }

  if (msg.attachments?.length) {
    const fileAttachments = [];

    for (const att of msg.attachments) {
      const dynamicData = getDynamicData(att);
      if (dynamicData) {
        hasChart = true;
        renderDynamicChart(bubble, dynamicData, att);
      } else {
        fileAttachments.push(att);
      }
    }

    if (fileAttachments.length) {
      const ul = document.createElement("ul");
      ul.className = "message-attachments";
      for (const att of fileAttachments) {
        renderFileAttachment(ul, att);
      }
      bubble.appendChild(ul);
    }
  }

  if (hasChart) {
    article.classList.add("has-chart");
  }

  const meta = document.createElement("time");
  meta.className = "message-meta";
  meta.dateTime = msg.timestamp;
  meta.textContent = formatTime(msg.timestamp);
  bubble.appendChild(meta);

  article.appendChild(bubble);
  return article;
}

function showEmptyState() {
  if (messagesEl.querySelector(".message")) return;
  const p = document.createElement("p");
  p.className = "empty-state";
  p.textContent = "No messages yet. Say hello or attach FASTA files.";
  messagesEl.appendChild(p);
}

function clearEmptyState() {
  messagesEl.querySelector(".empty-state")?.remove();
}

function scrollToBottom() {
  requestAnimationFrame(() => {
    messagesEl.scrollTop = messagesEl.scrollHeight;
    refreshAllCharts();
  });
}

function renderPreview() {
  previewEl.innerHTML = "";
  if (!pendingFiles.length) {
    previewEl.hidden = true;
    return;
  }
  previewEl.hidden = false;
  pendingFiles.forEach((file, index) => {
    const li = document.createElement("li");
    li.textContent = `${file.name} (${formatSize(file.size)}) `;
    const remove = document.createElement("button");
    remove.type = "button";
    remove.setAttribute("aria-label", "Remove file");
    remove.textContent = "×";
    remove.addEventListener("click", () => {
      pendingFiles.splice(index, 1);
      syncFileInput();
      renderPreview();
    });
    li.appendChild(remove);
    previewEl.appendChild(li);
  });
}

function syncFileInput() {
  const dt = new DataTransfer();
  pendingFiles.forEach((f) => dt.items.add(f));
  fileInputEl.files = dt.files;
}

function isDynamicMode() {
  return processTypeEl && processTypeEl.value !== "none";
}

fileInputEl.addEventListener("change", () => {
  pendingFiles = [...fileInputEl.files];
  renderPreview();
});

inputEl.addEventListener("input", () => {
  inputEl.style.height = "auto";
  inputEl.style.height = `${Math.min(inputEl.scrollHeight, 120)}px`;
});

inputEl.addEventListener("keydown", (e) => {
  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault();
    formEl.requestSubmit();
  }
});

formEl.addEventListener("submit", async (e) => {
  e.preventDefault();
  const text = inputEl.value.trim();
  if (!text && !pendingFiles.length) return;

  sendBtn.disabled = true;
  const formData = new FormData();
  if (text) formData.append("message", text);
  pendingFiles.forEach((f) => formData.append("files", f));

  const processType = processTypeEl?.value ?? "dynamic_programming";
  const dynamic = isDynamicMode();

  try {
    let sessionID = localStorage.getItem("session_id");
    if (!sessionID) {
      const sessionRes = await fetchWithTimeout("/api/chat/session", {
        method: "POST",
      });
      if (!sessionRes.ok) return;
      const sessionData = await sessionRes.json();
      sessionID = sessionData.session.id;
      localStorage.setItem("session_id", sessionID);
    }

    const params = new URLSearchParams({
      session_id: sessionID,
      process_type: processType,
    });

    const res = await fetchWithTimeout(`/api/chat?${params}`, {
      method: "POST",
      body: formData,
    });

    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      alert(err.error || "Failed to send message");
      return;
    }
    const data = await res.json();
    clearEmptyState();
    for (const msg of sortMessages(data.messages)) {
      if (messagesEl.querySelector(`[data-id="${msg.id}"]`)) continue;
      const el = renderMessage(msg);
      messagesEl.appendChild(el);
      scheduleCharts(el);
    }
    scrollToBottom();

    inputEl.value = "";
    inputEl.style.height = "auto";
    pendingFiles = [];
    fileInputEl.value = "";
    renderPreview();
  } catch (err) {
    alert(err.message || "Network error. Is the server running?");
  } finally {
    sendBtn.disabled = false;
  }
});

async function loadMessages() {
  try {
    const sessionID = localStorage.getItem("session_id");
    if (!sessionID) return;
    const res = await fetchWithTimeout(
      `/api/chat/session/${sessionID}`,
      {},
      30_000,
    );
    if (!res.ok) return;
    const data = await res.json();
    const messages = data.messages;
    if (!messages?.length) {
      showEmptyState();
      return;
    }
    clearEmptyState();
    for (const msg of sortMessages(messages)) {
      const el = renderMessage(msg);
      messagesEl.appendChild(el);
      scheduleCharts(el);
    }
    scrollToBottom();
  } catch {
    showEmptyState();
  }
}

loadMessages();
