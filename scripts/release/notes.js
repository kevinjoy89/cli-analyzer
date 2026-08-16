#!/usr/bin/env node
// 发布 release notes：读取 docs/release-notes/<tag>.md，PATCH 到 GitHub release。
// 用法：
//   GH_PAT=<token> node scripts/release/notes.js v0.3.9          # 提交 body + 发布
//   GH_PAT=<token> node scripts/release/notes.js v0.3.9 --verify  # 仅本地自检，不提交
// 自检规则（血泪教训）：源文件必须含 CJK 字符，否则拒绝提交（PS 5.1 曾把中文全变 ?）。
const fs = require('fs');
const path = require('path');

const repo = 'kevinjoy89/cli-analyzer';
const tag = process.argv[2];
const verifyOnly = process.argv.includes('--verify');
const token = process.env.GH_PAT;

if (!tag) { console.error('用法: node scripts/release/notes.js <tag> [--verify]'); process.exit(1); }
if (!verifyOnly && !token) { console.error('GH_PAT 未设置'); process.exit(1); }

const notesPath = path.join(__dirname, '..', '..', 'docs', 'release-notes', tag + '.md');
const notes = fs.readFileSync(notesPath, 'utf8');

// 自检 1：必须含中文（双语模板的强制项）
if (!/[\u4e00-\u9fff]/.test(notes)) {
  console.error(`FAIL: ${notesPath} 不含任何 CJK 字符 —— 双语 notes 缺失，拒绝提交`);
  process.exit(1);
}
// 自检 2：必须含英文段落分隔（---）与下载产物段落
if (!notes.includes('---') || !/CLI-Analyzer-|checksums\.txt/.test(notes)) {
  console.error('FAIL: notes 缺少中英分隔（---）或下载产物清单，拒绝提交');
  process.exit(1);
}
console.log('本地自检通过:', notesPath, `(${notes.length} chars)`);
if (verifyOnly) process.exit(0);

(async () => {
  // 按 tag 找 release id（不再硬编码）
  const list = await fetch(`https://api.github.com/repos/${repo}/releases/tags/${encodeURIComponent(tag)}`, {
    headers: { Accept: 'application/vnd.github+json', Authorization: 'Bearer ' + token },
  });
  if (!list.ok) { console.error('查找 release 失败:', list.status, await list.text()); process.exit(1); }
  const rel = await list.json();

  const res = await fetch(`https://api.github.com/repos/${repo}/releases/${rel.id}`, {
    method: 'PATCH',
    headers: {
      'Accept': 'application/vnd.github+json',
      'X-GitHub-Api-Version': '2022-11-28',
      'Authorization': 'Bearer ' + token,
      'Content-Type': 'application/json; charset=utf-8',
    },
    body: JSON.stringify({ body: notes, draft: false, prerelease: false }),
  });
  const data = await res.json();
  console.log('status:', res.status);
  console.log('tag:', data.tag_name, '| draft:', data.draft, '| assets:', data.assets ? data.assets.length : 'n/a');
  console.log('body head:', data.body ? data.body.slice(0, 70) : '(no body)');
  // 提交后回读校验 CJK（v0.3.8 血泪教训：提交后必须验证而非只看状态码）
  if (!/[\u4e00-\u9fff]/.test(data.body || '')) {
    console.error('FAIL: 提交后回读 body 不含 CJK —— 编码问题，需用 node fetch 方式重提');
    process.exit(1);
  }
  console.log('发布成功 ✓（draft=false）');
})().catch(e => { console.error('FAIL:', e.message); process.exit(1); });
