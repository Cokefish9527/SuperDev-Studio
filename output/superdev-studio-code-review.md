# superdev-studio - 代码审查指南

> **生成时间**: 自动生成
> **技术栈**: 前端 react | 后端 go | 平台 web

---

## 审查流程

### 第 1 步: 自动化检查
运行以下命令，确保所有检查通过：

```bash
# Linter
npm run lint  # 或 python -m pylint .

# 类型检查
npm run type-check  # 或 mypy .

# 格式检查
npm run format:check  # 或 black --check .

# 测试
npm test  # 或 pytest
```

### 第 2 步: 功能审查
- [ ] 代码实现了需求中的所有功能
- [ ] 边界条件处理正确
- [ ] 错误处理完善
- [ ] 日志记录恰当

### 第 3 步: 安全审查
- [ ] 输入验证完整
- [ ] 输出编码正确
- [ ] 认证授权正确实现
- [ ] 敏感数据已保护

### 第 4 步: 性能审查
- [ ] 无明显性能问题
- [ ] 数据库查询优化
- [ ] 缓存使用恰当
- [ ] 资源使用合理

---

## 通用审查项

### 代码质量

**命名规范:**
- [ ] 变量/函数命名清晰，见名知意
- [ ] 类名使用 PascalCase
- [ ] 函数名使用 camelCase (前端) 或 snake_case (后端)
- [ ] 常量使用 SCREAMING_SNAKE_CASE
- [ ] 布尔值使用 is/has/should 前缀

**函数设计:**
- [ ] 函数职责单一，不超过 50 行
- [ ] 参数数量合理 (不超过 5 个)
- [ ] 避免深层嵌套 (不超过 3 层)
- [ ] 返回值类型一致
- [ ] 无重复代码 (DRY 原则)

**错误处理:**
- [ ] 所有异常被捕获
- [ ] 错误信息清晰有用
- [ ] 不吞掉异常
- [ ] 资源正确释放 (使用 try-finally 或 with)
- [ ] 不返回 `null`/`None` 给前端 (使用统一错误响应)

**注释和文档:**
- [ ] 复杂逻辑有注释说明
- [ ] 公共 API 有文档注释
- [ ] TODO 注释有跟踪 issue
- [ ] 注释与代码保持一致

### 安全性

**输入验证:**
- [ ] 所有用户输入验证
- [ ] 验证类型、长度、格式
- [ ] 使用白名单而非黑名单
- [ ] SQL/命令注入防护 (使用参数化查询)

**认证授权:**
- [ ] 敏感操作需要认证
- [ ] 权限检查正确
- [ ] Token/Session 有效期合理
- [ ] HTTPS 强制使用

**数据保护:**
- [ ] 密码使用强哈希 (bcrypt/Argon2)
- [ ] 敏感数据不记录到日志
- [ ] 敏感数据不暴露给前端
- [ ] 加密存储敏感配置

### 性能

**数据库:**
- [ ] 查询使用索引
- [ ] 避免全列查询（优先明确列名）
- [ ] 避免 N+1 查询
- [ ] 使用连接池
- [ ] 分页大数据集

**API:**
- [ ] 响应大小合理
- [ ] 实施缓存策略
- [ ] 异步处理耗时操作
- [ ] 实施速率限制

**前端:**
- [ ] 避免不必要的重渲染
- [ ] 使用 React.memo/useMemo/useCallback
- [ ] 懒加载路由和组件
- [ ] 图片优化和懒加载

### 测试

**单元测试:**
- [ ] 核心逻辑有单元测试
- [ ] 测试覆盖率 > 80%
- [ ] 边界条件有测试
- [ ] 错误情况有测试

**集成测试:**
- [ ] API 端点有集成测试
- [ ] 关键流程有测试
- [ ] 数据库操作有测试

---

## 前端特定审查 (REACT)


**React 特定:**
- [ ] 组件拆分合理 (不超过 200 行)
- [ ] Props 类型定义完整 (TypeScript/PropTypes)
- [ ] State 使用恰当 (useState/useReducer)
- [ ] Effect 依赖正确 (useEffect)
- [ ] Context 使用合理 (避免过度使用)
- [ ] 性能优化 (memo/useMemo/useCallback)
- [ ] Key 属性正确 (列表渲染)
- [ ] 事件处理函数稳定 (useCallback)

