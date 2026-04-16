package server

const adminHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Bigbat Admin</title>
  <style>
    :root {
      --bg: #f5f7fb;
      --card: #ffffff;
      --text: #0f172a;
      --sub: #475569;
      --accent: #0ea5e9;
      --accent2: #0f766e;
      --border: #dbe4ef;
      --danger: #b91c1c;
      --ok: #15803d;
    }
    body {
      margin: 0;
      font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Noto Sans CJK SC", sans-serif;
      background: radial-gradient(circle at 10% 0%, #e0f2fe 0%, var(--bg) 40%);
      color: var(--text);
    }
    .wrap { max-width: 1100px; margin: 24px auto; padding: 0 16px 40px; }
    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 16px;
      box-shadow: 0 6px 24px rgba(2, 6, 23, 0.05);
    }
    h1 { margin: 0 0 16px; font-size: 24px; }
    h2 { margin: 0 0 12px; font-size: 18px; }
    p, li, label { color: var(--sub); }
    textarea, input {
      width: 100%;
      box-sizing: border-box;
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 10px;
      font-size: 14px;
      background: #f8fafc;
      color: var(--text);
    }
    textarea { min-height: 110px; }
    button {
      border: 0;
      border-radius: 10px;
      padding: 10px 14px;
      font-weight: 600;
      cursor: pointer;
      color: white;
      background: linear-gradient(120deg, var(--accent), var(--accent2));
    }
    button.secondary { background: #475569; }
    button.btn-sm { padding: 6px 10px; font-size: 12px; }
    .row { display: flex; gap: 10px; flex-wrap: wrap; }
    .row > * { min-width: 0; }
    .muted { font-size: 12px; color: #64748b; }
    .mono {
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      overflow-wrap: anywhere;
      word-break: break-word;
      white-space: pre-wrap;
    }
    .status { font-weight: 600; margin-top: 10px; }
    .ok { color: var(--ok); }
    .err { color: var(--danger); }
    table { width: 100%; border-collapse: collapse; }
    th, td {
      border-bottom: 1px solid var(--border);
      text-align: left;
      padding: 8px;
      font-size: 13px;
      overflow-wrap: anywhere;
      word-break: break-word;
      white-space: normal;
    }
    #state {
      max-height: 300px;
      overflow: auto;
      overflow-wrap: anywhere;
      word-break: break-word;
    }
    #cookies {
      min-height: 180px;
      resize: vertical;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-word;
    }
    .cookie-health {
      margin-top: 10px;
      border: 1px solid var(--border);
      border-radius: 10px;
      max-height: 220px;
      overflow: auto;
      background: #f8fafc;
      padding: 8px;
    }
    .cookie-health-item {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      gap: 8px;
      align-items: center;
      border-bottom: 1px dashed #cdd8e6;
      padding: 8px 4px;
    }
    .cookie-line {
      max-height: 5.4em;
      overflow: auto;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-word;
      padding-right: 4px;
    }
    .cookie-health-item:last-child { border-bottom: 0; }
    .debug-wrap {
      margin-top: 8px;
      background: #0f172a;
      color: #e2e8f0;
      border-radius: 8px;
      padding: 8px;
      font-size: 11px;
      max-height: 180px;
      overflow: auto;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-word;
    }
    .pill {
      border-radius: 999px;
      padding: 2px 8px;
      font-size: 12px;
      font-weight: 700;
      color: white;
      text-transform: uppercase;
      letter-spacing: .02em;
      white-space: nowrap;
    }
    .pill.healthy { background: #15803d; }
    .pill.expired { background: #b91c1c; }
    .pill.limited { background: #b45309; }
    .pill.blocked { background: #7c2d12; }
    .pill.error { background: #334155; }
    .legend {
      margin-top: 8px;
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      align-items: center;
    }
    .legend .muted { font-size: 11px; }
    .table-actions {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
    }
    .guide-box {
      margin-top: 10px;
      padding-top: 10px;
      border-top: 1px dashed var(--border);
    }
    .guide-grid {
      display: grid;
      grid-template-columns: 140px 1fr auto;
      gap: 8px;
      align-items: center;
      margin-top: 6px;
    }
    .guide-key {
      font-size: 12px;
      color: #64748b;
    }
    .guide-val {
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      background: #f8fafc;
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 6px 8px;
      overflow-wrap: anywhere;
      word-break: break-word;
    }
    .models-scroll {
      border: 1px solid var(--border);
      border-radius: 10px;
      max-height: 360px;
      overflow: auto;
    }
    .models-scroll table {
      margin: 0;
    }
    .models-scroll thead th {
      position: sticky;
      top: 0;
      background: #f1f5f9;
      z-index: 1;
    }
    .quick-copy {
      margin-top: 10px;
      padding-top: 10px;
      border-top: 1px dashed var(--border);
    }
    .proto-box {
      margin-top: 10px;
      padding-top: 10px;
      border-top: 1px dashed var(--border);
    }
    .proto-item {
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 8px;
      margin-top: 8px;
      background: #f8fafc;
    }
    .app-badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      background: linear-gradient(120deg, #0369a1, #0f766e);
      color: #fff;
      border-radius: 999px;
      padding: 6px 12px;
      font-size: 12px;
      font-weight: 700;
      margin-bottom: 10px;
    }
    @media (max-width: 900px) { .grid { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="app-badge">Big Bat <span id="appUptime">启动中...</span></div>
    <h1>Bigbat 管理后台</h1>
    <div class="grid">
      <section class="card">
        <h2>连接与状态</h2>
        <label>API Key</label>
        <input id="apiKey" type="password" placeholder="与你的 API_SECRET 一致" autocomplete="off" />
        <div class="row" style="margin-top:10px;">
          <button class="secondary" onclick="saveApiKey()">记住 Key</button>
          <button onclick="refreshState()">刷新状态</button>
        </div>
        <div id="apiStatus" class="status"></div>
        <pre id="state" class="mono"></pre>

        <div class="proto-box">
          <div class="muted">协议兼容状态</div>
          <div class="proto-item">
            <div><strong>OpenAI 兼容</strong>: /v1/chat/completions, /v1/models</div>
            <div class="row" style="margin-top:6px;">
              <button class="btn-sm secondary" onclick="copyQuickChat(false)">复制 OpenAI 示例</button>
              <button class="btn-sm secondary" onclick="checkOpenAICompatibility()">检测 OpenAI 探针</button>
            </div>
            <div id="openaiCompatStatus" class="status"></div>
          </div>
          <div class="proto-item">
            <div><strong>Anthropic 兼容</strong>: /v1/messages, /v1/messages/health</div>
            <div class="row" style="margin-top:6px;">
              <button class="btn-sm secondary" onclick="copyQuickAnthropic()">复制 Anthropic 示例</button>
              <button class="btn-sm secondary" onclick="checkAnthropicCompatibility()">检测 Anthropic 探针</button>
            </div>
            <div id="anthropicCompatStatus" class="status"></div>
          </div>
        </div>

        <div class="guide-box">
          <div class="muted">接入参数（复制到客户端）</div>
          <div class="guide-grid">
            <div class="guide-key">Base URL</div>
            <div id="guideBaseUrl" class="guide-val"></div>
            <button class="btn-sm secondary" onclick="copyGuideValue('guideBaseUrl', 'quickCopyStatus', '已复制 Base URL')">复制</button>
          </div>
          <div class="guide-grid">
            <div class="guide-key">Anthropic URL</div>
            <div id="guideAnthropicUrl" class="guide-val"></div>
            <button class="btn-sm secondary" onclick="copyGuideValue('guideAnthropicUrl', 'quickCopyStatus', '已复制 Anthropic URL')">复制</button>
          </div>
          <div class="guide-grid">
            <div class="guide-key">Anthropic Health</div>
            <div id="guideAnthropicHealthUrl" class="guide-val"></div>
            <button class="btn-sm secondary" onclick="copyGuideValue('guideAnthropicHealthUrl', 'quickCopyStatus', '已复制 Anthropic Health URL')">复制</button>
          </div>
          <div class="guide-grid">
            <div class="guide-key">API Key(Token)</div>
            <div id="guideApiKey" class="guide-val"></div>
            <button class="btn-sm secondary" onclick="copyGuideValue('guideApiKey', 'quickCopyStatus', '已复制 API Key')">复制</button>
          </div>
          <div class="guide-grid">
            <div class="guide-key">默认模型</div>
            <div id="guideDefaultModel" class="guide-val"></div>
            <button class="btn-sm secondary" onclick="copyGuideValue('guideDefaultModel', 'quickCopyStatus', '已复制默认模型')">复制</button>
          </div>
          <div class="guide-grid">
            <div class="guide-key">Opus4.6 模型名</div>
            <div id="guideOpusModel" class="guide-val"></div>
            <button class="btn-sm secondary" onclick="copyGuideValue('guideOpusModel', 'quickCopyStatus', '已复制 Opus 模型名')">复制</button>
          </div>
        </div>

        <div class="quick-copy">
          <div class="muted">常用调用（可直接复制）</div>
          <div class="row" style="margin-top:8px;">
            <button class="btn-sm" onclick="copyQuickChat(false)">复制 Opus4.6 非流式</button>
            <button class="btn-sm secondary" onclick="copyQuickChat(true)">复制 Opus4.6 流式</button>
            <button class="btn-sm secondary" onclick="copyQuickModels()">复制 models 调用</button>
          </div>
          <div id="quickCopyStatus" class="status"></div>
        </div>
      </section>

      <section class="card">
        <h2>Cookies 管理</h2>
        <p class="muted">每行一个 cookie（可只填 session_id 值，后端会自动补齐前缀）。</p>
        <textarea id="cookies"></textarea>
        <div class="row" style="margin-top:10px;">
          <button onclick="saveCookies()">保存 Cookies</button>
          <button class="secondary" onclick="loadCookies()">加载 Cookies</button>
          <button class="secondary" onclick="checkCookiesHealth()">健康检查</button>
          <button class="secondary" onclick="toggleCookieDebug()">切换调试视图</button>
        </div>
        <div id="cookieStatus" class="status"></div>
        <div id="cookieHealthSummary" class="muted"></div>
        <div class="legend">
          <span class="pill healthy">healthy</span><span class="muted">可用</span>
          <span class="pill expired">expired</span><span class="muted">登录态失效</span>
          <span class="pill limited">limited</span><span class="muted">限流/额度</span>
          <span class="pill blocked">blocked</span><span class="muted">风控/403/挑战</span>
          <span class="pill error">error</span><span class="muted">其他异常</span>
        </div>
        <div id="cookieHealthList" class="cookie-health mono"></div>
      </section>

      <section class="card">
        <h2>运行配置</h2>
        <label>请求限速（每分钟）</label>
        <input id="rate" type="number" min="0" />
        <label style="margin-top:10px;display:block;">API Keys（每行一个）</label>
        <textarea id="secrets" style="min-height:80px;"></textarea>
        <div class="row" style="margin-top:10px;">
          <button onclick="saveConfig()">保存配置</button>
        </div>
        <div id="cfgStatus" class="status"></div>
      </section>

      <section class="card">
        <h2>模型与调用方式</h2>
        <div class="row" style="margin-bottom:8px;">
          <button onclick="loadModels()">刷新模型</button>
          <input id="modelFilter" placeholder="输入模型名过滤" style="max-width:220px;" oninput="renderModels()" />
        </div>
        <div class="models-scroll">
          <table>
            <thead><tr><th>模型</th><th>类型</th><th>调用路由</th><th>一键复制</th></tr></thead>
            <tbody id="modelsBody"></tbody>
          </table>
        </div>
        <div id="modelsMeta" class="muted" style="margin-top:8px;">仅限制可视高度，完整模型列表可通过滚动查看。</div>
        <div id="modelActionStatus" class="status"></div>
      </section>
    </div>
  </div>

  <script>
    const KEY_STORAGE = 'bigbat_admin_api_key';
    let cookieDebugEnabled = false;
    let modelsCache = [];

    function currentApiKey() {
      return document.getElementById('apiKey').value.trim();
    }

    function saveApiKey() {
      const key = currentApiKey();
      if (!key) {
        localStorage.removeItem(KEY_STORAGE);
        setStatus('apiStatus', false, 'API Key 为空，已清除本地保存');
        updateGuide();
        return;
      }
      localStorage.setItem(KEY_STORAGE, key);
      setStatus('apiStatus', true, 'API Key 已保存到浏览器本地');
      updateGuide();
      loadModels();
    }

    function ensureApiKey(statusElementId) {
      const key = currentApiKey();
      if (key) return true;
      const msg = '请先输入 API Key（即 API_SECRET）';
      if (statusElementId) setStatus(statusElementId, false, msg);
      setStatus('apiStatus', false, msg);
      document.getElementById('apiKey').focus();
      return false;
    }

    function authHeaders() {
      const key = currentApiKey();
      const h = { 'Content-Type': 'application/json' };
      if (key) h['Authorization'] = 'Bearer ' + key;
      return h;
    }

    function apiPrefix() {
      const base = adminBase();
      if (base.endsWith('/admin')) {
        return base.slice(0, -6);
      }
      return '';
    }

    function fullApiURL(path) {
      return window.location.origin + apiPrefix() + path;
    }

    function currentTokenOrPlaceholder() {
      const key = currentApiKey();
      return key || '<API_KEY>';
    }

    function updateGuide() {
      const base = fullApiURL('/v1').replace(/\/$/, '');
      document.getElementById('guideBaseUrl').textContent = base;
      document.getElementById('guideAnthropicUrl').textContent = window.location.origin + apiPrefix();
      document.getElementById('guideAnthropicHealthUrl').textContent = window.location.origin + apiPrefix() + '/v1/messages/health';
      document.getElementById('guideApiKey').textContent = currentTokenOrPlaceholder();
      document.getElementById('guideDefaultModel').textContent = 'gpt-5-pro';
      document.getElementById('guideOpusModel').textContent = 'opus4.6 (别名) / claude-opus-4-6 (标准名)';
    }

    async function copyGuideValue(elementId, statusId, okMessage) {
      const el = document.getElementById(elementId);
      if (!el) return;
      const text = (el.textContent || '').trim();
      await copyText(text, statusId, okMessage);
    }

    async function copyText(text, statusId, okMessage) {
      try {
        if (navigator.clipboard && navigator.clipboard.writeText) {
          await navigator.clipboard.writeText(text);
        } else {
          const ta = document.createElement('textarea');
          ta.value = text;
          ta.style.position = 'fixed';
          ta.style.opacity = '0';
          document.body.appendChild(ta);
          ta.focus();
          ta.select();
          document.execCommand('copy');
          document.body.removeChild(ta);
        }
        const msg = okMessage || '已复制';
        setStatus(statusId, true, msg);
        if (!currentApiKey() && text.includes('<API_KEY>')) {
          setStatus(statusId, false, msg + '，请把 <API_KEY> 替换为真实值');
        }
      } catch (e) {
        setStatus(statusId, false, '复制失败: ' + String(e));
      }
    }

    function buildCurlSnippet(method, path, body, stream) {
      const token = currentTokenOrPlaceholder();
      const url = fullApiURL(path);
      const upperMethod = String(method || 'POST').toUpperCase();
      if (upperMethod === 'GET') {
        return [
          'curl -s "' + url + '" \\\\',
          '  -H "Authorization: Bearer ' + token + '"'
        ].join('\\n');
      }
      const payload = JSON.stringify(body || {}, null, 2);
      const flag = stream ? '-N' : '-s';
      return [
        "cat <<'JSON' | curl " + flag + ' "' + url + '" \\\\',
        '  -H "Authorization: Bearer ' + token + '" \\\\',
        '  -H "Content-Type: application/json" \\\\',
        '  -X ' + upperMethod + ' \\\\',
        '  -d @-',
        payload,
        'JSON'
      ].join('\\n');
    }
    function modelExamplePayload(model) {
      if (model && model.example && model.example.path) {
        return {
          method: model.method || 'POST',
          path: model.example.path,
          body: model.example.body || {}
        };
      }
      return {
        method: 'POST',
        path: '/v1/chat/completions',
        body: {
          model: model && model.id ? model.id : 'opus4.6',
          stream: false,
          messages: [{ role: 'user', content: 'hello' }]
        }
      };
    }

    async function copyQuickChat(stream) {
      const payload = {
        model: 'opus4.6',
        stream: !!stream,
        messages: [{ role: 'user', content: stream ? '请给我3条学习建议' : '请用一句话介绍你自己' }]
      };
      const text = buildCurlSnippet('POST', '/v1/chat/completions', payload, !!stream);
      await copyText(text, 'quickCopyStatus', stream ? '已复制 Opus4.6 流式调用' : '已复制 Opus4.6 非流式调用');
    }

    async function copyQuickModels() {
      const text = buildCurlSnippet('GET', '/v1/models');
      await copyText(text, 'quickCopyStatus', '已复制 models 调用');
    }

    async function copyQuickAnthropic() {
      const token = currentTokenOrPlaceholder();
      const url = fullApiURL('/v1/messages');
      const payload = {
        model: 'opus4.6',
        max_tokens: 256,
        messages: [{ role: 'user', content: 'Hello from Claude-compatible client' }]
      };
      const text = [
        'cat <<\'JSON\' | curl -s "' + url + '" \\\\',
        '  -H "x-api-key: ' + token + '" \\\\',
        '  -H "anthropic-version: 2023-06-01" \\\\',
        '  -H "content-type: application/json" \\\\',
        '  -X POST \\\\',
        '  -d @-',
        JSON.stringify(payload, null, 2),
        'JSON'
      ].join('\\n');
      await copyText(text, 'quickCopyStatus', '已复制 Anthropic 调用示例');
    }

    async function checkAnthropicCompatibility() {
      if (!ensureApiKey('anthropicCompatStatus')) return;
      try {
        const data = await req(apiPrefix() + '/v1/messages/health');
        const ready = !!(data && data.ready);
        const msg = ready
          ? ('Anthropic 兼容可用: ready=true, healthy=' + (data.cookies_healthy || 0) + '/' + (data.cookies_total || 0))
          : ('Anthropic 兼容未就绪: ' + (data.reason || 'unknown'));
        setStatus('anthropicCompatStatus', ready, msg);
      } catch (e) {
        setStatus('anthropicCompatStatus', false, String(e));
      }
    }

    async function checkOpenAICompatibility() {
      if (!ensureApiKey('openaiCompatStatus')) return;
      try {
        const data = await req(apiPrefix() + '/v1/chat/health');
        const ready = !!(data && data.ready);
        const msg = ready
          ? ('OpenAI 兼容可用: ready=true, healthy=' + (data.cookies_healthy || 0) + '/' + (data.cookies_total || 0))
          : ('OpenAI 兼容未就绪: ' + (data.reason || 'unknown'));
        setStatus('openaiCompatStatus', ready, msg);
      } catch (e) {
        setStatus('openaiCompatStatus', false, String(e));
      }
    }

    async function req(url, options = {}) {
      const headers = Object.assign({}, authHeaders(), options.headers || {});
      const res = await fetch(url, Object.assign({}, options, { headers }));
      if (!res.ok) {
        const txt = await res.text();
        let msg = txt || ('HTTP ' + res.status);
        try {
          const parsed = JSON.parse(txt || '{}');
          if (parsed && parsed.error && parsed.error.message) {
            msg = parsed.error.message;
          }
        } catch (_) {}
        if (res.status === 401) {
          msg = 'API Key 无效，请检查 API_SECRET';
        }
        throw new Error(msg);
      }
      const ct = res.headers.get('content-type') || '';
      return ct.includes('application/json') ? res.json() : res.text();
    }

    function setStatus(id, ok, msg) {
      const el = document.getElementById(id);
      el.className = 'status ' + (ok ? 'ok' : 'err');
      el.textContent = msg;
    }

    function adminBase() {
      const p = window.location.pathname;
      if (p.endsWith('/ui')) {
        return p.slice(0, -3);
      }
      return p;
    }

    async function refreshState() {
      if (!ensureApiKey('apiStatus')) {
        document.getElementById('state').textContent = '请先输入 API Key（即 API_SECRET），再点击刷新状态';
        return;
      }
      try {
        const data = await req(adminBase() + '/state');
        document.getElementById('state').textContent = JSON.stringify(data, null, 2);
        document.getElementById('rate').value = data.config.request_rate_limit_per_minute;
        if (data.app && data.app.uptime) {
          document.getElementById('appUptime').textContent = '运行时长 ' + data.app.uptime;
        }
        loadModels();
      } catch (e) {
        document.getElementById('state').textContent = String(e);
      }
    }

    async function loadCookies() {
      if (!ensureApiKey('cookieStatus')) return;
      try {
        const data = await req(adminBase() + '/cookies');
        document.getElementById('cookies').value = (data.cookies || []).join('\n');
        setStatus('cookieStatus', true, '已加载');
        await checkCookiesHealth();
      } catch (e) {
        setStatus('cookieStatus', false, String(e));
      }
    }

    async function saveCookies() {
      if (!ensureApiKey('cookieStatus')) return;
      try {
        const lines = document.getElementById('cookies').value.split('\n').map(s => s.trim()).filter(Boolean);
        const data = await req(adminBase() + '/cookies', { method: 'POST', body: JSON.stringify({ cookies: lines }) });
        setStatus('cookieStatus', true, '保存成功，总数: ' + data.total);
        await checkCookiesHealth();
      } catch (e) {
        setStatus('cookieStatus', false, String(e));
      }
    }

    function renderCookieHealth(payload) {
      const summary = payload && payload.summary ? payload.summary : {};
      const items = payload && payload.items ? payload.items : [];
      const summaryEl = document.getElementById('cookieHealthSummary');
      summaryEl.textContent =
        '总数: ' + (summary.total || 0) +
        ' | 可用: ' + (summary.healthy || 0) +
        ' | 失效: ' + (summary.expired || 0) +
        ' | 限流: ' + (summary.limited || 0) +
        ' | 风控: ' + (summary.blocked || 0) +
        ' | 异常: ' + (summary.error || 0);

      const box = document.getElementById('cookieHealthList');
      if (!items.length) {
        box.innerHTML = '<div class="muted">暂无 cookies</div>';
        return;
      }
      box.innerHTML = items.map(function (it) {
        const status = String(it.status || 'error');
        const msg = it.message ? (' - ' + escapeHtml(it.message)) : '';
        const code = it.http_status ? (' (HTTP ' + it.http_status + ')') : '';
        const debug = cookieDebugEnabled && it.debug ?
          '<div class="debug-wrap">' + escapeHtml(JSON.stringify(it.debug, null, 2)) + '</div>' : '';
        const source = it.source ? (' | source: ' + escapeHtml(it.source)) : '';
        return '<div class="cookie-health-item">' +
          '<div><div class="cookie-line"><strong>#' + (it.index || 0) + '</strong> ' + escapeHtml(it.cookie || '') + '</div>' +
          '<div class="muted">' + msg + code + source + '</div>' + debug + '</div>' +
          '<span class="pill ' + status + '">' + status + '</span>' +
        '</div>';
      }).join('');
    }

    function escapeHtml(s) {
      return String(s)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#039;');
    }

    async function checkCookiesHealth() {
      if (!ensureApiKey('cookieStatus')) return;
      const box = document.getElementById('cookieHealthList');
      box.innerHTML = '<div class="muted">检查中...</div>';
      try {
        const url = adminBase() + '/cookies/health' + (cookieDebugEnabled ? '?debug=1' : '');
        const data = await req(url);
        renderCookieHealth(data);
        await checkOpenAICompatibility();
        await checkAnthropicCompatibility();
      } catch (e) {
        box.innerHTML = '<div class="err">' + escapeHtml(String(e)) + '</div>';
      }
    }

    async function toggleCookieDebug() {
      cookieDebugEnabled = !cookieDebugEnabled;
      setStatus('cookieStatus', true, cookieDebugEnabled ? '调试视图已开启' : '调试视图已关闭');
      await checkCookiesHealth();
    }

    async function saveConfig() {
      if (!ensureApiKey('cfgStatus')) return;
      try {
        const rate = parseInt(document.getElementById('rate').value || '0', 10);
        const secrets = document.getElementById('secrets').value.split('\n').map(s => s.trim()).filter(Boolean);
        await req(adminBase() + '/config', {
          method: 'PATCH',
          body: JSON.stringify({ request_rate_limit_per_minute: rate, api_secrets: secrets })
        });
        setStatus('cfgStatus', true, '保存成功');
      } catch (e) {
        setStatus('cfgStatus', false, String(e));
      }
    }

    function renderModels() {
      const tbody = document.getElementById('modelsBody');
      tbody.innerHTML = '';
      const q = (document.getElementById('modelFilter').value || '').trim().toLowerCase();
      const list = modelsCache.filter(function(m){
        if (!q) return true;
        const id = String(m.id || '').toLowerCase();
        const t = String(m.type || '').toLowerCase();
        return id.includes(q) || t.includes(q);
      });
      list.forEach(m => {
        try {
          const tr = document.createElement('tr');
          const idTd = document.createElement('td');
          idTd.className = 'mono';
          idTd.textContent = m.id || '';

          const typeTd = document.createElement('td');
          typeTd.textContent = m.type || '';

          const routeTd = document.createElement('td');
          routeTd.className = 'mono';
          routeTd.textContent = (m.routes || []).join(', ');

          const actionTd = document.createElement('td');
          const actionWrap = document.createElement('div');
          actionWrap.className = 'table-actions';

          const copyCurlBtn = document.createElement('button');
          copyCurlBtn.className = 'btn-sm';
          copyCurlBtn.textContent = '复制 cURL';
          copyCurlBtn.onclick = async function() {
            const ex = modelExamplePayload(m);
            const stream = !!(ex.body && ex.body.stream);
            const snippet = buildCurlSnippet(ex.method, ex.path, ex.body, stream);
            await copyText(snippet, 'modelActionStatus', '已复制 ' + (m.id || 'model') + ' 的 cURL 调用');
          };

          const copyBodyBtn = document.createElement('button');
          copyBodyBtn.className = 'btn-sm secondary';
          copyBodyBtn.textContent = '复制 Body';
          copyBodyBtn.onclick = async function() {
            const ex = modelExamplePayload(m);
            const bodyText = JSON.stringify(ex.body || {}, null, 2);
            await copyText(bodyText, 'modelActionStatus', '已复制 ' + (m.id || 'model') + ' 请求体');
          };

          actionWrap.appendChild(copyCurlBtn);
          actionWrap.appendChild(copyBodyBtn);
          actionTd.appendChild(actionWrap);

          tr.appendChild(idTd);
          tr.appendChild(typeTd);
          tr.appendChild(routeTd);
          tr.appendChild(actionTd);
          tbody.appendChild(tr);
        } catch (e) {
          const tr = document.createElement('tr');
          tr.innerHTML = '<td colspan="4">render error: ' + escapeHtml(String(e)) + '</td>';
          tbody.appendChild(tr);
        }
      });
      document.getElementById('modelsMeta').textContent = '共 ' + modelsCache.length + ' 个模型，当前显示 ' + list.length + ' 个。';
    }

    async function loadModels() {
      if (!ensureApiKey('apiStatus')) return;
      try {
        const data = await req(adminBase() + '/models');
        modelsCache = Array.isArray(data.models) ? data.models : [];
        renderModels();
      } catch (e) {
        const tbody = document.getElementById('modelsBody');
        tbody.innerHTML = '<tr><td colspan="4">' + escapeHtml(String(e)) + '</td></tr>';
        document.getElementById('modelsMeta').textContent = '模型加载失败';
      }
    }

    (function init() {
      const saved = localStorage.getItem(KEY_STORAGE) || '';
      if (saved) {
        document.getElementById('apiKey').value = saved;
        setStatus('apiStatus', true, '已加载本地保存的 API Key');
        refreshState();
        loadCookies();
        loadModels();
        checkCookiesHealth();
        checkOpenAICompatibility();
        checkAnthropicCompatibility();
      } else {
        document.getElementById('state').textContent = '请先输入 API Key（即 API_SECRET），再点击刷新状态。';
        document.getElementById('cookieHealthList').innerHTML = '<div class="muted">请输入 API Key 后进行检查</div>';
      }
      updateGuide();
      document.getElementById('apiKey').addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
          saveApiKey();
          refreshState();
        }
      });
      document.getElementById('apiKey').addEventListener('input', updateGuide);
      if (saved) {
        checkOpenAICompatibility();
        checkAnthropicCompatibility();
      }
    })();
  </script>
</body>
</html>`
