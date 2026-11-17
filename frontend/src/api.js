const base = '/api/v1'

async function handle(res) {
  if (!res.ok) {
    let msg = ''
    try { msg = (await res.json()).error } catch (e) { msg = await res.text() }
    throw new Error(msg || ('HTTP ' + res.status))
  }
  return res.json()
}

export const api = {
  profiles: (cap) => fetch(`${base}/profiles${cap ? '?capability=' + cap : ''}`).then(handle),
  createProfile: (b) => fetch(`${base}/profiles`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(b) }).then(handle),
  deleteProfile: (id) => fetch(`${base}/profiles/${id}`, { method: 'DELETE' }).then(handle),
  kbs: () => fetch(`${base}/knowledge-bases`).then(handle),
  createKB: (name) => fetch(`${base}/knowledge-bases`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }).then(handle),
  deleteKB: (id) => fetch(`${base}/knowledge-bases/${id}`, { method: 'DELETE' }).then(handle),
  kbVersions: (id) => fetch(`${base}/knowledge-bases/${id}/versions`).then(handle),
  uploadKB: (id, fd) => fetch(`${base}/knowledge-bases/${id}/versions`, { method: 'POST', body: fd }).then(handle),
  tasks: () => fetch(`${base}/tasks`).then(handle),
  task: (id) => fetch(`${base}/tasks/${id}`).then(handle),
  taskResult: (id) => fetch(`${base}/tasks/${id}/result`).then(handle),
  createTask: (fd) => fetch(`${base}/tasks`, { method: 'POST', body: fd }).then(handle),
  retryTask: (id) => fetch(`${base}/tasks/${id}/retry`, { method: 'POST' }).then(handle),
  deleteTask: (id) => fetch(`${base}/tasks/${id}`, { method: 'DELETE' }).then(handle),
  fileUrl: (id, name) => `${base}/tasks/${id}/files/${name}`,
}
