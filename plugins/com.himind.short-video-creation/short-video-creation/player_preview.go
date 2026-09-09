package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"himind-plugin/short-video-creation/internal/himindjsonrpc"
)

// startPlayerPreview builds a single-file Remotion Player HTML that plays the
// project's recipe with real motion, without waiting for a full MP4 render.
// The player bundle is built once per runtime version and cached at the
// runtime root, so repeat previews are instant after the first build.
func startPlayerPreview(root string, item *project, recipe map[string]any) (any, *himindjsonrpc.Error) {
	if _, err := exec.LookPath("node"); err != nil {
		return playerRuntimeBlocked("Node.js")
	}
	runtimeRoot := filepath.Join(root, metadataDir, "runtime", "remotion-"+remotionVersion)
	if err := ensureRemotionRuntime(runtimeRoot); err != nil {
		return playerRuntimeBlocked("Remotion 运行时未就绪: " + err.Error())
	}
	bundlePath := filepath.Join(runtimeRoot, "player.bundle.js")
	if _, err := os.Stat(bundlePath); err != nil {
		if err := buildPlayerBundle(runtimeRoot, bundlePath); err != nil {
			return playerRuntimeBlocked("Player 打包失败: " + err.Error())
		}
	}
	jobRoot := filepath.Join(root, metadataDir, jobDir)
	artifactRoot := filepath.Join(root, metadataDir, artifactDir)
	if err := os.MkdirAll(jobRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := os.MkdirAll(artifactRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	jobID := newID("preview")
	artifactID := newID("player-artifact")
	artifactPath := filepath.Join(artifactRoot, artifactID+".html")
	if err := writePlayerPreview(artifactPath, bundlePath, recipe); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	templateItem, _ := findTemplate(asString(recipe["template_id"]))
	artifactItem, err := makeArtifact(artifactID, "preview/player", artifactPath, item.ID, "remotion-player", true)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	jobItem := job{ID: jobID, ProjectID: item.ID, Stage: "preview", Status: "completed", Progress: 100, ArtifactID: artifactItem.ID, ArtifactPath: artifactPath, Message: "动效预览已生成，可在工作台直接播放", CreatedAt: now(), UpdatedAt: now()}
	if err := writeJSON(filepath.Join(jobRoot, jobID+".json"), jobItem); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	adapter := "remotion-player"
	if templateItem.Adapter == "hyperframes" {
		adapter = "hyperframes"
	}
	return map[string]any{"job": jobItem, "artifact": artifactItem, "preview": map[string]any{"adapter": adapter, "path": artifactPath, "playable": true}, "next": "记录反馈后可再次调用 preview.player 或 preview.start"}, nil
}

func playerRuntimeBlocked(detail string) (any, *himindjsonrpc.Error) {
	return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("player_runtime_missing", "runtime", "动效预览需要 "+detail, "先调用 preview.start 生成静态故事板，或安装 Node.js LTS（含 npm）后用 preview.player 播放动效")}, "retryable": true}, nil
}

// buildPlayerBundle bundles player.jsx with esbuild into a single browser iife
// script. The runtime already ships esbuild (a @remotion/cli dependency), so
// no extra dependency is required beyond the fixed-version Remotion runtime.
func buildPlayerBundle(runtimeRoot, bundlePath string) error {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return err
	}
	esbuildCli := filepath.Join(runtimeRoot, "node_modules", "esbuild", "bin", "esbuild")
	if _, err := os.Stat(esbuildCli); err != nil {
		return fmt.Errorf("esbuild not found in Remotion runtime")
	}
	entry := filepath.Join(runtimeRoot, "player.jsx")
	args := []string{esbuildCli, entry, "--bundle", "--format=iife", "--platform=browser", "--loader:.jsx=jsx", "--jsx=automatic", "--log-level=warning", "--outfile=" + bundlePath}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, nodePath, args...)
	command.Dir = runtimeRoot
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("esbuild: %s", conciseProcessOutput(output, err))
	}
	info, err := os.Stat(bundlePath)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("esbuild did not produce a player bundle")
	}
	return nil
}

// writePlayerPreview inlines the bundled player script and the preview spec
// into a single HTML document. The spec is embedded as window.__PREVIEW_SPEC__
// so the same file works both inside a srcdoc iframe (no URL query) and when
// opened directly in a browser.
func writePlayerPreview(path, bundlePath string, recipe map[string]any) error {
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		return err
	}
	spec := playerSpecFromRecipe(recipe)
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>")
	buf.WriteString(escHTML(asString(recipe["title"])))
	buf.WriteString("</title><style>html,body{margin:0;height:100%;background:#0a0f1f}body{overflow:hidden}#root{width:100vw;height:100vh}</style></head><body><div id=\"root\"></div><script>window.__PREVIEW_SPEC__ = ")
	buf.Write(specJSON)
	buf.WriteString(";</script><script>")
	buf.Write(bundle)
	buf.WriteString("</script></body></html>")
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// playerSpecFromRecipe maps the Recipe (the same data used for headless
// rendering) onto the props contract the compositions expect. Keeping one data
// source means the on-screen preview matches the final MP4.
func playerSpecFromRecipe(recipe map[string]any) map[string]any {
	spec := map[string]any{
		"composition":      asString(recipe["composition"]),
		"title":            asString(recipe["title"]),
		"subtitle":         asString(recipe["subtitle"]),
		"period":           asString(recipe["period"]),
		"unit":             asString(recipe["unit"]),
		"source":           asString(recipe["source"]),
		"highlight":        asString(recipe["highlight"]),
		"duration_seconds": numberInt(recipe["duration_seconds"]),
	}
	if spec["composition"] == "" {
		if templateItem, ok := findTemplate(asString(recipe["template_id"])); ok {
			spec["composition"] = templateItem.Composition
		}
		if spec["composition"] == "" {
			spec["composition"] = "LeaderboardTech"
		}
	}
	if canvas, ok := recipe["canvas"].(map[string]any); ok {
		spec["canvas"] = canvas
	}
	if safe, ok := recipe["safe_area"].(map[string]any); ok {
		spec["safe_area"] = safe
	}
	spec["style_id"] = asString(recipe["style_id"])
	if data, ok := recipe["data"].(map[string]any); ok {
		if items, ok := data["items"].([]any); ok {
			spec["items"] = items
		}
		spec["data"] = data
	}
	return spec
}