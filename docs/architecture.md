# 短视频扩展架构

## 分层

HiMind Agent 只负责通用 Capability Registry、MCP 暴露、插件生命周期和本地文件边界。短视频业务规则位于扩展源：插件提供确定性文件/Recipe/Job 能力，Skill 提供对话编排和沉淀方法。Dashboard 连接后只增加组织调度、审核和分发，不改变本地创作协议。

## 可替换适配器

模板描述 `adapter` 和 `composition`，Recipe 只携带数据与视觉参数。`adapter` 严格限定为 `remotion | hyperframes`：Remotion 是内置主渲染引擎，以固定版本、后台 Job 和工作区内运行时生成 MP4；HyperFrames 是同级可插拔实现。Three.js、地图等技术只能作为模板实现或引擎组合能力接入。HyperFrames Adapter 读取 `render.start` 返回的 `recipe_path`，把最终视频写回工作区，再调用 `render.complete` 登记 Artifact，因此所有实现共享同一个项目、任务和产物模型。契约细节见 [renderer-adapter.md](renderer-adapter.md)。

插件 UI 只调用项目、预览、渲染和 Artifact Capability，`.himind-video` 是唯一状态源。UI 关闭或 Agent 重启后，后台任务和产物仍可由 `project.get`、`render.status` 恢复查看。

## 反馈到资产

每个预览反馈按类别写入项目目录，项目 Recipe 通过 `project.update` 保存并递增 `revision`。风格 Skill 只在反馈被接受且可复现时提出候选，候选先保存到工作区，不直接覆盖扩展源：

```text
feedback.list -> candidate.save -> evidence -> template variant / style pack / quality rule
  -> golden storyboard -> quality.check -> 新版本资产
```

资产以 Git 版本控制，项目 Recipe 固定引用 `id + version`，支持回滚和复现实验。`expected_updated_at` 用于并发编辑保护，避免多个 AI 会话互相覆盖。

## 独立分发

GitHub 仓库是扩展源，插件 `.hmpkg` 和 Skill `.hmskill` 是不可变制品。独立模式安装默认注册本地能力；Dashboard 模式可在相同制品上增加组织审核和分发状态。目录工作区与扩展源码工作区分离，视频项目允许任意用户目录。

外部 AI 客户端通过 Agent 的 `extension.workspace.bind` 绑定本仓库根目录。Agent 读取 `extensions.json` 后发现插件和 Skill；`extension.workspace.current` 返回绑定结果，`extension.workspace.clear` 可恢复到会话目录。绑定只约束扩展创作与候选包，短视频项目的 `workspace_root` 不受此限制。

## 失败诊断

所有可恢复失败返回 `state: blocked` 与 `blockers[]`，每项包含 `code`、`stage`、`message`、`remediation`、`retryable`。外部 AI 按稳定 code 处理，例如 `renderer_runtime_missing` 表示缺少 Remotion 运行时，`adapter_not_found` 表示 Recipe 未使用 `remotion` 或 `hyperframes`。
