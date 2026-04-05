package httpapi

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Uptime</title>
  <style>
    :root { --bg: #0f1419; --card: #1a2332; --text: #e7ecf3; --muted: #8b9cb3; --up: #3ecf8e; --down: #f56565; }
    * { box-sizing: border-box; }
    body { font-family: ui-sans-serif, system-ui, sans-serif; background: var(--bg); color: var(--text); margin: 0; min-height: 100vh; }
    header { padding: 1.5rem 1.25rem; border-bottom: 1px solid #2a3544; }
    h1 { margin: 0; font-size: 1.25rem; font-weight: 600; }
    p.sub { margin: 0.35rem 0 0; color: var(--muted); font-size: 0.875rem; }
    main { max-width: 960px; margin: 0 auto; padding: 1.25rem; }
    table { width: 100%; border-collapse: collapse; background: var(--card); border-radius: 8px; overflow: hidden; }
    th, td { text-align: left; padding: 0.75rem 1rem; border-bottom: 1px solid #2a3544; }
    th { color: var(--muted); font-weight: 500; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.04em; }
    tr:last-child td { border-bottom: none; }
    .badge { display: inline-block; padding: 0.2rem 0.5rem; border-radius: 4px; font-size: 0.75rem; font-weight: 600; }
    .badge.up { background: rgba(62, 207, 142, 0.15); color: var(--up); }
    .badge.down { background: rgba(245, 101, 101, 0.15); color: var(--down); }
    .badge.unknown { background: #2a3544; color: var(--muted); }
    .mono { font-family: ui-monospace, monospace; font-size: 0.8rem; word-break: break-all; }
    .err { color: var(--down); font-size: 0.8rem; margin-top: 0.25rem; }
    footer { padding: 1rem; text-align: center; color: var(--muted); font-size: 0.8rem; }
  </style>
</head>
<body>
  <header>
    <h1>Service uptime</h1>
    <p class="sub">Latest check per service · <span id="updated">Loading…</span></p>
  </header>
  <main>
    <table>
      <thead>
        <tr>
          <th>Service</th>
          <th>Endpoint</th>
          <th>Status</th>
          <th>Latency</th>
          <th>Last check</th>
        </tr>
      </thead>
      <tbody id="rows"></tbody>
    </table>
    <p id="loaderr" class="err" style="display:none"></p>
  </main>
  <footer>API: <span class="mono">GET /api/v1/status</span></footer>
  <script>
    async function load() {
      const errEl = document.getElementById('loaderr');
      errEl.style.display = 'none';
      try {
        const r = await fetch('/api/v1/status');
        if (!r.ok) throw new Error('HTTP ' + r.status);
        const data = await r.json();
        const tbody = document.getElementById('rows');
        tbody.innerHTML = '';
        for (const s of data.services || []) {
          const tr = document.createElement('tr');
          const st = s.status;
          let badgeClass = 'unknown';
          let label = '—';
          if (st === 'up') { badgeClass = 'up'; label = 'Up'; }
          else if (st === 'down') { badgeClass = 'down'; label = 'Down'; }
          const lat = s.latency_ms != null ? s.latency_ms + ' ms' : '—';
          const when = s.checked_at ? new Date(s.checked_at).toLocaleString() : '—';
          tr.innerHTML =
            '<td><strong>' + escapeHtml(s.name) + '</strong></td>' +
            '<td class="mono">' + escapeHtml(s.endpoint) + '</td>' +
            '<td><span class="badge ' + badgeClass + '">' + label + '</span>' +
            (s.error_message ? '<div class="err">' + escapeHtml(s.error_message) + '</div>' : '') + '</td>' +
            '<td>' + lat + '</td>' +
            '<td>' + when + '</td>';
          tbody.appendChild(tr);
        }
        document.getElementById('updated').textContent = 'Updated ' + new Date().toLocaleTimeString();
      } catch (e) {
        errEl.textContent = 'Failed to load: ' + e.message;
        errEl.style.display = 'block';
      }
    }
    function escapeHtml(t) {
      const d = document.createElement('div');
      d.textContent = t;
      return d.innerHTML;
    }
    load();
    setInterval(load, 15000);
  </script>
</body>
</html>
`
