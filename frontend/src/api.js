const base = '/api/v1'

function H(extra) {
  const t = localStorage.getItem('api_token') || ''
  const h = Object.assign({}, extra || {})
  if (t) h['X-API-Token'] = t
  return h
}

async function handle(res) {
  if (!res.ok) {
    let msg = ''
    try { msg = (await res.json()).error } catch (e) { msg = await res.text() }
    throw new Error(msg || ('HTTP ' + res.status))
  }
  return res.json()
}

export const api = {
  getSettings: () => fetch(`${base}/settings`).then(handle),
  saveSettings: (token) => fetch(`${base}/settings`, { method: 'PUT', headers: H({ 'Content-Type': 'application/json' }), body: JSON.stringify({ api_token: token }) }).then(handle),

  profiles: (cap) => fetch(`${base}/profiles${cap ? '?capability=' + cap : ''}`, { headers: H() }).then(handle),
  createProfile: (b) => fetch(`${base}/profiles`, { method: 'POST', headers: H({ 'Content-Type': 'application/json' }), body: JSON.stringify(b) }).then(handle),
  deleteProfile: (id) => fetch(`${base}/profiles/${id}`, { method: 'DELETE', headers: H() }).then(handle),

  kbs: () => fetch(`${base}/knowledge-bases`, { headers: H() }).then(handle),
  createKB: (name) => fetch(`${base}/knowledge-bases`, { method: 'POST', headers: H({ 'Content-Type': 'application/json' }), body: JSON.stringify({ name }) }).then(handle),
  deleteKB: (id) => fetch(`${base}/knowledge-bases/${id}`, { method: 'DELETE', headers: H() }).then(handle),
  kbVersions: (id) => fetch(`${base}/knowledge-bases/${id}/versions`, { headers: H() }).then(handle),
  uploadKB: (id, fd) => fetch(`${base}/knowledge-bases/${id}/versions`, { method: 'POST', headers: H(), body: fd }).then(handle),

  tasks: () => fetch(`${base}/tasks`, { headers: H() }).then(handle),
  task: (id) => fetch(`${base}/tasks/${id}`, { headers: H() }).then(handle),
  taskResult: (id) => fetch(`${base}/tasks/${id}/result`, { headers: H() }).then(handle),
  createTask: (fd) => fetch(`${base}/tasks`, { method: 'POST', headers: H(), body: fd }).then(handle),
  retryTask: (id) => fetch(`${base}/tasks/${id}/retry`, { method: 'POST', headers: H() }).then(handle),
  cleanupCache: (id) => fetch(`${base}/tasks/${id}/cache`, { method: 'DELETE', headers: H() }).then(handle),
  deleteTask: (id) => fetch(`${base}/tasks/${id}`, { method: 'DELETE', headers: H() }).then(handle),
  fileUrl: (id, name) => `${base}/tasks/${id}/files/${name}${localStorage.getItem('api_token') ? '?token=' + encodeURIComponent(localStorage.getItem('api_token')) : ''}`,
}
