const state = {
  backend: null,
  overview: null,
  knowledge: [],
  filter: "",
  filePath: "",
  appendDrafts: {},
  openAppendId: "",
  chat: [
    {
      role: "assistant",
      text: "桌面前端已接入。你可以在这里导入图片/PDF、直接管理记忆，或者像聊天一样继续使用现有命令。",
      time: nowLabel(),
    },
  ],
};

const promptExamples = [
  "记住：Windows 版先把桌面前端做稳",
  "/debug-search macOS 什么时候做？",
  "两小时后提醒我喝水",
  "现在我记了什么？",
];

document.addEventListener("DOMContentLoaded", () => {
  void init();
});

async function init() {
  bindStaticEvents();
  renderPrompts();
  renderChat();
  renderKnowledge();

  try {
    state.backend = await waitForBackend();
    bindRuntimeEvents();
    await refreshAll();
  } catch (error) {
    showBanner(asMessage(error), true);
  }
}

function bindStaticEvents() {
  document.getElementById("browse-file").addEventListener("click", () => {
    void browseFile();
  });

  document.getElementById("import-file").addEventListener("click", () => {
    void importFile();
  });

  document.getElementById("file-path").addEventListener("input", (event) => {
    state.filePath = event.target.value;
  });

  document.getElementById("memory-form").addEventListener("submit", (event) => {
    event.preventDefault();
    void createKnowledge();
  });

  document.getElementById("memory-filter").addEventListener("input", (event) => {
    state.filter = event.target.value.trim().toLowerCase();
    renderKnowledge();
  });

  document.getElementById("clear-memory").addEventListener("click", () => {
    void clearKnowledge();
  });

  document.getElementById("chat-form").addEventListener("submit", (event) => {
    event.preventDefault();
    void sendMessage();
  });

  document.getElementById("memory-list").addEventListener("click", (event) => {
    const target = event.target.closest("[data-action]");
    if (!target) {
      return;
    }

    const id = target.dataset.id || "";
    switch (target.dataset.action) {
      case "toggle-append":
        state.openAppendId = state.openAppendId === id ? "" : id;
        renderKnowledge();
        break;
      case "delete":
        void deleteKnowledge(id);
        break;
      case "save-append":
        void appendKnowledge(id);
        break;
      default:
        break;
    }
  });

  document.getElementById("memory-list").addEventListener("input", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLTextAreaElement) || !target.dataset.id) {
      return;
    }
    state.appendDrafts[target.dataset.id] = target.value;
  });
}

function bindRuntimeEvents() {
  if (!window.runtime || typeof window.runtime.EventsOn !== "function") {
    return;
  }

  window.runtime.EventsOn("reminder:due", (payload) => {
    const reminder = Array.isArray(payload) ? payload[0] : payload;
    if (!reminder) {
      return;
    }

    const shortId = reminder.shortId || reminder.id || "notice";
    const message = reminder.message || "提醒触发";
    state.chat.push({
      role: "system",
      text: `[提醒 #${shortId}] ${message}`,
      time: nowLabel(),
    });
    renderChat();
    showBanner(`提醒 #${shortId}: ${message}`, false);
  });
}

async function waitForBackend() {
  for (let index = 0; index < 80; index += 1) {
    const backend = window.go && window.go.main && window.go.main.DesktopApp;
    if (backend) {
      return backend;
    }
    await delay(50);
  }
  throw new Error("Wails 后端尚未就绪。");
}

async function refreshAll() {
  await Promise.all([refreshOverview(), refreshKnowledge()]);
}

async function refreshOverview() {
  state.overview = await state.backend.GetOverview();
  document.getElementById("data-dir").textContent = state.overview.dataDir;
  document.getElementById("ai-status").textContent = state.overview.aiAvailable ? "已配置" : "未配置";
  document.getElementById("ai-message").textContent = state.overview.aiMessage;
  document.getElementById("memory-count").textContent = String(state.overview.knowledgeCount);
}

async function refreshKnowledge() {
  state.knowledge = await state.backend.ListKnowledge();
  renderKnowledge();
}

async function browseFile() {
  try {
    const selected = await state.backend.OpenImportDialog();
    if (!selected) {
      return;
    }
    state.filePath = selected;
    document.getElementById("file-path").value = selected;
  } catch (error) {
    showBanner(asMessage(error), true);
  }
}

async function importFile() {
  if (!state.filePath.trim()) {
    showBanner("请先选择文件。", true);
    return;
  }

  try {
    const result = await state.backend.ImportFile(state.filePath);
    document.getElementById("file-path").value = "";
    state.filePath = "";
    await refreshAll();
    showBanner(result.message, false);
    state.chat.push({
      role: "system",
      text: `${result.message}\n${result.item.preview}`,
      time: nowLabel(),
    });
    renderChat();
  } catch (error) {
    showBanner(asMessage(error), true);
  }
}

async function createKnowledge() {
  const input = document.getElementById("memory-input");
  const text = input.value.trim();
  if (!text) {
    showBanner("请输入要保存的记忆内容。", true);
    return;
  }

  try {
    const result = await state.backend.CreateKnowledge(text);
    input.value = "";
    await refreshAll();
    showBanner(result.message, false);
  } catch (error) {
    showBanner(asMessage(error), true);
  }
}

async function appendKnowledge(id) {
  const draft = (state.appendDrafts[id] || "").trim();
  if (!draft) {
    showBanner("请输入补充内容。", true);
    return;
  }

  try {
    const result = await state.backend.AppendKnowledge(id, draft);
    state.appendDrafts[id] = "";
    state.openAppendId = "";
    await refreshAll();
    showBanner(result.message, false);
  } catch (error) {
    showBanner(asMessage(error), true);
  }
}

