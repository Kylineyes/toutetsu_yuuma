// 前端视觉与主题验收脚本：对 vite preview 产物逐页截图（明/暗），并验证主题切换与持久化。
// 依赖：playwright（需含浏览器；本机以 channel: msedge 调用系统 Edge，无需下载 chromium）。
// 用法：先在本仓库 frontend 下 `npx vite preview --port 4173 --strictPort`，
//       再在本目录（或在任意装有 playwright 的目录内）执行 `node capture.mjs`。
import { chromium } from 'playwright';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const BASE = process.env.PREVIEW_URL || 'http://localhost:4173';
const OUT = process.env.OUT_DIR ? path.resolve(process.env.OUT_DIR) : path.dirname(fileURLToPath(import.meta.url));

const tasks = [
  { id: 'task-20261119-001', filename: '2026新西兰酒展来了，云雾之湾世纪大和解！.mp4', status: 'done', progress: 100, created_at: '2026-11-19 03:12' },
  { id: 'task-20261119-002', filename: '船长测评：？分的玛莎诺娅坡地半甜红.mp4', status: 'running', progress: 62, created_at: '2026-11-19 04:05' },
  { id: 'task-20261119-003', filename: '勃艮第夜丘风土讲解.mp4', status: 'failed', progress: 38, created_at: '2026-11-19 04:40' },
  { id: 'task-20261119-004', filename: '香槟产区与酿造工艺.mp4', status: 'pending', progress: 0, created_at: '2026-11-19 04:52' },
];
const stages = [
  { id: 1, stage: 'intake', status: 'done', progress: 100, duration_ms: 4200, message: '7 类素材已登记' },
  { id: 2, stage: 'import', status: 'done', progress: 100, duration_ms: 1200, message: '' },
  { id: 3, stage: 'audio', status: 'done', progress: 100, duration_ms: 8600, message: '' },
  { id: 4, stage: 'asr', status: 'done', progress: 100, duration_ms: 128400, message: '云端 ASR · 57 分片' },
  { id: 5, stage: 'frame', status: 'skipped', progress: 0, duration_ms: 0, message: '已有可信原生字幕，无需抽帧' },
  { id: 6, stage: 'fuse', status: 'running', progress: 45, duration_ms: 0, message: '基底 native，ASR 补齐中' },
  { id: 7, stage: 'glossary', status: 'pending', progress: 0, duration_ms: 0, message: '' },
  { id: 8, stage: 'summary', status: 'pending', progress: 0, duration_ms: 0, message: '' },
  { id: 9, stage: 'quality', status: 'pending', progress: 0, duration_ms: 0, message: '' },
];
const task = {
  id: 'task-20261119-001', filename: '2026新西兰酒展来了，云雾之湾世纪大和解！.mp4',
  status: 'done', progress: 100, reading_depth: 2, screenshot_interval: 60,
  started_at: '2026-11-19 03:12', finished_at: '2026-11-19 03:48', error: '',
};
const summaryMd = '# 2026 新西兰酒展：云雾之湾的世纪大和解\n\n## 主线\n- 云雾之湾与竞争对手在同一展台完成和解，双方承认产区表达存在历史分歧。\n- 展会现场以盲品形式验证双方 2019/2021 两个年份的差异。\n\n## 关键结论\n1. 马尔堡产区的长相思风格正在从「果味主导」转向「结构主导」。\n2. 和解的直接动因是联合出口配额谈判。\n';
const knowledgeMd = '## 术语卡\n\n- **云雾之湾（Cloudy Bay）**：新西兰马尔堡产区代表酒庄。\n- **长相思（Sauvignon Blanc）**：芳香型白葡萄品种。\n- **盲品**：不提供酒标的品鉴形式。\n';
const result = { cost: { asr_seconds: 3412, vision_images: 0, prompt_tokens: 48210, completion_tokens: 3260 }, summary_md: summaryMd, knowledge_md: knowledgeMd };
const profiles = [
  { id: 1, capability: 'summary', name: 'deepseek-chat', protocol: 'openai_chat', model: 'deepseek-chat', base_url: 'https://api.deepseek.com', temperature: 0.3, timeout_seconds: 180, extra: '{}' },
  { id: 2, capability: 'asr', name: 'deepseek-asr', protocol: 'deepseek_asr', model: 'asr-1', base_url: 'https://api.deepseek.com', temperature: null, timeout_seconds: 300, extra: '{}' },
  { id: 3, capability: 'vision', name: 'qwen-vl', protocol: 'openai_chat', model: 'qwen2.5-vl-72b', base_url: 'https://dashscope.aliyuncs.com', temperature: 0.1, timeout_seconds: 120, extra: '{"max_tokens":4096}' },
  { id: 4, capability: 'reading', name: 'deepseek-reasoner', protocol: 'openai_chat', model: 'deepseek-reasoner', base_url: 'https://api.deepseek.com', temperature: null, timeout_seconds: 240, extra: '{}' },
];
const kbs = [
  { id: 3, name: '葡萄酒术语库', latest_version: 2 },
  { id: 5, name: '新西兰产区与酒庄', latest_version: 1 },
];
const versions = [
  { id: 9, version: 2, file_name: 'wine-terms-v2.csv', term_count: 128, size_bytes: 18342, sha256: 'a1b2c3d4e5f60718293a4b5c6d7e8f9012345678', created_at: '2026-11-18 22:10' },
  { id: 8, version: 1, file_name: 'wine-terms-v1.csv', term_count: 96, size_bytes: 14210, sha256: 'b2c3d4e5f60718293a4b5c6d7e8f90123456789a', created_at: '2026-11-12 09:33' },
];

