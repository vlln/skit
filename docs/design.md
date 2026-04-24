# skit 设计蓝图：兼容优先的 Skill 管理工具

> **版本**: 1.1  
> **更新**: 2026-04-24  
> **状态**: 草案

---

## 0. 项目目标与定位

**skit** 是一个用 **Go** 编写的高性能 Skill 管理 CLI 工具，定位为 **“Skill 生态的 go-style 工具链”**：启动快、行为可预测、锁文件可复现，并为后续跨 Agent 安装一致性打基础。

**使命**：
- 完全兼容 `agentskills.io` 定义的 `SKILL.md` 格式；
- 兼容当前 `skills` CLI 的常见 source syntax 和 project/global scope；Agent target 与 copy/symlink 同步作为后续能力；
- 参考 `clawhub` 的 registry、版本、元数据、安全扫描与 HTTP API 设计；
- 在兼容基础上逐步引入更强的 lockfile、source provider、系统依赖诊断和 `metadata.skit` 扩展协议；
- 长期成为 Agent 世界中 Skills 的标准运行时与分发中枢。

### 0.1 v1 非目标

v1 不追求一次性实现完整包管理理论。以下能力暂缓：

- 完整 MVS 依赖解析；
- 强制要求所有 Skill 提供独立 manifest 文件；
- 自动执行系统包管理器安装依赖；
- 自建中心化 registry；
- 管理 Agent target 目录、copy/symlink 同步状态；
- 替代 `agentskills.io` 标准。

---

## 1. 为什么选择 Go？

决策依据：**项目特性与 Go 的最佳区间完美重合**。

| 维度 | Go 的优势 | 对本项目的具体意义 |
|:---|:---|:---|
| **编译速度** | 秒级编译 | 生态初期需频繁迭代 Schema 和命令 |
| **交叉编译** | `GOOS/GOARCH` 一行搞定 | 需分发 macOS/Linux/Windows 三平台二进制 |
| **并发模型** | goroutine + channel | 并发下载多个 Git 仓库、并行解析元数据是刚需 |
| **标准库** | `net/http`, `os/exec`, `archive/zip` 完备 | 无需第三方即可处理 API、Git 调用、打包 |
| **工具链一致性** | `gofmt` 强制风格统一 | 降低未来社区贡献门槛 |
| **包管理哲学** | lockfile、可复现解析、显式来源 | 可借鉴 Go modules 的稳定性原则，但不机械套用 MVS |


---

## 2. Go 的包管理哲学及其对 skit 的启示

Go modules 的原则与 Skill 包管理面临的挑战相似，但不能机械套用。Skill 生态当前更接近“多来源文件包 + Agent 运行时指令”，不是传统代码库依赖图。

### 2.1 区分 Source 身份与 Runtime Skill 身份

`owner/repo` 不能作为 Skill 的唯一规范名，因为一个仓库可能包含多个 Skill，来源也不只 GitHub。

**skit 设计**：
- **Source identity** 表示文件从哪里来，例如 `github:vercel-labs/agent-skills#main:skills/frontend-design`。
- **Runtime skill identity** 来自 `SKILL.md` frontmatter 的 `name` 字段，例如 `frontend-design`。
- **Registry slug** 是注册中心公开标识，例如 ClawHub 的 `gifgrep`。
- lockfile 必须同时记录 source identity、runtime skill identity 和 registry slug（如存在）。

### 2.2 最小版本选择（MVS）——作为 v2 目标

Go 的 MVS 选择“满足所有约束的最小版本”，而非“最新版本”，以减少意外的不兼容变更。

**skit 设计**：
- v1 不默认实现 MVS。原因是 GitHub/GitLab/local source 没有统一版本索引，许多 Skill 只有 branch/tag/commit。
- v1 采用“精确 ref + lock hash”：安装时解析 ref，lockfile 固化 commit/hash。
- MVS 放到 v2，前提是 registry 能提供稳定版本列表、semver 约束和依赖元数据。

### 2.3 不执行源码，纯哈希校验——安装≠运行

Go 拉取依赖后只做完整性校验，不执行任何代码。

**skit 设计**：
- `skit install` 仅拉取文件并校验，绝不自动执行 Skill 中的任何脚本。
- 安全保证借鉴 `go.sum`：全量锁定递归依赖树的每个成员的提交哈希与内容校验和。

