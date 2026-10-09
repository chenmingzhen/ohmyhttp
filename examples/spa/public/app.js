const pages = {
  "/": {
    title: "SPA 示例",
    summary: "前端页面和 /api 接口由同一个 ohmyhttp 服务提供。",
  },
  "/about": {
    title: "关于这个示例",
    summary: "直接刷新 /about 时，服务端会回退到 index.html，再由前端完成路由渲染。",
  },
};

function renderPage() {
  const page = pages[location.pathname] ?? {
    title: "前端路由",
    summary: `当前路径是 ${location.pathname}，内容由前端渲染。`,
  };

  document.querySelector("#title").textContent = page.title;
  document.querySelector("#summary").textContent = page.summary;
  document.querySelector("#route").textContent = location.pathname;

  document.querySelectorAll("[data-link]").forEach((link) => {
    link.toggleAttribute("aria-current", link.pathname === location.pathname);
  });
}

async function loadMessage() {
  const output = document.querySelector("#api-result");
  output.textContent = "正在连接后端 API…";

  try {
    const response = await fetch("/api/message");
    const body = await response.json();
    output.textContent = body.message;
  } catch {
    output.textContent = "API 请求失败，请确认服务仍在运行。";
  }
}

document.addEventListener("click", (event) => {
  const link = event.target.closest("a[data-link]");
  if (!link || event.ctrlKey || event.metaKey || event.shiftKey) return;

  event.preventDefault();
  history.pushState(null, "", link.href);
  renderPage();
});

window.addEventListener("popstate", renderPage);
document.querySelector("#reload").addEventListener("click", loadMessage);

renderPage();
loadMessage();
