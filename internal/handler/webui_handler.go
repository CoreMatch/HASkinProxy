package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// WebUIHandler serves the resources consumed by the HRPAuth WEBUI through
// the microservice SDK mechanism (see HA-Contract microservices.md):
//   - GET /sdk/haskinproxy.js : the SDK JS file relayed by HRPAuth via
//     GET /services/sdk/HASkinProxy; it declares a Dashboard menu item.
//   - GET /customskinloader   : the CustomSkinLoader setup page embedded
//     in the Dashboard via iframe. It is reached through HRPAuth's relay
//     ({BackendUrl}/customskinloader), so the proxy must register a relay
//     rule on startup.
type WebUIHandler struct{}

func NewWebUIHandler() *WebUIHandler {
	return &WebUIHandler{}
}

// sdkJS declares the WEBUI SDK global object. It reads the backend
// callback URL from window.__BACKEND_URL__, which HRPAuth injects into
// the page before loading microservice SDKs. dashboard.url is formed by
// appending /customskinloader to that URL. If __BACKEND_URL__ is absent,
// dashboard stays absent so the WEBUI simply does not render the menu
// item.
const sdkJS = `(function () {
  var base = window.__BACKEND_URL__ || '';
  var sdk = {
    name: 'HASkinProxy',
    version: '1.0.0',
    dashboard: { label: 'CustomSkinLoader', url: base.replace(/\/+$/, '') + '/customskinloader' },
  };
  window['HASkinProxy-sdk'] = sdk;
})();
`

// GetSDKJS handles GET /sdk/haskinproxy.js.
// The response must be served with Content-Type application/javascript so
// HRPAuth relays it unchanged to the WEBUI.
func (h *WebUIHandler) GetSDKJS(c *gin.Context) {
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.String(http.StatusOK, sdkJS)
}

// GetCSLPage handles GET /customskinloader. It renders a self-contained
// setup page: a CustomSkinLoader config.json generator (the API root is
// derived from the main service's callback URL + /csl/, the relay dest)
// plus usage instructions.
func (h *WebUIHandler) GetCSLPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, cslPage)
}

const cslPage = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>CustomSkinLoader 配置</title>
<style>
  :root { color-scheme: light dark; }
  body { font-family: system-ui, -apple-system, "Segoe UI", sans-serif; margin: 0; padding: 24px; }
  h1 { font-size: 1.25rem; margin: 0 0 16px; }
  h2 { font-size: 1rem; margin: 24px 0 8px; }
  .card { border: 1px solid var(--border, #d0d7de); border-radius: 8px; padding: 16px; margin-bottom: 16px; }
  label { display: block; font-size: 0.85rem; margin-bottom: 6px; }
  input { width: 100%; box-sizing: border-box; padding: 8px; border: 1px solid var(--border, #d0d7de); border-radius: 6px; background: transparent; color: inherit; }
  pre { background: rgba(127,127,127,.12); border-radius: 6px; padding: 12px; overflow-x: auto; font-size: 0.8rem; }
  .actions { display: flex; gap: 8px; margin-top: 12px; }
  button { padding: 8px 14px; border: none; border-radius: 6px; cursor: pointer; background: #1976d2; color: #fff; }
  button.secondary { background: transparent; border: 1px solid var(--border, #d0d7de); color: inherit; }
  ol { padding-left: 20px; margin: 8px 0; }
  li { margin: 4px 0; }
  code { background: rgba(127,127,127,.15); padding: 1px 5px; border-radius: 4px; font-size: 0.85em; }
</style>
</head>
<body>
  <h1>CustomSkinLoader 接入配置</h1>

  <div class="card">
    <h2>1. 生成配置文件</h2>
    <label for="api-root">CustomSkinAPI 根地址（必须以 / 结尾）</label>
    <input id="api-root" type="text" spellcheck="false">
    <pre id="config-preview"></pre>
    <div class="actions">
      <button id="btn-copy">复制 config.json</button>
      <button id="btn-download" class="secondary">下载 config.json</button>
    </div>
  </div>

  <div class="card">
    <h2>2. 使用说明</h2>
    <ol>
      <li>安装 <a href="https://github.com/xfl03/MCCustomSkinLoader" target="_blank" rel="noopener">CustomSkinLoader</a> 到你的 Minecraft 版本目录（建议使用官方或兼容的加载器）。</li>
      <li>将生成的 <code>config.json</code> 放入 <code>.minecraft/customskinloader/</code> 目录（没有则新建）。</li>
      <li>启动游戏，登录后皮肤与披风将通过本服务自动加载。</li>
      <li>若其他玩家的皮肤加载异常，请确认其客户端可以访问上面的根地址。</li>
    </ol>
  </div>

<script>
(function () {
  var rootInput = document.getElementById('api-root');
  var preview = document.getElementById('config-preview');
  var copied = false;

  // 本代理的 CSL API 经主服务 relay 到 /csl/（dest=/csl，见 HA-Contract
  // 微服务约定）。默认根地址取 window.__BACKEND_URL__（由 HRPAuth 注入）+
  // /csl/；缺失时回退到当前页面 origin + /csl/。
  var base = window.__BACKEND_URL__ || '';
  var defaultRoot = base.replace(/\/+$/, '') + '/csl/' || location.origin + '/csl/';

  rootInput.value = defaultRoot;

  function buildConfig() {
    var root = rootInput.value.trim();
    if (root && !root.endsWith('/')) root += '/';
    return JSON.stringify({
      enable: true,
      loadlist: [
        {
          name: 'HASkinProxy',
          type: 'CustomSkinAPI',
          root: root || defaultRoot
        }
      ]
    }, null, 2);
  }

  function render() {
    preview.textContent = buildConfig();
  }
  rootInput.addEventListener('input', render);
  render();

  document.getElementById('btn-copy').addEventListener('click', function () {
    navigator.clipboard.writeText(buildConfig()).then(function () {
      var btn = document.getElementById('btn-copy');
      btn.textContent = '已复制!';
      if (copied) return;
      copied = true;
      setTimeout(function () { btn.textContent = '复制 config.json'; copied = false; }, 1500);
    });
  });

  document.getElementById('btn-download').addEventListener('click', function () {
    var blob = new Blob([buildConfig()], { type: 'application/json' });
    var a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'config.json';
    a.click();
    URL.revokeObjectURL(a.href);
  });
})();
</script>
</body>
</html>
`