### 2.4 go.sum 全量锁定——lockfile 必须锁定整个依赖树

Go 的 `go.sum` 记录所有直接和间接依赖的哈希，任何篡改都会被检测到。

**skit 设计**：
- `.agent/skills/skit.lock` 记录完整递归依赖树中每个 Skill 的精确版本、提交哈希和校验和。
- 格式采用 JSON，便于 CLI 稳定读写、测试和跨语言工具消费。

### 2.5 go mod tidy 自动清理——`skit doctor` 诊断完整性

Go 提供 `go mod tidy` 清理无用依赖。

**skit 设计**：
- 提供 `skit doctor` 命令，检查系统依赖、lockfile 一致性、循环依赖等。
- 提供 `skit tidy` 清理未使用的 Skill 或失效锁条目。

---

## 3. 架构：标准兼容下的元数据分层

### 3.1 设计原则

**保持 `SKILL.md` 标准兼容，同时把包管理、依赖和诊断元数据逐步分层**。`name` 和 `description` 必须留在 `SKILL.md` 中；更重的包管理信息优先放入 `metadata.skit`。

| 文件 | 职责 | 读者 | 加载时机 |
| :--- | :--- | :--- | :--- |
| `SKILL.md` 中的 `metadata.skit` | skit 扩展协议：版本、依赖、系统依赖、平台、关键字等 | skit CLI、注册中心 | `install` / `publish` / `search` |
| `skill.yaml` | 可选未来标准 manifest，内容协议等同于 `metadata.skit` | skit CLI、注册中心 | `install` / `publish` / `search` |
| `SKILL.md` | 运行时指令：任务目标、步骤、规范、约束 | Agent（模型） | Agent 激活 Skill 时注入上下文 |
| `.agent/skills/skit.lock` | 递归依赖树的精确版本锁定 | skit CLI | `install` / `update` 时自动生成 |

### 3.2 `metadata.skit` 规范 Schema

v1 不强制引入独立 manifest 文件。默认读取 `SKILL.md` frontmatter 和生态已有扩展字段；如存在 `skill.yaml`，其内容协议必须与 `metadata.skit` 完全一致。

推荐优先级：

1. `SKILL.md` 标准字段：`name`、`description`、`license`、`compatibility`、`metadata`、`allowed-tools`。
2. `metadata.skit`：skit 专属扩展。
3. `metadata.clawdbot` / `metadata.clawdis` / `metadata.openclaw`：兼容 ClawHub/OpenClaw 生态；这些块是增量运行时元数据，不是完整 YAML 头副本。
4. `skill.yaml`：可选同构 manifest，仅作为文件形式的 `metadata.skit`。

`metadata.skit` 示例：

```yaml
---
name: pdf-tools
description: Extract, merge, compress, and inspect PDF files.
metadata:
  skit:
    dependencies:
      - source: github:example/pdf-core
        ref: v1.2.0
        skill: pdf-core
    requires:
      bins:
        - pdftotext
        - qpdf
      env:
        - PDF_API_KEY
---
```

可选 `skill.yaml` 示例：

```yaml
schema: skit.skill/v1
version: 1.2.0
dependencies:
  - source: github:example/pdf-core
    ref: v1.2.0
    skill: pdf-core
requires:
  bins:
    - pdftotext
    - qpdf
  env:
    - PDF_API_KEY
platforms:
  os:
    - linux
    - darwin
keywords:
  - pdf
  - document
```

规则：

- `SKILL.md` 是标准文件名；为兼容 ClawHub/OpenClaw 生态，v1 可接受小写 `skill.md`，但必须给 warning；若两者同时存在，优先使用 `SKILL.md` 并 warning。
- `SKILL.md` 的 `name` 和 `description` 仍是必需标准字段，不在外部 manifest 中重复定义。
- `license`、`compatibility`、`allowed-tools` 等官方标准字段仍留在 `SKILL.md`；`skill.yaml` 不重复定义。
- `schema: skit.skill/v1` 是可选 manifest 格式标记，不属于 `metadata.skit` 业务字段。
- 外部 manifest 的顶层字段协议等同于 `metadata.skit`，不能另起一套 schema。
- `metadata.skit` 与外部 manifest 在 v1 中互斥；若两者同时存在，默认报错，不进行合并。
- v1 可读取 `skill.yaml`，但不默认生成它；`metadata.skit` 是 v1 的实际承载位置。

