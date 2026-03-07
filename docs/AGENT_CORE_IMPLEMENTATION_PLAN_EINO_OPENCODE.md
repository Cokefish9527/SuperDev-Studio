# SuperDev Studio Eino Agent Core 落地方案

## 1. 文档目标

本文是 `docs/AGENT_CORE_IMPLEMENTATION_PLAN.md` 的并行补充方案，不覆盖原有 LangGraph 方向方案。

本文目标是回答三个问题：

1. 在当前服务端以 Go 为主的前提下，为什么 `Eino` 更适合作为第一阶段 Agent Core 技术底座。
2. 如何将 `Eino` 与 `OpenCode / Oh-My-OpenCode` 风格扩展层组合，形成适合 `SuperDev Studio` 的可落地方案。
3. 如何在不推翻现有 `Project -> ChangeBatch -> PipelineRun -> Context Hub` 控制面的情况下完成渐进式迁移。

## 2. 执行摘要

### 2.1 结论

建议采用以下路线：

> 保留现有 Studio 控制面，使用 `Eino` 在 Go 后端内实现第一阶段 `Agent Core`，并在其上引入 `OpenCode` 风格的 `agents / modes / skills / hooks / commands` 扩展层。

该路线相比原 LangGraph 方案的变化点是：

- **不单独引入 Python sidecar**，优先保持 Go 单栈。
- **不推翻原方案中的架构思想**，仅将 Agent Runtime 实现从 Python/LangGraph 改为 Go/Eino。
- **扩展层仍借鉴 OpenCode**，因为插件化与技能化属于产品层抽象，与底层 Agent Runtime 不冲突。

### 2.2 为什么现在更推荐 Eino

当前 `SuperDev Studio` 的核心特征是：

- 业务编排和状态持久化都在 Go 后端
- 现有 LLM 供应商是 `Volcengine Ark`
- 核心执行器是 `super-dev` CLI / pipeline
- UI 需要强依赖运行事件、工件和追踪记录

因此第一阶段最重要的不是拿到“理论上最强”的 Agent runtime，而是拿到：

- 更低接入成本
- 更高的工程贴合度
- 可复用现有 `store` / `pipeline` / `api` / `event` 结构
- 能快速让 `step_by_step` 模式具备真正的自主推进能力

从这个角度看，`Eino` 比引入 Python sidecar 更合适。

## 3. 当前问题与保留原则

## 3.1 当前问题不在模型，而在系统结构

当前系统中，LLM 仍主要承担：

- 任务拆分
- 任务执行建议
- 任务完成判定
- 迭代修复建议
- 验收总结
- 构思稿 / 设计稿 / 复盘稿

这些都属于“局部增强”，并没有形成真正的 Agent 主链路。

当前不足主要表现为：

1. 主执行链路仍是固定流程，不是 Agent 决策流。
2. 上下文检索仍是规则式召回，不是证据驱动的检索链路。
3. 没有统一的计划-执行-观察-评估-重试状态机。
4. 工具调用与推理没有统一建模。
5. 缺少步骤级 trace，难以解释“为什么这么做”。

## 3.2 必须保留的现有资产

以下设计不应推翻：

- `Project`：默认执行画像与项目级配置中心
- `ChangeBatch`：目标与范围边界
- `PipelineRun`：运行记录与追踪闭环
- `RunEvent`：可观测事件流
- `Context Hub`：记忆、知识、上下文包的聚合体验

因此本次方案原则是：

- **保留控制面**
- **重构执行面**
- **增加扩展层**

## 4. Eino 与原方案的对比

## 4.1 对比对象

本节比较三条路线：

1. 继续沿现有 Go 逻辑堆叠 prompt 和 fallback
2. 采用 LangGraph + Python sidecar
3. 采用 Eino + Go 原生 Agent Runtime

## 4.2 对比结果

| 维度 | 现有逻辑增强 | LangGraph 方案 | Eino 方案 |
| --- | --- | --- | --- |
| 与当前 Go 后端贴合度 | 高 | 中 | 很高 |
| 引入新运行时成本 | 低 | 高 | 低 |
| Agent 编排能力 | 低 | 高 | 中高 |
| 对 Ark / Go CLI / Go store 集成 | 中 | 中 | 高 |
| 调试与生态成熟度 | 低 | 很高 | 中高 |
| 可渐进接入难度 | 中 | 中高 | 低 |
| 第一阶段落地速度 | 中 | 中 | 高 |
| 长期多 agent 上限 | 低 | 很高 | 高 |

## 4.3 结论

- 如果目标是 **尽快在现有 Go 服务内做出可运行的 Agent Core MVP**，优先选 `Eino`。
- 如果目标是 **追求最成熟的持久化执行与图调试生态**，`LangGraph` 仍有优势。
- 对 `SuperDev Studio` 当前阶段，推荐先走 `Eino-first` 路线，但必须保留 `AgentRuntime` 抽象，避免未来被单一框架锁死。

