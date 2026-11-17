const base = '/api/v1'

async function handle(res) {
  if (!res.ok) {
    let msg = ''
    try { msg = (await res.json()).error } catch (e) { msg = await res.text() }
    const err = new Error(msg || ('HTTP ' + res.status))
    err.status = res.status
    throw err
  }
  return res.json()
}
const j = (method) => (url, body) => fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then(handle)

export const api = {
  session: () => fetch(base + '/session').then((r) => r.json()),
  login: (password) => fetch(base + '/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ password }) }).then(handle),
  logout: () => fetch(base + '/logout', { method: 'POST' }).then(handle),

  profiles: (cap) => fetch(`${base}/profiles${cap ? '?capability=' + cap : ''}`).then(handle),
  createProfile: (b) => j('POST')(`${base}/profiles`, b),
  deleteProfile: (id) => fetch(`${base}/profiles/${id}`, { method: 'DELETE' }).then(handle),

  kbs: () => fetch(base + '/knowledge-bases').then(handle),
  createKB: (name) => j('POST')(base + '/knowledge-bases', { name }),
  deleteKB: (id) => fetch(`${base}/knowledge-bases/${id}`, { method: 'DELETE' }).then(handle),
  kbVersions: (id) => fetch(`${base}/knowledge-bases/${id}/versions`).then(handle),
  uploadKB: (id, fd) => fetch(`${base}/knowledge-bases/${id}/versions`, { method: 'POST', body: fd }).then(handle),
  deleteKBVersion: (id, v) => fetch(`${base}/knowledge-bases/${id}/versions/${v}`, { method: 'DELETE' }).then(handle),

  tasks: () => fetch(base + '/tasks').then(handle),
  task: (id) => fetch(`${base}/tasks/${id}`).then(handle),
  taskResult: (id) => fetch(`${base}/tasks/${id}/result`).then(handle),
  taskStages: (id) => fetch(`${base}/tasks/${id}/stages`).then(handle),
  createTask: (fd) => fetch(base + '/tasks', { method: 'POST', body: fd }).then(handle),
  retryTask: (id) => fetch(`${base}/tasks/${id}/retry`, { method: 'POST' }).then(handle),
  cleanupCache: (id) => fetch(`${base}/tasks/${id}/cache`, { method: 'DELETE' }).then(handle),
  deleteTask: (id) => fetch(`${base}/tasks/${id}`, { method: 'DELETE' }).then(handle),
  fileUrl: (id, name) => `${base}/tasks/${id}/files/${name}`,
}
