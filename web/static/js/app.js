// NebulaPanel Frontend Application
(function(){
'use strict';

// ── State ──
const state = {
  token: localStorage.getItem('token') || '',
  role: localStorage.getItem('role') || '',
  username: localStorage.getItem('username') || '',
  page: 'dashboard',
};

// ── API ──
async function api(method, path, body) {
  const opts = { method, headers: {'Content-Type':'application/json'} };
  if (state.token) opts.headers['Authorization'] = 'Bearer ' + state.token;
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(path, opts);
  if (res.status === 401) { logout(); throw new Error('Unauthorized'); }
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) {
    const json = await res.json();
    if (json.code !== 0 && json.code !== undefined) throw new Error(json.message || 'Error');
    return json.data !== undefined ? json.data : json;
  }
  return res;
}

// ── Toast ──
function toast(msg, type='info') {
  const el = document.createElement('div');
  el.className = 'toast toast-' + type;
  el.innerHTML = msg + '<span style="cursor:pointer;margin-left:12px" onclick="this.parentElement.remove()">&times;</span>';
  document.getElementById('toasts').appendChild(el);
  setTimeout(() => el.remove(), 4000);
}

// ── Util ──
function formatBytes(b) {
  if (!b || b === 0) return '0 B';
  const u = ['B','KB','MB','GB','TB'];
  const i = Math.floor(Math.log(b) / Math.log(1024));
  return (b / Math.pow(1024, i)).toFixed(2) + ' ' + u[i];
}
function escHtml(s) {
  if (s === null || s === undefined) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}
function $(sel) { return document.querySelector(sel); }

// ── Icons (inline SVG) ──
const icons = {
  dashboard: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>',
  users: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>',
  nodes: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4m0 14v4M4.22 4.22l2.83 2.83m9.9 9.9l2.83 2.83M1 12h4m14 0h4M4.22 19.78l2.83-2.83m9.9-9.9l2.83-2.83"/></svg>',
  agents: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>',
  templates: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
  settings: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',
  logs: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>',
  traffic: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',
  logout: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>',
};

// ── Auth ──
function setAuth(token, role, username) {
  state.token = token; state.role = role; state.username = username;
  localStorage.setItem('token', token);
  localStorage.setItem('role', role);
  localStorage.setItem('username', username);
}
function logout() {
  state.token = ''; state.role = ''; state.username = '';
  localStorage.clear();
  render();
}

// ── Router ──
function navigate(page) {
  state.page = page;
  render();
}

// ── Render ──
function render() {
  const app = document.getElementById('app');
  if (!state.token) {
    renderLogin(app);
  } else {
    renderApp(app);
  }
}

// ── Login / Register ──
let loginMode = 'login';
let captchaId = '';
let captchaSvg = '';
let needCaptcha = false;

async function loadCaptcha() {
  try {
    const d = await api('GET', '/api/captcha');
    captchaId = d.captcha_id;
    captchaSvg = d.captcha_svg;
  } catch(e) {}
}

async function checkNeedCaptcha() {
  try {
    const d = await api('GET', '/api/login/need-captcha');
    needCaptcha = d.need;
  } catch(e) {}
}

function renderLogin(app) {
  const isReg = loginMode === 'register';
  app.innerHTML = `<div class="login-page"><div class="login-box">
    <h1>NebulaPanel</h1>
    <p class="subtitle">${isReg ? 'Create Account' : 'Sign In'}</p>
    <form id="authForm">
      <div class="form-group"><label>Username</label><input class="form-control" id="authUser" placeholder="Username" required minlength="3"></div>
      <div class="form-group"><label>Password</label><input class="form-control" id="authPass" type="password" placeholder="Password" required minlength="6"></div>
      <div id="captchaArea" style="display:${isReg || needCaptcha ? 'block' : 'none'}">
        <div class="captcha-row">
          <div class="form-group"><label>Captcha</label><input class="form-control" id="authCaptcha" placeholder="Enter captcha"></div>
          <div id="captchaImg" style="cursor:pointer;margin-bottom:16px" title="Click to refresh"></div>
        </div>
      </div>
      <button class="btn btn-primary" style="width:100%;justify-content:center;padding:12px" type="submit">${isReg ? 'Register' : 'Login'}</button>
    </form>
    <div class="switch-link">${isReg
      ? 'Already have an account? <a href="#" id="switchAuth">Login</a>'
      : 'No account? <a href="#" id="switchAuth">Register</a>'}</div>
  </div></div>`;

  if (isReg || needCaptcha) loadCaptchaUI();

  document.getElementById('switchAuth').onclick = async (e) => {
    e.preventDefault();
    loginMode = loginMode === 'login' ? 'register' : 'login';
    if (loginMode === 'register') await loadCaptcha();
    render();
  };

  document.getElementById('authForm').onsubmit = async (e) => {
    e.preventDefault();
    const user = document.getElementById('authUser').value.trim();
    const pass = document.getElementById('authPass').value;
    const cap = document.getElementById('authCaptcha')?.value?.trim() || '';
    try {
      if (isReg) {
        await api('POST', '/api/register', {username:user, password:pass, captcha_id:captchaId, captcha:cap});
        toast('Registration successful! Please login.', 'success');
        loginMode = 'login';
        render();
      } else {
        const d = await api('POST', '/api/login', {username:user, password:pass, captcha_id:captchaId, captcha:cap});
        setAuth(d.token, d.role, d.username);
        toast('Welcome back, ' + d.username, 'success');
        state.page = 'dashboard';
        render();
      }
    } catch(err) {
      toast(err.message, 'error');
      await checkNeedCaptcha();
      if (needCaptcha) { await loadCaptcha(); render(); }
    }
  };
}

async function loadCaptchaUI() {
  await loadCaptcha();
  const el = document.getElementById('captchaImg');
  if (el) {
    el.innerHTML = captchaSvg;
    el.onclick = async () => { await loadCaptcha(); el.innerHTML = captchaSvg; };
  }
}

// ── Main App ──
function renderApp(app) {
  const isAdmin = state.role === 'admin';
  const navItems = [
    {id:'dashboard', label:'Dashboard', icon:icons.dashboard},
    {id:'traffic', label:'Traffic', icon:icons.traffic},
  ];
  if (isAdmin) {
    navItems.push(
      {id:'users', label:'Users', icon:icons.users},
      {id:'nodes', label:'Nodes', icon:icons.nodes},
      {id:'agents', label:'Agents', icon:icons.agents},
      {id:'templates', label:'Templates', icon:icons.templates},
      {id:'settings', label:'Settings', icon:icons.settings},
      {id:'logs', label:'Logs', icon:icons.logs},
    );
  }

  app.innerHTML = `<div class="app">
    <aside class="sidebar">
      <div class="sidebar-brand"><h1>NebulaPanel</h1><small>Proxy Management</small></div>
      <nav class="sidebar-nav">${navItems.map(n =>
        `<a href="#" data-page="${n.id}" class="${state.page===n.id?'active':''}">${n.icon}<span>${n.label}</span></a>`
      ).join('')}</nav>
      <div class="sidebar-footer">
        <div style="margin-bottom:8px">${escHtml(state.username)} <span class="badge badge-${isAdmin?'primary':'info'}">${state.role}</span></div>
        <a href="#" id="logoutBtn" style="color:var(--danger);display:flex;align-items:center;gap:6px">${icons.logout}<span>Logout</span></a>
      </div>
    </aside>
    <main class="main" id="mainContent"><div class="loading"><div class="spinner"></div>Loading...</div></main>
  </div>`;

  document.querySelectorAll('.sidebar-nav a').forEach(a => {
    a.onclick = (e) => { e.preventDefault(); navigate(a.dataset.page); };
  });
  document.getElementById('logoutBtn').onclick = (e) => { e.preventDefault(); logout(); };

  loadPage(state.page);
}

async function loadPage(page) {
  const main = document.getElementById('mainContent');
  try {
    switch(page) {
      case 'dashboard': await renderDashboard(main); break;
      case 'traffic': await renderTraffic(main); break;
      case 'users': await renderUsers(main); break;
      case 'nodes': await renderNodes(main); break;
      case 'agents': await renderAgents(main); break;
      case 'templates': await renderTemplates(main); break;
      case 'settings': await renderSettings(main); break;
      case 'logs': await renderLogs(main); break;
      default: main.innerHTML = '<div class="empty">Page not found</div>';
    }
  } catch(err) { main.innerHTML = `<div class="empty">Error: ${escHtml(err.message)}</div>`; }
}

// ── Dashboard ──
async function renderDashboard(el) {
  const d = await api('GET', '/api/dashboard');
  const u = d.user;
  const isAdmin = state.role === 'admin';

  let html = `<div class="topbar"><h2>Dashboard</h2><div class="topbar-actions"><span style="color:var(--text-dim);font-size:13px">${new Date().toLocaleDateString()}</span></div></div>`;

  // User stats
  html += `<div class="stats-grid">
    <div class="stat-card info"><div class="label">Upload</div><div class="value">${formatBytes(u.traffic_up)}</div></div>
    <div class="stat-card primary"><div class="label">Download</div><div class="value">${formatBytes(u.traffic_down)}</div></div>
    <div class="stat-card ${u.traffic_limit > 0 && u.traffic_used >= u.traffic_limit ? 'danger' : 'success'}">
      <div class="label">Traffic Used / Limit</div>
      <div class="value">${formatBytes(u.traffic_used)}</div>
      <div class="sub">${u.traffic_limit > 0 ? 'Limit: ' + formatBytes(u.traffic_limit) : 'Unlimited'}</div>
    </div>
    <div class="stat-card ${u.expire_at ? 'warning' : 'success'}">
      <div class="label">Expire</div>
      <div class="value" style="font-size:18px">${u.expire_at || 'Never'}</div>
      <div class="sub">${u.speed_limit > 0 ? 'Speed: ' + u.speed_limit + ' Mbps' : 'No speed limit'}</div>
    </div>
  </div>`;

  // Admin overview
  if (isAdmin && d.admin) {
    const a = d.admin;
    html += `<div class="stats-grid">
      <div class="stat-card primary"><div class="label">Total Users</div><div class="value">${a.total_users}</div></div>
      <div class="stat-card info"><div class="label">Total Nodes</div><div class="value">${a.total_nodes}</div></div>
      <div class="stat-card success"><div class="label">Agents Online</div><div class="value">${a.online_agents} / ${a.total_agents}</div></div>
      <div class="stat-card warning"><div class="label">Total Traffic</div><div class="value">${formatBytes(a.total_traffic_up + a.total_traffic_down)}</div></div>
    </div>`;
  }

  // Subscription info
  const me = await api('GET', '/api/me');
  html += `<div class="card"><div class="card-header"><h3>Subscription</h3></div>
    <div class="copy-wrap">
      <input class="form-control" readonly value="${location.origin}/api/sub/${escHtml(me.sub_token)}?format=clash" id="subUrl">
      <button class="btn btn-primary copy-btn" onclick="copyText('subUrl')">Copy</button>
    </div>
    <div style="margin-top:8px;font-size:12px;color:var(--text-dim)">Formats: ?format=clash | base64 | surge</div>
  </div>`;

  // Today traffic chart
  html += `<div class="card"><div class="card-header"><h3>Today Traffic Trend</h3></div>
    <div class="chart-container"><canvas id="dashChart"></canvas></div></div>`;

  el.innerHTML = html;

  // Render chart
  if (d.today_chart) renderHourlyChart('dashChart', d.today_chart);
}

// ── Traffic Page ──
async function renderTraffic(el) {
  const stats = await api('GET', '/api/traffic/stats');
  const today = new Date().toISOString().split('T')[0];

  let html = `<div class="topbar"><h2>Traffic Statistics</h2></div>`;
  html += `<div class="stats-grid">
    <div class="stat-card info"><div class="label">Upload</div><div class="value">${formatBytes(stats.traffic_up)}</div></div>
    <div class="stat-card primary"><div class="label">Download</div><div class="value">${formatBytes(stats.traffic_down)}</div></div>
    <div class="stat-card success"><div class="label">Total Used</div><div class="value">${formatBytes(stats.traffic_used)}</div>
      <div class="sub">${stats.traffic_limit > 0 ? 'Limit: ' + formatBytes(stats.traffic_limit) : 'Unlimited'}</div></div>
    <div class="stat-card warning"><div class="label">Speed Limit</div><div class="value">${stats.speed_limit > 0 ? stats.speed_limit + ' Mbps' : 'None'}</div></div>
  </div>`;

  html += `<div class="card"><div class="card-header"><h3>Hourly Traffic</h3>
    <div class="chart-toolbar"><input type="date" id="chartDate" value="${today}"><button class="btn btn-sm btn-outline" id="loadChartBtn">Load</button></div>
  </div><div class="chart-container"><canvas id="trafficChart"></canvas></div></div>`;

  el.innerHTML = html;

  async function loadChart(date) {
    const cd = await api('GET', '/api/traffic/chart?date=' + date);
    renderHourlyChart('trafficChart', cd.chart);
  }
  await loadChart(today);

  document.getElementById('loadChartBtn').onclick = () => {
    const d = document.getElementById('chartDate').value;
    if (d) loadChart(d);
  };
}

function renderHourlyChart(canvasId, data) {
  const canvas = document.getElementById(canvasId);
  if (!canvas) return;
  const ctx = canvas.getContext('2d');

  if (canvas._chart) canvas._chart.destroy();

  canvas._chart = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: data.map(d => d.hour),
      datasets: [
        { label: 'Upload', data: data.map(d => d.up), backgroundColor: 'rgba(99,102,241,0.7)', borderRadius: 4 },
        { label: 'Download', data: data.map(d => d.down), backgroundColor: 'rgba(59,130,246,0.7)', borderRadius: 4 },
      ]
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        tooltip: {
          callbacks: {
            title: (items) => { const i = items[0].dataIndex; return data[i].time; },
            label: (ctx) => ctx.dataset.label + ': ' + formatBytes(ctx.raw),
            footer: (items) => { const i = items[0].dataIndex; return 'Total: ' + formatBytes(data[i].total); }
          }
        },
        legend: { labels: { color: '#8b8fa3' } }
      },
      scales: {
        x: { ticks: { color: '#8b8fa3' }, grid: { color: 'rgba(42,45,58,0.5)' } },
        y: { ticks: { color: '#8b8fa3', callback: v => formatBytes(v) }, grid: { color: 'rgba(42,45,58,0.5)' } }
      }
    }
  });
}

