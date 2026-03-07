# SuperDev Studio 重构蓝图

## 1. 为什么需要重构

基于当前代码与产品设计，我认为原方案里有几处关键问题需要被明确质疑：

### 1.1 信息架构按“功能模块”拆得太散
- `Projects`、`Pipeline`、`Memory`、`Knowledge`、`Context Optimizer` 分别成页，但用户真正的目标并不是访问模块，而是推进一次变更。
- 这会导致“我应该先去哪里开始”不够明确，页面之间也缺少自然闭环。

### 1.2 项目推进与流水线启动存在双入口
- `ProjectsPage` 上存在“执行项目推进”，`PipelinePage` 上也可以启动运行。
- 两者语义接近，但上下文、默认值和结果追踪方式不同，容易让用户误以为是两套执行系统。

### 1.3 Task 不是交付事实来源，却承担了过多叙事
- 当前任务看板更像规划工具，而真实执行单元其实是 `super-dev change` / pipeline run。
- 如果让 Task、Run、Change 各自维护状态，会出现推进状态漂移。

### 1.4 上下文能力被拆成三个孤立页面
- 记忆、知识、上下文打包在实现上有关联，但使用体验被拆开。
- 用户无法直观看到“这次运行到底注入了哪些记忆与知识”。

### 1.5 项目默认执行配置没有成为稳定中心
- 如果每次运行都重新填写平台、前后端、上下文策略，运行结果的一致性会快速下降。
- 这些值应该属于工作区级配置，而不是每次任务临时输入。

### 1.6 运行记录缺少变更链路追踪字段
- 没有 `change_batch_id` / 外部 `change_id` 时，很难回答“这次改版是哪一轮变更触发的”“失败重试继承的是哪条链路”。

## 2. 重构后的产品信息架构

将产品从“功能罗列”调整为“change 驱动交付工作台”。

### 2.1 一级导航
1. **工作台**：整体概览，查看当前工作区最近运行与关键指标
2. **工作区**：管理项目、规划任务、维护任务排期
3. **变更中心**：创建/选择 change batch，定义本轮改版目标
4. **交付运行**：基于当前工作区 + 当前变更批次启动 `super-dev`
5. **上下文中心**：统一查看 Context Pack / 记忆 / 知识库
6. **项目设置**：维护默认技术栈、上下文策略、回写策略

### 2.2 核心交互路径
1. 在 **工作区** 选择项目
2. 在 **变更中心** 创建本轮 `ChangeBatch`
3. 在 **项目设置** 确认默认执行配置
4. 在 **交付运行** 启动一次 `super-dev` 流程
5. 在 **上下文中心** 回看上下文沉淀，形成下一轮输入

### 2.3 结构调整原则
- `Project` 负责稳定配置
- `ChangeBatch` 负责交付目标与范围
- `PipelineRun` 负责执行记录与产物追踪
- `Task` 退回为规划视角，不与真实交付状态竞争事实来源

## 3. 重构后的数据模型

### 3.1 关系总览

```text
Project 1 ── N ChangeBatch 1 ── N PipelineRun 1 ── N RunEvent
   │
   ├── N Task
   ├── N Memory
   └── N KnowledgeDocument 1 ── N KnowledgeChunk
```

### 3.2 关键实体

#### Project
工作区根对象，新增“默认执行画像”：

| 字段 | 说明 |
| --- | --- |
| `id` | 项目 ID |
| `name` / `description` / `repo_path` / `status` | 基础信息 |
| `default_platform` | 默认平台，如 `web` |
| `default_frontend` | 默认前端，如 `react` |
| `default_backend` | 默认后端，如 `go` |
| `default_domain` | 默认业务领域 |
| `default_context_mode` | `off/auto/manual` |
| `default_context_token_budget` | 默认 token 预算 |
| `default_context_max_items` | 最大上下文条目数 |
| `default_context_dynamic` | 是否按阶段动态召回 |
| `default_memory_writeback` | 是否在运行结束回写记忆 |

#### ChangeBatch
本轮改版的业务执行单元：

| 字段 | 说明 |
| --- | --- |
| `id` / `project_id` | 归属关系 |
| `title` | 批次标题 |
| `goal` | 本轮目标 |
| `status` | `draft/queued/running/completed/failed` |
| `mode` | `step_by_step` / `full_cycle` |
| `external_change_id` | 对应的 `super-dev change_id` |
| `latest_run_id` | 最近一次运行 |

#### PipelineRun
执行记录对象，新增追踪字段：

| 字段 | 说明 |
| --- | --- |
| `id` / `project_id` | 基础归属 |
| `change_batch_id` | 关联改版批次 |
| `external_change_id` | 外部 `super-dev` 变更 ID |
| `prompt` | 本次执行指令 |
| `status` / `stage` / `progress` | 执行状态 |
| `retry_of` | 重试来源 |
| `started_at` / `finished_at` | 生命周期时间 |

### 3.3 模型调整原则
- `ChangeBatch` 解决“目标”和“执行”脱节问题
- `Project defaults` 解决每次运行重复输入问题
- `PipelineRun metadata` 解决结果不可追踪问题

## 4. 页面改版草图

