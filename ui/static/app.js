let showArchive = false;
let expandedId = null;
let currentTab = 'tasks';
let currentNoteId = null;
let noteHasUnsavedChanges = false;

const PRIORITY_LABEL = { 1: 'High', 2: 'Medium', 3: 'Low' };
const PRIORITY_CLASS  = { 1: 'p1',   2: 'p2',    3: 'p3'  };
const STATUS_OPTIONS  = ['todo', 'in_progress', 'blocked', 'done'];

// Markdown shortcut buttons shown in the editor toolbar
const SHORTCUTS = [
  { label: '# H1',      text: '# '       },
  { label: '## H2',     text: '## '      },
  { label: '### H3',    text: '### '     },
  { label: '**bold**',  text: '**bold**' },
  { label: '_italic_',  text: '_italic_' },
  { label: '`code`',    text: '`code`'   },
  { label: '```block',  text: '```\n\n```' },
  { label: '> quote',   text: '> '       },
  { label: '- list',    text: '- '       },
  { label: '1. list',   text: '1. '      },
  { label: '[ ] todo',  text: '- [ ] '   },
  { label: '---',       text: '---\n'    },
];

// ─── HTTP helper ─────────────────────────────────────────────────────────────

async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const r = await fetch(path, opts);
  if (r.status === 204) return null;
  const data = await r.json().catch(() => ({}));
  if (!r.ok) {
    const msg = data.error || `HTTP ${r.status}`;
    alert(msg);
    throw new Error(msg);
  }
  return data;
}

// ─── Tab switching ────────────────────────────────────────────────────────────

function switchTab(tab) {
  currentTab = tab;
  document.getElementById('tab-tasks').classList.toggle('active', tab === 'tasks');
  document.getElementById('tab-schedules').classList.toggle('active', tab === 'schedules');
  document.getElementById('tab-notes').classList.toggle('active', tab === 'notes');
  document.getElementById('tasks-view').classList.toggle('hidden', tab !== 'tasks');
  document.getElementById('schedules-view').classList.toggle('hidden', tab !== 'schedules');
  document.getElementById('notes-view').classList.toggle('hidden', tab !== 'notes');
  document.getElementById('archive-toggle').style.display = tab === 'tasks' ? '' : 'none';
  if (tab === 'tasks') loadTasks();
  else if (tab === 'schedules') loadSchedules();
  else loadNotes();
}

// ─── Notes ───────────────────────────────────────────────────────────────────

async function loadNotes() {
  const notes = await api('GET', '/notes');
  renderNotesList(notes || []);
}

function renderNotesList(notes) {
  const el = document.getElementById('notes-list');
  if (!notes.length) {
    el.innerHTML = '<p style="color:#636366;padding:20px 14px;font-size:13px;">No notes. Add one below.</p>';
    return;
  }

  // Group by task_id
  const byTask = {};
  const standalone = [];
  notes.forEach(n => {
    if (n.task_id != null) {
      (byTask[n.task_id] = byTask[n.task_id] || []).push(n);
    } else {
      standalone.push(n);
    }
  });

  let html = '';

  if (standalone.length) {
    html += '<div class="group-label">Standalone Notes</div>';
    standalone.forEach(n => { html += noteRowHtml(n); });
  }

  for (const [taskId, taskNotes] of Object.entries(byTask)) {
    const tasks = window.activeTasks || [];
    const task = tasks.find(t => t.id === parseInt(taskId));
    const label = task ? escHtml(task.title) : `Task #${taskId}`;
    html += `<div class="group-label">${label}</div>`;
    taskNotes.forEach(n => { html += noteRowHtml(n); });
  }

  el.innerHTML = html;
}

