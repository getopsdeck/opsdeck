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
    background: linear-gradient(135deg, var(--blue), #5b8def);
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
    padding: 8px 20px;
    border-radius: 6px;
    font-size: 14px;
    font-weight: 600;
    min-height: 44px;
    display: inline-flex;
    align-items: center;
  }
  .stat-busy { background: #9ece6a22; color: var(--green); border: 1px solid #9ece6a44; }
  .stat-waiting { background: #e0af6822; color: var(--yellow); border: 1px solid #e0af6844; }
  .stat-idle { background: #565f8922; color: var(--fg-dark); border: 1px solid #565f8944; }
  .stat-dead { background: #f7768e22; color: var(--red); border: 1px solid #f7768e44; }
  @keyframes pulse { 0%,100% { opacity:1 } 50% { opacity:0.5 } }
  @keyframes shimmer { 0% { background-position: -200% 0; } 100% { background-position: 200% 0; } }
  .skeleton {
    background: linear-gradient(90deg, var(--bg-highlight) 25%, var(--border) 50%, var(--bg-highlight) 75%);
    background-size: 200% 100%;
    animation: shimmer 1.5s infinite;
    height: 14px;
    border-radius: 4px;
    margin: 6px 0;
  }
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
  tr { cursor: pointer; transition: background 0.15s, transform 0.15s; }
  tr:hover { background: var(--bg-highlight); transform: translateY(-1px); }
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
    max-height: 0;
    opacity: 0;
    overflow: hidden;
    transition: max-height 0.3s ease, opacity 0.3s ease;
  }
  .detail-panel.open { display: block; max-height: 2000px; opacity: 1; }
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
    font-size: 11px;
    margin-top: 8px;
    opacity: 0.6;
    letter-spacing: 0.3px;
  }
  @media (max-width: 768px) {
    .container { padding: 8px; }
    table { font-size: 12px; }
    th, td { padding: 6px 4px; }
    .stats { gap: 6px; }
  }
  .timeline-container {
    margin-top: 12px;
    background: var(--bg-dark);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 16px;
    display: none;
  }
  .timeline-container.open { display: block; }
  .timeline-title {
    font-size: 13px;
    color: var(--fg-dark);
    margin-bottom: 8px;
  }
  .timeline-bar {
    height: 24px;
    display: flex;
    border-radius: 4px;
    overflow: hidden;
    background: var(--bg-highlight);
    margin-bottom: 6px;
  }
  .timeline-segment {
    min-width: 2px;
    height: 100%;
    position: relative;
    cursor: pointer;
    transition: opacity 0.1s;
  }
  .timeline-segment:hover { opacity: 0.7; }
  .tl-tool { background: var(--blue); }
  .tl-text { background: var(--green); }
  .tl-user { background: var(--cyan); }
  .tl-error { background: var(--red); }
  .tl-idle { background: var(--bg-highlight); }
  .timeline-legend {
    display: flex;
    gap: 16px;
    font-size: 11px;
    color: var(--fg-dark);
    margin-top: 4px;
  }
  .timeline-legend span::before {
    content: '';
    display: inline-block;
    width: 10px;
    height: 10px;
    border-radius: 2px;
    margin-right: 4px;
    vertical-align: middle;
  }
  .tl-legend-tool::before { background: var(--blue); }
  .tl-legend-text::before { background: var(--green); }
  .tl-legend-user::before { background: var(--cyan); }
  .tl-legend-error::before { background: var(--red); }
  .empty {
    text-align: center;
    padding: 60px 0;
    color: var(--fg-dark);
  }
  .stat-badge {
    cursor: pointer;
    transition: box-shadow 0.15s, transform 0.1s;
    user-select: none;
  }
  .stat-badge:hover { transform: translateY(-1px); }
  .stat-badge.active-filter {
    box-shadow: 0 0 0 2px currentColor;
    transform: translateY(-1px);
  }
  .search-bar {
    width: 100%;
    background: var(--bg-dark);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--fg);
    font-family: 'SF Mono', 'Cascadia Code', 'JetBrains Mono', monospace;
    font-size: 13px;
    padding: 8px 14px;
    outline: none;
    transition: border-color 0.15s;
    margin-bottom: 4px;
  }
  .search-bar::placeholder { color: var(--fg-dark); }
  .search-bar:focus { border-color: var(--blue); }
  .cost-badge {
    font-size: 12px;
    font-weight: normal;
    color: var(--cyan);
    margin-left: 12px;
    opacity: 0.9;
  }
  .brief-card {
    background: var(--bg-dark);
    border: 1px solid var(--border);
    border-radius: 8px;
    margin-bottom: 12px;
    overflow: hidden;
  }
  .brief-header {
    padding: 10px 16px;
    cursor: pointer;
    color: var(--blue);
    font-weight: 600;
    font-size: 14px;
  }
  .brief-header:hover { background: var(--bg-highlight); }
  .brief-body { padding: 0 16px 12px; }
  .brief-attention { color: var(--yellow); margin: 4px 0; font-size: 13px; }
  .brief-update { color: var(--fg); margin: 4px 0; font-size: 13px; }
  .brief-idle { color: var(--fg-dark); margin: 4px 0; font-size: 13px; }
</style>
</head>
<body>
<header>
  <div>OpsDeck <span class="subtitle">Chief of Staff for Claude Code</span></div>
  <div><span class="live-dot"></span><span id="session-count">Loading...</span><span class="cost-badge" id="cost-today"></span></div>
</header>
<div class="container">
  <div class="stats" id="stats"></div>
  <div class="brief-card" id="brief-card">
    <div class="brief-header" onclick="toggleBrief()">
      Morning Brief <span id="brief-toggle">&#9656;</span>
    </div>
    <div class="brief-body" id="brief-body" style="display:none">
      <div id="brief-content">
        <div class="skeleton" style="width:80%"></div>
        <div class="skeleton" style="width:65%"></div>
        <div class="skeleton" style="width:72%"></div>
      </div>
    </div>
  </div>
  <input class="search-bar" id="search" type="text" placeholder="Search by project, session ID, or activity..." autocomplete="off">
  <table>
    <thead>
      <tr>
        <th style="width:30px"></th>
        <th>Project</th>
        <th>Branch</th>
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
    <div id="detail-resume" style="margin-top:12px"></div>
  </div>
  <div class="timeline-container" id="timeline">
    <div class="timeline-title" id="timeline-title">Session Timeline (last 24h)</div>
    <div class="timeline-bar" id="timeline-bar"></div>
    <div class="timeline-legend">
      <span class="tl-legend-tool">Tool calls</span>
      <span class="tl-legend-text">Text output</span>
      <span class="tl-legend-user">User input</span>
      <span class="tl-legend-error">Errors</span>
    </div>
  </div>
  <div class="last-updated" id="last-updated"></div>
</div>
<script>
const stateIcons = { busy: '●', waiting: '◐', idle: '○', dead: '✕' };
let selectedId = null;
let allSessions = [];
let activeFilter = null;
let searchQuery = '';

// Search input
document.getElementById('search').addEventListener('input', function() {
  searchQuery = this.value.trim().toLowerCase();
  renderFiltered();
});

function connect() {
  const es = new EventSource('/api/events');
  es.onmessage = (e) => {
    allSessions = JSON.parse(e.data);
    render(allSessions);
  };
  es.onerror = () => {
    es.close();
    setTimeout(connect, 3000);
  };
}

function render(sessions) {
  // Stats — count across all sessions (unfiltered)
  const counts = { busy: 0, waiting: 0, idle: 0, dead: 0 };
  sessions.forEach(s => counts[s.state] = (counts[s.state] || 0) + 1);
  document.getElementById('stats').innerHTML =
    Object.entries(counts).map(([k,v]) =>
      '<div class="stat-badge stat-' + k + (activeFilter === k ? ' active-filter' : '') +
      '" data-state="' + k + '">' + v + ' ' + k + '</div>'
    ).join('');
  // Attach filter click handlers
  document.querySelectorAll('.stat-badge[data-state]').forEach(badge => {
    badge.addEventListener('click', () => {
      const st = badge.dataset.state;
      activeFilter = activeFilter === st ? null : st;
      renderFiltered();
    });
  });

  // Total cost
  const totalCost = sessions.reduce((sum, s) => sum + (s.est_cost_usd || 0), 0);
  const costEl = document.getElementById('cost-today');
  costEl.textContent = totalCost > 0 ? '· $' + totalCost.toFixed(2) + ' today' : '';

  document.getElementById('session-count').textContent =
    sessions.length + ' session' + (sessions.length !== 1 ? 's' : '');

  renderFiltered();
}

function renderFiltered() {
  // Re-render badges to reflect activeFilter
  document.querySelectorAll('.stat-badge[data-state]').forEach(badge => {
    badge.classList.toggle('active-filter', badge.dataset.state === activeFilter);
  });

  // Apply search + state filter
  const q = searchQuery;
  let filtered = allSessions.filter(s => {
    if (activeFilter && s.state !== activeFilter) return false;
    if (q) {
      const haystack = (s.project + ' ' + s.id + ' ' + (s.working_on || '') + ' ' + (s.git_branch || '')).toLowerCase();
      if (!haystack.includes(q)) return false;
    }
    return true;
  });

  // Group by project
  const groups = {};
  filtered.forEach(s => {
    if (!groups[s.project]) groups[s.project] = [];
    groups[s.project].push(s);
  });

  // Table
  const tbody = document.getElementById('sessions');
  tbody.innerHTML = '';

  if (filtered.length === 0) {
    const tr = document.createElement('tr');
    tr.innerHTML = '<td colspan="7" style="text-align:center;padding:40px 0;color:var(--fg-dark)">No sessions match the current filter</td>';
    tbody.appendChild(tr);
  } else {
    Object.keys(groups).sort().forEach(project => {
      const tr = document.createElement('tr');
      tr.innerHTML = '<td colspan="7" class="project-group">' + escapeHtml(project) + '/ (' + groups[project].length + ')</td>';
      tbody.appendChild(tr);

      groups[project].forEach(s => {
        const row = document.createElement('tr');
        row.className = 'state-' + s.state + (s.id === selectedId ? ' selected' : '');
        row.onclick = () => selectSession(s.id);
        const ago = timeAgo(new Date(s.started_at));
        const branchLabel = s.git_branch ? escapeHtml(s.git_branch) + (s.git_dirty ? ' *' : '') : '-';
        const branchStyle = s.git_dirty ? 'color:var(--yellow)' : 'color:var(--cyan)';
        const burnHtml = s.state === 'busy' ? formatBurnRate(s.burn_rate) : '';
        const activityText = s.working_on ? escapeHtml(s.working_on) : formatStats(s);
        row.innerHTML =
          '<td class="state-icon">' + (stateIcons[s.state] || '?') + '</td>' +
          '<td>' + escapeHtml(s.project) + '</td>' +
          '<td style="font-size:12px;' + branchStyle + '">' + branchLabel + '</td>' +
          '<td style="font-family:monospace;font-size:12px">' + escapeHtml(s.id.substring(0,12)) + '</td>' +
          '<td>' + escapeHtml(s.state.toUpperCase()) + '</td>' +
          '<td>' + ago + '</td>' +
          '<td style="color:var(--fg-dark)">' + activityText + (burnHtml ? ' ' + burnHtml : '') + '</td>';
        tbody.appendChild(row);
      });
    });
  }

  document.getElementById('last-updated').textContent =
    'Updated ' + new Date().toLocaleTimeString();
}

function formatBurnRate(rate) {
  if (!rate || rate <= 0) return '';
  const label = '$' + rate.toFixed(2) + '/hr';
  if (rate < 5) return '<span style="color:var(--green);font-size:12px">' + label + '</span>';
  if (rate < 20) return '<span style="color:var(--yellow);font-size:12px">' + label + '</span>';
  return '<span style="color:var(--red);font-size:12px">' + label + '</span>';
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
  if (!selectedId) {
    panel.classList.remove('open');
    document.getElementById('timeline').classList.remove('open');
    return;
  }

  fetch('/api/session/' + id)
    .then(r => r.json())
    .then(s => {
      let titleText = (stateIcons[s.state] || '') + '  ' + s.id + '  (PID ' + s.pid + ')';
      if (s.git_branch) {
        titleText += '  \u2014 ' + s.git_branch + (s.git_dirty ? ' *' : '');
      }
      document.getElementById('detail-title').textContent = titleText;

      const stats = [];
      if (s.edit_count) stats.push('<span>' + s.edit_count + '</span> edits');
      if (s.bash_count) stats.push('<span>' + s.bash_count + '</span> commands');
      if (s.files_changed) stats.push('<span>' + s.files_changed + '</span> files');
      if (s.error_count) stats.push('<span>' + s.error_count + '</span> errors');
      if (s.messages) stats.push('<span>' + s.messages + '</span> messages');
      if (s.est_cost_usd > 0) stats.push('<span>$' + s.est_cost_usd.toFixed(2) + '</span> est.');
      if (s.git_ahead) stats.push('<span>' + s.git_ahead + '</span> ahead');
      if (s.git_behind) stats.push('<span>' + s.git_behind + '</span> behind');
      if (s.git_last_commit) stats.push('<span style="color:var(--cyan)">' + escapeHtml(s.git_last_commit) + '</span>');
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

      // Resume command button.
      const resumeCmd = 'opsdeck resume ' + s.id.substring(0, 12);
      document.getElementById('detail-resume').innerHTML =
        '<button onclick="navigator.clipboard.writeText(\'' + resumeCmd + '\').then(()=>this.textContent=\'Copied!\')" ' +
        'style="background:var(--bg-highlight);color:var(--cyan);border:1px solid var(--border);border-radius:4px;' +
        'padding:4px 12px;font-family:monospace;font-size:12px;cursor:pointer">' +
        '\u2398 ' + escapeHtml(resumeCmd) + '</button>';

      panel.classList.add('open');
      panel.scrollIntoView({ behavior: 'smooth', block: 'nearest' });

      // Fetch and render timeline.
      loadTimeline(id);
    });

  // Re-render to update selected row
  document.querySelectorAll('#sessions tr').forEach(tr => {
    tr.classList.toggle('selected', tr.onclick && tr.querySelector('td:nth-child(3)')?.textContent === id.substring(0,12));
  });
}

function loadTimeline(sessionId) {
  const container = document.getElementById('timeline');
  const bar = document.getElementById('timeline-bar');

  fetch('/api/timeline/' + sessionId)
    .then(r => r.json())
    .then(tl => {
      if (!tl.events || tl.events.length === 0) {
        container.classList.remove('open');
        return;
      }

      bar.innerHTML = '';
      const totalDuration = tl.events.reduce((sum, e) => sum + Math.max(e.duration, 1), 0);

      tl.events.forEach(e => {
        const seg = document.createElement('div');
        const pct = Math.max((Math.max(e.duration, 1) / totalDuration) * 100, 0.3);
        seg.className = 'timeline-segment tl-' + e.type;
        seg.style.width = pct + '%';
        seg.title = e.type + (e.tool ? ': ' + e.tool : '') +
          ' (' + e.duration + 's) ' +
          new Date(e.timestamp).toLocaleTimeString();
        bar.appendChild(seg);
      });

      document.getElementById('timeline-title').textContent =
        'Session Timeline \u2014 ' + tl.events.length + (tl.events.length === 1 ? ' event, ' : ' events, ') +
        new Date(tl.started_at).toLocaleTimeString() + ' \u2013 ' +
        new Date(tl.ended_at).toLocaleTimeString();

      container.classList.add('open');
    })
    .catch(() => container.classList.remove('open'));
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

function toggleBrief() {
  const body = document.getElementById('brief-body');
  const toggle = document.getElementById('brief-toggle');
  const open = body.style.display === 'none';
  body.style.display = open ? 'block' : 'none';
  toggle.innerHTML = open ? '&#9662;' : '&#9656;';
}

function loadBrief() {
  fetch('/api/brief')
    .then(r => r.json())
    .then(brief => {
      const content = document.getElementById('brief-content');
      const parts = [];

      // Waiting sessions (from enriched WaitingSessions)
      let waitCount = 0;
      if (brief.Projects) {
        brief.Projects.forEach(p => {
          if (p.WaitingSessions && p.WaitingSessions.length > 0) {
            p.WaitingSessions.forEach(ws => {
              waitCount++;
              if (waitCount <= 5) {
                const msg = ws.LastUserMsg ? ' \u2014 ' + escapeHtml(ws.LastUserMsg) : '';
                parts.push('<div class="brief-attention">! ' + escapeHtml(p.Name) + ' (' + escapeHtml(ws.SessionID.substring(0,8)) + ')' + msg + '</div>');
              }
            });
          }
        });
      }

      // Project updates using OneLine
      if (brief.Projects) {
        brief.Projects.forEach(p => {
          const summary = p.OneLine || (p.TotalEdits > 0 ? p.TotalEdits + ' edits' : '');
          if (!summary && p.ActiveCount === 0) return;
          const branch = p.Branch ? ' [' + escapeHtml(p.Branch) + (p.IsDirty ? '*' : '') + ']' : '';
          const tag = p.LatestTag ? ' ' + escapeHtml(p.LatestTag) : '';
          if (summary) {
            parts.push('<div class="brief-update"><b>' + escapeHtml(p.Name) + '</b> \u2014 ' + escapeHtml(summary) + '<span style="color:var(--cyan)">' + branch + tag + '</span></div>');
          }
        });
      }

      // Cost + totals
      const footer = [];
      if (brief.TotalEdits) footer.push(brief.TotalEdits + ' edits');
      if (brief.TotalCommands) footer.push(brief.TotalCommands + ' commands');
      if (brief.CostEstimate > 0) footer.push('~$' + Math.round(brief.CostEstimate) + ' est.');
      if (footer.length > 0) {
        parts.push('<div class="brief-idle" style="margin-top:8px;border-top:1px solid var(--border);padding-top:8px">' + footer.join(' \u00b7 ') + '</div>');
      }

      content.innerHTML = parts.length > 0 ? parts.join('') : '<div class="brief-idle">No activity today.</div>';
    })
    .catch(() => {
      document.getElementById('brief-content').innerHTML = '<div class="brief-idle">Brief unavailable.</div>';
    });
}

connect();
loadBrief();
</script>
</body>
</html>`