**Hooks 使用:**
- [ ] 自定义 Hook 可复用
- [ ] Hook 规则遵守 (只在顶层调用)
- [ ] Effect 依赖数组完整
- [ ] 清理函数正确返回

**状态管理:**
- [ ] 全局状态使用 Redux/Zustand
- [ ] 本地状态使用 useState
- [ ] 表单状态使用 Form 库
- [ ] 服务端状态使用 React Query/SWR


---

## 后端特定审查 (GO)


**通用后端:**
- [ ] API 设计 RESTful
- [ ] 请求验证完整
- [ ] 错误处理统一
- [ ] 日志记录恰当
- [ ] 依赖注入使用


---

## 领域特定审查 (通用)


**通用领域:**
- [ ] 业务规则正确
- [ ] 数据一致性
- [ ] 事务处理合理
- [ ] 幂等性保证


---

## 审查清单模板

### Pull Request 审查清单

**提交前自检:**
- [ ] 代码符合团队规范
- [ ] Linter 无错误
- [ ] 所有测试通过
- [ ] 新增代码有测试
- [ ] 文档已更新

**Reviewers 检查:**
- [ ] 代码逻辑正确
- [ ] 无安全漏洞
- [ ] 无性能问题
- [ ] 错误处理完善
- [ ] 代码可维护

---

## 常见问题检查表

### 1. 空值处理
- [ ] 使用可选链 `?.` 访问属性
- [ ] 使用空值合并 `??` 提供默认值
- [ ] 函数参数验证空值
- [ ] 数据库查询结果检查空值

### 2. 异步处理
- [ ] async/await 正确使用
- [ ] Promise/异常正确处理
- [ ] 并发控制合理
- [ ] 超时处理设置

### 3. 资源管理
- [ ] 文件句柄正确关闭
- [ ] 数据库连接释放
- [ ] 订阅/监听器清理
- [ ] 定时器清理

### 4. 状态管理
- [ ] 状态不可变修改
- [ ] 状态更新正确
- [ ] 副作用隔离
- [ ] 状态持久化合理

---

## 审查工具

### 推荐工具

**前端 (react):**
- ESLint - 代码检查
- Prettier - 代码格式化
- TypeScript - 类型检查
- Jest - 单元测试

**后端 (go):**
- pylint/flake8 - 代码检查
- black/autopep8 - 代码格式化
- mypy - 类型检查 (Python)
- pytest - 单元测试

### CI/CD 集成

在 CI 流水线中自动运行以下检查：

```yaml
# .github/workflows/review.yml
name: Code Review
on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Lint
        run: npm run lint
      - name: Type Check
        run: npm run type-check
      - name: Test
        run: npm test -- --coverage
      - name: Security Scan
        run: npm audit
```

---

## 审查反馈模板

### 建设性反馈

**示例:**

> 我注意到在 `UserService.ts:45` 中，直接使用了 `user.email` 而没有检查 `user` 是否为 `null`。这可能导致运行时错误。
>
> 建议: 使用可选链 `user?.email` 或在函数开始时验证参数。
>
> 参考: https://github.com/our-team/frontend-handbook/blob/main/null-safety.md

### 需要修改的反馈

**示例:**

> [需要修改] 在 `AuthController.java:123` 中，SQL 查询使用了字符串拼接，存在 SQL 注入风险。
>
> 请修改为参数化查询:
> ```java
> // 错误：通过字符串拼接构造数据库查询语句
> String query = buildQueryByConcat(email);
>
> // 正确：使用参数占位符并绑定参数
> String query = buildPreparedQueryWithParams(email);
> ```
>
> 参考: OWASP SQL Injection Prevention Cheat Sheet

---

## 审查完成后

### 通过审查
- [ ] 批准并合并 PR
- [ ] 感谢贡献者
- [ ] 更新相关文档

### 需要修改
- [ ] 在 PR 中留下具体反馈
- [ ] 标注需要修改的文件和行号
- [ ] 提供修改建议和参考资料
- [ ] 设置修改后的截止时间

---

## 附录: 代码审查最佳实践

1. **及时响应**: 收到审查请求后 24 小时内完成审查
2. **建设性反馈**: 提供具体、可操作的改进建议
3. **尊重贡献者**: 评论针对代码而非个人
4. **讨论复杂问题**: 面对面或视频会议讨论
5. **持续学习**: 分享审查中学到的经验
