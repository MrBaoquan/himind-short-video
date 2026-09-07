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

func TestLeaderboardRecipeUsesVideoFlowStyleBaseline(t *testing.T) {
	recipe := defaultRecipe("leaderboard-tech", "tech-blue", map[string]any{
		"items": []any{map[string]any{"name": "A", "value": 96}},
	})
	if recipe["template_version"] != "1.1.0" {
		t.Fatalf("leaderboard template baseline must be version 1.1.0: %#v", recipe["template_version"])
	}
	if recipe["style_version"] != "1.1.0" {
		t.Fatalf("tech-blue style baseline must be version 1.1.0: %#v", recipe["style_version"])
	}
	canvas := recipe["canvas"].(map[string]any)
	if canvas["width"] != 2160 || canvas["height"] != 3840 {
		t.Fatalf("video-flow baseline must use 2160x3840 canvas: %#v", canvas)
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
	if outputDir := os.Getenv("HIMIND_REMOTION_OUTPUT_DIR"); outputDir != "" {
		root = outputDir
		if err := os.MkdirAll(root, 0755); err != nil {
			t.Fatal(err)
		}
	}
	runtimeRoot := filepath.Join(root, "runtime")
	recipePath := filepath.Join(root, "recipe.json")
	outputPath := filepath.Join(root, "result.mp4")
	recipe := defaultRecipe("leaderboard-tech", "tech-blue", map[string]any{"items": []any{
		map[string]any{"name": "黄山风景区", "value": 3860, "sub_value": "+12.5%", "icon": "🏔️"},
		map[string]any{"name": "九华山", "value": 2940, "sub_value": "+8.3%", "icon": "⛩️"},
		map[string]any{"name": "天柱山", "value": 2510, "sub_value": "+15.2%", "icon": "🗻"},
		map[string]any{"name": "西递宏村", "value": 2280, "sub_value": "+6.7%", "icon": "🏘️"},
		map[string]any{"name": "三河古镇", "value": 1950, "sub_value": "+9.1%", "icon": "🏯"},
		map[string]any{"name": "芜湖方特", "value": 1820, "sub_value": "+22.4%", "icon": "🎢"},
		map[string]any{"name": "琅琊山", "value": 1560, "sub_value": "+4.8%", "icon": "🌿"},
		map[string]any{"name": "万佛湖", "value": 1340, "sub_value": "+11.6%", "icon": "🌊"},
		map[string]any{"name": "八里河", "value": 1180, "sub_value": "+7.2%", "icon": "🌸"},
		map[string]any{"name": "太平湖", "value": 980, "sub_value": "+13.9%", "icon": "💧"},
	}})
	recipe["title"] = "安徽省热门景区\n游客量排行榜"
	recipe["subtitle"] = "统计周期：2025年1月 - 2025年12月 · 单位：万人次"
	recipe["period"] = "2025年度"
	recipe["unit"] = "万人次"
	recipe["source"] = "数据来源：安徽省文化和旅游厅（示例数据）"
	recipe["total"] = "2.04 亿"
	recipe["average"] = "+11.2%"
	recipe["highlight"] = "芜湖方特"
	// Keep the integration fixture long enough to show the complete ten-row
	// composition and its summary footer, matching the production baseline.
	recipe["duration_seconds"] = 12
	recipe["canvas"] = map[string]any{"width": 540, "height": 960, "fps": 30}
	recipe["safe_area"] = map[string]any{"top": 81, "right": 66, "bottom": 112, "left": 66}
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