以下为面向交付闭环的低保真草图。

### 4.1 全局框架

```text
+---------------------------------------------------------------+
| Header: 当前工作区 | 当前变更批次 | 快捷操作                   |
+-----------+-----------------------------------------------+
| Nav       | 主内容区                                       |
| 工作台    |                                               |
| 工作区    |                                               |
| 变更中心  |                                               |
| 交付运行  |                                               |
| 上下文中心|                                               |
| 项目设置  |                                               |
+-----------+-----------------------------------------------+
```

### 4.2 工作台

```text
+---------------------------------------------------------------+
| 工作台总览                                                   |
| [工作区数] [计划任务] [交付运行] [记忆条目] [知识文档]       |
+---------------------------------------------------------------+
| 最近交付运行                                                 |
| - run status / stage / progress / prompt / change batch     |
+---------------------------------------------------------------+
```

### 4.3 工作区

```text
+----------------------+----------------------------------------+
| 工作区列表           | 计划任务看板 / 甘特图                  |
| - 选择项目           | - 任务规划                             |
| - 新建项目           | - 自动排期                             |
| - 快速推进           | - 仅做 planning，不直接代表交付事实    |
+----------------------+----------------------------------------+
```

### 4.4 变更中心

```text
+---------------------------------------------------------------+
| 变更中心                                                     |
| [新建变更批次]                                               |
+---------------------------------------------------------------+
| 批次列表: title | mode | status | external change_id | run    |
+---------------------------------------------------------------+
| 当前选中批次                                               |
| 目标 / 说明 / 下一步建议：去交付运行启动 super-dev            |
+---------------------------------------------------------------+
```

### 4.5 交付运行

```text
+------------------------------+--------------------------------+
| 运行配置表单                 | 当前变更批次                  |
| prompt                       | title / goal / status         |
| platform/frontend/backend    | external change_id            |
| context strategy             | latest run                    |
| [启动流水线]                 |                                |
+------------------------------+--------------------------------+
| 运行列表: status / stage / progress / batch / change_id      |
| 运行详情: events / checklist / artifacts                     |
+---------------------------------------------------------------+
```

### 4.6 上下文中心

```text
+---------------------------------------------------------------+
| 上下文中心                                                   |
| [Context Pack] [记忆库] [知识库]                             |
| 统一围绕“本次交付需要什么上下文”来组织                        |
+---------------------------------------------------------------+
```

### 4.7 项目设置

```text
+---------------------------------------------------------------+
| 项目设置                                                     |
| repo_path / status                                            |
| 默认平台 / 默认前端 / 默认后端 / 默认领域                    |
| 默认上下文模式 / token budget / max items                    |
| 动态召回 / 记忆回写                                          |
+---------------------------------------------------------------+
```

## 5. 本轮已落地的代码调整

### 5.1 后端
- 扩展 `Project` 默认执行配置
- 新增 `ChangeBatch` 存储模型和接口
- 为 `PipelineRun` 增加 `change_batch_id` 与 `external_change_id`
- 启动与重试运行时保留批次链路，并回写批次状态

### 5.2 前端
- 调整一级导航为工作台/工作区/变更中心/交付运行/上下文中心/项目设置
- 新增 `ChangeCenterPage`、`ProjectSettingsPage`、`ContextHubPage`
- 让 `PipelinePage` 绑定当前 `ChangeBatch`，并优先使用项目默认配置
- 将当前活动项目与活动变更批次持久化到前端状态中

## 6. 仍值得继续追问的地方

以下不是本轮必须改完，但属于下一轮应该继续收敛的问题：

1. **是否保留 `ProjectsPage` 里的“一键推进”按钮？**
   - 建议后续改成“创建变更并进入运行页”，避免继续形成第二执行入口。

2. **Task 是否应该直接映射到 ChangeBatch？**
   - 建议下一阶段把 Task 升级成“规划任务 → 关联 change batch”的模型，而不是单独漂浮。

3. **是否继续保留 `/memory` 与 `/knowledge` 独立路由？**
   - 当前为了兼容保留，但长期更建议统一收敛到 `/context`。

4. **前端包体偏大**
   - 当前 `vite build` 已出现 chunk size 警告，建议后续用路由级懒加载切分页面。

## 7. 使用 super-dev 推进本次改版的方式

本轮采用的是 `super-dev` 的增量 SDD 流程：

1. `super-dev spec init`
2. `super-dev spec propose refactor-studio-workspace`
3. 为 `workspace` / `change-batch` / `project-defaults` / `pipeline-metadata` / `context-hub` 添加 requirement
4. 用 `super-dev pipeline ... --skip-scaffold --skip-redteam --skip-quality-gate` 生成 PRD / 架构 / UIUX 参考文档
5. 基于变更规格手工完成前后端实现
6. 通过 `super-dev spec validate` 与 `super-dev quality --type all` 做收口

## 8. 调整方案结论

如果只用一句话概括这次改版：

> 把 SuperDev Studio 从“若干 AI 研发功能页的集合”，重构成“以 ChangeBatch 为执行主线、以 Project 默认配置为稳定底座、以 PipelineRun 为追踪闭环”的交付工作台。