function noteRowHtml(n) {
  const modified = n.modified_at ? new Date(n.modified_at).toLocaleString() : '—';
  return `
    <div class="note-row" onclick="openNote(${n.id})">
      <div style="display:flex;align-items:center;gap:6px;">
        <div style="flex:1;font-size:13px;color:#f2f2f7;">${escHtml(n.title)}</div>
        <button class="btn-sm" style="font-size:11px;padding:2px 8px;color:#ff453a;" onclick="event.stopPropagation();deleteNote(${n.id})">Delete</button>
      </div>
      <div style="font-size:11px;color:#636366;">${modified}</div>
    </div>`;
}

async function deleteNote(id) {
  if (!confirm('Delete this note? This cannot be undone.')) return;
  await api('DELETE', `/notes/${id}`);
  loadNotes();
}

async function deleteCurrentNote() {
  if (currentNoteId === null) return;
  if (!confirm('Delete this note? This cannot be undone.')) return;
  const id = currentNoteId;
  currentNoteId = null;
  noteHasUnsavedChanges = false;
  await api('DELETE', `/notes/${id}`);
  document.getElementById('note-editor-container').classList.add('hidden');
  document.getElementById('notes-list-panel').classList.remove('hidden');
  loadNotes();
}

async function addNote() {
  const input = document.getElementById('add-note-title');
  const title = input.value.trim();
  if (!title) return;

  const note = await api('POST', '/notes', { title });
  input.value = '';
  if (note && note.id) {
    await loadNotes();
    openNote(note.id);
  } else {
    loadNotes();
  }
}

async function openNote(id) {
  currentNoteId = id;
  noteHasUnsavedChanges = false;

  const note = await api('GET', `/notes/${id}`);
  document.getElementById('note-title-display').textContent = note.title;
  document.getElementById('note-editor-text').value = note.content || '';
  updatePreview();

  document.getElementById('notes-list-panel').classList.add('hidden');
  document.getElementById('note-editor-container').classList.remove('hidden');

  document.getElementById('note-editor-text').focus();
}

function closeEditor() {
  if (noteHasUnsavedChanges) {
    saveNote();
  }
  currentNoteId = null;
  noteHasUnsavedChanges = false;

  document.getElementById('note-editor-container').classList.add('hidden');
  document.getElementById('notes-list-panel').classList.remove('hidden');
  loadNotes();
}

function updatePreview() {
  const content = document.getElementById('note-editor-text').value;
  const previewEl = document.getElementById('note-preview-content');
  previewEl.innerHTML = marked.parse(content);
  if (typeof Prism !== 'undefined') {
    Prism.highlightAllUnder(previewEl);
  }
}

function insertShortcut(snippet) {
  const textarea = document.getElementById('note-editor-text');
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const before = textarea.value.substring(0, start);
  const after  = textarea.value.substring(end);
  textarea.value = before + snippet + after;
  textarea.selectionStart = start + snippet.length;
  textarea.selectionEnd   = start + snippet.length;
  textarea.focus();
  updatePreview();
  noteHasUnsavedChanges = true;
  setSaveBtnDirty();
}

async function saveNote() {
  if (currentNoteId === null) return;
  const content = document.getElementById('note-editor-text').value;
  await api('PUT', `/notes/${currentNoteId}`, { content });
  noteHasUnsavedChanges = false;
  setSaveBtnSaved();
}

function setSaveBtnDirty() {
  const btn = document.getElementById('save-note-btn');
  if (btn) { btn.textContent = 'Save'; btn.style.opacity = '1'; }
}

function setSaveBtnSaved() {
  const btn = document.getElementById('save-note-btn');
  if (btn) {
    btn.textContent = 'Saved';
    btn.style.opacity = '0.6';
    setTimeout(() => { btn.textContent = 'Save'; btn.style.opacity = '1'; }, 1500);
  }
}

