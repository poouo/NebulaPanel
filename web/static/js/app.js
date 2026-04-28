// NebulaPanel Frontend Application
// Features: i18n (zh/en), Light/Dark theme, full SPA
(function(){
'use strict';

// ══════════════════════════════════════
// ── i18n 国际化
// ══════════════════════════════════════
const i18n = {
zh: {
  // Auth
  sign_in: '登录', sign_up: '注册', username: '用户名', password: '密码',
  captcha: '验证码', captcha_refresh: '点击刷新', login: '登录', register: '注册',
  no_account: '没有账号？', have_account: '已有账号？', reg_success: '注册成功，请登录',
  welcome_back: '欢迎回来，',
  leave_empty_keep: '（留空保持不变）',
  // Nav
  dashboard: '仪表盘', traffic: '流量统计', users: '用户管理', nodes: '节点管理',
  agents: 'Agent管理', templates: '订阅模板', settings: '系统设置', logs: '系统日志',
  logout: '退出登录',
  // Dashboard
  upload: '上行流量', download: '下行流量', traffic_used_limit: '已用 / 限额',
  unlimited: '无限制', expire: '到期时间', never: '永不过期', speed_limit: '速率限制',
  no_speed_limit: '无速率限制', total_users: '总用户数', total_nodes: '总节点数',
  agents_online: 'Agent在线', total_traffic: '总流量', subscription: '订阅链接',
  copy: '复制', formats_hint: '格式: ?format=clash | base64 | surge',
  today_traffic: '今日流量趋势', copied: '已复制！',
  // Traffic
  traffic_stats: '流量统计', hourly_traffic: '每小时流量', load: '加载',
  // Users
  add_user: '添加用户', edit_user: '编辑用户', role: '角色', traffic_used: '已用流量',
  limit: '限额', speed: '速率', status: '状态', actions: '操作',
  active: '启用', disabled: '禁用', edit: '编辑', reset: '重置', del: '删除',
  confirm_delete_user: '确定删除该用户？', user_deleted: '用户已删除',
  confirm_reset_traffic: '确定重置该用户流量？', traffic_reset: '流量已重置',
  assign_nodes: '分配节点', nodes_assigned: '节点已分配',
  traffic_limit_gb: '流量限额 (GB, 0=无限)', speed_limit_mbps: '速率限制 (Mbps, 0=无限)',
  expire_date: '到期日期', reset_day: '流量重置日 (0=不自动重置)',
  save: '保存', cancel: '取消', user_created: '用户已创建', user_updated: '用户已更新',
  admin: '管理员', user: '普通用户',
  // Nodes
  add_node: '添加节点', edit_node: '编辑节点', name: '名称', address: '地址',
  protocol: '协议', transport: '传输', enabled: '已启用',
  confirm_delete_node: '确定删除该节点？', node_deleted: '节点已删除',
  node_toggled: '节点状态已切换', enable: '启用', disable: '禁用',
  port: '端口', uuid_password: 'UUID / 密码', alter_id: 'Alter ID',
  tls: 'TLS', tls_sni: 'TLS SNI', extra_config: '额外配置 (JSON)', sort_order: '排序',
  node_created: '节点已创建', node_updated: '节点已更新',
  // Agents
  add_agent: '添加Agent', edit_agent: '编辑Agent', install_script: '安装脚本',
  host: '主机', cpu: 'CPU', memory: '内存', net_in_out: '网络 入/出',
  uptime: '运行时间', last_hb: '最后心跳',
  confirm_delete_agent: '确定删除该Agent？', agent_deleted: 'Agent已删除',
  agent_created: 'Agent已创建', agent_updated: 'Agent已更新',
  install_cmd_hint: '一键安装命令（在目标服务器上执行）：',
  uninstall_cmd_hint: '卸载命令：',
  close: '关闭',
  // Templates
  add_template: '添加模板', edit_template: '编辑模板', format: '格式',
  is_default: '默认', content: '内容', yes: '是', no: '否',
  confirm_delete_tpl: '确定删除该模板？', tpl_deleted: '模板已删除',
  tpl_created: '模板已创建', tpl_updated: '模板已更新',
  tpl_vars: '模板变量',
  // Settings
  general: '常规设置', site_name: '站点名称',
  panel_host: '面板地址（用于Agent脚本，如 your-domain.com:3001）',
  allow_register: '允许注册', comm_key: '通信密钥',
  comm_key_desc: '所有Agent使用此密钥进行AES-256-GCM加密通信。修改后需重新部署所有Agent。',
  data_management: '数据管理', export_data: '导出数据', import_data: '导入数据',
  import_confirm: '导入将覆盖所有现有数据，确定继续？',
  export_success: '数据已导出', import_success: '数据导入成功',
  import_failed: '导入失败：', settings_saved: '设置已保存',
  save_settings: '保存设置',
  // Logs
  retention: '保留期限：30天', time: '时间', level: '级别', module: '模块', message: '消息',
  prev: '上一页', next: '下一页',
  // Theme
  light_mode: '浅色', dark_mode: '深色',
  // Misc
  no_data: '暂无数据', error: '错误', page_not_found: '页面未找到',
  online: '在线', offline: '离线',
},
en: {
  sign_in: 'Sign In', sign_up: 'Sign Up', username: 'Username', password: 'Password',
  captcha: 'Captcha', captcha_refresh: 'Click to refresh', login: 'Login', register: 'Register',
  no_account: 'No account?', have_account: 'Already have an account?', reg_success: 'Registration successful! Please login.',
  welcome_back: 'Welcome back, ',
  leave_empty_keep: '(leave empty to keep)',
  dashboard: 'Dashboard', traffic: 'Traffic', users: 'Users', nodes: 'Nodes',
  agents: 'Agents', templates: 'Templates', settings: 'Settings', logs: 'Logs',
  logout: 'Logout',
  upload: 'Upload', download: 'Download', traffic_used_limit: 'Used / Limit',
  unlimited: 'Unlimited', expire: 'Expire', never: 'Never', speed_limit: 'Speed Limit',
  no_speed_limit: 'No speed limit', total_users: 'Total Users', total_nodes: 'Total Nodes',
  agents_online: 'Agents Online', total_traffic: 'Total Traffic', subscription: 'Subscription',
  copy: 'Copy', formats_hint: 'Formats: ?format=clash | base64 | surge',
  today_traffic: 'Today Traffic Trend', copied: 'Copied!',
  traffic_stats: 'Traffic Statistics', hourly_traffic: 'Hourly Traffic', load: 'Load',
  add_user: 'Add User', edit_user: 'Edit User', role: 'Role', traffic_used: 'Traffic Used',
  limit: 'Limit', speed: 'Speed', status: 'Status', actions: 'Actions',
  active: 'Active', disabled: 'Disabled', edit: 'Edit', reset: 'Reset', del: 'Del',
  confirm_delete_user: 'Delete this user?', user_deleted: 'User deleted',
  confirm_reset_traffic: 'Reset traffic for this user?', traffic_reset: 'Traffic reset',
  assign_nodes: 'Assign Nodes', nodes_assigned: 'Nodes assigned',
  traffic_limit_gb: 'Traffic Limit (GB, 0=unlimited)', speed_limit_mbps: 'Speed Limit (Mbps, 0=unlimited)',
  expire_date: 'Expire Date', reset_day: 'Traffic Reset Day (0=no auto reset)',
  save: 'Save', cancel: 'Cancel', user_created: 'User created', user_updated: 'User updated',
  admin: 'Admin', user: 'User',
  add_node: 'Add Node', edit_node: 'Edit Node', name: 'Name', address: 'Address',
  protocol: 'Protocol', transport: 'Transport', enabled: 'Enabled',
  confirm_delete_node: 'Delete this node?', node_deleted: 'Node deleted',
  node_toggled: 'Node toggled', enable: 'Enable', disable: 'Disable',
  port: 'Port', uuid_password: 'UUID / Password', alter_id: 'Alter ID',
  tls: 'TLS', tls_sni: 'TLS SNI', extra_config: 'Extra Config (JSON)', sort_order: 'Sort Order',
  node_created: 'Node created', node_updated: 'Node updated',
  add_agent: 'Add Agent', edit_agent: 'Edit Agent', install_script: 'Install Script',
  host: 'Host', cpu: 'CPU', memory: 'Memory', net_in_out: 'Net In/Out',
  uptime: 'Uptime', last_hb: 'Last HB',
  confirm_delete_agent: 'Delete this agent?', agent_deleted: 'Agent deleted',
  agent_created: 'Agent created', agent_updated: 'Agent updated',
  install_cmd_hint: 'One-click install command (run on target server):',
  uninstall_cmd_hint: 'Uninstall command:',
  close: 'Close',
  add_template: 'Add Template', edit_template: 'Edit Template', format: 'Format',
  is_default: 'Default', content: 'Content', yes: 'Yes', no: 'No',
  confirm_delete_tpl: 'Delete this template?', tpl_deleted: 'Template deleted',
  tpl_created: 'Template created', tpl_updated: 'Template updated',
  tpl_vars: 'Template Variables',
  general: 'General', site_name: 'Site Name',
  panel_host: 'Panel Host (for Agent script, e.g. your-domain.com:3001)',
  allow_register: 'Allow Registration', comm_key: 'Communication Key',
  comm_key_desc: 'All agents use this key for AES-256-GCM encrypted communication. Changing it requires re-deploying all agents.',
  data_management: 'Data Management', export_data: 'Export Data', import_data: 'Import Data',
  import_confirm: 'This will overwrite all existing data. Continue?',
  export_success: 'Data exported', import_success: 'Data imported successfully',
  import_failed: 'Import failed: ', settings_saved: 'Settings saved',
  save_settings: 'Save Settings',
  retention: 'Retention: 30 days', time: 'Time', level: 'Level', module: 'Module', message: 'Message',
  prev: 'Prev', next: 'Next',
  light_mode: 'Light', dark_mode: 'Dark',
  no_data: 'No data', error: 'Error', page_not_found: 'Page not found',
  online: 'Online', offline: 'Offline',
}
};

// ══════════════════════════════════════
// ── State
// ══════════════════════════════════════
const state = {
  token: localStorage.getItem('token') || '',
  role: localStorage.getItem('role') || '',
  username: localStorage.getItem('username') || '',
  page: 'dashboard',
  lang: localStorage.getItem('lang') || 'zh',
  theme: localStorage.getItem('theme') || 'light',
};

function t(key) { return (i18n[state.lang] || i18n.zh)[key] || key; }

// ── Theme ──
function applyTheme() {
  document.documentElement.setAttribute('data-theme', state.theme);
  localStorage.setItem('theme', state.theme);
}
function toggleTheme() {
  state.theme = state.theme === 'light' ? 'dark' : 'light';
  applyTheme();
  render();
}
function toggleLang() {
  state.lang = state.lang === 'zh' ? 'en' : 'zh';
  localStorage.setItem('lang', state.lang);
  render();
}
applyTheme();

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

// ── Icons ──
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
  sun: '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>',
  moon: '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>',
};

// ── Auth ──
function setAuth(tk, role, uname) {
  state.token = tk; state.role = role; state.username = uname;
  localStorage.setItem('token', tk);
  localStorage.setItem('role', role);
  localStorage.setItem('username', uname);
}
function logout() {
  state.token = ''; state.role = ''; state.username = '';
  localStorage.removeItem('token'); localStorage.removeItem('role'); localStorage.removeItem('username');
  render();
}
function navigate(page) { state.page = page; render(); }

// ══════════════════════════════════════
// ── Render
// ══════════════════════════════════════
function render() {
  const app = document.getElementById('app');
  if (!state.token) { renderLogin(app); } else { renderApp(app); }
}

// ── Login ──
let loginMode = 'login';
let captchaId = '', captchaSvg = '', needCaptcha = false;

async function loadCaptcha() {
  try { const d = await api('GET', '/api/captcha'); captchaId = d.captcha_id; captchaSvg = d.captcha_svg; } catch(e) {}
}
async function checkNeedCaptcha() {
  try { const d = await api('GET', '/api/login/need-captcha'); needCaptcha = d.need; } catch(e) {}
}

function renderLogin(app) {
  const isReg = loginMode === 'register';
  app.innerHTML = `<div class="login-page"><div class="login-box">
    <h1>NebulaPanel</h1>
    <p class="subtitle">${isReg ? t('sign_up') : t('sign_in')}</p>
    <form id="authForm">
      <div class="form-group"><label>${t('username')}</label><input class="form-control" id="authUser" placeholder="${t('username')}" required minlength="3"></div>
      <div class="form-group"><label>${t('password')}</label><input class="form-control" id="authPass" type="password" placeholder="${t('password')}" required minlength="6"></div>
      <div id="captchaArea" style="display:${isReg || needCaptcha ? 'block' : 'none'}">
        <div class="captcha-row">
          <div class="form-group"><label>${t('captcha')}</label><input class="form-control" id="authCaptcha" placeholder="${t('captcha')}"></div>
          <div id="captchaImg" style="cursor:pointer;margin-bottom:16px" title="${t('captcha_refresh')}"></div>
        </div>
      </div>
      <button class="btn btn-primary" style="width:100%;justify-content:center;padding:12px" type="submit">${isReg ? t('register') : t('login')}</button>
    </form>
    <div class="switch-link">${isReg
      ? t('have_account') + ' <a href="#" id="switchAuth">' + t('login') + '</a>'
      : t('no_account') + ' <a href="#" id="switchAuth">' + t('register') + '</a>'}</div>
    <div class="login-footer">
      <button class="theme-toggle-btn" id="loginThemeBtn">${state.theme==='light' ? icons.moon + ' ' + t('dark_mode') : icons.sun + ' ' + t('light_mode')}</button>
      <button class="lang-toggle-btn" id="loginLangBtn">${state.lang==='zh' ? 'EN' : '中文'}</button>
    </div>
  </div></div>`;

  if (isReg || needCaptcha) loadCaptchaUI();
  document.getElementById('switchAuth').onclick = async (e) => {
    e.preventDefault(); loginMode = loginMode === 'login' ? 'register' : 'login';
    if (loginMode === 'register') await loadCaptcha(); render();
  };
  document.getElementById('loginThemeBtn').onclick = toggleTheme;
  document.getElementById('loginLangBtn').onclick = toggleLang;

  document.getElementById('authForm').onsubmit = async (e) => {
    e.preventDefault();
    const user = document.getElementById('authUser').value.trim();
    const pass = document.getElementById('authPass').value;
    const cap = document.getElementById('authCaptcha')?.value?.trim() || '';
    try {
      if (isReg) {
        await api('POST', '/api/register', {username:user, password:pass, captcha_id:captchaId, captcha:cap});
        toast(t('reg_success'), 'success'); loginMode = 'login'; render();
      } else {
        const d = await api('POST', '/api/login', {username:user, password:pass, captcha_id:captchaId, captcha:cap});
        setAuth(d.token, d.role, d.username);
        toast(t('welcome_back') + d.username, 'success'); state.page = 'dashboard'; render();
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
  if (el) { el.innerHTML = captchaSvg; el.onclick = async () => { await loadCaptcha(); el.innerHTML = captchaSvg; }; }
}

// ── Main App ──
function renderApp(app) {
  const isAdmin = state.role === 'admin';
  const navItems = [
    {id:'dashboard', label:t('dashboard'), icon:icons.dashboard},
    {id:'traffic', label:t('traffic'), icon:icons.traffic},
  ];
  if (isAdmin) {
    navItems.push(
      {id:'users', label:t('users'), icon:icons.users},
      {id:'nodes', label:t('nodes'), icon:icons.nodes},
      {id:'agents', label:t('agents'), icon:icons.agents},
      {id:'templates', label:t('templates'), icon:icons.templates},
      {id:'settings', label:t('settings'), icon:icons.settings},
      {id:'logs', label:t('logs'), icon:icons.logs},
    );
  }
  app.innerHTML = `<div class="app">
    <aside class="sidebar">
      <div class="sidebar-brand"><h1>NebulaPanel</h1><small>Proxy Management</small></div>
      <nav class="sidebar-nav">${navItems.map(n =>
        `<a href="#" data-page="${n.id}" class="${state.page===n.id?'active':''}">${n.icon}<span>${n.label}</span></a>`
      ).join('')}</nav>
      <div class="sidebar-footer">
        <div style="margin-bottom:8px">${escHtml(state.username)} <span class="badge badge-${isAdmin?'primary':'info'}">${isAdmin?t('admin'):t('user')}</span></div>
        <div class="sidebar-footer-actions">
          <button class="theme-toggle-btn" id="themeBtn">${state.theme==='light' ? icons.moon + ' ' + t('dark_mode') : icons.sun + ' ' + t('light_mode')}</button>
          <button class="lang-toggle-btn" id="langBtn">${state.lang==='zh' ? 'EN' : '中文'}</button>
        </div>
        <a href="#" id="logoutBtn" style="color:var(--danger);display:flex;align-items:center;gap:6px;margin-top:10px">${icons.logout}<span>${t('logout')}</span></a>
      </div>
    </aside>
    <main class="main" id="mainContent"><div class="loading"><div class="spinner"></div></div></main>
  </div>`;

  document.querySelectorAll('.sidebar-nav a').forEach(a => {
    a.onclick = (e) => { e.preventDefault(); navigate(a.dataset.page); };
  });
  document.getElementById('logoutBtn').onclick = (e) => { e.preventDefault(); logout(); };
  document.getElementById('themeBtn').onclick = toggleTheme;
  document.getElementById('langBtn').onclick = toggleLang;
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
      default: main.innerHTML = '<div class="empty">' + t('page_not_found') + '</div>';
    }
  } catch(err) { main.innerHTML = `<div class="empty">${t('error')}: ${escHtml(err.message)}</div>`; }
}

// ══════════════════════════════════════
// ── Dashboard
// ══════════════════════════════════════
async function renderDashboard(el) {
  const d = await api('GET', '/api/dashboard');
  const u = d.user;
  const isAdmin = state.role === 'admin';
  let html = `<div class="topbar"><h2>${t('dashboard')}</h2><div class="topbar-actions"><span style="color:var(--text-dim);font-size:13px">${new Date().toLocaleDateString()}</span></div></div>`;
  html += `<div class="stats-grid">
    <div class="stat-card info"><div class="label">${t('upload')}</div><div class="value">${formatBytes(u.traffic_up)}</div></div>
    <div class="stat-card primary"><div class="label">${t('download')}</div><div class="value">${formatBytes(u.traffic_down)}</div></div>
    <div class="stat-card ${u.traffic_limit > 0 && u.traffic_used >= u.traffic_limit ? 'danger' : 'success'}">
      <div class="label">${t('traffic_used_limit')}</div><div class="value">${formatBytes(u.traffic_used)}</div>
      <div class="sub">${u.traffic_limit > 0 ? t('limit') + ': ' + formatBytes(u.traffic_limit) : t('unlimited')}</div></div>
    <div class="stat-card ${u.expire_at ? 'warning' : 'success'}">
      <div class="label">${t('expire')}</div><div class="value" style="font-size:18px">${u.expire_at || t('never')}</div>
      <div class="sub">${u.speed_limit > 0 ? t('speed_limit') + ': ' + u.speed_limit + ' Mbps' : t('no_speed_limit')}</div></div>
  </div>`;
  if (isAdmin && d.admin) {
    const a = d.admin;
    html += `<div class="stats-grid">
      <div class="stat-card primary"><div class="label">${t('total_users')}</div><div class="value">${a.total_users}</div></div>
      <div class="stat-card info"><div class="label">${t('total_nodes')}</div><div class="value">${a.total_nodes}</div></div>
      <div class="stat-card success"><div class="label">${t('agents_online')}</div><div class="value">${a.online_agents} / ${a.total_agents}</div></div>
      <div class="stat-card warning"><div class="label">${t('total_traffic')}</div><div class="value">${formatBytes(a.total_traffic_up + a.total_traffic_down)}</div></div>
    </div>`;
  }
  const me = await api('GET', '/api/me');
  html += `<div class="card"><div class="card-header"><h3>${t('subscription')}</h3></div>
    <div class="copy-wrap"><input class="form-control" readonly value="${location.origin}/api/sub/${escHtml(me.sub_token)}?format=clash" id="subUrl"><button class="btn btn-primary copy-btn" onclick="copyText('subUrl')">${t('copy')}</button></div>
    <div style="margin-top:8px;font-size:12px;color:var(--text-dim)">${t('formats_hint')}</div></div>`;
  html += `<div class="card"><div class="card-header"><h3>${t('today_traffic')}</h3></div><div class="chart-container"><canvas id="dashChart"></canvas></div></div>`;
  el.innerHTML = html;
  if (d.today_chart) renderHourlyChart('dashChart', d.today_chart);
}

// ── Traffic ──
async function renderTraffic(el) {
  const stats = await api('GET', '/api/traffic/stats');
  const today = new Date().toISOString().split('T')[0];
  let html = `<div class="topbar"><h2>${t('traffic_stats')}</h2></div>`;
  html += `<div class="stats-grid">
    <div class="stat-card info"><div class="label">${t('upload')}</div><div class="value">${formatBytes(stats.traffic_up)}</div></div>
    <div class="stat-card primary"><div class="label">${t('download')}</div><div class="value">${formatBytes(stats.traffic_down)}</div></div>
    <div class="stat-card success"><div class="label">${t('traffic_used_limit')}</div><div class="value">${formatBytes(stats.traffic_used)}</div>
      <div class="sub">${stats.traffic_limit > 0 ? t('limit') + ': ' + formatBytes(stats.traffic_limit) : t('unlimited')}</div></div>
    <div class="stat-card warning"><div class="label">${t('speed_limit')}</div><div class="value">${stats.speed_limit > 0 ? stats.speed_limit + ' Mbps' : t('no_speed_limit')}</div></div>
  </div>`;
  html += `<div class="card"><div class="card-header"><h3>${t('hourly_traffic')}</h3>
    <div class="chart-toolbar"><input type="date" id="chartDate" value="${today}"><button class="btn btn-sm btn-outline" id="loadChartBtn">${t('load')}</button></div>
  </div><div class="chart-container"><canvas id="trafficChart"></canvas></div></div>`;
  el.innerHTML = html;
  async function loadChart(date) { const cd = await api('GET', '/api/traffic/chart?date=' + date); renderHourlyChart('trafficChart', cd.chart); }
  await loadChart(today);
  document.getElementById('loadChartBtn').onclick = () => { const d = document.getElementById('chartDate').value; if (d) loadChart(d); };
}

function renderHourlyChart(canvasId, data) {
  const canvas = document.getElementById(canvasId);
  if (!canvas) return;
  if (canvas._chart) canvas._chart.destroy();
  const isDark = state.theme === 'dark';
  const gridColor = isDark ? 'rgba(42,45,58,0.5)' : 'rgba(0,0,0,0.06)';
  const tickColor = isDark ? '#8b8fa3' : '#6b7280';
  canvas._chart = new Chart(canvas.getContext('2d'), {
    type: 'bar',
    data: {
      labels: data.map(d => d.hour),
      datasets: [
        { label: t('upload'), data: data.map(d => d.up), backgroundColor: 'rgba(99,102,241,0.7)', borderRadius: 4 },
        { label: t('download'), data: data.map(d => d.down), backgroundColor: 'rgba(59,130,246,0.7)', borderRadius: 4 },
      ]
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        tooltip: {
          callbacks: {
            title: (items) => data[items[0].dataIndex].time,
            label: (ctx) => ctx.dataset.label + ': ' + formatBytes(ctx.raw),
            footer: (items) => { const i = items[0].dataIndex; return 'Total: ' + formatBytes(data[i].total); }
          }
        },
        legend: { labels: { color: tickColor } }
      },
      scales: {
        x: { ticks: { color: tickColor }, grid: { color: gridColor } },
        y: { ticks: { color: tickColor, callback: v => formatBytes(v) }, grid: { color: gridColor } }
      }
    }
  });
}

// ── Users ──
async function renderUsers(el) {
  const users = await api('GET', '/api/users');
  let html = `<div class="topbar"><h2>${t('users')}</h2><button class="btn btn-primary" id="addUserBtn">+ ${t('add_user')}</button></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>ID</th><th>${t('username')}</th><th>${t('role')}</th><th>${t('traffic_used')}</th><th>${t('limit')}</th><th>${t('speed')}</th><th>${t('expire')}</th><th>${t('status')}</th><th>${t('actions')}</th></tr></thead>
    <tbody>${users.map(u => `<tr>
      <td>${u.id}</td><td>${escHtml(u.username)}</td>
      <td><span class="badge badge-${u.role==='admin'?'primary':'info'}">${u.role==='admin'?t('admin'):t('user')}</span></td>
      <td>${formatBytes(u.traffic_used)}</td>
      <td>${u.traffic_limit > 0 ? formatBytes(u.traffic_limit) : t('unlimited')}</td>
      <td>${u.speed_limit > 0 ? u.speed_limit + ' Mbps' : '-'}</td>
      <td>${u.expire_at || t('never')}</td>
      <td><span class="badge badge-${u.enabled?'success':'danger'}">${u.enabled?t('active'):t('disabled')}</span></td>
      <td><div class="btn-group">
        <button class="btn btn-sm btn-outline" onclick="editUser(${u.id})">${t('edit')}</button>
        <button class="btn btn-sm btn-outline" onclick="resetTraffic(${u.id})">${t('reset')}</button>
        <button class="btn btn-sm btn-outline" onclick="assignNodes(${u.id})">${t('nodes')}</button>
        ${u.role!=='admin'?`<button class="btn btn-sm btn-danger" onclick="deleteUser(${u.id})">${t('del')}</button>`:''}
      </div></td></tr>`).join('')}${users.length===0?`<tr><td colspan="9" class="empty">${t('no_data')}</td></tr>`:''}</tbody></table></div></div>`;
  el.innerHTML = html;
  document.getElementById('addUserBtn').onclick = () => showUserModal();
}

window.editUser = async (id) => { const users = await api('GET', '/api/users'); const u = users.find(x => x.id === id); if (u) showUserModal(u); };
window.deleteUser = async (id) => { if (!confirm(t('confirm_delete_user'))) return; await api('DELETE', '/api/users/' + id); toast(t('user_deleted'), 'success'); loadPage('users'); };
window.resetTraffic = async (id) => { if (!confirm(t('confirm_reset_traffic'))) return; await api('POST', '/api/users/' + id + '/reset-traffic'); toast(t('traffic_reset'), 'success'); loadPage('users'); };
window.assignNodes = async (uid) => {
  const [nodes, assigned] = await Promise.all([api('GET', '/api/nodes'), api('GET', '/api/users/' + uid + '/nodes')]);
  const set = new Set(assigned);
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal">
    <div class="modal-header"><h3>${t('assign_nodes')}</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body">${nodes.map(n => `<label style="display:flex;align-items:center;gap:8px;padding:6px 0;cursor:pointer">
      <input type="checkbox" value="${n.id}" ${set.has(n.id)?'checked':''} class="nodeCheck"> ${escHtml(n.name)} (${n.address}:${n.port})
    </label>`).join('')}${nodes.length===0?`<div class="empty">${t('no_data')}</div>`:''}</div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">${t('cancel')}</button><button class="btn btn-primary" id="saveNodesBtn">${t('save')}</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);
  document.getElementById('saveNodesBtn').onclick = async () => {
    const ids = [...document.querySelectorAll('.nodeCheck:checked')].map(c => parseInt(c.value));
    await api('PUT', '/api/users/' + uid + '/nodes', {node_ids: ids});
    toast(t('nodes_assigned'), 'success'); closeModal();
  };
};

function showUserModal(user) {
  const isEdit = !!user;
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal">
    <div class="modal-header"><h3>${isEdit ? t('edit_user') : t('add_user')}</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body"><form id="userForm">
      <div class="form-group"><label>${t('username')}</label><input class="form-control" id="fUsername" value="${isEdit?escHtml(user.username):''}" ${isEdit?'':'required'} minlength="3"></div>
      <div class="form-group"><label>${t('password')} ${isEdit?t('leave_empty_keep'):''}</label><input class="form-control" id="fPassword" type="password" ${isEdit?'':'required'} minlength="6"></div>
      <div class="form-row">
        <div class="form-group"><label>${t('role')}</label><select class="form-control" id="fRole"><option value="user" ${isEdit&&user.role==='user'?'selected':''}>${t('user')}</option><option value="admin" ${isEdit&&user.role==='admin'?'selected':''}>${t('admin')}</option></select></div>
        <div class="form-group"><label>${t('status')}</label><select class="form-control" id="fEnabled"><option value="1" ${!isEdit||user.enabled?'selected':''}>${t('active')}</option><option value="0" ${isEdit&&!user.enabled?'selected':''}>${t('disabled')}</option></select></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>${t('traffic_limit_gb')}</label><input class="form-control" id="fTrafficLimit" type="number" value="${isEdit ? Math.round((user.traffic_limit||0)/1073741824) : 0}" min="0"></div>
        <div class="form-group"><label>${t('speed_limit_mbps')}</label><input class="form-control" id="fSpeedLimit" type="number" value="${isEdit?user.speed_limit||0:0}" min="0"></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>${t('expire_date')}</label><input class="form-control" id="fExpire" type="datetime-local" value="${isEdit&&user.expire_at?user.expire_at.replace(' ','T'):''}"></div>
        <div class="form-group"><label>${t('reset_day')}</label><input class="form-control" id="fResetDay" type="number" value="${isEdit?user.reset_day||0:0}" min="0" max="31"></div>
      </div>
    </form></div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">${t('cancel')}</button><button class="btn btn-primary" id="saveUserBtn">${t('save')}</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);
  document.getElementById('saveUserBtn').onclick = async () => {
    const data = { username: document.getElementById('fUsername').value.trim(), role: document.getElementById('fRole').value, enabled: parseInt(document.getElementById('fEnabled').value), traffic_limit: parseInt(document.getElementById('fTrafficLimit').value) * 1073741824, speed_limit: parseInt(document.getElementById('fSpeedLimit').value), expire_at: document.getElementById('fExpire').value ? document.getElementById('fExpire').value.replace('T',' ') + ':00' : '', reset_day: parseInt(document.getElementById('fResetDay').value) };
    const pw = document.getElementById('fPassword').value;
    if (pw) data.password = pw;
    try {
      if (isEdit) await api('PUT', '/api/users/' + user.id, data); else { data.password = pw; await api('POST', '/api/users', data); }
      toast(isEdit ? t('user_updated') : t('user_created'), 'success'); closeModal(); loadPage('users');
    } catch(e) { toast(e.message, 'error'); }
  };
}

// ── Nodes ──
async function renderNodes(el) {
  const nodes = await api('GET', '/api/nodes');
  let html = `<div class="topbar"><h2>${t('nodes')}</h2><button class="btn btn-primary" id="addNodeBtn">+ ${t('add_node')}</button></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>ID</th><th>${t('name')}</th><th>${t('address')}</th><th>${t('protocol')}</th><th>${t('transport')}</th><th>${t('tls')}</th><th>${t('status')}</th><th>${t('actions')}</th></tr></thead>
    <tbody>${nodes.map(n => `<tr>
      <td>${n.id}</td><td>${escHtml(n.name)}</td><td>${escHtml(n.address)}:${n.port}</td>
      <td><span class="badge badge-info">${n.protocol}</span></td><td>${n.transport}</td><td>${n.tls?t('yes'):t('no')}</td>
      <td><span class="badge badge-${n.enabled?'success':'danger'}">${n.enabled?t('enabled'):t('disabled')}</span></td>
      <td><div class="btn-group">
        <button class="btn btn-sm btn-outline" onclick="toggleNode(${n.id})">${n.enabled?t('disable'):t('enable')}</button>
        <button class="btn btn-sm btn-outline" onclick="editNode(${n.id})">${t('edit')}</button>
        <button class="btn btn-sm btn-danger" onclick="deleteNode(${n.id})">${t('del')}</button>
      </div></td></tr>`).join('')}${nodes.length===0?`<tr><td colspan="8" class="empty">${t('no_data')}</td></tr>`:''}</tbody></table></div></div>`;
  el.innerHTML = html;
  document.getElementById('addNodeBtn').onclick = () => showNodeModal();
}
window.toggleNode = async (id) => { await api('PUT', '/api/nodes/' + id + '/toggle'); toast(t('node_toggled'), 'success'); loadPage('nodes'); };
window.editNode = async (id) => { const nodes = await api('GET', '/api/nodes'); const n = nodes.find(x => x.id === id); if (n) showNodeModal(n); };
window.deleteNode = async (id) => { if (!confirm(t('confirm_delete_node'))) return; await api('DELETE', '/api/nodes/' + id); toast(t('node_deleted'), 'success'); loadPage('nodes'); };

function showNodeModal(node) {
  const isEdit = !!node;
  const protocols = ['vmess','vless','trojan','ss','hysteria2'];
  const transports = ['tcp','ws','grpc','h2','quic'];
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal">
    <div class="modal-header"><h3>${isEdit?t('edit_node'):t('add_node')}</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body"><form id="nodeForm">
      <div class="form-group"><label>${t('name')}</label><input class="form-control" id="nName" value="${isEdit?escHtml(node.name):''}" required></div>
      <div class="form-row">
        <div class="form-group"><label>${t('address')}</label><input class="form-control" id="nAddr" value="${isEdit?escHtml(node.address):''}" required></div>
        <div class="form-group"><label>${t('port')}</label><input class="form-control" id="nPort" type="number" value="${isEdit?node.port:443}" required></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>${t('protocol')}</label><select class="form-control" id="nProto">${protocols.map(p=>`<option value="${p}" ${isEdit&&node.protocol===p?'selected':''}>${p}</option>`).join('')}</select></div>
        <div class="form-group"><label>${t('transport')}</label><select class="form-control" id="nTrans">${transports.map(tp=>`<option value="${tp}" ${isEdit&&node.transport===tp?'selected':''}>${tp}</option>`).join('')}</select></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>${t('uuid_password')}</label><input class="form-control" id="nUUID" value="${isEdit?escHtml(node.uuid):''}"></div>
        <div class="form-group"><label>${t('alter_id')}</label><input class="form-control" id="nAltID" type="number" value="${isEdit?node.alter_id:0}"></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label>${t('tls')}</label><select class="form-control" id="nTLS"><option value="0" ${isEdit&&!node.tls?'selected':''}>Off</option><option value="1" ${isEdit&&node.tls?'selected':''}>On</option></select></div>
        <div class="form-group"><label>${t('tls_sni')}</label><input class="form-control" id="nSNI" value="${isEdit?escHtml(node.tls_sni):''}"></div>
      </div>
      <div class="form-group"><label>${t('extra_config')}</label><textarea class="form-control" id="nExtra">${isEdit?escHtml(node.extra_config):''}</textarea></div>
      <div class="form-group"><label>${t('sort_order')}</label><input class="form-control" id="nSort" type="number" value="${isEdit?node.sort_order:0}"></div>
    </form></div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">${t('cancel')}</button><button class="btn btn-primary" id="saveNodeBtn">${t('save')}</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);
  document.getElementById('saveNodeBtn').onclick = async () => {
    const data = { name: document.getElementById('nName').value.trim(), address: document.getElementById('nAddr').value.trim(), port: parseInt(document.getElementById('nPort').value), protocol: document.getElementById('nProto').value, transport: document.getElementById('nTrans').value, uuid: document.getElementById('nUUID').value.trim(), alter_id: parseInt(document.getElementById('nAltID').value), tls: parseInt(document.getElementById('nTLS').value), tls_sni: document.getElementById('nSNI').value.trim(), extra_config: document.getElementById('nExtra').value.trim(), sort_order: parseInt(document.getElementById('nSort').value) };
    try {
      if (isEdit) await api('PUT', '/api/nodes/' + node.id, data); else await api('POST', '/api/nodes', data);
      toast(isEdit?t('node_updated'):t('node_created'), 'success'); closeModal(); loadPage('nodes');
    } catch(e) { toast(e.message, 'error'); }
  };
}

// ── Agents ──
async function renderAgents(el) {
  const agents = await api('GET', '/api/agents');
  let html = `<div class="topbar"><h2>${t('agents')}</h2><div class="topbar-actions">
    <button class="btn btn-primary" id="showScriptBtn">${t('install_script')}</button>
  </div></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>ID</th><th>${t('name')}</th><th>${t('host')}</th><th>${t('status')}</th><th>${t('cpu')}</th><th>${t('memory')}</th><th>${t('net_in_out')}</th><th>${t('uptime')}</th><th>${t('last_hb')}</th><th>${t('actions')}</th></tr></thead>
    <tbody>${agents.map(a => `<tr>
      <td>${a.id}</td><td>${escHtml(a.name)}</td><td>${escHtml(a.host)}:${a.port}</td>
      <td><span class="badge badge-${a.status==='online'?'success':'danger'}">${a.status==='online'?t('online'):t('offline')}</span></td>
      <td>${a.cpu_usage.toFixed(1)}%</td><td>${a.mem_usage.toFixed(1)}%</td>
      <td>${formatBytes(a.net_in)} / ${formatBytes(a.net_out)}</td>
      <td>${a.uptime > 0 ? Math.floor(a.uptime/3600)+'h' : '-'}</td>
      <td style="font-size:12px">${a.last_heartbeat||'-'}</td>
      <td><div class="btn-group">
        <button class="btn btn-sm btn-danger" onclick="deleteAgent(${a.id})">${t('del')}</button>
      </div></td></tr>`).join('')}${agents.length===0?`<tr><td colspan="10" class="empty">${t('no_data')}</td></tr>`:''}</tbody></table></div></div>`;
  el.innerHTML = html;
  document.getElementById('showScriptBtn').onclick = async () => {
    // 获取真实的通信密钥
    let commKey = '';
    try {
      const settings = await api('GET', '/api/settings');
      commKey = settings.comm_key || '';
    } catch(e) { commKey = 'YOUR_COMM_KEY'; }
    const ghUrl = 'https://raw.githubusercontent.com/poouo/NebulaPanel/main/agent/install.sh';
    const panelUrl = location.origin;
    const installCmd = `bash <(curl -sL --connect-timeout 15 ${ghUrl} || curl -sL ${panelUrl}/static/agent/install.sh) install ${panelUrl} ${commKey}`;
    const uninstallCmd = `bash <(curl -sL --connect-timeout 15 ${ghUrl} || curl -sL ${panelUrl}/static/agent/install.sh) uninstall`;
    let mhtml = `<div class="modal-overlay" id="modalOverlay"><div class="modal" style="max-width:750px">
      <div class="modal-header"><h3>${t('install_script')}</h3><button class="btn-icon" id="scriptCloseBtn">&times;</button></div>
      <div class="modal-body">
        <p style="margin-bottom:12px;color:var(--text-dim);font-size:13px">${t('install_cmd_hint')}</p>
        <div class="copy-wrap" style="margin-bottom:16px">
          <input class="form-control" readonly id="installCmd" style="font-family:monospace;font-size:12px">
          <button class="btn btn-primary copy-btn" id="copyInstallBtn">${t('copy')}</button>
        </div>
        <p style="margin-bottom:8px;color:var(--text-dim);font-size:13px">${t('uninstall_cmd_hint')}</p>
        <div class="copy-wrap">
          <input class="form-control" readonly id="uninstallCmd" style="font-family:monospace;font-size:12px">
          <button class="btn btn-primary copy-btn" id="copyUninstallBtn">${t('copy')}</button>
        </div>
      </div>
      <div class="modal-footer"><button class="btn btn-outline" id="scriptCloseBtn2">${t('close')}</button></div>
    </div></div>`;
    document.body.insertAdjacentHTML('beforeend', mhtml);
    document.getElementById('installCmd').value = installCmd;
    document.getElementById('uninstallCmd').value = uninstallCmd;
    document.getElementById('copyInstallBtn').onclick = () => doCopy('installCmd');
    document.getElementById('copyUninstallBtn').onclick = () => doCopy('uninstallCmd');
    document.getElementById('scriptCloseBtn').onclick = closeModal;
    document.getElementById('scriptCloseBtn2').onclick = closeModal;
  };
}
window.deleteAgent = async (id) => { if (!confirm(t('confirm_delete_agent'))) return; await api('DELETE', '/api/agents/' + id); toast(t('agent_deleted'), 'success'); loadPage('agents'); };

// ── Templates ──
async function renderTemplates(el) {
  const tpls = await api('GET', '/api/templates');
  let html = `<div class="topbar"><h2>${t('templates')}</h2><button class="btn btn-primary" id="addTplBtn">+ ${t('add_template')}</button></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>ID</th><th>${t('name')}</th><th>${t('format')}</th><th>${t('is_default')}</th><th>${t('actions')}</th></tr></thead>
    <tbody>${tpls.map(tp => `<tr>
      <td>${tp.id}</td><td>${escHtml(tp.name)}</td>
      <td><span class="badge badge-info">${tp.format}</span></td>
      <td>${tp.is_default?`<span class="badge badge-success">${t('yes')}</span>`:'-'}</td>
      <td><div class="btn-group">
        <button class="btn btn-sm btn-outline" onclick="editTemplate(${tp.id})">${t('edit')}</button>
        <button class="btn btn-sm btn-danger" onclick="deleteTemplate(${tp.id})">${t('del')}</button>
      </div></td></tr>`).join('')}${tpls.length===0?`<tr><td colspan="5" class="empty">${t('no_data')}</td></tr>`:''}</tbody></table></div></div>`;
  html += `<div class="card"><div class="card-header"><h3>${t('tpl_vars')}</h3></div>
    <div style="font-size:13px;color:var(--text-dim)"><code>{{PROXIES}}</code> - Proxy list &nbsp; <code>{{PROXY_NAMES}}</code> - Proxy name list</div></div>`;
  el.innerHTML = html;
  document.getElementById('addTplBtn').onclick = () => showTemplateModal();
}
window.editTemplate = async (id) => { const tpls = await api('GET', '/api/templates'); const tp = tpls.find(x => x.id === id); if (tp) showTemplateModal(tp); };
window.deleteTemplate = async (id) => { if (!confirm(t('confirm_delete_tpl'))) return; await api('DELETE', '/api/templates/' + id); toast(t('tpl_deleted'), 'success'); loadPage('templates'); };

function showTemplateModal(tpl) {
  const isEdit = !!tpl;
  let html = `<div class="modal-overlay" id="modalOverlay"><div class="modal" style="max-width:700px">
    <div class="modal-header"><h3>${isEdit?t('edit_template'):t('add_template')}</h3><button class="btn-icon" onclick="closeModal()">&times;</button></div>
    <div class="modal-body">
      <div class="form-row">
        <div class="form-group"><label>${t('name')}</label><input class="form-control" id="tName" value="${isEdit?escHtml(tpl.name):''}" required></div>
        <div class="form-group"><label>${t('format')}</label><select class="form-control" id="tFormat">
          <option value="clash" ${isEdit&&tpl.format==='clash'?'selected':''}>Clash/Mihomo</option>
          <option value="surge" ${isEdit&&tpl.format==='surge'?'selected':''}>Surge</option>
          <option value="base64" ${isEdit&&tpl.format==='base64'?'selected':''}>Base64</option>
        </select></div>
      </div>
      <div class="form-group"><label>${t('is_default')}</label><select class="form-control" id="tDefault"><option value="0" ${isEdit&&!tpl.is_default?'selected':''}>${t('no')}</option><option value="1" ${isEdit&&tpl.is_default?'selected':''}>${t('yes')}</option></select></div>
      <div class="form-group"><label>${t('content')}</label><textarea class="form-control" id="tContent" style="min-height:200px;font-size:12px">${isEdit?escHtml(tpl.content):''}</textarea></div>
    </div>
    <div class="modal-footer"><button class="btn btn-outline" onclick="closeModal()">${t('cancel')}</button><button class="btn btn-primary" id="saveTplBtn">${t('save')}</button></div>
  </div></div>`;
  document.body.insertAdjacentHTML('beforeend', html);
  document.getElementById('saveTplBtn').onclick = async () => {
    const data = { name: document.getElementById('tName').value.trim(), format: document.getElementById('tFormat').value, is_default: parseInt(document.getElementById('tDefault').value), content: document.getElementById('tContent').value };
    try {
      if (isEdit) await api('PUT', '/api/templates/' + tpl.id, data); else await api('POST', '/api/templates', data);
      toast(isEdit?t('tpl_updated'):t('tpl_created'), 'success'); closeModal(); loadPage('templates');
    } catch(e) { toast(e.message, 'error'); }
  };
}

// ── Settings ──
async function renderSettings(el) {
  const settings = await api('GET', '/api/settings');
  let html = `<div class="topbar"><h2>${t('settings')}</h2></div>`;
  html += `<div class="card"><div class="card-header"><h3>${t('general')}</h3></div>
    <div class="form-group"><label>${t('site_name')}</label><input class="form-control" id="sSiteName" value="${escHtml(settings.site_name||'NebulaPanel')}"></div>
    <div class="form-group"><label>${t('panel_host')}</label><input class="form-control" id="sPanelHost" value="${escHtml(settings.panel_host||'')}"></div>
    <div class="form-group"><label>${t('allow_register')}</label><select class="form-control" id="sAllowReg"><option value="true" ${settings.allow_register==='true'?'selected':''}>${t('yes')}</option><option value="false" ${settings.allow_register!=='true'?'selected':''}>${t('no')}</option></select></div>
  </div>`;
  html += `<div class="card"><div class="card-header"><h3>${t('comm_key')}</h3></div>
    <div class="form-group">
      <div class="copy-wrap"><input class="form-control" id="sCommKey" value="${escHtml(settings.comm_key||'')}" style="font-family:monospace;font-size:12px"><button class="btn btn-primary copy-btn" onclick="copyText('sCommKey')">${t('copy')}</button></div>
    </div>
    <p style="font-size:12px;color:var(--text-dim);margin-top:4px">${t('comm_key_desc')}</p>
  </div>`;
  html += `<div class="card"><div class="card-header"><h3>${t('data_management')}</h3></div>
    <div class="btn-group">
      <button class="btn btn-primary" id="exportBtn">${t('export_data')}</button>
      <label class="btn btn-outline" style="cursor:pointer">${t('import_data')}<input type="file" accept=".json" id="importFile" style="display:none"></label>
    </div></div>`;
  html += `<div style="margin-top:16px"><button class="btn btn-primary" id="saveSettingsBtn">${t('save_settings')}</button></div>`;
  el.innerHTML = html;

  document.getElementById('saveSettingsBtn').onclick = async () => {
    const data = { site_name: document.getElementById('sSiteName').value.trim(), panel_host: document.getElementById('sPanelHost').value.trim(), allow_register: document.getElementById('sAllowReg').value, comm_key: document.getElementById('sCommKey').value.trim() };
    await api('PUT', '/api/settings', data); toast(t('settings_saved'), 'success');
  };
  document.getElementById('exportBtn').onclick = async () => {
    const res = await fetch('/api/export', { headers: {'Authorization': 'Bearer ' + state.token} });
    const blob = await res.blob(); const a = document.createElement('a');
    a.href = URL.createObjectURL(blob); a.download = 'nebula_backup_' + new Date().toISOString().slice(0,10) + '.json'; a.click();
    toast(t('export_success'), 'success');
  };
  document.getElementById('importFile').onchange = async (e) => {
    const file = e.target.files[0]; if (!file) return;
    if (!confirm(t('import_confirm'))) return;
    const text = await file.text();
    try { JSON.parse(text); await api('POST', '/api/import', JSON.parse(text)); toast(t('import_success'), 'success'); loadPage('settings'); }
    catch(err) { toast(t('import_failed') + err.message, 'error'); }
  };
}

// ── Logs ──
let logPage = 1;
async function renderLogs(el) {
  const d = await api('GET', `/api/logs?page=${logPage}&page_size=20`);
  let html = `<div class="topbar"><h2>${t('logs')}</h2><span style="color:var(--text-dim);font-size:13px">${t('retention')}</span></div>`;
  html += `<div class="card"><div class="table-wrap"><table>
    <thead><tr><th>${t('time')}</th><th>${t('level')}</th><th>${t('module')}</th><th>${t('message')}</th></tr></thead>
    <tbody>${d.logs.map(l => `<tr>
      <td style="font-size:12px;white-space:nowrap">${l.created_at}</td>
      <td><span class="badge badge-${l.level==='error'?'danger':l.level==='warn'?'warning':'info'}">${l.level}</span></td>
      <td>${escHtml(l.module)}</td>
      <td style="font-size:13px">${escHtml(l.message)}</td>
    </tr>`).join('')}${d.logs.length===0?`<tr><td colspan="4" class="empty">${t('no_data')}</td></tr>`:''}</tbody></table></div>
    <div class="pagination">
      <button ${logPage<=1?'disabled':''} onclick="logNav(${logPage-1})">${t('prev')}</button>
      <button class="active">${logPage}</button>
      <button ${d.logs.length<20?'disabled':''} onclick="logNav(${logPage+1})">${t('next')}</button>
    </div></div>`;
  el.innerHTML = html;
}
window.logNav = (p) => { logPage = p; loadPage('logs'); };

// ── Helpers ──
window.closeModal = () => { const m = document.getElementById('modalOverlay'); if (m) m.remove(); };
function doCopy(id) {
  const el = document.getElementById(id); if (!el) return;
  const text = el.value || el.textContent || '';
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text).then(() => toast(t('copied'), 'success')).catch(() => fallbackCopy(el));
  } else {
    fallbackCopy(el);
  }
}
function fallbackCopy(el) {
  const ta = document.createElement('textarea');
  ta.value = el.value || el.textContent || '';
  ta.style.cssText = 'position:fixed;left:-9999px';
  document.body.appendChild(ta);
  ta.select();
  try { document.execCommand('copy'); toast(t('copied'), 'success'); } catch(e) { toast('Copy failed', 'error'); }
  document.body.removeChild(ta);
}
window.copyText = doCopy;
window.doCopy = doCopy;

// ── Init ──
checkNeedCaptcha().then(() => render());

})();
