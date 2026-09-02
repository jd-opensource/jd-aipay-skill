#!/usr/bin/env node

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');

// ── 常量 ──────────────────────────────────────────────
const SKILL_NAME = 'jd-aipay-onboarding';

// npm 包根目录（bin/cli.js 的上级）
const PKG_ROOT = path.resolve(__dirname, '..');

// 需要拷贝到 skill 目录的内容
const ENTRIES = ['SKILL.md', 'playbooks', 'reference', 'assets', 'scripts'];

// ── 平台配置 ─────────────────────────────────────────
const PLATFORMS = {
  claude: {
    name: 'Claude Code',
    skillDir: path.join(os.homedir(), '.claude', 'skills', SKILL_NAME),
    install(force) {
      return installSkillFiles(this.skillDir, force);
    },
    uninstall() {
      return uninstallSkillDir(this.skillDir, this.name);
    },
  },
  codex: {
    name: 'Codex',
    skillDir: path.join(os.homedir(), '.agents', 'skills', SKILL_NAME),
    install(force) {
      return installSkillFiles(this.skillDir, force);
    },
    uninstall() {
      return uninstallSkillDir(this.skillDir, this.name);
    },
  },
  cursor: {
    name: 'Cursor',
    skillDir: path.join(os.homedir(), '.cursor', 'skills', SKILL_NAME),
    ruleFile: path.join(os.homedir(), '.cursor', 'rules', `${SKILL_NAME}.mdc`),
    install(force) {
      // 1) 安装完整 Skill 资料到 ~/.cursor/skills/jd-aipay-onboarding/
      const result = installSkillFiles(this.skillDir, force);
      if (!result) return false;

      // 2) 生成 .mdc 规则文件到 ~/.cursor/rules/
      const rulesDir = path.dirname(this.ruleFile);
      fs.mkdirSync(rulesDir, { recursive: true });

      const mdcContent = generateCursorRule(this.skillDir);
      fs.writeFileSync(this.ruleFile, mdcContent, 'utf8');
      console.log(c.green(`  ✓  ${path.basename(this.ruleFile)}`) + c.dim(' (Cursor 规则文件)'));

      return true;
    },
    uninstall() {
      // 删除规则文件
      if (fs.existsSync(this.ruleFile)) {
        fs.unlinkSync(this.ruleFile);
        console.log(c.green(`  ✓  已删除 Cursor 规则文件：${this.ruleFile}`));
      }
      // 删除 Skill 资料目录
      return uninstallSkillDir(this.skillDir, this.name);
    },
  },
};

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
  if (fs.rmSync) {
    fs.rmSync(dir, { recursive: true, force: true });
  } else {
    fs.rmdirSync(dir, { recursive: true });
  }
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

