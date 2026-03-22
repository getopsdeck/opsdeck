package web

import (
	"html/template"
	"net/http"
)

var dashboardTmpl = template.Must(template.New("dashboard").Parse(dashboardHTML))

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	dashboardTmpl.Execute(w, nil)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>OpsDeck</title>
<style>
  :root {
    --bg: #1a1b26;
    --bg-dark: #16161e;
    --bg-highlight: #292e42;
    --fg: #c0caf5;
    --fg-dark: #565f89;
    --blue: #7aa2f7;
    --green: #9ece6a;
    --yellow: #e0af68;
    --red: #f7768e;
    --cyan: #7dcfff;
    --border: #3b4261;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'SF Mono', 'Cascadia Code', 'JetBrains Mono', monospace;
    background: var(--bg);
    color: var(--fg);
    font-size: 14px;
    line-height: 1.5;
  }
  header {
    background: var(--blue);
    color: var(--bg);
    padding: 12px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: bold;
  }
  header .subtitle { font-weight: normal; opacity: 0.8; }
  .container { max-width: 1200px; margin: 0 auto; padding: 16px; }
  .stats {
    display: flex;
    gap: 12px;
    margin: 16px 0;
    flex-wrap: wrap;
  }
  .stat-badge {
    padding: 6px 16px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 600;
  }
  .stat-busy { background: #9ece6a22; color: var(--green); border: 1px solid #9ece6a44; }
  .stat-waiting { background: #e0af6822; color: var(--yellow); border: 1px solid #e0af6844; }
  .stat-idle { background: #565f8922; color: var(--fg-dark); border: 1px solid #565f8944; }
  .stat-dead { background: #f7768e22; color: var(--red); border: 1px solid #f7768e44; }
  @keyframes pulse { 0%,100% { opacity:1 } 50% { opacity:0.5 } }
  .state-waiting .state-icon { animation: pulse 2s ease-in-out infinite; }
  .state-busy .state-icon { animation: pulse 1s ease-in-out infinite; }
  .live-dot { display:inline-block; width:8px; height:8px; background:var(--green); border-radius:50%; margin-right:8px; animation: pulse 2s ease-in-out infinite; }
  table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 16px;
  }
  th {
    text-align: left;
    padding: 10px 12px;
    border-bottom: 2px solid var(--border);
    color: var(--fg-dark);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  td {
    padding: 10px 12px;
    border-bottom: 1px solid var(--bg-highlight);
  }
  tr { cursor: pointer; transition: background 0.15s; }
  tr:hover { background: var(--bg-highlight); }
  tr.selected { background: var(--bg-highlight); border-left: 3px solid var(--blue); }
  .state-icon { font-size: 16px; }
  .state-busy .state-icon { color: var(--green); }
  .state-waiting .state-icon { color: var(--yellow); }
  .state-idle .state-icon { color: var(--fg-dark); }
  .state-dead .state-icon { color: var(--red); }
  .project-group {
    background: var(--bg-dark);
    padding: 8px 12px;
    font-weight: 600;
    color: var(--blue);
    border-bottom: 1px solid var(--border);
  }
  .detail-panel {
    margin-top: 16px;
    background: var(--bg-dark);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 20px;
    display: none;
  }
  .detail-panel.open { display: block; }
  .detail-title {
    font-size: 16px;
    font-weight: bold;
    color: var(--blue);
    margin-bottom: 12px;
  }
  .detail-stats {
    display: flex;
    gap: 16px;
    margin-bottom: 12px;
    color: var(--fg-dark);
    font-size: 13px;
  }
  .detail-stats span { color: var(--fg); font-weight: 600; }
  .activity-list { list-style: none; padding: 0; }
  .activity-list li {
    padding: 4px 0;
    color: var(--fg);
    font-size: 13px;
  }
  .activity-list li::before {
    content: "* ";
    color: var(--fg-dark);
  }
  .last-updated {
    text-align: right;
    color: var(--fg-dark);
    font-size: 12px;
    margin-top: 8px;
  }
  .empty {
    text-align: center;
    padding: 60px 0;
    color: var(--fg-dark);
  }
</style>
</head>
<body>
<header>
  <div>OpsDeck <span class="subtitle">Chief of Staff for Claude Code</span></div>
  <div><span class="live-dot"></span><span id="session-count">Loading...</span></div>
</header>
<div class="container">
  <div class="stats" id="stats"></div>
  <table>
    <thead>
      <tr>
        <th style="width:30px"></th>
        <th>Project</th>
        <th>Session</th>
        <th>State</th>
        <th>Started</th>
        <th>Activity</th>
      </tr>
    </thead>
    <tbody id="sessions"></tbody>
  </table>
  <div class="detail-panel" id="detail">
    <div class="detail-title" id="detail-title"></div>
    <div class="detail-stats" id="detail-stats"></div>
    <ul class="activity-list" id="detail-activities"></ul>
    <div id="detail-request" style="margin-top:12px;color:var(--fg-dark);font-size:13px"></div>
  </div>
  <div class="last-updated" id="last-updated"></div>
</div>
<script>
const stateIcons = { busy: '●', waiting: '◐', idle: '○', dead: '✕' };
let selectedId = null;

function connect() {
  const es = new EventSource('/api/events');
  es.onmessage = (e) => {
    const sessions = JSON.parse(e.data);
    render(sessions);
  };
  es.onerror = () => {
    es.close();
    setTimeout(connect, 3000);
  };
}

function render(sessions) {
  // Stats
  const counts = { busy: 0, waiting: 0, idle: 0, dead: 0 };
  sessions.forEach(s => counts[s.state] = (counts[s.state] || 0) + 1);
  document.getElementById('stats').innerHTML =
    Object.entries(counts).map(([k,v]) =>
      '<div class="stat-badge stat-' + k + '">' + v + ' ' + k + '</div>'
    ).join('');
  document.getElementById('session-count').textContent =
    sessions.length + ' sessions';

  // Group by project
  const groups = {};
  sessions.forEach(s => {
    if (!groups[s.project]) groups[s.project] = [];
    groups[s.project].push(s);
  });

  // Table
  const tbody = document.getElementById('sessions');
  tbody.innerHTML = '';
  Object.keys(groups).sort().forEach(project => {
    const tr = document.createElement('tr');
    tr.innerHTML = '<td colspan="6" class="project-group">' + project + '/ (' + groups[project].length + ')</td>';
    tbody.appendChild(tr);

    groups[project].forEach(s => {
      const row = document.createElement('tr');
      row.className = 'state-' + s.state + (s.id === selectedId ? ' selected' : '');
      row.onclick = () => selectSession(s.id);
      const ago = timeAgo(new Date(s.started_at));
      row.innerHTML =
        '<td class="state-icon">' + (stateIcons[s.state] || '?') + '</td>' +
        '<td>' + s.project + '</td>' +
        '<td style="font-family:monospace;font-size:12px">' + s.id.substring(0,12) + '</td>' +
        '<td>' + s.state.toUpperCase() + '</td>' +
        '<td>' + ago + '</td>' +
        '<td style="color:var(--fg-dark)">' + (s.working_on || formatStats(s)) + '</td>';
      tbody.appendChild(row);
    });
  });

  document.getElementById('last-updated').textContent =
    'Updated ' + new Date().toLocaleTimeString();
}

function formatStats(s) {
  const parts = [];
  if (s.edit_count) parts.push(s.edit_count + ' edits');
  if (s.bash_count) parts.push(s.bash_count + ' cmds');
  if (s.messages) parts.push(s.messages + ' msgs');
  return parts.join(', ') || '-';
}

function selectSession(id) {
  selectedId = selectedId === id ? null : id;
  const panel = document.getElementById('detail');
  if (!selectedId) { panel.classList.remove('open'); return; }

  fetch('/api/session/' + id)
    .then(r => r.json())
    .then(s => {
      document.getElementById('detail-title').textContent =
        (stateIcons[s.state] || '') + '  ' + s.id + '  (PID ' + s.pid + ')';

      const stats = [];
      if (s.edit_count) stats.push('<span>' + s.edit_count + '</span> edits');
      if (s.bash_count) stats.push('<span>' + s.bash_count + '</span> commands');
      if (s.files_changed) stats.push('<span>' + s.files_changed + '</span> files');
      if (s.error_count) stats.push('<span>' + s.error_count + '</span> errors');
      if (s.messages) stats.push('<span>' + s.messages + '</span> messages');
      document.getElementById('detail-stats').innerHTML = stats.join(' &middot; ');

      const al = document.getElementById('detail-activities');
      al.innerHTML = '';
      (s.activities || []).forEach(a => {
        const li = document.createElement('li');
        li.textContent = a;
        al.appendChild(li);
      });

      document.getElementById('detail-request').innerHTML =
        s.last_request ? '<b>Last request:</b> ' + escapeHtml(s.last_request) : '';

      panel.classList.add('open');
    });

  // Re-render to update selected row
  document.querySelectorAll('#sessions tr').forEach(tr => {
    tr.classList.toggle('selected', tr.onclick && tr.querySelector('td:nth-child(3)')?.textContent === id.substring(0,12));
  });
}

function timeAgo(date) {
  const mins = Math.floor((Date.now() - date.getTime()) / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return mins + 'm ago';
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return hrs + 'h ago';
  return Math.floor(hrs / 24) + 'd ago';
}

function escapeHtml(s) {
  const div = document.createElement('div');
  div.textContent = s;
  return div.innerHTML;
}

connect();
</script>
</body>
</html>`
