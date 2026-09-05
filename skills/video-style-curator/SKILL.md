---
name: video-style-curator
description: 将短视频创作反馈沉淀为模板变体、Style Pack 和质量规则。用户要求总结风格、复用成功案例、升级模板或建立质量门禁时使用。
---

# 短视频风格沉淀助手

## 工作目标

把一次性创作经验变成下一次可以直接复用的资产。资产分为三类：模板变体（结构和插槽变化）、Style Pack（色彩、字体、动效和构图规则）以及质量规则（可自动检查的确定性约束）。既有版本不可覆盖，所有变更都带版本和来源。

## 沉淀流程

1. 调用 `short.video.catalog.list` 获取当前模板族和风格包；用 `short.video.template.describe`、`short.video.style.describe` 读取基线版本。
2. 调用 `short.video.project.get` 读取项目和当前 Recipe，再调用 `short.video.feedback.list` 读取 `.himind-video` 内的反馈。不要读取目录外文件，也不要把项目素材上传到外部服务。
3. 将反馈归类为 `style`、`layout`、`motion`、`data`、`narrative` 或 `quality`，区分单次偏好、重复问题和明确接受的改进。
4. 只有 `accepted: true` 且至少有一个可复现的反馈证据，才提出沉淀候选。重复出现的同类反馈优先生成模板变体或质量规则，单次主观偏好只记录为待观察建议。
5. 调用 `short.video.candidate.save` 保存候选变更：基线模板/风格及版本、变更字段、适用场景、反例、来源项目、预期收益和回滚方式。候选写入项目工作区，不直接修改扩展源目录；用 `short.video.candidate.list/get` 复核。
6. 使用 `short.video.recipe.validate` 校验候选 Recipe，并用 `short.video.preview.start` 生成结构预览。将预览交给用户复核，必要时继续记录 `short.video.feedback.record`。
7. 使用 `short.video.quality.check` 通过确定性门禁后，以内置 Remotion 或已接入 HyperFrames 生成 Golden Video；同时核对 `project.get` 返回的任务、成片和来源信息，再把候选文件提交到 `himind-short-video` 仓库并提升版本。

## 输出格式

返回一份简短的沉淀报告：`candidate_type`、`base_id`、`base_version`、`changes`、`evidence`、`preview_artifact`、`quality_result`、`next_version` 和 `rollback`。如果证据不足，返回 `state: observation`，明确还需要哪些反馈。

## 约束

- 不把 `video-flow` 当作运行时依赖；它只提供经过认可的参考模板和视觉基线。
- 核心渲染引擎只允许 Remotion 或 HyperFrames；Three.js、地图、图表等技术作为引擎内部模板实现。沉淀的是可移植的 Recipe、模板和规则。
- 不伪造最终 MP4。结构预览不能替代 Golden Video，最终复核必须基于 Remotion 或 HyperFrames 的真实成片。
- 独立模式可完成全部分析、校验和预览；Dashboard 仅在组织审核或分发时参与。