// ── Cursor .mdc 规则文件生成 ─────────────────────────
function generateCursorRule(skillDir) {
  return `---
description: "京东 AI付一站式接入 Agent。当用户提到 接入AI付、集成AI付、联调、上线AI付、jdpay-aipay、AI付快速接入、一站式接入、沙箱联调、生产配置替换、上线验证、产品开通、证书密钥、createOrder、queryPayResult、refund、queryRefundResult，或咨询 AI付/订阅 接入问题时触发。"
alwaysApply: false
---

# 京东 AI付一站式接入 Agent

## 定位

让接入这件事交给 AI 来做。你要引导并执行京东 AI付服务端代码集成、沙箱联调、生产配置替换与上线验证，同时提供产品开通、实名认证、证书密钥获取等平台侧操作指引。

## Skill 资料目录

完整的 playbooks、API 参考文档、示例工程和脚本位于：

\`${skillDir}\`

请从该目录读取所有接入编排和参考资料。

## 能力边界

- Java、Node.js、Python 服务端真实项目接入
- 沙箱联调和接口排障
- 代码集成完成后的产品开通并行提醒
- 实名认证、产品开通、证书密钥获取、上线验证的平台侧操作指引
- 生产配置替换和上线前检查
- 接入过程中的产品、流程、接口、签名、加密、状态码、错误码答疑

客户端 SDK 接入需以京东 AI付商务或技术支持提供的 SDK 包和正式方案为准。

## 默认流程

用户提出"帮我接入 AI付"或类似需求时：

1. 先阅读 \`${skillDir}/playbooks/00-response-style.md\`，保持对外商户友好的接入顾问口径。
2. 检测当前工作区是否为服务端项目，并识别 Java / Node.js / Python 语言与框架。
3. 如果识别到服务端项目，阅读 \`${skillDir}/playbooks/02-detect-server-project.md\` 和对应语言 playbook，在现有项目内完成代码集成。
4. 代码集成完成后，阅读 \`${skillDir}/playbooks/06-product-opening-parallel-reminder.md\`，提醒产品开通。
5. 执行沙箱联调（\`${skillDir}/playbooks/07-sandbox-joint-debug.md\`）。
6. 引导证书密钥准备（\`${skillDir}/playbooks/08-certificate-secret-guide.md\`）。
7. 生产配置替换（\`${skillDir}/playbooks/09-production-cutover.md\`）。
8. 上线验证（\`${skillDir}/playbooks/10-go-live-verification.md\`）。

## 知识路由

- 产品认知 QA：\`${skillDir}/reference/product/product-and-onboarding-qa.md\`
- API 协议：\`${skillDir}/reference/api/\`
- 示例工程：\`${skillDir}/assets/server-examples/\`

## 平台侧边界

- 不登录官网、企业站或商户后台
- 不直接查询商户私有信息
- 不在回复中展示完整密钥、证书、手机号、银行卡号等敏感信息

## 常用官网入口

- AI付官网：https://aipay.jdpay.com/
- 快速接入页：https://aipay.jdpay.com/developers
- 一站式接入：https://aipay.jdpay.com/onboarding?product=AI_PAY
- 技术支持：jdpay-bd@jd.com / 400-098-8500
`;
}