async function deleteKnowledge(id) {
  try {
    const ok = await state.backend.ConfirmAction("删除记忆", `确认删除 #${id.slice(0, 8)} 吗？`);
    if (!ok) {
      return;
    }
    const result = await state.backend.DeleteKnowledge(id);
    await refreshAll();
    showBanner(result.message, false);
  } catch (error) {
    showBanner(asMessage(error), true);
  }
}

async function clearKnowledge() {
  try {
    const ok = await state.backend.ConfirmAction("清空知识库", "确认清空全部记忆吗？这个动作不可撤销。");
    if (!ok) {
      return;
    }
    const result = await state.backend.ClearKnowledge();
    await refreshAll();
    showBanner(result.message, false);
  } catch (error) {
    showBanner(asMessage(error), true);
  }
}

async function sendMessage() {
  const input = document.getElementById("chat-input");
  const text = input.value.trim();
  if (!text) {
    return;
  }

  state.chat.push({ role: "user", text, time: nowLabel() });
  renderChat();
  input.value = "";

  try {
    const result = await state.backend.SendMessage(text);
    state.chat.push({
      role: "assistant",
      text: result.reply,
      time: result.timestamp || nowLabel(),
    });
    renderChat();
    await refreshAll();
  } catch (error) {
    state.chat.push({
      role: "system",
      text: asMessage(error),
      time: nowLabel(),
    });
    renderChat();
    showBanner(asMessage(error), true);
  }
}

function renderPrompts() {
  const container = document.getElementById("prompt-row");
  container.innerHTML = promptExamples
    .map(
      (prompt) =>
        `<button class="prompt-chip" type="button" data-prompt="${escapeAttribute(prompt)}">${escapeHTML(prompt)}</button>`,
    )
    .join("");

  container.querySelectorAll("[data-prompt]").forEach((button) => {
    button.addEventListener("click", () => {
      document.getElementById("chat-input").value = button.dataset.prompt || "";
      document.getElementById("chat-input").focus();
    });
  });
}

function renderKnowledge() {
  const container = document.getElementById("memory-list");
  const filtered = state.knowledge.filter((item) => {
    if (!state.filter) {
      return true;
    }
    const haystack = [
      item.id,
      item.shortId,
      item.source,
      item.text,
      ...(item.keywords || []),
    ]
      .join(" ")
      .toLowerCase();
    return haystack.includes(state.filter);
  });

  if (filtered.length === 0) {
    container.innerHTML = `<div class="empty-state">当前没有符合条件的记忆。</div>`;
    return;
  }

  container.innerHTML = filtered
    .map((item) => {
      const isOpen = state.openAppendId === item.id;
      const keywords = (item.keywords || [])
        .slice(0, 6)
        .map((keyword) => `<span class="memory-badge">${escapeHTML(keyword)}</span>`)
        .join("");
      return `
        <article class="memory-card">
          <div class="memory-top">
            <div>
              <div class="memory-meta">
                <span class="memory-badge memory-id">#${escapeHTML(item.shortId)}</span>
                ${item.isFile ? `<span class="memory-badge">文件摘要</span>` : ""}
                ${item.source ? `<span class="memory-badge">${escapeHTML(item.source)}</span>` : ""}
                <span class="memory-badge">${escapeHTML(item.recordedAt)}</span>
              </div>
            </div>
          </div>
          <p class="memory-preview">${escapeHTML(item.preview)}</p>
          ${keywords ? `<div class="memory-meta">${keywords}</div>` : ""}
          <div class="memory-actions">
            <button class="inline-button" type="button" data-action="toggle-append" data-id="${escapeAttribute(item.id)}">
              ${isOpen ? "收起补充" : "补充记忆"}
            </button>
            <button class="inline-button" type="button" data-action="delete" data-id="${escapeAttribute(item.id)}">
              删除
            </button>
          </div>
          ${
            isOpen
              ? `
                <div class="append-box">
                  <textarea rows="3" data-id="${escapeAttribute(item.id)}" placeholder="补充这一条记忆的新增事实。">${escapeHTML(state.appendDrafts[item.id] || "")}</textarea>
                  <button class="primary-button" type="button" data-action="save-append" data-id="${escapeAttribute(item.id)}">保存补充</button>
                </div>
              `
              : ""
          }
          <details class="details">
            <summary>查看完整内容</summary>
            <pre>${escapeHTML(item.text)}</pre>
          </details>
        </article>
      `;
    })
    .join("");
}

function renderChat() {
  const container = document.getElementById("chat-list");
  container.innerHTML = state.chat
    .map(
      (message) => `
        <div class="bubble ${escapeAttribute(message.role)}">
          ${escapeHTML(message.text)}
          <span class="bubble-time">${escapeHTML(message.time)}</span>
        </div>
      `,
    )
    .join("");
  container.scrollTop = container.scrollHeight;
}

let bannerTimer = 0;

function showBanner(message, isError) {
  const banner = document.getElementById("banner");
  banner.hidden = false;
  banner.textContent = message;
  banner.style.background = isError ? "rgba(128, 40, 30, 0.92)" : "rgba(42, 28, 18, 0.88)";

  window.clearTimeout(bannerTimer);
  bannerTimer = window.setTimeout(() => {
    banner.hidden = true;
  }, 3200);
}

function delay(ms) {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

function nowLabel() {
  return new Date().toLocaleString("zh-CN", {
    hour12: false,
  });
}

function asMessage(error) {
  if (!error) {
    return "发生未知错误。";
  }
  if (typeof error === "string") {
    return error;
  }
  if (error.message) {
    return error.message;
  }
  return String(error);
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function escapeAttribute(value) {
  return escapeHTML(value).replaceAll("`", "&#96;");
}
