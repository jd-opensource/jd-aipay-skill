#!/usr/bin/env node

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');

// ── 常量 ──────────────────────────────────────────────
const SKILL_NAME = 'jd-aipay-onboarding';
const SKILLS_DIR = path.join(os.homedir(), '.claude', 'skills');
const TARGET_DIR = path.join(SKILLS_DIR, SKILL_NAME);

// npm 包根目录（bin/cli.js 的上级）
const PKG_ROOT = path.resolve(__dirname, '..');

// 需要拷贝到 skill 目录的内容
const ENTRIES = ['SKILL.md', 'playbooks', 'reference', 'assets', 'scripts'];

// ── 颜色输出 ──────────────────────────────────────────
const c = {
  green: (s) => `\x1b[32m${s}\x1b[0m`,
  red: (s) => `\x1b[31m${s}\x1b[0m`,
  yellow: (s) => `\x1b[33m${s}\x1b[0m`,
  cyan: (s) => `\x1b[36m${s}\x1b[0m`,
  bold: (s) => `\x1b[1m${s}\x1b[0m`,
  dim: (s) => `\x1b[2m${s}\x1b[0m`,
};

// ── 工具函数 ──────────────────────────────────────────
function copyRecursive(src, dest) {
  const stat = fs.statSync(src);
  if (stat.isDirectory()) {
    fs.mkdirSync(dest, { recursive: true });
    for (const entry of fs.readdirSync(src)) {
      copyRecursive(path.join(src, entry), path.join(dest, entry));
    }
  } else {
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    fs.copyFileSync(src, dest);
  }
}

function removeRecursive(dir) {
  if (!fs.existsSync(dir)) return;
  // Node 14.14+ 支持 rm recursive
  if (fs.rmSync) {
    fs.rmSync(dir, { recursive: true, force: true });
  } else {
    // 低版本 fallback
    fs.rmdirSync(dir, { recursive: true });
  }
}

// ── install ───────────────────────────────────────────
function install(force) {
  console.log();
  console.log(c.bold('  ⚡ 京东 AI付 · Claude Code Skill 安装器'));
  console.log(c.dim('  ─────────────────────────────────────────'));
  console.log();

  // 确保 ~/.claude/skills/ 存在
  fs.mkdirSync(SKILLS_DIR, { recursive: true });

  // 检查是否已安装
  if (fs.existsSync(TARGET_DIR)) {
    if (!force) {
      console.log(c.yellow(`  ⚠  技能目录已存在：${TARGET_DIR}`));
      console.log(c.yellow('     使用 --force 参数覆盖安装'));
      console.log();
      process.exit(1);
    }
    console.log(c.yellow('  ♻  检测到已有安装，覆盖中...'));
    removeRecursive(TARGET_DIR);
  }

  // 拷贝各入口
  fs.mkdirSync(TARGET_DIR, { recursive: true });
  let fileCount = 0;
  for (const entry of ENTRIES) {
    const src = path.join(PKG_ROOT, entry);
    const dest = path.join(TARGET_DIR, entry);
    if (!fs.existsSync(src)) {
      console.log(c.dim(`  ⏭  跳过不存在的：${entry}`));
      continue;
    }
    copyRecursive(src, dest);
    const stat = fs.statSync(src);
    if (stat.isDirectory()) {
      const count = countFiles(src);
      fileCount += count;
      console.log(c.green(`  ✓  ${entry}/`) + c.dim(` (${count} 个文件)`));
    } else {
      fileCount += 1;
      console.log(c.green(`  ✓  ${entry}`));
    }
  }

  console.log();
  console.log(c.green(c.bold('  ✅ 安装完成！')));
  console.log(c.dim(`     共 ${fileCount} 个文件 → ${TARGET_DIR}`));
  console.log();
  console.log(c.bold('  📖 使用方式：'));
  console.log('     打开 Claude Code，在你的服务端项目中输入：');
  console.log(c.cyan('     「帮我接入 AI付」'));
  console.log();
  console.log(c.dim('  卸载：npx -y @jdpay/aipay uninstall'));
  console.log();
}

function countFiles(dir) {
  let count = 0;
  const stat = fs.statSync(dir);
  if (stat.isDirectory()) {
    for (const entry of fs.readdirSync(dir)) {
      count += countFiles(path.join(dir, entry));
    }
  } else {
    count = 1;
  }
  return count;
}

// ── uninstall ─────────────────────────────────────────
function uninstall() {
  console.log();
  console.log(c.bold('  ⚡ 京东 AI付 · Claude Code Skill 卸载'));
  console.log(c.dim('  ─────────────────────────────────────────'));
  console.log();

  if (!fs.existsSync(TARGET_DIR)) {
    console.log(c.yellow('  ⚠  技能未安装，无需卸载'));
    console.log();
    return;
  }

  removeRecursive(TARGET_DIR);

  console.log(c.green('  ✅ 已卸载 jd-aipay-onboarding 技能'));
  console.log(c.dim(`     已删除：${TARGET_DIR}`));
  console.log();
  console.log(c.dim('  重新安装：npx -y @jdpay/aipay@latest install'));
  console.log();
}

// ── help ──────────────────────────────────────────────
function showHelp() {
  console.log(`
  ${c.bold('⚡ @jdpay/aipay')} — 京东 AI付 Claude Code Skill 安装器

  ${c.bold('用法：')}
    npx -y @jdpay/aipay@latest install          安装 AI付 技能到 Claude Code
    npx -y @jdpay/aipay@latest install --force   覆盖已有安装
    npx -y @jdpay/aipay@latest uninstall         卸载 AI付 技能

  ${c.bold('说明：')}
    安装京东 AI付 一站式接入技能（jd-aipay-onboarding）到
    ~/.claude/skills/ 目录，使 Claude Code 能够引导你完成
    AI付 服务端集成的全流程：

    • 项目检测 → 代码生成（Java / Node.js / Python）
    • 沙箱联调 → 生产配置替换 → 上线验证
    • 产品开通指引 & 常见问题解答

  ${c.bold('更多信息：')}
    https://github.com/jd-opensource/jd-aipay-skill
`);
}

// ── 主入口 ────────────────────────────────────────────
const args = process.argv.slice(2);
const command = args[0];

switch (command) {
  case 'install':
    install(args.includes('--force'));
    break;
  case 'uninstall':
  case 'remove':
    uninstall();
    break;
  case '--help':
  case '-h':
  case 'help':
    showHelp();
    break;
  default:
    showHelp();
    break;
}
