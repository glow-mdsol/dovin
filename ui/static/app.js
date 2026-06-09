// ui/static/app.js
let showArchive = false;
let expandedId = null;

const PRIORITY_LABEL = { 1: 'High', 2: 'Medium', 3: 'Low' };
const PRIORITY_CLASS = { 1: 'p1', 2: 'p2', 3: 'p3' };
const STATUS_OPTIONS = ['todo', 'in_progress', 'blocked'];

async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const r = await fetch(path, opts);
  if (r.status === 204) return null;
  return r.json();
}

async function loadTasks() {
  if (window.location.hash === '#add') {
    window.location.hash = '';
    showAddForm();
  }
  const url = showArchive ? '/tasks?archive=1' : '/tasks';
  const tasks = await api('GET', url);
  renderTasks(tasks || []);
}

function renderTasks(tasks) {
  const el = document.getElementById('task-list');
  if (!tasks.length) {
    el.innerHTML = '<p style="color:#636366;padding:20px 14px;font-size:13px;">No tasks.</p>';
    return;
  }

  const groups = [
    { key: 'in_progress', label: 'In Progress' },
    { key: 'todo',        label: 'Todo' },
    { key: 'blocked',     label: 'Blocked' },
    { key: 'archived',    label: 'Archived' },
  ];

  let html = '';
  for (const g of groups) {
    const items = tasks.filter(t => t.status === g.key);
    if (!items.length) continue;
    html += `<div class="group-label">${g.label}</div>`;
    for (const t of items) html += renderTaskRow(t);
  }
  el.innerHTML = html;
}

function renderTaskRow(t) {
  const prio = t.priority || 2;
  const pClass = PRIORITY_CLASS[prio] || 'p2';
  const subtaskCount = t.subtasks ? t.subtasks.length : 0;
  const subtaskDone = t.subtasks ? t.subtasks.filter(s => s.status === 'done').length : 0;
  const chip = subtaskCount ? `<span class="subtask-chip">${subtaskDone}/${subtaskCount}</span>` : '';
  const notesIcon = t.notes_path ? '📄' : '';
  const expanded = expandedId === t.id;

  let html = `
    <div class="task-row ${expanded ? 'expanded' : ''}" onclick="toggleExpand(${t.id})">
      <span class="priority-dot ${pClass}"></span>
      <span class="task-title">${escHtml(t.title)}</span>
      ${chip}
      <span class="notes-icon">${notesIcon}</span>
    </div>`;

  if (expanded) html += renderDetail(t);
  return html;
}

function renderDetail(t) {
  const prio = t.priority || 2;
  const statusOpts = STATUS_OPTIONS.map(s =>
    `<option value="${s}" ${t.status === s ? 'selected' : ''}>${s.replace('_', ' ')}</option>`
  ).join('');
  const prioOpts = [1, 2, 3].map(p =>
    `<option value="${p}" ${prio === p ? 'selected' : ''}>${PRIORITY_LABEL[p]}</option>`
  ).join('');

  let subtasksHtml = '';
  if (t.subtasks && t.subtasks.length) {
    subtasksHtml = t.subtasks.map(s => `
      <div class="subtask-item">
        <input type="checkbox" ${s.status === 'done' ? 'checked' : ''}
               onchange="setSubtaskDone(${s.id}, this.checked)">
        <span>${escHtml(s.title)}</span>
        <button class="subtask-promote" onclick="promoteSubtask(${s.id})" title="Promote to task">↑</button>
      </div>`).join('');
  }

  return `
    <div class="detail-panel">
      <div class="detail-row">
        <label>Status</label>
        <select class="inline" onchange="setStatus(${t.id}, this.value)">${statusOpts}</select>
      </div>
      <div class="detail-row">
        <label>Priority</label>
        <select class="inline" onchange="setPriority(${t.id}, this.value)">${prioOpts}</select>
      </div>
      <div class="subtask-list">${subtasksHtml}</div>
      <div class="add-subtask-row">
        <input type="text" id="sub-input-${t.id}" placeholder="Add subtask…">
        <button onclick="addSubtask(${t.id})">Add</button>
      </div>
      <button class="btn-notes" onclick="openNotes(${t.id})">📄 Notes</button>
    </div>`;
}

function toggleExpand(id) {
  expandedId = expandedId === id ? null : id;
  loadTasks();
}

async function setStatus(id, status) {
  await api('POST', `/tasks/${id}/status`, { status });
  loadTasks();
}

async function setPriority(id, priority) {
  await api('POST', `/tasks/${id}/priority`, { priority: parseInt(priority) });
  loadTasks();
}

async function setSubtaskDone(id, done) {
  await api('POST', `/tasks/${id}/status`, { done });
  loadTasks();
}

async function promoteSubtask(id) {
  await api('POST', `/tasks/${id}/promote`, {});
  expandedId = null;
  loadTasks();
}

async function openNotes(id) {
  await api('POST', `/tasks/${id}/notes`, {});
}

async function addSubtask(parentId) {
  const input = document.getElementById(`sub-input-${parentId}`);
  const title = input.value.trim();
  if (!title) return;
  await api('POST', '/tasks', { title, parent_id: parentId });
  input.value = '';
  loadTasks();
}

function showAddForm() {
  document.getElementById('add-trigger').style.display = 'none';
  document.getElementById('add-form').classList.remove('hidden');
  document.getElementById('add-title').focus();
}

async function submitTask(e) {
  e.preventDefault();
  const title = document.getElementById('add-title').value.trim();
  const priority = parseInt(document.getElementById('add-priority').value);
  const schedule = document.getElementById('add-schedule').value.trim();
  if (!title) return;

  if (schedule) {
    // create recurrence first — task will appear when due
    await api('POST', '/recurrences', { title, priority, schedule });
  } else {
    await api('POST', '/tasks', { title, priority });
  }

  document.getElementById('add-title').value = '';
  document.getElementById('add-schedule').value = '';
  document.getElementById('add-form').classList.add('hidden');
  document.getElementById('add-trigger').style.display = '';
  loadTasks();
}

function toggleArchive() {
  showArchive = !showArchive;
  const btn = document.getElementById('archive-toggle');
  btn.classList.toggle('active', showArchive);
  loadTasks();
}

function escHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// Poll for changes every 30s (scheduler may add tasks)
loadTasks();
setInterval(loadTasks, 30000);