## 5. 目标方案

## 5.1 方案一句话描述

> 在现有 Go 后端内新增 `Eino Agent Runtime`，以统一状态机方式驱动 `super-dev` 工具调用、上下文检索、结果评估与自动修复；同时在其上增加 `OpenCode` 风格的扩展配置层，使 Agent 行为可以被项目级配置和复用。

## 5.2 两层模型

### 第一层：Eino Runtime Layer

解决“系统如何真正自主运行”。

职责包括：

- 计划生成
- 工具调用
- 上下文检索
- 结果评估
- 重试策略
- 人工介入点
- 步骤级追踪

### 第二层：OpenCode-style Extension Layer

解决“系统如何被团队化、场景化、产品化配置”。

职责包括：

- `agents`：角色
- `modes`：运行模式
- `skills`：技能包
- `hooks`：阶段钩子
- `commands`：高阶动作命令

## 6. 架构原则

1. **单栈优先**：第一阶段不引入 Python sidecar。
2. **接口解耦**：定义 `AgentRuntime` 接口，Eino 只是实现者之一。
3. **工具统一入口**：Agent 只通过 Tool Gateway 调用外部能力。
4. **状态可回放**：所有关键步骤必须可追踪、可解释。
5. **先单 Agent 后多 Agent**：先完成 `Delivery Agent`，再扩展多 agent。
6. **先 step_by_step 后 full_cycle**：优先改造可控、易观测模式。

## 7. 目标架构

```text
Frontend
  └─ Studio UI

Go Backend
  ├─ API / Store / Context Hub / PipelineRun / ChangeBatch
  ├─ Agent Runtime Interface
  ├─ Eino Runtime Implementation
  ├─ Tool Gateway
  ├─ Retrieval Service
  ├─ Evaluation Service
  └─ Extension Registry (agents / modes / skills / hooks / commands)

Execution Targets
  ├─ super-dev CLI
  ├─ file/artifact access
  ├─ memory / knowledge / context APIs
  └─ preview / quality / deploy tools
```

## 8. 第一阶段边界

第一阶段只覆盖以下内容：

- `step_by_step` 模式
- 单个 `delivery-agent`
- 基础检索
- 核心工具调用
- 步骤级 trace
- 基础评估与 repair loop

以下内容不在第一阶段：

- 多 agent 协作
- 插件市场 UI
- 大规模向量平台
- 自动部署闭环默认开启
- 跨项目技能共享平台

## 9. 建议产出物

本方案建议配套以下文档与交付物：

1. `docs/AGENT_CORE_IMPLEMENTATION_PLAN_EINO_OPENCODE.md`
2. `docs/AGENT_CORE_DETAILED_DESIGN_EINO_OPENCODE.md`
3. `docs/AGENT_CORE_DEVELOPMENT_PLAN_EINO_OPENCODE.md`
4. `.super-dev/changes/eino-opencode-agent-core/`

## 10. 成功标准

当以下目标达成时，可认为 Eino 方案第一阶段成功：

1. `step_by_step` 模式能够由 Agent 自主推进 70% 以上的标准链路。
2. 每个运行都能展示：
   - 当前计划
   - 当前步骤
   - 调用工具
   - 使用证据
   - 评估结果
3. 失败场景可区分：
   - 自动修复
   - 需要补上下文
   - 需要人工介入
4. Agent 行为可通过 `agents / modes / skills / hooks / commands` 配置演进，而不是继续堆到 `manager.go` 中。

## 11. 参考资料

- Eino Overview: `https://www.cloudwego.io/docs/eino/overview/`
- Eino ADK Overview: `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_preview/`
- Eino Checkpoint & Interrupt: `https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/checkpoint_interrupt/`
- Eino Callback: `https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/callback_manual/`
- Eino Visual Debug Plugin: `https://www.cloudwego.io/docs/eino/core_modules/devops/visual_debug_plugin_guide/`
- Eino Ark Chat Model: `https://www.cloudwego.io/docs/eino/ecosystem_integration/chat_model/chat_model_ark/`
- Eino MCP Tool: `https://www.cloudwego.io/docs/eino/ecosystem_integration/tool/tool_mcp/`
- Eino Commandline Tool: `https://www.cloudwego.io/docs/eino/ecosystem_integration/tool/tool_commandline/`
- OpenCode Agents: `https://opencode.ai/docs/agents`
- OpenCode Plugins: `https://opencode.ai/docs/plugins`
- OpenCode Skills: `https://opencode.ai/docs/skills`
- Oh-My-OpenCode: `https://ohmyopencode.org/`