### 3.3 推荐的 `SKILL.md`

至少保留 Agent 所需的标准元数据（`name`、`description`）与全部指令。是否保留 `license`、`compatibility`、`metadata`、`allowed-tools` 取决于目标生态兼容性，不应由 `skit migrate` 默认删除。

```yaml
---
name: advanced-pdf-workflow
description: Complete PDF pipeline for extraction, watermarking, and compression.
---
# Step-by-step instructions...
```

### 3.4 元数据解析优先级（兼容性保障）

skit 读取一个 Skill 时：

1. **始终解析 `SKILL.md` YAML 头**。这是官方标准和 Agent 运行时入口。
2. **读取 `metadata.skit`**。这是 skit 的首选扩展位置。
3. **读取 `metadata.clawdbot` / `metadata.clawdis` / `metadata.openclaw`**。兼容 ClawHub/OpenClaw 的运行需求声明；不读取并特殊处理 `metadata.clawhub`，除非未来上游定义该命名空间。
4. **读取可选 `skill.yaml`**。仅当 `metadata.skit` 不存在时读取；两者同时存在时报错，不进行合并；不得替代 `SKILL.md` 的标准字段。
5. **`skit migrate` 默认不破坏兼容性**。默认只生成 `metadata.skit`；只有用户显式传未来的 manifest 选项时才生成 `skill.yaml`，只有传 `--minimal-frontmatter` 时才精简非标准字段。

此设计最大化兼容现有生态，同时为新式 Skill 提供更清晰的包管理扩展路径。

---

## 4. 依赖解析与系统依赖处理

### 4.1 挑战：非注册 Skill 的依赖可能未声明

许多现有 Skill 没有 `metadata.skit` 或 `dependencies` 字段，导致依赖树断裂。

**应对策略**：
- **分层降级**：`metadata.skit` / 外部同构 manifest / 兼容运行时元数据 → 视为无 skit 依赖。
- **隐式探测**：检查 `SKILL.md` 文本中是否引用其他 Skill 的名录，给出“可能缺失的依赖”警告（不自动解析）。
- **严格安装、宽松诊断**：显式依赖安装失败时默认阻断；隐式探测只给 warning，不自动解析。

v1 修订：

- 显式依赖必须声明 `source`、可选 `ref`、可选 `skill`。
- ref 缺省时使用 source provider 默认分支或 registry latest，但 lockfile 必须固化 resolved ref。
- 依赖安装失败时默认阻断，除非用户传 `--ignore-deps`。
- 循环依赖必须检测并报错。
- 同一 `source + skill + ref` 的重复依赖可去重；同一 `source + skill` 但 `ref` 不同的依赖在 v1 视为冲突。
- 复杂 semver constraint 和 MVS 放到 v2。

### 4.2 挑战：系统依赖（CLI 工具）的自动安装

不同操作系统的包管理器完全不同（brew, apt, choco…），且可能无安装权限。

**应对策略**：

1. **声明即需求**：只声明规范名、二进制、环境变量、配置文件和平台限制，不默认包含安装逻辑。
2. **兼容生态字段**：规范依赖与系统需求优先读取 `metadata.skit` 和外部同构 manifest；ClawHub/OpenClaw 兼容字段按 `metadata.clawdbot`、`metadata.clawdis`、`metadata.openclaw`、顶层 `clawdis`、顶层 fallback 字段顺序读取，用于诊断和展示，不执行其 `install` / `nix` / `config` 安装逻辑。
3. **手动模式默认**：`skit doctor` 打印清晰的安装建议，例如 `brew install ffmpeg`、`apt install ffmpeg`。
4. **交互修复可选**：后续可提供 `skit doctor --fix`，执行前必须明确确认。
5. **内置二进制谨慎引入**：对部分工具可拉取预编译二进制到 `~/.skit/bin`，但必须校验来源 hash 和签名。

---

## 5. 分发与“源”抽象设计

### 5.1 现状：skills.sh 并非“源”，而是搜索引擎

skills.sh 不托管 Skill 文件，只维护搜索索引，底层安装直接 `git clone` GitHub，故无“换源”概念。

### 5.2 skit 引入“源”的必要性

为解锁企业私有源、镜像加速、离线自托管等场景，skit 必须抽象出“源”层。

