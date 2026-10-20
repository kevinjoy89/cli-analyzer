# 用户规则包（rules.d）

内置规则表编译进二进制，新出现的 AI 工具只能等发版才能被正确归因。规则包让用户自己补条目，不必等版本。

## 放在哪

```
<数据目录>/rules.d/*.json
```

数据目录就是应用自己的 `DataRoot()`：

| 平台 | 路径 |
|---|---|
| macOS | `~/Library/Application Support/cli-analyzer/rules.d/` |
| Linux | `~/.local/share/cli-analyzer/rules.d/` |
| Windows | `%LOCALAPPDATA%\cli-analyzer\rules.d\` |

目录不存在是正常情况（多数用户不会放规则包）。文件按**文件名排序**加载，同一工具在多个文件里出现时后加载者生效。

## 文件格式

```json
{
  "schemaVersion": 1,
  "rules": [
    {
      "name": "mytool",
      "displayName": "My Tool",
      "aliases": ["mytool-cli"],
      "installer": "other",
      "homepage": "https://example.com",
      "description": "一句话说明",
      "category": "agent",
      "apps": ["MyTool"],
      "dataDirs": [
        {"root": "home", "sub": ".mytool", "tier": "user", "kind": "data"},
        {"root": "macos-application-support", "sub": "MyTool", "tier": "user", "kind": "data", "form": "desktop"}
      ],
      "cleanables": [
        {"root": "home", "sub": ".mytool/cache", "tier": "safe", "kind": "cache", "desc": "可再生成的缓存"}
      ]
    }
  ]
}
```

### 字段

| 字段 | 说明 |
|---|---|
| `root` | 数据根类型：`xdg-cache` / `xdg-data` / `xdg-config` / `xdg-state` / `home` / `macos-application-support` / `macos-caches` / `macos-preferences` / `appdata` / `localappdata`。当前平台不存在的根自动跳过 |
| `sub` | 相对该根的目录名；支持尾部 `*` 前缀匹配（`codex-runtime-install-*`） |
| `path` | 字面绝对路径，优先于 `root`+`sub`。**必须是绝对路径** |
| `tier` | `safe` / `user`——只是标签，不是删除门槛 |
| `kind` | 目录类型（`cache` / `config` / `data` / `old-version` / `install` / `session` / `index` …），影响展示与 `clean --all` 的默认批量范围 |
| `form` | `desktop` 表示桌面端形态；留空按 CLI 计 |
| `category` | `agent` / `ai-ide`；非空即参与 AI 编目（已卸载判定、形态小计） |
| `apps` | 桌面端宿主应用名（macOS `.app` 基名），用于「已卸载」判定与应用包测量 |

## 优先级与容错

- **同名条目整条替换内置条目**（不做字段级合并——合并会让「我明明写对了却还是老行为」变成猜谜）。想改内置条目时，把整条抄过来改。
- 新名字**追加**，与内置条目走完全相同的归因 / 清理 / 卸载管线。
- **单条不合法只跳过该条**，其余条目与内置表照常工作；整个文件坏掉也只影响该文件。扫描绝不因规则包失败。
- 加载问题会出现在 `scan` 的人类可读输出末尾，以及 `scan --json` 的 `ruleErrors` 字段里——「我写的规则为什么没生效」有据可查。
- `schemaVersion` 只接受 `1`（或省略）；其它值视为未来格式，整份文件跳过并报错，不猜着解析。

## 校验规则

加载时逐条检查：`name` 必填；`root` 必须是上表里的值；`path` 必须是绝对路径；`form` 只能是 `desktop`。`dataDirs` / `cleanables` 每条至少要有一个 `root` 或 `path`。

## 注意

规则包只影响**归因与标注**，不改变删除行为：每个归因目录仍然是「可处置项」，删不删由用户在 GUI/CLI 里决定，删除前照常经过完整性守卫（绝对路径、无 `..`、不在系统根、不是回收站根、不是当前版本路径）。
