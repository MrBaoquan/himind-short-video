package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"himind-plugin/short-video-creation/internal/himindjsonrpc"
)

const remotionVersion = "4.0.521"

type remotionWorkerSpec struct {
	WorkspaceRoot string `json:"workspace_root"`
	JobPath       string `json:"job_path"`
	RecipePath    string `json:"recipe_path"`
	ArtifactID    string `json:"artifact_id"`
	ArtifactPath  string `json:"artifact_path"`
	RuntimeRoot   string `json:"runtime_root"`
	ProjectID     string `json:"project_id"`
	Composition   string `json:"composition"`
}

func startRemotionRender(root string, item *project, recipe map[string]any) (any, *himindjsonrpc.Error) {
	if _, err := exec.LookPath("node"); err != nil {
		return rendererRuntimeBlocked("Node.js")
	}
	if _, err := lookPathAny("npm.cmd", "npm"); err != nil {
		return rendererRuntimeBlocked("npm")
	}
	jobRoot := filepath.Join(root, metadataDir, jobDir)
	artifactRoot := filepath.Join(root, metadataDir, artifactDir)
	if err := os.MkdirAll(jobRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := os.MkdirAll(artifactRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	jobID := newID("render")
	recipePath := filepath.Join(jobRoot, jobID+".recipe.json")
	if err := writeJSON(recipePath, recipe); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	recipeHash, err := fileSHA256(recipePath)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	artifactID := newID("video-artifact")
	artifactPath := filepath.Join(artifactRoot, artifactID+".mp4")
	jobPath := filepath.Join(jobRoot, jobID+".json")
	jobItem := job{ID: jobID, ProjectID: item.ID, Stage: "render", Status: "queued", Progress: 0, RecipePath: recipePath, RecipeSHA256: recipeHash, Message: "等待 Remotion 渲染", CreatedAt: now(), UpdatedAt: now()}
	if err := writeJSON(jobPath, jobItem); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	templateItem, _ := findTemplate(asString(recipe["template_id"]))
	composition := templateItem.Composition
	if composition == "" {
		composition = "LeaderboardTech"
	}
	spec := remotionWorkerSpec{WorkspaceRoot: root, JobPath: jobPath, RecipePath: recipePath, ArtifactID: artifactID, ArtifactPath: artifactPath, RuntimeRoot: filepath.Join(root, metadataDir, "runtime", "remotion-"+remotionVersion), ProjectID: item.ID, Composition: composition}
	specPath := filepath.Join(jobRoot, jobID+".worker.json")
	if err := writeJSON(specPath, spec); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	logFile, err := os.OpenFile(filepath.Join(jobRoot, jobID+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	command := exec.Command(executable, "--render-job", specPath)
	command.Dir = root
	command.Stdout = logFile
	command.Stderr = logFile
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return nil, himindjsonrpc.InternalError("无法启动 Remotion 渲染任务: " + err.Error())
	}
	_ = logFile.Close()
	jobItem.Status = "preparing"
	jobItem.Progress = 5
	jobItem.WorkerPID = command.Process.Pid
	jobItem.Message = "正在准备 Remotion 运行时"
	jobItem.UpdatedAt = now()
	_ = writeJSON(jobPath, jobItem)
	_ = command.Process.Release()
	return map[string]any{"job": jobItem, "adapter": "remotion", "recipe_path": recipePath, "next": "调用 render.status 查询后台任务；完成后使用 artifact.export 或 artifact.open"}, nil
}

func rendererRuntimeBlocked(name string) (any, *himindjsonrpc.Error) {
	return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("renderer_runtime_missing", "runtime", "Remotion 运行时缺少 "+name, "安装 Node.js LTS（含 npm）后重试 runtime.status 和 render.start")}, "retryable": true}, nil
}

func remotionRuntimeReady(root string) bool {
	path := filepath.Join(root, metadataDir, "runtime", "remotion-"+remotionVersion, "node_modules", "@remotion", "cli", "remotion-cli.js")
	_, err := os.Stat(path)
	return err == nil
}

func runRemotionWorker(specPath string) error {
	var spec remotionWorkerSpec
	if err := readJSON(specPath, &spec); err != nil {
		return err
	}
	root, err := workspace(spec.WorkspaceRoot)
	if err != nil {
		return err
	}
	for _, candidate := range []string{spec.JobPath, spec.RecipePath, spec.ArtifactPath, spec.RuntimeRoot, specPath} {
		if _, err := workspaceFile(root, candidate); err != nil {
			return err
		}
	}
	var jobItem job
	if err := readJSON(spec.JobPath, &jobItem); err != nil {
		return err
	}
	update := func(status string, progress int, message string) {
		jobItem.Status = status
		jobItem.Progress = progress
		jobItem.Message = message
		jobItem.UpdatedAt = now()
		_ = writeJSON(spec.JobPath, jobItem)
	}
	fail := func(err error) error {
		update("failed", 100, "Remotion 渲染失败: "+err.Error())
		return err
	}
	update("preparing", 10, "正在准备固定版本 Remotion 运行时")
	if err := ensureRemotionRuntime(spec.RuntimeRoot); err != nil {
		return fail(err)
	}
	update("rendering", 35, "Remotion 正在生成视频")
	if err := invokeRemotion(spec.RuntimeRoot, spec.RecipePath, spec.ArtifactPath, spec.Composition); err != nil {
		return fail(err)
	}
	artifactItem, err := makeArtifact(spec.ArtifactID, "video/mp4", spec.ArtifactPath, spec.ProjectID, "remotion", true)
	if err != nil {
		return fail(err)
	}
	jobItem.Status = "completed"
	jobItem.Progress = 100
	jobItem.ArtifactID = artifactItem.ID
	jobItem.ArtifactPath = artifactItem.Path
	jobItem.Message = "Remotion 视频已生成"
	jobItem.UpdatedAt = now()
	return writeJSON(spec.JobPath, jobItem)
}

func ensureRemotionRuntime(root string) error {
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	for _, name := range []string{"package.json", "index.jsx"} {
		content, err := os.ReadFile(filepath.Join(pluginRoot(), "renderer", "remotion", name))
		if err != nil {
			return fmt.Errorf("读取 Remotion 资源 %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(root, name), content, 0644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules", "@remotion", "cli", "remotion-cli.js")); err == nil {
		return nil
	}
	npmPath, err := lookPathAny("npm.cmd", "npm")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, npmPath, "install", "--no-audit", "--no-fund")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install: %s", conciseProcessOutput(output, err))
	}
	return nil
}