// ── Users ──
async function renderUsers(el) {
  const users = await api('GET', '/api/users');
  let html = `<div class="topbar"><h2>Users</h2><button class="btn btn-primary" id="addUserBtn">+ Add User</button></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>ID</th><th>Username</th><th>Role</th><th>Traffic Used</th><th>Limit</th><th>Speed</th><th>Expire</th><th>Status</th><th>Actions</th></tr></thead>
    <tbody>${users.map(u => `<tr>
      <td>${u.id}</td><td>${escHtml(u.username)}</td>
      <td><span class="badge badge-${u.role==='admin'?'primary':'info'}">${u.role}</span></td>
      <td>${formatBytes(u.traffic_used)}</td>
      <td>${u.traffic_limit > 0 ? formatBytes(u.traffic_limit) : 'Unlimited'}</td>
      <td>${u.speed_limit > 0 ? u.speed_limit + ' Mbps' : '-'}</td>
      <td>${u.expire_at || 'Never'}</td>
      <td><span class="badge badge-${u.enabled?'success':'danger'}">${u.enabled?'Active':'Disabled'}</span></td>
      <td><div class="btn-group">
        <button class="btn btn-sm btn-outline" onclick="editUser(${u.id})">Edit</button>
        <button class="btn btn-sm btn-outline" onclick="resetTraffic(${u.id})">Reset</button>
        <button class="btn btn-sm btn-outline" onclick="assignNodes(${u.id})">Nodes</button>
        ${u.role!=='admin'?`<button class="btn btn-sm btn-danger" onclick="deleteUser(${u.id})">Del</button>`:''}
      </div></td>
    </tr>`).join('')}</tbody></table></div></div>`;
  el.innerHTML = html;

  document.getElementById('addUserBtn').onclick = () => showUserModal();
}