// Wire up textarea events after DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  const ta = document.getElementById('note-editor-text');
  if (ta) {
    ta.addEventListener('input', () => {
      updatePreview();
      noteHasUnsavedChanges = true;
      setSaveBtnDirty();
    });
    ta.addEventListener('blur', () => {
      if (noteHasUnsavedChanges) saveNote();
    });
  }

  // Ctrl/Cmd+S saves
  document.addEventListener('keydown', e => {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      if (currentNoteId !== null) saveNote();
    }
  });

  // Build shortcut buttons
  const grid = document.getElementById('shortcut-grid');
  if (grid) {
    SHORTCUTS.forEach(sc => {
      const btn = document.createElement('button');
      btn.className = 'shortcut-btn';
      btn.textContent = sc.label;
      btn.onclick = () => insertShortcut(sc.text);
      grid.appendChild(btn);
    });
  }
});

// ─── Tasks ────────────────────────────────────────────────────────────────────

async function loadTasks() {
  if (window.location.hash === '#add') {
    window.location.hash = '';
    showAddForm();
  }
  const url = showArchive ? '/tasks?archive=1' : '/tasks';
  const tasks = await api('GET', url);
  window.activeTasks = tasks || [];
  renderTasks(window.activeTasks);
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
  const prio   = t.priority || 2;
  const pClass = PRIORITY_CLASS[prio] || 'p2';
  const subtaskCount = t.subtasks ? t.subtasks.length : 0;
  const subtaskDone  = t.subtasks ? t.subtasks.filter(s => s.status === 'done').length : 0;
  const chip     = subtaskCount ? `<span class="subtask-chip">${subtaskDone}/${subtaskCount}</span>` : '';
  const notesIcon = t.notes_path ? '📄' : '';
  const expanded  = expandedId === t.id;

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
  const result = await api('POST', `/tasks/${id}/notes`, {});
  switchTab('notes');
  if (result && result.note_id) {
    await openNote(result.note_id);
  } else {
    await loadNotes();
  }
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
  const title    = document.getElementById('add-title').value.trim();
  const priority = parseInt(document.getElementById('add-priority').value);
  const schedule = document.getElementById('add-schedule').value.trim();
  if (!title) return;

  if (schedule) {
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

// ─── Schedules ────────────────────────────────────────────────────────────────

async function loadSchedules() {
  const recs = await api('GET', '/recurrences');
  renderSchedules(recs || []);
}

function renderSchedules(recs) {
  const el = document.getElementById('schedule-list');
  if (!recs.length) {
    el.innerHTML = '<p style="color:#636366;padding:20px 14px;font-size:13px;">No scheduled tasks.</p>';
    return;
  }
  el.innerHTML = recs.map(r => {
    const nextDue = r.next_due_at
      ? new Date(r.next_due_at).toLocaleString('default', {
          weekday: 'short', month: 'short', day: 'numeric',
          hour: 'numeric', minute: '2-digit',
        })
      : '—';
    const toggleBtn = r.active
      ? `<button class="btn-sm" onclick="toggleSchedule(${r.id}, false)">Pause</button>`
      : `<button class="btn-sm" onclick="toggleSchedule(${r.id}, true)">Resume</button>`;
    return `
      <div class="schedule-row">
        <div class="schedule-title${r.active ? '' : ' inactive'}">${escHtml(r.title)}</div>
        <div class="schedule-meta">${escHtml(r.schedule)} · next: ${nextDue}</div>
        <div class="schedule-actions">
          ${toggleBtn}
          <button class="btn-sm danger" onclick="deleteSchedule(${r.id})">Delete</button>
        </div>
      </div>`;
  }).join('');
}

async function toggleSchedule(id, active) {
  await api('POST', `/recurrences/${id}`, { active });
  loadSchedules();
}

async function deleteSchedule(id) {
  await api('DELETE', `/recurrences/${id}`);
  loadSchedules();
}

// ─── Shared ───────────────────────────────────────────────────────────────────

function escHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// ─── Boot ─────────────────────────────────────────────────────────────────────

loadTasks();

setInterval(() => {
  if (currentTab === 'tasks') loadTasks();
  else if (currentTab === 'schedules') loadSchedules();
  // notes are loaded on demand
}, 30000);
