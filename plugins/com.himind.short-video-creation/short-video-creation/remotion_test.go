package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRuntimeStatusUsesRemotionAsPrimaryRenderer(t *testing.T) {
	result, rpcError := runtimeStatus(requestInput{})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	renderers := result.(map[string]any)["renderers"].([]any)
	renderer := renderers[0].(map[string]any)
	if renderer["id"] != "remotion" || renderer["role"] != "primary" {
		t.Fatalf("unexpected primary renderer: %#v", renderer)
	}
}

func TestDefaultRecipeUsesRemotion(t *testing.T) {
	recipe := defaultRecipe("leaderboard-tech", "tech-blue", nil)
	if recipe["adapter"] != "remotion" {
		t.Fatalf("default adapter must be remotion: %#v", recipe["adapter"])
	}
}

func TestDefaultRecipeUsesTemplateRenderer(t *testing.T) {
	recipe := defaultRecipe("photo-narrative", "minimal-white", nil)
	if recipe["adapter"] != "hyperframes" {
		t.Fatalf("template renderer must be preserved: %#v", recipe["adapter"])
	}
}

func TestCoreRenderersAreLimitedToRemotionAndHyperFrames(t *testing.T) {
	items := loadAdapters()
	if len(items) != 2 || asString(items[0]["id"]) != "remotion" || asString(items[1]["id"]) != "hyperframes" {
		t.Fatalf("unexpected renderer catalog: %#v", items)
	}
}

func TestRemotionBrowserDiscovery(t *testing.T) {
	if runtime.GOOS == "windows" && findRemotionBrowser() == "" {
		t.Fatal("installed Chrome or Edge should be reused by Remotion")
	}
}

func TestRemotionRenderIntegration(t *testing.T) {
	if os.Getenv("HIMIND_REMOTION_INTEGRATION") != "1" {
		t.Skip("set HIMIND_REMOTION_INTEGRATION=1 to run the real renderer")
	}
	root := t.TempDir()
	runtimeRoot := filepath.Join(root, "runtime")
	recipePath := filepath.Join(root, "recipe.json")
	outputPath := filepath.Join(root, "result.mp4")
	recipe := defaultRecipe("leaderboard-tech", "tech-blue", map[string]any{"items": []any{map[string]any{"name": "深圳科技馆", "value": 96}, map[string]any{"name": "数字体验", "value": 88}, map[string]any{"name": "互动展项", "value": 81}}})
	recipe["title"] = "科技体验热度榜"
	recipe["subtitle"] = "基于 HiMind 模板生成"
	recipe["duration_seconds"] = 2
	recipe["canvas"] = map[string]any{"width": 540, "height": 960, "fps": 24}
	if err := writeJSON(recipePath, recipe); err != nil {
		t.Fatal(err)
	}
	if err := ensureRemotionRuntime(runtimeRoot); err != nil {
		t.Fatal(err)
	}
	if err := invokeRemotion(runtimeRoot, recipePath, outputPath, "LeaderboardTech"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() < 10*1024 {
		t.Fatalf("invalid MP4 output: size=%d err=%v", info.Size(), err)
	}
}