**源的定义**：一个可解析、列出、获取、校验 Skill 的 provider。HTTP registry 只是 source provider 的一种，不应把所有来源都抽象成 HTTP。

v1 MVP provider 类型：

- `local`：本地目录；
- `github`：GitHub shorthand、URL、tree subpath；
- `git`：任意 git URL；
- `gitlab`：GitLab URL、tree subpath。

后置可选 provider：

- `registry`：HTTP registry，例如 ClawHub 风格 API；
- `well-known`：读取 `.well-known` 配置发现 registry。

v1 开工范围以 `local + github + gitlab + generic git + import-lock` 为准。registry / well-known 能力可以在 M6 之后加入，但不能阻塞 git/local 闭环，也不能要求 v1 初始实现 source 配置命令。

内部接口草案：

```go
type Provider interface {
    Parse(input string) (Source, error)
    ListSkills(ctx context.Context, source Source) ([]SkillRef, error)
    Fetch(ctx context.Context, ref SkillRef) (FetchedSkill, error)
    Resolve(ctx context.Context, ref SkillRef) (ResolvedSkill, error)
}
```

registry provider 额外支持：

- search；
- versions；
- metadata；
- download；
- file fetch；
- moderation/security status。

### 5.3 源配置（后置能力，`~/.config/skit/config.yaml`）

```yaml
schema: skit.config/v1
sources:
  - name: clawhub
    type: registry
    url: https://clawhub.ai
    priority: 10
  - name: github
    type: github
    priority: 20
cache:
  ttl: 24h
```

配置位置：

- `$XDG_CONFIG_HOME/skit/config.yaml`
- 缺省：`~/.config/skit/config.yaml`

### 5.4 源的命令集成（后置能力）

- `skit source list`：列出所有配置的源及优先级。
- `skit source add <name> <url> [--type]`：添加新源。
- `skit install` / `search`：按优先级顺序查询各源并合并结果。

**初期默认源**：内置 GitHub/local provider。GitLab/generic git、ClawHub/skills.sh 风格 registry 可作为后置能力配置，但不要把 `skills.sh` 硬编码为唯一源。

---

## 5.5 Lockfile 设计

lockfile 的目标是让安装可复现、可审计、可更新，而不是在 v1 实现完整依赖求解。

现有生态已有 lock 机制：

- `skills` 项目 lock：`skills-lock.json`，字段包含 `source`、`ref`、`sourceType`、`computedHash`；无时间戳，按 skill name 排序，适合提交到 VCS。
- `skills` 全局 lock：`$XDG_STATE_HOME/skills/.skill-lock.json` 或 `~/.agents/.skill-lock.json`，字段包含 `source`、`sourceType`、`sourceUrl`、`ref`、`skillPath`、`skillFolderHash`、安装/更新时间和交互偏好。
- `clawhub` 项目 lock：`.clawhub/lock.json`，字段很小：registry slug -> `{ version, installedAt }`。
- `clawhub` skill origin：`<skill>/.clawhub/origin.json`，记录 `registry`、`slug`、`installedVersion`、`installedAt`。

skit 不应直接复用其中任一格式作为唯一格式。原因：

- `skills-lock.json` 对 git/local/node_modules 友好，但不能表达 registry slug、targets、resolved commit 等完整信息。
- `.clawhub/lock.json` 对 registry 安装足够，但不能表达任意 git source 和 subpath。
- 全局 lock 含本机时间戳和交互状态，不适合作为可提交 lock。

### 5.5.1 skit lock 分层

skit v1 采用内容锁文件；本机安装状态推迟到后续版本：

- **项目 lock**：`.agent/skills/skit.lock`，与项目 active root 放在一起，不写入本机时间戳，字段排序稳定，目标是可复现和低冲突。
- **全局 lock**：`~/.agent/skills/skit.lock`，schema 与项目 lock 相同，用于记录全局激活的 Skill。
- **本机安装 state**：v1 不单独引入。active 目录中的 Skill 均为指向全局 store 的软链接。

### 5.5.2 项目 lock schema 草案

项目 lock 不记录 `installedAt` / `updatedAt` / Agent targets，避免无意义 merge conflict 和本机状态污染。

