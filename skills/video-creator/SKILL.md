---
name: video-creator
description: 通过对话创建模板化短视频项目。用户提出做排行榜、地图数据、图片叙事、数据可视化短视频，或要求预览、修改、构建视频时使用。
---

# 短视频创作助手

## 目标

将自然语言需求转化为可重复的短视频 Recipe。优先复用已有模板和风格，只让用户确认数据、标题、时长和少量定制字段。项目可位于任意已有目录，不要求目录属于扩展源码仓库，也不依赖 Dashboard。

## 标准闭环

1. 调用 `short.video.catalog.list`，了解模板族、风格包和渲染引擎；需要筛选时使用 `short.video.template.list` 和 `short.video.style.list`。
2. 结合需求选择模板和风格。先调用 `short.video.template.describe` 或 `short.video.style.describe` 获取插槽、兼容性和质量规则，不凭名称猜测字段。
3. 如果用户是在做同类视频或已有成熟项目，优先调用 `short.video.project.duplicate`。传入源项目的 `workspace_root`、`project_id` 和新的目标目录/名称；它会继承模板、风格、Recipe、数据结构和 brief，并允许只替换本次标题、数据、模板、风格或 brief。不要重新从零创建同类项目。
4. 没有可复用项目时调用 `short.video.project.create`，传入用户明确给出的 `workspace_root`、项目名称、`template_id`、`style_id` 和数据。`workspace_root` 可以是 Unity、Remotion、HyperFrames 或普通文件夹；中文项目名称会保留为展示名，并由插件生成安全的内部项目 ID。后续始终使用返回的 `project.id`，不要使用项目展示名。
5. 使用返回的 Recipe 生成内容。数据放在 `data` 或模板声明的插槽中；`adapter` 只允许 `remotion` 或 `hyperframes`，通常沿用模板声明，不要把 Three.js、地图或 FFmpeg 写成 adapter。
6. 调用 `short.video.recipe.validate`。发现 `blockers` 时按 `code` 修复，不能跳过模板兼容性、版本、画布、时长、数据和安全区检查。
7. 用 `short.video.project.update` 保存修订后的 Recipe，并传入上次 `project.get` 返回的 `updated_at` 做并发保护；然后调用 `short.video.preview.start` 生成 HTML 故事板，向用户返回 `artifact.path` 供浏览器预览。
8. 根据用户反馈再次更新 Recipe、重新校验并预览。确认有效反馈后调用 `short.video.feedback.record`，分类使用 `style`、`layout`、`motion`、`data`、`narrative` 或 `quality`；需要分析历史反馈时调用 `short.video.feedback.list`。
9. 调用 `short.video.quality.check`。只有 `passed: true` 才进入构建。
10. 调用 `short.video.runtime.status` 检查渲染运行时，再调用 `short.video.render.start`。默认 `remotion` 由插件内置后台任务生成真实 MP4；返回后轮询 `short.video.render.status`，不要因调用已返回就判断构建完成。
11. `hyperframes` 与 Remotion 是同级可插拔引擎；Three.js、地图、图表等专业能力只能在两种引擎的模板内部组合。HyperFrames 适配器读取 `recipe_path` 后调用 `short.video.render.complete` 回写结果。任务完成后用 `short.video.artifact.export` 获取路径和 SHA-256，需要本机查看时调用 `short.video.artifact.open`。

## 持续复用

成功项目的 Recipe、反馈和产物元数据都保存在 `<workspace_root>/.himind-video/`。重复出现的反馈交给 `video-style-curator` 评估，形成新的模板变体、Style Pack 版本或质量规则；不要直接覆盖既有版本。

## 约束

- 不执行任意 Shell，不自行拼接 Remotion、HyperFrames 或外部 URL 命令；只调用插件提供的固定参数能力。
- 不把 Dashboard 服务当作前置条件；本 Skill 的本地能力在独立模式可用。
- 大文件只通过 Artifact 元数据传递，不把视频内容塞入 MCP 响应。
- `blockers` 必须向调用方保留 `code`、`stage`、`message`、`remediation` 和 `retryable`。