// ── 通用安装逻辑 ─────────────────────────────────────
function installSkillFiles(targetDir, force) {
  const parentDir = path.dirname(targetDir);
  fs.mkdirSync(parentDir, { recursive: true });

  if (fs.existsSync(targetDir)) {
    if (!force) {
      console.log(c.yellow(`  ⚠  目录已存在：${targetDir}`));
      console.log(c.yellow('     使用 --force 参数覆盖安装'));
      return false;
    }
    console.log(c.yellow('  ♻  检测到已有安装，覆盖中...'));
    removeRecursive(targetDir);
  }

  fs.mkdirSync(targetDir, { recursive: true });
  let fileCount = 0;
  for (const entry of ENTRIES) {
    const src = path.join(PKG_ROOT, entry);
    const dest = path.join(targetDir, entry);
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

  console.log(c.dim(`     共 ${fileCount} 个文件 → ${targetDir}`));
  return true;
}

// ── 通用卸载逻辑 ─────────────────────────────────────
function uninstallSkillDir(targetDir, platformName) {
  if (!fs.existsSync(targetDir)) {
    console.log(c.yellow(`  ⚠  ${platformName} 技能未安装，无需卸载`));
    return false;
  }
  removeRecursive(targetDir);
  console.log(c.green(`  ✓  已删除 ${platformName} 技能目录：${targetDir}`));
  return true;
}

// ── 解析目标平台 ─────────────────────────────────────
function resolveTargets(args) {
  const hasAll = args.includes('--all');
  const hasClaude = args.includes('--claude');
  const hasCodex = args.includes('--codex');
  const hasCursor = args.includes('--cursor');

  if (hasAll) {
    return ['claude', 'codex', 'cursor'];
  }

  const targets = [];
  if (hasClaude) targets.push('claude');
  if (hasCodex) targets.push('codex');
  if (hasCursor) targets.push('cursor');

  // 默认只安装 Claude Code
  if (targets.length === 0) {
    targets.push('claude');
  }

  return targets;
}

// ── install ───────────────────────────────────────────
function install(args) {
  const force = args.includes('--force');
  const targets = resolveTargets(args);

  console.log();
  console.log(c.bold('  ⚡ 京东 AI付 · Skill 安装器'));
  console.log(c.dim('  ─────────────────────────────────────────'));
  console.log();
  console.log(c.dim(`  目标平台：${targets.map((t) => PLATFORMS[t].name).join('、')}`));
  console.log();

  let success = 0;
  let fail = 0;

  for (const key of targets) {
    const platform = PLATFORMS[key];
    console.log(c.bold(`  ── ${platform.name} ──`));
    console.log();
    const ok = platform.install(force);
    if (ok) {
      success++;
    } else {
      fail++;
    }
    console.log();
  }

  // 汇总
  if (success > 0) {
    console.log(c.green(c.bold(`  ✅ 安装完成！`)) + c.dim(` (成功 ${success} 个平台${fail > 0 ? `，跳过 ${fail} 个` : ''})`));
    console.log();
    console.log(c.bold('  📖 使用方式：'));

    if (targets.includes('claude')) {
      console.log('     Claude Code — 在你的服务端项目中输入：');
      console.log(c.cyan('     「帮我接入 AI付」'));
    }
    if (targets.includes('codex')) {
      console.log('     Codex — 在你的服务端项目中输入：');
      console.log(c.cyan('     「帮我接入 AI付」'));
    }
    if (targets.includes('cursor')) {
      console.log('     Cursor — 在 Composer/Chat 中输入：');
      console.log(c.cyan('     「帮我接入 AI付」'));
    }

    console.log();
    console.log(c.dim('  卸载：npx -y @jdpay/aipay@latest remove' + (targets.length > 1 ? ' --all' : '')));
  } else {
    console.log(c.yellow('  ⚠  未成功安装任何平台，请检查上方提示'));
  }
  console.log();
}

// ── uninstall ─────────────────────────────────────────
function uninstall(args) {
  const targets = resolveTargets(args);

  console.log();
  console.log(c.bold('  ⚡ 京东 AI付 · Skill 卸载'));
  console.log(c.dim('  ─────────────────────────────────────────'));
  console.log();
  console.log(c.dim(`  目标平台：${targets.map((t) => PLATFORMS[t].name).join('、')}`));
  console.log();

  for (const key of targets) {
    const platform = PLATFORMS[key];
    console.log(c.bold(`  ── ${platform.name} ──`));
    platform.uninstall();
    console.log();
  }

  console.log(c.green('  ✅ 卸载完成'));
  console.log();
  console.log(c.dim('  重新安装：npx -y @jdpay/aipay@latest add'));
  console.log();
}

// ── help ──────────────────────────────────────────────
function showHelp() {
  console.log(`
  ${c.bold('⚡ @jdpay/aipay')} — 京东 AI付 Skill 安装器

  ${c.bold('用法：')}
    npx -y @jdpay/aipay@latest add                 安装到 Claude Code ${c.dim('(默认)')}
    npx -y @jdpay/aipay@latest add --codex          安装到 Codex
    npx -y @jdpay/aipay@latest add --cursor         安装到 Cursor
    npx -y @jdpay/aipay@latest add --all            安装到所有平台
    npx -y @jdpay/aipay@latest add --force          覆盖已有安装
    npx -y @jdpay/aipay@latest remove               卸载 ${c.dim('(可加 --codex / --cursor / --all)')}

  ${c.bold('支持的平台：')}
    Claude Code   安装到 ~/.claude/skills/
    Codex         安装到 ~/.agents/skills/
    Cursor        安装到 ~/.cursor/rules/ + ~/.cursor/skills/

  ${c.bold('说明：')}
    安装京东 AI付一站式接入技能（jd-aipay-onboarding），
    使 AI 编程工具能够引导你完成 AI付服务端集成的全流程：

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
  case 'add':
  case 'install':
    install(args.slice(1));
    break;
  case 'remove':
  case 'uninstall':
    uninstall(args.slice(1));
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