```json
{
  "schema": "skit.lock/v1",
  "skills": {
    "frontend-design": {
      "name": "frontend-design",
      "description": "Design frontend interfaces...",
      "source": {
        "type": "github",
        "locator": "vercel-labs/agent-skills",
        "url": "https://github.com/vercel-labs/agent-skills",
        "ref": "main",
        "resolvedRef": "4f2c...",
        "subpath": "skills/frontend-design",
        "skill": "frontend-design"
      },
      "registry": {
        "name": "clawhub",
        "url": "https://clawhub.ai",
        "slug": "frontend-design",
        "version": "1.2.3",
        "digest": "sha256-..."
      },
      "download": {
        "url": "https://clawhub.ai/api/v1/skills/frontend-design/versions/1.2.3/download",
        "sha256": "sha256-..."
      },
      "hashes": {
        "tree": "sha256-...",
        "skillMd": "sha256-..."
      },
      "dependencies": [],
      "incomplete": false,
      "warnings": []
    }
  }
}
```

字段说明：

- `source.skill` 记录最终选中的 runtime Skill name；它替代输入中的模糊 `@skill` shortcut。
- `registry` 仅在 registry/import 来源存在时写入；local/GitHub 安装可以省略。
- `download` 仅在安装来源有下载包时写入；local/GitHub 安装可以省略。
- `registry.digest` 表示 registry 版本或包元数据的稳定摘要；`download.sha256` 表示下载包字节校验。
- `dependencies` 记录已解析的依赖边；被依赖 Skill 同时拥有自己的顶层 lock entry。
- `incomplete: true` 仅用于兼容导入时无法形成可复现 restore 的条目；`skit install` 不自动恢复这类条目，而是提示用户重新 add 或 inspect。
- `warnings` 记录小写 `skill.md`、兼容字段降级、导入信息丢失等非致命诊断。

### 5.5.3 后续本机 state schema 草案

以下 schema 不属于 v1 实现范围，仅作为后续 Agent target 同步设计草案。

```json
{
  "schema": "skit.state/v1",
  "skills": {
    "frontend-design": {
      "installedAt": "2026-04-24T00:00:00Z",
      "updatedAt": "2026-04-24T00:00:00Z",
      "targets": [
        {
          "agent": "codex",
          "scope": "global",
          "path": "~/.codex/skills/frontend-design",
          "mode": "symlink"
        }
      ]
    }
  },
  "lastSelectedAgents": ["codex"]
}
```

### 5.5.4 兼容策略

skit 应提供读取兼容，不默认双写其他工具的 lock：

- 若当前目录无 `.agent/skills/skit.lock`，但存在 `skills-lock.json`，`skit install` 可读取并恢复。
- 若当前目录无 `.agent/skills/skit.lock`，但存在 `.clawhub/lock.json`，`skit install` 可按 registry slug/version 恢复。
- `skit import-lock skills` 将 `skills-lock.json` 转为 `.agent/skills/skit.lock`。
- `skit import-lock clawhub` 将 `.clawhub/lock.json` 和 `.clawhub/origin.json` 转为 `.agent/skills/skit.lock`。
- `skit export-lock skills` / `skit export-lock clawhub` 可作为后续显式兼容命令，但普通安装不默认写入其他工具的 lock，避免多工具互相覆盖。
- 兼容导入若无法还原 source/hash，必须生成 `incomplete: true` 条目并报告丢失字段。

v1 至少记录：

- `SKILL.md` 内容 hash；
- Skill 目录 tree hash；
- git commit hash 或 registry version digest；
- 下载包 sha256（仅下载包来源）。

hash 只证明内容未变，不证明内容安全。

---

## 5.6 安全模型

`skit install` 必须遵守：

- 不执行 Skill 内脚本；
- 不执行 postinstall/preinstall；
- 不自动运行包管理器；
- 默认拒绝路径穿越；
- 默认拒绝写入 skit store 之外；
- v1 默认拒绝 symlink 和其他非普通文件，不跟随链接；
- 本地路径安装必须明确标记为 local/untrusted。

registry publish/download 应考虑：

- 文本文件 allowlist；
- bundle size limit；
- 单文件 size limit；
- zip slip 防护；
- symlink 拒绝策略；
- 可选忽略文件：`.skitignore`、`.gitignore`；
- 明确是否允许二进制 assets。

skit 不应声称 hash 等于安全。可提供：

- `skit inspect`：展示文件、hash、来源、需求声明；
- `skit doctor`：检查声明与本地环境；
- registry security status 展示；
- 对明显危险内容给 warning，例如混淆 shell、`curl | sh`、base64 decode execute。

