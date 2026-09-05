# Renderer Adapter 契约

短视频创作插件内置 Remotion 主渲染引擎，并允许 HyperFrames 作为同级 Renderer Adapter 接入。Recipe 的 `adapter` 只允许 `remotion` 或 `hyperframes`；Three.js、地图、图表和其他技术属于两种引擎内部的模板实现能力，不形成第三套项目、任务或产物协议。

## 调用顺序

1. AI 调用 `short.video.render.start`，得到 `job.job_id`、`job.recipe_path` 和 `adapter`。
2. 适配器读取 `recipe_path`，在同一个 `workspace_root` 内运行自己的渲染工具链。禁止读取工作区外的素材，禁止执行未声明的任意命令。
3. 适配器把 `video/mp4`、`video/webm` 或其他明确的 `video/*` 文件写入工作区内，再调用 `short.video.render.complete`，传入 `job_id`、`status: completed`、`artifact_path` 和 `renderer`。
4. 失败时传入 `status: failed` 和结构化可读的 `message`。AI 使用 `render.status` 查询状态，使用 `artifact.export` 获取制品路径与 SHA-256。

## 设计约束

- `recipe_path` 是一次渲染的不可变快照，避免项目后续编辑影响正在运行的 Job。
- `render.complete` 会拒绝工作区外路径、目录和非 `video/*` Artifact。
- 故事板预览是独立的结构/风格验证能力，不是 Renderer Adapter，也不代表最终视频。
- Remotion 可在模板内部组合 Three.js、地图、图表和音视频处理；HyperFrames 也通过同一 Recipe、Job 和 Artifact 契约扩展。
