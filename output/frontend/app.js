const DATA = {
  "requirements": [
    {
      "spec_name": "core",
      "req_name": "business-core-flow",
      "description": "系统应完整支持以下业务目标：Refactor SuperDev Studio into a change-driven delivery workspace. Unify the product IA around workspace, change center, delivery runs, context hub, and project settings. Persist project default execution profile. Add change batch and run traceability metadata. Upgrade the React plus Go app pages and APIs accordingly.。请结合以下上下文实现：本地知识参考: PRODUCT_DESIGN - # SuperDev Studio 产品设计说明；外部最佳实践: Refactor an Existing Codebase using Prompt Driven Development - DEV Community - January 13, 2026 -The API exposes endpoints for managing products and categories, including batch operations. Before refactoring, the API is executed locally and exercised through its endpoints to confirm current behavio；外部最佳实践: Supercharge Developer Workflows with GitHub Copilot Workspace Extensions | by David Minkovski | Medium - February 14, 2025 -Use the same UI/UX, but with our own AI logic.；外部最佳实践: 4 steps to connect change management and DevOps | CIO - December 1, 2023 -Businesses must emphasize the importance ofclear processes for requesting, reviewing, approving, and implementing changes, rigorous testing and validation, and continuous improvementto ensure successful",
      "scenarios": [
        {
          "given": "用户进入系统首页",
          "when": "按业务路径完成主要操作",
          "then": "系统成功返回结果并展示下一步引导"
        }
      ]
    },
    {
      "spec_name": "profile",
      "req_name": "profile-management",
      "description": "用户应可查看和更新个人资料与偏好设置。",
      "scenarios": [
        {
          "given": "用户已经登录",
          "when": "在个人中心提交更新",
          "then": "资料变更被持久化并反馈成功状态"
        }
      ]
    }
  ],
  "phases": [
    {
      "id": "phase-1",
      "title": "增量需求与影响分析",
      "objective": "确认变更边界、兼容性和风险。",
      "deliverables": [
        "变更影响矩阵",
        "兼容性策略",
        "回滚方案"
      ]
    },
    {
      "id": "phase-2",
      "title": "前端模块扩展",
      "objective": "优先扩展用户可感知模块并保持设计一致性。",
      "deliverables": [
        "新增页面/组件",
        "交互更新",
        "文案与埋点更新"
      ]
    },
    {
      "id": "phase-3",
      "title": "后端能力扩展",
      "objective": "按规范增加接口与数据能力，避免破坏存量系统。",
      "deliverables": [
        "增量 API",
        "迁移脚本",
        "灰度开关"
      ]
    },
    {
      "id": "phase-4",
      "title": "回归验证与发布",
      "objective": "覆盖关键链路并完成灰度/正式发布。",
      "deliverables": [
        "回归测试结果",
        "发布报告",
        "监控告警确认"
      ]
    },
    {
      "id": "phase-5",
      "title": "持续优化",
      "objective": "围绕 business-core-flow, profile-management 持续迭代优化。",
      "deliverables": [
        "性能优化清单",
        "体验优化清单",
        "后续版本计划"
      ]
    }
  ],
  "docs": {
    "prd": "D:\\Work\\agent-demo\\SuperDev-Studio\\output\\superdev-studio-prd.md",
    "architecture": "D:\\Work\\agent-demo\\SuperDev-Studio\\output\\superdev-studio-architecture.md",
    "uiux": "D:\\Work\\agent-demo\\SuperDev-Studio\\output\\superdev-studio-uiux.md",
    "plan": "D:\\Work\\agent-demo\\SuperDev-Studio\\output\\superdev-studio-execution-plan.md",
    "frontend_blueprint": "D:\\Work\\agent-demo\\SuperDev-Studio\\output\\superdev-studio-frontend-blueprint.md"
  }
};

const docContainer = document.getElementById("doc-links");
const reqContainer = document.getElementById("requirements");
const timelineContainer = document.getElementById("timeline");

const docList = [
  { title: "PRD 文档", desc: "产品目标、需求边界、验收标准", path: DATA.docs.prd },
  { title: "架构文档", desc: "模块划分、接口契约、部署策略", path: DATA.docs.architecture },
  { title: "UI/UX 文档", desc: "视觉系统、交互规则、页面结构", path: DATA.docs.uiux },
  { title: "执行路线图", desc: "0-1 与 1-N+1 的分阶段推进", path: DATA.docs.plan },
  { title: "前端蓝图", desc: "前端模块拆分与先行交付策略", path: DATA.docs.frontend_blueprint },
];

for (const doc of docList) {
  if (!doc.path) continue;
  const link = document.createElement("a");
  link.className = "doc-item";
  link.href = relativePath(doc.path);
  link.target = "_blank";
  link.rel = "noreferrer";
  link.innerHTML = `<b>${doc.title}</b><span>${doc.desc}</span>`;
  docContainer.appendChild(link);
}

for (const req of DATA.requirements) {
  const chip = document.createElement("span");
  chip.className = "chip";
  chip.textContent = `${req.spec_name} · ${req.req_name}`;
  reqContainer.appendChild(chip);
}

for (const phase of DATA.phases) {
  const li = document.createElement("li");
  li.innerHTML = `<b>${phase.title}</b><p>${phase.objective}</p>`;
  timelineContainer.appendChild(li);
}

function relativePath(path) {
  if (!path) return "#";
  const normalized = String(path).replace(/\\/g, "/");
  const marker = "/output/";
  const index = normalized.lastIndexOf(marker);
  if (index >= 0) {
    return ".." + normalized.slice(index + marker.length - 1);
  }
  return "#";
}
