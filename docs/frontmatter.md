# SKILL.md Frontmatter Schema

`SKILL.md` 文件以 YAML frontmatter 开头，后跟 Markdown 正文。所有字段均为可选。

## 字段参考

### name

| 属性 | 值 |
|------|-----|
| 类型 | `string` |
| 必需 | 否 |
| 来源 | Claude Code |

显示名称。默认使用目录名。

```yaml
name: pdf-processing
```

约束：

- 1–64 字符
- Unicode 小写字母、数字、连字符（`-`）
- 不能以 `-` 开头或结尾
- 不能包含连续 `--`
- 建议匹配目录名

### description

| 属性 | 值 |
|------|-----|
| 类型 | `string` |
| 必需 | 推荐 |
| 来源 | Claude Code |

描述技能的功能和使用场景。Agent 用它决定是否激活技能。若省略，使用 Markdown 正文的第一段。

```yaml
description: >
  Extract text and tables from PDF files, fill PDF forms, and
  merge multiple PDFs. Use when working with PDF documents or
  when the user mentions PDFs, forms, or document extraction.
```

约束：

- 推荐不超过 1024 字符
- 与 `when-to-use` 合并后截断于 1536 字符

### when-to-use

| 属性 | 值 |
|------|-----|
| 类型 | `string` |
| 必需 | 否 |
| 来源 | Claude Code |

额外的触发上下文。追加到 `description` 后，与 `description` 合并计入 1536 字符限制。

```yaml
when-to-use: >
  Also use when the user asks about document extraction, form
  filling, or mentions scanned documents.
```

### allowed-tools

| 属性 | 值 |
|------|-----|
| 类型 | `string` 或 `[]string` |
| 必需 | 否 |
| 来源 | Claude Code |

技能激活期间预批准的工具。接受空格分隔字符串、逗号分隔字符串或 YAML 列表。

```yaml
# YAML 列表（推荐）
allowed-tools:
  - Bash(git:*)
  - Bash(jq:*)
  - Read

# 或空格分隔字符串
allowed-tools: Bash(git:*) Bash(jq:*) Read
```

### disallowed-tools

| 属性 | 值 |
|------|-----|
| 类型 | `string` 或 `[]string` |
| 必需 | 否 |
| 来源 | Claude Code |

技能激活期间从可用池中移除的工具。限制在用户发送下一条消息时清除。

```yaml
disallowed-tools:
  - AskUserQuestion
```

### argument-hint

| 属性 | 值 |
|------|-----|
| 类型 | `string` |
| 必需 | 否 |
| 来源 | Claude Code |

自动补全提示，指示预期参数。

```yaml
argument-hint: <file> [format]
```

### arguments

| 属性 | 值 |
|------|-----|
| 类型 | `string` 或 `[]string` |
| 必需 | 否 |
| 来源 | Claude Code |

命名位置参数，用于技能内容中的 `$name` 替换。名称按位置映射到参数顺序。

```yaml
arguments:
  - file
  - format
```

### disable-model-invocation

| 属性 | 值 |
|------|-----|
| 类型 | `boolean` |
| 必需 | 否 |
| 来源 | Claude Code |

设为 `true` 阻止 Agent 自动加载此技能。仅通过手动 `/name` 触发。同时阻止技能预加载到 subagent。

```yaml
disable-model-invocation: false
```

### user-invocable

| 属性 | 值 |
|------|-----|
| 类型 | `boolean` |
| 必需 | 否 |
| 来源 | Claude Code |

设为 `false` 从 `/` 菜单中隐藏。用于用户不应直接调用的后台知识技能。

```yaml
user-invocable: true
```

### license

| 属性 | 值 |
|------|-----|
| 类型 | `string` |
| 必需 | 否 |
| 来源 | Agent Skills Spec |

许可证名称或对捆绑许可证文件的引用。

```yaml
license: MIT
```

### metadata

| 属性 | 值 |
|------|-----|
| 类型 | `map[string]string` |
| 必需 | 否 |
| 来源 | Agent Skills Spec |

附加注解。键和值均为字符串。常用键：

| 键 | 含义 |
|----|------|
| `author` | 技能作者 |
| `version` | 技能版本号（语义化版本） |

```yaml
metadata:
  author: example-org
  version: "1.0"
```

---

## skit 扩展

### requires

| 属性 | 值 |
|------|-----|
| 类型 | `map` |
| 必需 | 否 |
| 来源 | skit |

结构化运行时需求和约束。用于自动化诊断和依赖检查。

```yaml
requires:
  bins:
    - skit
    - git
  any-bins:
    - pdftotext
    - mutool
  env:
    - PDF_API_KEY
  config:
    - ~/.config/pdf-tools
  skills:
    - github:owner/repo@required-skill
  platforms:
    os:
      - linux
      - darwin
    arch:
      - amd64
      - arm64
```

#### requires.bins

| 属性 | 值 |
|------|-----|
| 类型 | `[]string` |
| 必需 | 否 |

全部必须存在的 CLI 工具。

#### requires.any-bins

| 属性 | 值 |
|------|-----|
| 类型 | `[]string` |
| 必需 | 否 |

至少有一个存在即可的 CLI 工具。

#### requires.env

| 属性 | 值 |
|------|-----|
| 类型 | `[]string` |
| 必需 | 否 |

必须设置的环境变量。

#### requires.config

| 属性 | 值 |
|------|-----|
| 类型 | `[]string` |
| 必需 | 否 |

必须存在的配置文件路径。

#### requires.skills

| 属性 | 值 |
|------|-----|
| 类型 | `[]string` |
| 必需 | 否 |

依赖的其他技能，使用 skit 安装目标格式。

```
github:owner/repo@skill-name
```

#### requires.platforms

| 属性 | 值 |
|------|-----|
| 类型 | `map` |
| 必需 | 否 |

支持的平台。

```yaml
requires:
  platforms:
    os: [linux, darwin, windows]
    arch: [amd64, arm64]
```

---

## 完整示例

```yaml
---
name: pdf-processing
description: >
  Extract text and tables from PDF files, fill PDF forms, and
  merge multiple PDFs. Use when working with PDF documents or
  when the user mentions PDFs, forms, or document extraction.
when-to-use: >
  Also use when the user asks about document extraction, form
  filling, or mentions scanned documents.
allowed-tools:
  - Bash(git:*)
  - Bash(jq:*)
  - Read
argument-hint: <file> [format]
arguments:
  - file
  - format
disable-model-invocation: false
user-invocable: true
license: MIT
metadata:
  author: example-org
  version: "1.0"

requires:
  bins:
    - git
  any-bins:
    - pdftotext
    - mutool
  env:
    - PDF_API_KEY
  skills:
    - github:example/pdf-core@pdf-core
  platforms:
    os:
      - linux
      - darwin
    arch:
      - amd64
      - arm64
---
```