window.editUser = async (id) => {
  const users = await api('GET', '/api/users');
  const u = users.find(x => x.id === id);
  if (u) showUserModal(u);
};

window.deleteUser = async (id) => {
  if (!confirm('Delete this user?')) return;
  await api('DELETE', '/api/users/' + id);
  toast('User deleted', 'success');
  loadPage('users');
};

window.resetTraffic = async (id) => {
  if (!confirm('Reset traffic for this user?')) return;
  await api('POST', '/api/users/' + id + '/reset-traffic');
  toast('Traffic reset', 'success');
  loadPage('users');
};

window.assignNodes = async (uid) => {
  const [nodes, assigned] = await Promise.all([api('GET', '/api/nodes'), api('GET', '/api/users/' + uid + '/nodes')]);
  const set = new Set(assigned);
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal">
    <div class="modal-header"><h3>Assign Nodes</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body">${nodes.map(n => `<label style="display:flex;align-items:center;gap:8px;padding:6px 0;cursor:pointer">
      <input type="checkbox" value="${n.id}" ${set.has(n.id)?'checked':''} class="nodeCheck"> ${escHtml(n.name)} (${n.address}:${n.port})
    </label>`).join('')}
    ${nodes.length===0?'<div class="empty">No nodes</div>':''}
    </div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">Cancel</button><button class="btn btn-primary" id="saveNodesBtn">Save</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);
  document.getElementById('saveNodesBtn').onclick = async () => {
    const ids = [...document.querySelectorAll('.nodeCheck:checked')].map(c => parseInt(c.value));
    await api('PUT', '/api/users/' + uid + '/nodes', {node_ids: ids});
    toast('Nodes assigned', 'success');
    closeModal();
  };
};

function showUserModal(user) {
  const isEdit = !!user;
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal">
    <div class="modal-header"><h3>${isEdit ? 'Edit User' : 'Add User'}</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body"><form id="userForm">
      <div class="form-group"><label>Username</label><input class="form-control" id="fUsername" value="${isEdit?escHtml(user.username):''}" ${isEdit?'':'required'} minlength="3"></div>
      <div class="form-group"><label>Password ${isEdit?'(leave empty to keep)':''}</label><input class="form-control" id="fPassword" type="password" ${isEdit?'':'required'} minlength="6"></div>
      <div class="form-row">
        <div class="form-group"><label>Role</label><select class="form-control" id="fRole"><option value="user" ${isEdit&&user.role==='user'?'selected':''}>User</option><option value="admin" ${isEdit&&user.role==='admin'?'selected':''}>Admin</option></select></div>
        <div class="form-group"><label>Status</label><select class="form-control" id="fEnabled"><option value="1" ${!isEdit||user.enabled?'selected':''}>Active</option><option value="0" ${isEdit&&!user.enabled?'selected':''}>Disabled</option></select></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>Traffic Limit (GB, 0=unlimited)</label><input class="form-control" id="fTrafficLimit" type="number" value="${isEdit ? Math.round((user.traffic_limit||0)/1073741824) : 0}" min="0"></div>
        <div class="form-group"><label>Speed Limit (Mbps, 0=unlimited)</label><input class="form-control" id="fSpeedLimit" type="number" value="${isEdit?user.speed_limit||0:0}" min="0"></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>Expire Date</label><input class="form-control" id="fExpire" type="datetime-local" value="${isEdit&&user.expire_at?user.expire_at.replace(' ','T'):''}"></div>
        <div class="form-group"><label>Traffic Reset Day (0=no auto reset)</label><input class="form-control" id="fResetDay" type="number" value="${isEdit?user.reset_day||0:0}" min="0" max="31"></div>
      </div>
    </form></div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">Cancel</button><button class="btn btn-primary" id="saveUserBtn">Save</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);

  document.getElementById('saveUserBtn').onclick = async () => {
    const data = {
      username: document.getElementById('fUsername').value.trim(),
      role: document.getElementById('fRole').value,
      enabled: parseInt(document.getElementById('fEnabled').value),
      traffic_limit: parseInt(document.getElementById('fTrafficLimit').value) * 1073741824,
      speed_limit: parseInt(document.getElementById('fSpeedLimit').value),
      expire_at: document.getElementById('fExpire').value ? document.getElementById('fExpire').value.replace('T',' ') + ':00' : '',
      reset_day: parseInt(document.getElementById('fResetDay').value),
    };
    const pw = document.getElementById('fPassword').value;
    if (pw) data.password = pw;
    try {
      if (isEdit) { await api('PUT', '/api/users/' + user.id, data); }
      else { data.password = pw; await api('POST', '/api/users', data); }
      toast(isEdit ? 'User updated' : 'User created', 'success');
      closeModal(); loadPage('users');
    } catch(e) { toast(e.message, 'error'); }
  };
}

// ── Nodes ──
async function renderNodes(el) {
  const nodes = await api('GET', '/api/nodes');
  let html = `<div class="topbar"><h2>Nodes</h2><button class="btn btn-primary" id="addNodeBtn">+ Add Node</button></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>ID</th><th>Name</th><th>Address</th><th>Protocol</th><th>Transport</th><th>TLS</th><th>Status</th><th>Actions</th></tr></thead>
    <tbody>${nodes.map(n => `<tr>
      <td>${n.id}</td><td>${escHtml(n.name)}</td><td>${escHtml(n.address)}:${n.port}</td>
      <td><span class="badge badge-info">${n.protocol}</span></td>
      <td>${n.transport}</td><td>${n.tls?'Yes':'No'}</td>
      <td><span class="badge badge-${n.enabled?'success':'danger'}">${n.enabled?'Enabled':'Disabled'}</span></td>
      <td><div class="btn-group">
        <button class="btn btn-sm btn-outline" onclick="toggleNode(${n.id})">${n.enabled?'Disable':'Enable'}</button>
        <button class="btn btn-sm btn-outline" onclick="editNode(${n.id})">Edit</button>
        <button class="btn btn-sm btn-danger" onclick="deleteNode(${n.id})">Del</button>
      </div></td>
    </tr>`).join('')}${nodes.length===0?'<tr><td colspan="8" class="empty">No nodes yet</td></tr>':''}</tbody></table></div></div>`;
  el.innerHTML = html;
  document.getElementById('addNodeBtn').onclick = () => showNodeModal();
}

window.toggleNode = async (id) => {
  await api('PUT', '/api/nodes/' + id + '/toggle');
  toast('Node toggled', 'success');
  loadPage('nodes');
};

window.editNode = async (id) => {
  const nodes = await api('GET', '/api/nodes');
  const n = nodes.find(x => x.id === id);
  if (n) showNodeModal(n);
};

window.deleteNode = async (id) => {
  if (!confirm('Delete this node?')) return;
  await api('DELETE', '/api/nodes/' + id);
  toast('Node deleted', 'success');
  loadPage('nodes');
};

function showNodeModal(node) {
  const isEdit = !!node;
  const protocols = ['vmess','vless','trojan','ss','hysteria2'];
  const transports = ['tcp','ws','grpc','h2','quic'];
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal">
    <div class="modal-header"><h3>${isEdit?'Edit Node':'Add Node'}</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body"><form id="nodeForm">
      <div class="form-group"><label>Name</label><input class="form-control" id="nName" value="${isEdit?escHtml(node.name):''}" required></div>
      <div class="form-row">
        <div class="form-group"><label>Address</label><input class="form-control" id="nAddr" value="${isEdit?escHtml(node.address):''}" required></div>
        <div class="form-group"><label>Port</label><input class="form-control" id="nPort" type="number" value="${isEdit?node.port:443}" required></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>Protocol</label><select class="form-control" id="nProto">${protocols.map(p=>`<option value="${p}" ${isEdit&&node.protocol===p?'selected':''}>${p}</option>`).join('')}</select></div>
        <div class="form-group"><label>Transport</label><select class="form-control" id="nTrans">${transports.map(t=>`<option value="${t}" ${isEdit&&node.transport===t?'selected':''}>${t}</option>`).join('')}</select></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>UUID / Password</label><input class="form-control" id="nUUID" value="${isEdit?escHtml(node.uuid):''}"></div>
        <div class="form-group"><label>Alter ID</label><input class="form-control" id="nAltID" type="number" value="${isEdit?node.alter_id:0}"></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>TLS</label><select class="form-control" id="nTLS"><option value="0" ${isEdit&&!node.tls?'selected':''}>Off</option><option value="1" ${isEdit&&node.tls?'selected':''}>On</option></select></div>
        <div class="form-group"><label>TLS SNI</label><input class="form-control" id="nSNI" value="${isEdit?escHtml(node.tls_sni):''}"></div>
      </div>
      <div class="form-group"><label>Extra Config (JSON)</label><textarea class="form-control" id="nExtra">${isEdit?escHtml(node.extra_config):''}</textarea></div>
      <div class="form-group"><label>Sort Order</label><input class="form-control" id="nSort" type="number" value="${isEdit?node.sort_order:0}"></div>
    </form></div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">Cancel</button><button class="btn btn-primary" id="saveNodeBtn">Save</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);

  document.getElementById('saveNodeBtn').onclick = async () => {
    const data = {
      name: document.getElementById('nName').value.trim(),
      address: document.getElementById('nAddr').value.trim(),
      port: parseInt(document.getElementById('nPort').value),
      protocol: document.getElementById('nProto').value,
      transport: document.getElementById('nTrans').value,
      uuid: document.getElementById('nUUID').value.trim(),
      alter_id: parseInt(document.getElementById('nAltID').value),
      tls: parseInt(document.getElementById('nTLS').value),
      tls_sni: document.getElementById('nSNI').value.trim(),
      extra_config: document.getElementById('nExtra').value.trim(),
      sort_order: parseInt(document.getElementById('nSort').value),
    };
    try {
      if (isEdit) await api('PUT', '/api/nodes/' + node.id, data);
      else await api('POST', '/api/nodes', data);
      toast(isEdit?'Node updated':'Node created', 'success');
      closeModal(); loadPage('nodes');
    } catch(e) { toast(e.message, 'error'); }
  };
}

// ── Agents ──
async function renderAgents(el) {
  const agents = await api('GET', '/api/agents');
  let html = `<div class="topbar"><h2>Agents</h2><div class="topbar-actions">
    <button class="btn btn-outline" id="showScriptBtn">Install Script</button>
    <button class="btn btn-primary" id="addAgentBtn">+ Add Agent</button>
  </div></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>ID</th><th>Name</th><th>Host</th><th>Status</th><th>CPU</th><th>Memory</th><th>Net In/Out</th><th>Uptime</th><th>Last HB</th><th>Actions</th></tr></thead>
    <tbody>${agents.map(a => `<tr>
      <td>${a.id}</td><td>${escHtml(a.name)}</td><td>${escHtml(a.host)}:${a.port}</td>
      <td><span class="badge badge-${a.status==='online'?'success':'danger'}">${a.status}</span></td>
      <td>${a.cpu_usage.toFixed(1)}%</td><td>${a.mem_usage.toFixed(1)}%</td>
      <td>${formatBytes(a.net_in)} / ${formatBytes(a.net_out)}</td>
      <td>${a.uptime > 0 ? Math.floor(a.uptime/3600)+'h' : '-'}</td>
      <td style="font-size:12px">${a.last_heartbeat||'-'}</td>
      <td><div class="btn-group">
        <button class="btn btn-sm btn-outline" onclick="editAgent(${a.id})">Edit</button>
        <button class="btn btn-sm btn-danger" onclick="deleteAgent(${a.id})">Del</button>
      </div></td>
    </tr>`).join('')}${agents.length===0?'<tr><td colspan="10" class="empty">No agents</td></tr>':''}</tbody></table></div></div>`;
  el.innerHTML = html;

  document.getElementById('addAgentBtn').onclick = () => showAgentModal();
  document.getElementById('showScriptBtn').onclick = async () => {
    const d = await api('GET', '/api/agents/install-script');
    let mhtml = `<div class="modal-overlay" id="modalOverlay"><div class="modal" style="max-width:700px">
      <div class="modal-header"><h3>Agent Install Script</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
      <div class="modal-body">
        <p style="margin-bottom:12px;color:var(--text-dim);font-size:13px">One-click install command (copy and run on target server):</p>
        <div class="copy-wrap" style="margin-bottom:16px">
          <input class="form-control" readonly value="curl -sL ${location.origin}/static/agent/install.sh | bash" id="installCmd" style="font-family:monospace;font-size:12px">
          <button class="btn btn-primary copy-btn" onclick="copyText('installCmd')">Copy</button>
        </div>
        <p style="margin-bottom:8px;color:var(--text-dim);font-size:13px">Uninstall command:</p>
        <div class="copy-wrap" style="margin-bottom:16px">
          <input class="form-control" readonly value="curl -sL ${location.origin}/static/agent/install.sh | bash -s uninstall" id="uninstallCmd" style="font-family:monospace;font-size:12px">
          <button class="btn btn-primary copy-btn" onclick="copyText('uninstallCmd')">Copy</button>
        </div>
        <details><summary style="cursor:pointer;color:var(--primary);font-size:13px">View full script</summary>
          <pre style="background:var(--bg);padding:12px;border-radius:6px;font-size:11px;overflow-x:auto;margin-top:8px;max-height:300px">${escHtml(d.script)}</pre>
        </details>
      </div>
      <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">Close</button></div>
    </div></div>`;
    document.body.insertAdjacentHTML('beforeend', mhtml);
  };
}

window.editAgent = async (id) => {
  const agents = await api('GET', '/api/agents');
  const a = agents.find(x => x.id === id);
  if (a) showAgentModal(a);
};

window.deleteAgent = async (id) => {
  if (!confirm('Delete this agent?')) return;
  await api('DELETE', '/api/agents/' + id);
  toast('Agent deleted', 'success');
  loadPage('agents');
};

function showAgentModal(agent) {
  const isEdit = !!agent;
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal">
    <div class="modal-header"><h3>${isEdit?'Edit Agent':'Add Agent'}</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body">
      <div class="form-group"><label>Name</label><input class="form-control" id="aName" value="${isEdit?escHtml(agent.name):''}" required></div>
      <div class="form-row">
        <div class="form-group"><label>Host</label><input class="form-control" id="aHost" value="${isEdit?escHtml(agent.host):''}" required></div>
        <div class="form-group"><label>Port</label><input class="form-control" id="aPort" type="number" value="${isEdit?agent.port:9527}"></div>
      </div>
    </div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">Cancel</button><button class="btn btn-primary" id="saveAgentBtn">Save</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);

  document.getElementById('saveAgentBtn').onclick = async () => {
    const data = { name: document.getElementById('aName').value.trim(), host: document.getElementById('aHost').value.trim(), port: parseInt(document.getElementById('aPort').value) };
    try {
      if (isEdit) await api('PUT', '/api/agents/' + agent.id, data);
      else await api('POST', '/api/agents', data);
      toast(isEdit?'Agent updated':'Agent created', 'success');
      closeModal(); loadPage('agents');
    } catch(e) { toast(e.message, 'error'); }
  };
}

// ── Templates ──
async function renderTemplates(el) {
  const tpls = await api('GET', '/api/templates');
  let html = `<div class="topbar"><h2>Subscription Templates</h2><button class="btn btn-primary" id="addTplBtn">+ Add Template</button></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>ID</th><th>Name</th><th>Format</th><th>Default</th><th>Actions</th></tr></thead>
    <tbody>${tpls.map(t => `<tr>
      <td>${t.id}</td><td>${escHtml(t.name)}</td>
      <td><span class="badge badge-info">${t.format}</span></td>
      <td>${t.is_default?'<span class="badge badge-success">Yes</span>':'-'}</td>
      <td><div class="btn-group">
        <button class="btn btn-sm btn-outline" onclick="editTemplate(${t.id})">Edit</button>
        <button class="btn btn-sm btn-danger" onclick="deleteTemplate(${t.id})">Del</button>
      </div></td>
    </tr>`).join('')}${tpls.length===0?'<tr><td colspan="5" class="empty">No templates</td></tr>':''}</tbody></table></div></div>`;
  html += `<div class="card"><div class="card-header"><h3>Template Variables</h3></div>
    <div style="font-size:13px;color:var(--text-dim)"><code>{{PROXIES}}</code> - Proxy list &nbsp; <code>{{PROXY_NAMES}}</code> - Proxy name list</div></div>`;
  el.innerHTML = html;
  document.getElementById('addTplBtn').onclick = () => showTemplateModal();
}

window.editTemplate = async (id) => {
  const tpls = await api('GET', '/api/templates');
  const t = tpls.find(x => x.id === id);
  if (t) showTemplateModal(t);
};

window.deleteTemplate = async (id) => {
  if (!confirm('Delete this template?')) return;
  await api('DELETE', '/api/templates/' + id);
  toast('Template deleted', 'success');
  loadPage('templates');
};

function showTemplateModal(tpl) {
  const isEdit = !!tpl;
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal" style="max-width:700px">
    <div class="modal-header"><h3>${isEdit?'Edit Template':'Add Template'}</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body">
      <div class="form-row">
        <div class="form-group"><label>Name</label><input class="form-control" id="tName" value="${isEdit?escHtml(tpl.name):''}" required></div>
        <div class="form-group"><label>Format</label><select class="form-control" id="tFormat">
          <option value="clash" ${isEdit&&tpl.format==='clash'?'selected':''}>Clash/Mihomo</option>
          <option value="surge" ${isEdit&&tpl.format==='surge'?'selected':''}>Surge</option>
          <option value="base64" ${isEdit&&tpl.format==='base64'?'selected':''}>Base64</option>
        </select></div>
      </div>
      <div class="form-group"><label>Default</label><select class="form-control" id="tDefault"><option value="0" ${isEdit&&!tpl.is_default?'selected':''}>No</option><option value="1" ${isEdit&&tpl.is_default?'selected':''}>Yes</option></select></div>
      <div class="form-group"><label>Content</label><textarea class="form-control" id="tContent" style="min-height:200px;font-size:12px">${isEdit?escHtml(tpl.content):''}</textarea></div>
    </div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">Cancel</button><button class="btn btn-primary" id="saveTplBtn">Save</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);

  document.getElementById('saveTplBtn').onclick = async () => {
    const data = { name: document.getElementById('tName').value.trim(), format: document.getElementById('tFormat').value, is_default: parseInt(document.getElementById('tDefault').value), content: document.getElementById('tContent').value };
    try {
      if (isEdit) await api('PUT', '/api/templates/' + tpl.id, data);
      else await api('POST', '/api/templates', data);
      toast(isEdit?'Template updated':'Template created', 'success');
      closeModal(); loadPage('templates');
    } catch(e) { toast(e.message, 'error'); }
  };
}

// ── Settings ──
async function renderSettings(el) {
  const settings = await api('GET', '/api/settings');
  let html = `<div class="topbar"><h2>Settings</h2></div>`;
  html += `<div class="card"><div class="card-header"><h3>General</h3></div>
    <div class="form-group"><label>Site Name</label><input class="form-control" id="sSiteName" value="${escHtml(settings.site_name||'NebulaPanel')}"></div>
    <div class="form-group"><label>Panel Host (for Agent script, e.g. your-domain.com:3000)</label><input class="form-control" id="sPanelHost" value="${escHtml(settings.panel_host||'')}"></div>
    <div class="form-group"><label>Allow Registration</label><select class="form-control" id="sAllowReg"><option value="true" ${settings.allow_register==='true'?'selected':''}>Yes</option><option value="false" ${settings.allow_register!=='true'?'selected':''}>No</option></select></div>
  </div>`;
  html += `<div class="card"><div class="card-header"><h3>Communication Key</h3></div>
    <div class="form-group"><label>Encryption Key (used for Agent communication)</label>
      <div class="copy-wrap"><input class="form-control" id="sCommKey" value="${escHtml(settings.comm_key||'')}" style="font-family:monospace;font-size:12px"><button class="btn btn-primary copy-btn" onclick="copyText('sCommKey')">Copy</button></div>
    </div>
    <p style="font-size:12px;color:var(--text-dim);margin-top:4px">All agents use this key for AES-256-GCM encrypted communication. Change will require re-deploying all agents.</p>
  </div>`;
  html += `<div class="card"><div class="card-header"><h3>Data Management</h3></div>
    <div class="btn-group">
      <button class="btn btn-primary" id="exportBtn">Export Data</button>
      <label class="btn btn-outline" style="cursor:pointer">Import Data<input type="file" accept=".json" id="importFile" style="display:none"></label>
    </div></div>`;
  html += `<div style="margin-top:16px"><button class="btn btn-primary" id="saveSettingsBtn">Save Settings</button></div>`;
  el.innerHTML = html;

  document.getElementById('saveSettingsBtn').onclick = async () => {
    const data = {
      site_name: document.getElementById('sSiteName').value.trim(),
      panel_host: document.getElementById('sPanelHost').value.trim(),
      allow_register: document.getElementById('sAllowReg').value,
      comm_key: document.getElementById('sCommKey').value.trim(),
    };
    await api('PUT', '/api/settings', data);
    toast('Settings saved', 'success');
  };

  document.getElementById('exportBtn').onclick = async () => {
    const res = await fetch('/api/export', { headers: {'Authorization': 'Bearer ' + state.token} });
    const blob = await res.blob();
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'nebula_backup_' + new Date().toISOString().slice(0,10) + '.json';
    a.click();
    toast('Data exported', 'success');
  };

  document.getElementById('importFile').onchange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    if (!confirm('This will overwrite all existing data. Continue?')) return;
    const text = await file.text();
    try {
      JSON.parse(text);
      await api('POST', '/api/import', JSON.parse(text));
      toast('Data imported successfully', 'success');
      loadPage('settings');
    } catch(err) { toast('Import failed: ' + err.message, 'error'); }
  };
}

// ── Logs ──
let logPage = 1;
async function renderLogs(el) {
  const d = await api('GET', `/api/logs?page=${logPage}&page_size=20`);
  let html = `<div class="topbar"><h2>Logs</h2><span style="color:var(--text-dim);font-size:13px">Retention: 30 days</span></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>Time</th><th>Level</th><th>Module</th><th>Message</th></tr></thead>
    <tbody>${d.logs.map(l => `<tr>
      <td style="font-size:12px;white-space:nowrap">${l.created_at}</td>
      <td><span class="badge badge-${l.level==='error'?'danger':l.level==='warn'?'warning':'info'}">${l.level}</span></td>
      <td>${escHtml(l.module)}</td>
      <td style="font-size:13px">${escHtml(l.message)}</td>
    </tr>`).join('')}${d.logs.length===0?'<tr><td colspan="4" class="empty">No logs</td></tr>':''}</tbody></table></div>
    <div class="pagination">
      <button ${logPage<=1?'disabled':''} onclick="logNav(${logPage-1})">Prev</button>
      <button class="active">${logPage}</button>
      <button ${d.logs.length<20?'disabled':''} onclick="logNav(${logPage+1})">Next</button>
    </div></div>`;
  el.innerHTML = html;
}
window.logNav = (p) => { logPage = p; loadPage('logs'); };

// ── Helpers ──
window.closeModal = () => {
  const m = document.getElementById('modalOverlay');
  if (m) m.remove();
};

window.copyText = (id) => {
  const el = document.getElementById(id);
  if (!el) return;
  navigator.clipboard.writeText(el.value).then(() => toast('Copied!', 'success')).catch(() => {
    el.select(); document.execCommand('copy'); toast('Copied!', 'success');
  });
};

// ── Init ──
checkNeedCaptcha().then(() => render());

})();
