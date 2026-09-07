# himind-short-video

HiMind Agent 的独立短视频创作扩展源。通过 GitHub 分发，不依赖 Dashboard；安装后由 Agent 将本地插件 Capability 和 Skill 暴露给 HiMind AI 及兼容 Agent Skills 的 AI 工具。

## 安装与更新

在 HiMind Agent 的“扩展源”中添加 `https://github.com/MrBaoquan/himind-short-video`。仓库根目录的 `.himind/catalog.json` 是 Agent 使用的正式目录；插件与两个 Skill 作为独立制品安装和更新，不依赖 Dashboard。

扩展源默认由用户主动添加，安装和更新也由用户决定。发布制品使用不可变版本号和 GitHub Release，目录记录文件大小、SHA-256 与签名；源码仓库不提交本地构建目录、Remotion 运行时或视频项目产物。

## 组成

- `extensions.json`：独立扩展源目录清单，供 Agent 绑定和发现。
- `plugins/com.himind.short-video-creation/short-video-creation`：本地 JSON-RPC stdio 插件。
- `skills/video-creator`：从需求到预览、反馈和构建的创作编排。
- `skills/video-style-curator`：把反馈沉淀为模板变体、Style Pack 和质量规则。
- `catalog/`：版本化模板、风格和 Renderer Adapter 声明。
- `schemas/recipe.json`：Recipe 结构约束。
- `dsh/agent-presets/himind-short-video/`：可选的 DSH“短视频创作”预设。只有安装本扩展源中的短视频插件或功能包后，Agent 才会把它同步到 DSH；未使用扩展源时不会出现在 DSH 预设列表。

## 创作闭环

```text
对话需求 -> catalog/template/style -> project.create -> recipe.validate
  -> preview.start -> 用户反馈 -> feedback.record -> 再次预览
  -> quality.check -> render.start -> Remotion 后台任务 -> render.status -> artifact.export
```

每个项目只在调用方传入的 `workspace_root` 下写入：

```text
<workspace_root>/.himind-video/
├── projects/<project_id>/project.json
├── projects/<project_id>/recipe.json
├── projects/<project_id>/feedback.jsonl
├── candidates/<candidate_id>.json
├── jobs/
└── artifacts/
```

## 渲染边界

插件以 Remotion 作为内置主渲染引擎：首次构建会在当前视频工作区准备固定版本运行时，随后由后台 Job 生成真实 MP4，不新增常驻端口。HTML 故事板承担快速结构预览；HyperFrames 与 Remotion 是同级可插拔引擎。Recipe 的渲染引擎只允许 `remotion` 或 `hyperframes`，Three.js、地图、图表等能力必须作为两种引擎内部的模板实现，不形成第三套项目或渲染协议。HyperFrames 适配器仍使用 `render.start -> render.complete` 契约。

插件详情中的“短视频项目”窗口用于加载任意视频工作区、创建和切换项目、生成预览、构建视频、查看任务进度以及打开产物。界面只消费插件 Capability 和 `.himind-video` 状态，不维护第二份业务数据。

## 模板沉淀

`video-flow` 只是认可模板和视觉调性的参考来源，不是运行时依赖。迁移时将 `DataRanking`、`GDPRanking`、`ChinaGDPMap`、`AnhuiMap`、`AnhuiHSR`、`AnhuiHSRTime` 的结构、颜色、字号和动效整理为版本化模板/风格资产。新项目尽量通过替换 `data.items`、标题和少量插槽完成复用；重复反馈由风格 Skill 生成候选，不直接覆盖旧版本。

## 本地验证

在插件目录执行：

```text
go test ./...
go build -o bin/short-video-creation.exe .
```

随后使用 HiMind Agent 的扩展开发 Capability 完成 `plugin.validate -> plugin.build -> plugin.package -> candidate.save -> extension.test`，Skill 使用对应的 validate/package/candidate/test 链。独立模式保留候选包，不把 Dashboard 提审当成本地创作前置条件。

外部 AI 工具首次使用本仓库时，先调用 `extension.workspace.bind`，传入仓库根目录；再调用 `extension.workspace.current` 确认绑定结果。清除绑定使用 `extension.workspace.clear`。视频项目的 `workspace_root` 仍可指向任意项目目录，不要求放在扩展源内。GitHub 安装默认进入独立模式；连接 Dashboard 只增加组织审核、分发和调度，不改变本地协议。

安装短视频插件后，Agent 会按目录中的 SHA-256 从同一 GitHub 版本读取 DSH 预设，写入当前 HiMind DSH 交互 Profile 的用户预设目录。预设复用 DSH 标准模式的完整工具集，仅增加模板选择、预览、反馈、质量检查和真实渲染的创作约束；卸载扩展不会删除用户自行创建的同名预设。

发布新版本时必须先提升对应 Manifest 的语义版本并更新 `release_notes`，再完成 `validate -> build/package -> candidate.save -> extension.test`。通过后为每个制品创建独立 GitHub Release，并机械更新 `.himind/catalog.json`；同版本制品不得覆盖。