func invokeRemotion(runtimeRoot, recipePath, outputPath, composition string) error {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return err
	}
	cli := filepath.Join(runtimeRoot, "node_modules", "@remotion", "cli", "remotion-cli.js")
	if strings.TrimSpace(composition) == "" {
		composition = "LeaderboardTech"
	}
	// A single browser page keeps local Agent renders deterministic. Multiple
	// concurrent tabs can exceed Chrome's startup/render budget on first run
	// and surface a misleading delayRender timeout even when the composition
	// itself is valid.
	args := []string{cli, "render", filepath.Join(runtimeRoot, "index.jsx"), composition, outputPath, "--props=" + recipePath, "--codec=h264", "--crf=18", "--concurrency=1", "--log=error", "--overwrite"}
	if browserPath := findRemotionBrowser(); browserPath != "" {
		args = append(args, "--browser-executable="+browserPath)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, nodePath, args...)
	command.Dir = runtimeRoot
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remotion render: %s", conciseProcessOutput(output, err))
	}
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("Remotion 未生成有效 MP4")
	}
	return nil
}

func findRemotionBrowser() string {
	candidates := make([]string, 0, 6)
	switch runtime.GOOS {
	case "windows":
		for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")} {
			if root == "" {
				continue
			}
			candidates = append(candidates, filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"))
		}
	case "darwin":
		candidates = append(candidates, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge")
	default:
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge"} {
			if path, err := exec.LookPath(name); err == nil {
				return path
			}
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func conciseProcessOutput(output []byte, fallback error) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return fallback.Error()
	}
	runes := []rune(text)
	if len(runes) > 800 {
		return string(runes[len(runes)-800:])
	}
	return text
}