const json = (route, body) => route.fulfill({ status: 200, contentType: 'application/json; charset=utf-8', body: JSON.stringify(body) });

async function mock(route) {
  const p = new URL(route.request().url()).pathname.replace(/^\/api\/v1/, '');
  if (p === '/session') return json(route, { login_required: false, authenticated: true });
  if (p === '/tasks') return json(route, tasks);
  if (/^\/tasks\/[^/]+\/stages$/.test(p)) return json(route, stages);
  if (/^\/tasks\/[^/]+\/result$/.test(p)) return json(route, result);
  if (/^\/tasks\/[^/]+$/.test(p)) return json(route, task);
  if (p === '/profiles') return json(route, profiles);
  if (p === '/knowledge-bases') return json(route, kbs);
  if (/^\/knowledge-bases\/\d+\/versions$/.test(p)) return json(route, versions);
  return json(route, {});
}

const pages = [
  ['login', '/login'],
  ['tasks', '/tasks'],
  ['tasks-new', '/tasks/new'],
  ['task-detail', '/tasks/task-20261119-001'],
  ['profiles', '/profiles'],
  ['knowledge-bases', '/knowledge-bases'],
];

const browser = await chromium.launch({ channel: 'msedge', headless: true });
const report = { base: BASE, themes: {}, toggle: null, mobile: null, consoleErrors: [] };

for (const theme of ['light', 'dark']) {
  const dir = path.join(OUT, theme);
  fs.mkdirSync(dir, { recursive: true });
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1, colorScheme: theme });
  await ctx.addInitScript((t) => localStorage.setItem('ty-theme', t), theme);
  await ctx.route('**/api/v1/**', mock);
  const page = await ctx.newPage();
  page.on('console', (m) => { if (m.type() === 'error') report.consoleErrors.push(theme + ' ' + page.url() + ' :: ' + m.text()); });
  page.on('pageerror', (e) => report.consoleErrors.push(theme + ' PAGEERROR ' + page.url() + ' :: ' + e.message));
  const shots = [];
  for (const [name, pathname] of pages) {
    await page.goto(BASE + pathname, { waitUntil: 'networkidle' });
    await page.waitForTimeout(700);
    const themeAttr = await page.evaluate(() => document.documentElement.dataset.theme);
    const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
    await page.screenshot({ path: path.join(dir, name + '.png'), fullPage: true });
    shots.push({ name, themeAttr, bg });
  }
  report.themes[theme] = shots;
  await ctx.close();
}

// 主题切换：默认跟随系统（light）→ 点击切深色 → 刷新后保持 → 再切回
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: 'light' });
  await ctx.route('**/api/v1/**', mock);
  const page = await ctx.newPage();
  await page.goto(BASE + '/tasks', { waitUntil: 'networkidle' });
  const read = () => page.evaluate(() => ({ theme: document.documentElement.dataset.theme, stored: localStorage.getItem('ty-theme') }));
  const initial = await read();
  await page.click('button[aria-label="切换到深色主题"]');
  await page.waitForTimeout(500);
  const afterToggle = await read();
  await page.screenshot({ path: path.join(OUT, 'toggle-after-click.png'), fullPage: true });
  await page.reload({ waitUntil: 'networkidle' });
  const afterReload = await read();
  await page.click('button[aria-label="切换到浅色主题"]');
  await page.waitForTimeout(400);
  const afterToggleBack = await read();
  report.toggle = { initial, afterToggle, afterReload, afterToggleBack };

  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForTimeout(400);
  report.mobile = await page.evaluate(() => ({ scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth }));
  await page.screenshot({ path: path.join(OUT, 'mobile-tasks.png'), fullPage: true });
  await ctx.close();
}

await browser.close();
fs.writeFileSync(path.join(OUT, 'report.json'), JSON.stringify(report, null, 2));
console.log(JSON.stringify(report, null, 2));