---

## 6. 关键命令清单

v1 推荐命令语义：

| 命令 | 功能 |
|:---|:---|
| `skit install [source...]` | 从 source 安装 Skill；无 source 时根据 lockfile 恢复 active symlink |
| `skit list` / `skit ls` | 列出已安装 Skill |
| `skit remove <name>` / `skit rm` | 删除已安装 Skill |
| `skit update [name]` | 更新 Skill 并刷新 lockfile |
| `skit init [name]` | 创建 `SKILL.md` 模板 |
| `skit sync` | 后续版本：同步已安装 Skill 到多个 Agent 目标目录 |
| `skit doctor` | 检查 lockfile、系统依赖、环境变量 |
| `skit inspect <source-or-name>` | 展示来源、hash、metadata、风险提示 |
| `skit search <query>` / `skit find` | 后置版本：搜索 registry 或本地缓存 |
| `skit source <add/list/remove>` | 后置版本：管理 registry/source 配置 |

- v1 优先兼容 `--global`、`--project`、`--skill`、`--yes`、`--all`。
- `--skill` 只允许出现一次，但可跟多个用空格分隔的 skill name；只适用于单个 source。多个 source 使用 `owner/repo@skill` 这类 inline selector。
- `--agent`、`--copy` 与 `skit sync` 归入后续 Agent target 同步能力。
- `--global` 与 `--project` 互斥；`--all` 与 `--skill` 互斥；`--yes` 只跳过确认，不隐含选择全部 Skill。

---

## 6.1 安装目录模型

skit v1 维护全局 canonical store、lockfile 和 active symlink。默认 project active root 为 `.agent/skills`，global active root 为 `~/.agent/skills`。

Canonical store：

- project 不单独维护 store
- global store：`$XDG_DATA_HOME/skit/store/`，缺省 `~/.local/share/skit/store/`
- v1 store layout 固定为 `<store>/<hashes.tree>/<skill-name>/`，其中 `hashes.tree` 已包含 `sha256-` 前缀。
- 安装写入必须先进入临时目录，完成校验与 hash 后再原子移动到 content-addressed store；失败安装不得留下半成品最终目录。
- `skit install` 默认在 active root 创建指向 store snapshot 的软链接；`--no-active` 可仅写 store/lock。
- `skit remove` 默认移除 lock entry 和 active symlink；只有能确认 store 内容未被任何 lock 引用时才可清理目录。更完整的清理留给后续 `skit store prune`。

后续 Agent target 是实际被 Agent 发现的目录，例如：

- Codex global：`~/.codex/skills`
- Codex project：`.agents/skills`
- Claude Code global：`~/.claude/skills`
- Claude Code project：`.claude/skills`

后续安装模式：

- `symlink`：默认推荐，target 指向 canonical store；
- `copy`：兼容不支持 symlink 的平台或团队希望提交完整文件。

---

## 7. 兼容性承诺与生态推动

- **分发渠道优先兼容**：v1 先兼容 `skills` CLI 已支持的 GitHub/local/GitLab/generic git source；不把任何单一服务硬编码为唯一渠道。
- **格式无损扩展**：`SKILL.md` 的 YAML 头仍保留官方字段，现有工具不受影响。
- **迁移工具降低切换成本**：`skit migrate` 默认只增加扩展元数据，不默认删除现有 frontmatter。
- **主动参与标准演进**：当 `metadata.skit` / 外部同构 manifest 被实践验证后，再向 agentskills.io 社区提交 RFC。

---

## 8. 扩展蓝图：从 CLI 到 Skill Runtime

未来 skit 可演进为所有 Agent 统一的 **Skill Runtime**：
任何 Agent 都可以调用 skit 作为 Skill 的下载、管理和使用工具。

建议路线：

- v1：兼容、快速、可复现。实现 Go 单二进制、主流 source syntax、project/global store、lockfile、local/GitHub/GitLab/generic git provider、doctor/inspect；registry search/download 作为 M6 后可选增强，不阻塞 v1 MVP。
- v2：增强 manifest 与依赖。稳定 `metadata.skit` 协议、显式 Skill 依赖、registry version index、semver constraint、可选 MVS。
- v3：Runtime 与生态标准。企业私有源、离线镜像、安全扫描集成、自动化系统依赖安装，并向 `agentskills.io` 提交 RFC。
