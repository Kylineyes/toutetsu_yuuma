<script setup>
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <div class="app">
    <aside class="sidebar">
      <h1>toutetsu_yuuma</h1>
      <p class="sub">视频阅读 · 总结 + 知识点</p>
      <nav>
        <RouterLink to="/tasks">任务</RouterLink>
        <RouterLink to="/tasks/new">新建任务</RouterLink>
        <RouterLink to="/profiles">配置方案</RouterLink>
        <RouterLink to="/knowledge-bases">知识库</RouterLink>
      </nav>
    </aside>
    <main class="content">
      <RouterView />
    </main>
  </div>
</template>

<style>
* { box-sizing: border-box; }
body { margin: 0; font-family: -apple-system, "Segoe UI", "Microsoft YaHei", sans-serif; color: #1f2937; background: #f8fafc; }
.app { display: flex; min-height: 100vh; }
.sidebar { width: 220px; background: #0f172a; color: #e2e8f0; padding: 20px 16px; flex-shrink: 0; }
.sidebar h1 { font-size: 16px; margin: 0 0 4px; }
.sidebar .sub { font-size: 12px; color: #94a3b8; margin: 0 0 20px; }
.sidebar nav { display: flex; flex-direction: column; gap: 6px; }
.sidebar nav a { color: #cbd5e1; text-decoration: none; padding: 8px 10px; border-radius: 6px; font-size: 14px; }
.sidebar nav a.router-link-active { background: #1e293b; color: #fff; }
.content { flex: 1; padding: 28px 36px; max-width: 1100px; }
h2 { margin-top: 0; }
input, select, textarea { font: inherit; padding: 7px 9px; border: 1px solid #cbd5e1; border-radius: 6px; background: #fff; }
button { font: inherit; padding: 7px 14px; border: 0; border-radius: 6px; background: #2563eb; color: #fff; cursor: pointer; }
button.danger { background: #dc2626; }
button.ghost { background: #e2e8f0; color: #0f172a; }
table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; }
th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid #eef2f7; font-size: 14px; vertical-align: top; }
th { background: #f1f5f9; font-weight: 600; }
.card { background: #fff; border: 1px solid #e2e8f0; border-radius: 10px; padding: 18px; margin-bottom: 18px; }
.row { display: flex; gap: 12px; flex-wrap: wrap; align-items: center; margin-bottom: 12px; }
label.fld { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: #475569; }
.tag { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 12px; background: #e2e8f0; }
.tag.done { background: #dcfce7; color: #166534; }
.tag.failed { background: #fee2e2; color: #991b1b; }
.tag.running { background: #dbeafe; color: #1e40af; }
pre.md { white-space: pre-wrap; background: #fff; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; font-family: ui-monospace, Consolas, monospace; font-size: 13px; }
</style>
