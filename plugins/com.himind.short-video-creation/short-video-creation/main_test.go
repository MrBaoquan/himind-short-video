package main

import (
	"himind-plugin/short-video-creation/internal/himindjsonrpc"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortVideoCreationLoop(t *testing.T) {
	workspace := t.TempDir()
	created, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"leaderboard-demo","template_id":"leaderboard-tech","style_id":"tech-blue"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	project := created.(map[string]any)["project"].(project)
	if project.ID != "leaderboard-demo" {
		t.Fatalf("unexpected project id: %s", project.ID)
	}

	validated, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.recipe.validate",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"leaderboard-demo"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	if !validated.(map[string]any)["valid"].(bool) {
		t.Fatalf("recipe should be valid: %#v", validated)
	}

	preview, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.preview.start",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"leaderboard-demo"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	previewMap := preview.(map[string]any)
	artifact := previewMap["artifact"].(artifact)
	if !artifact.Ready || artifact.Kind != "preview/html" {
		t.Fatalf("unexpected preview artifact: %#v", artifact)
	}
	if _, err := os.Stat(artifact.Path); err != nil {
		t.Fatalf("preview file missing: %v", err)
	}

	feedback, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.feedback.record",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"leaderboard-demo","category":"style","feedback":"标题层级清晰","accepted":true}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	if !feedback.(map[string]any)["recorded"].(bool) {
		t.Fatal("feedback was not recorded")
	}
}

func TestProjectCreateCarriesEditorialFieldsIntoRecipe(t *testing.T) {
	workspace := t.TempDir()
	result, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"年度榜单","template_id":"leaderboard-tech","style_id":"tech-blue","title":"安徽省热门景区\n游客量排行榜","subtitle":"统计周期：2025年1月 - 2025年12月","unit":"万人次","period":"2025年度","source":"安徽省文化和旅游厅（示例数据）","total":"2.04 亿","average":"+11.2%","highlight":"芜湖方特","data":{"items":[{"name":"黄山风景区","value":3860}]}}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	item := result.(map[string]any)["project"].(project)
	expectations := map[string]any{
		"title":     "安徽省热门景区\n游客量排行榜",
		"subtitle":  "统计周期：2025年1月 - 2025年12月",
		"unit":      "万人次",
		"period":    "2025年度",
		"source":    "安徽省文化和旅游厅（示例数据）",
		"total":     "2.04 亿",
		"average":   "+11.2%",
		"highlight": "芜湖方特",
	}
	for key, expected := range expectations {
		if item.Recipe[key] != expected {
			t.Fatalf("recipe %s = %#v, want %#v", key, item.Recipe[key], expected)
		}
	}
}

func TestProjectBriefIsPersistedAndUpdated(t *testing.T) {
	workspace := t.TempDir()
	created, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"brief-demo","template_id":"leaderboard-tech","style_id":"tech-blue","creative_brief":"保持科技蓝基调，突出前三名和数据来源。"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	item := created.(map[string]any)["project"].(project)
	if item.CreativeBrief == "" {
		t.Fatal("creative brief was not returned")
	}
	briefPath := filepath.Join(workspace, metadataDir, projectDir, item.ID, "brief.md")
	brief, err := os.ReadFile(briefPath)
	if err != nil || !strings.Contains(string(brief), "科技蓝") {
		t.Fatalf("brief file was not persisted: %v %q", err, brief)
	}
	updated, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.update",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"brief-demo","creative_brief":"改为更快节奏，保留安全区。"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	if updated.(map[string]any)["project"].(project).CreativeBrief != "改为更快节奏，保留安全区。" {
		t.Fatalf("brief update was not applied: %#v", updated)
	}
}

func TestProjectDuplicatePreservesRecipeAndSupportsDataOverride(t *testing.T) {
	workspace := t.TempDir()
	created, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"rank-base","description":"成熟榜单模板","creative_brief":"使用已经验证的排行榜节奏。","template_id":"leaderboard-tech","style_id":"tech-blue","title":"基准榜单","data":{"items":[{"name":"旧数据","value":80}]}}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	base := created.(map[string]any)["project"].(project)
	duplicated, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.duplicate",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","source_project_id":"` + base.ID + `","name":"rank-2026","title":"2026 年榜单","creative_brief":"换成 2026 年数据，沿用基准动效。","data":{"items":[{"name":"新数据","value":99}]}}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	copyItem := duplicated.(map[string]any)["project"].(project)
	if copyItem.ID == base.ID || copyItem.Template != base.Template || copyItem.Style != base.Style {
		t.Fatalf("duplicate did not preserve project identity fields: %#v", copyItem)
	}
	if copyItem.CreativeBrief != "换成 2026 年数据，沿用基准动效。" || copyItem.Recipe["title"] != "2026 年榜单" {
		t.Fatalf("duplicate editorial fields were not replaced: %#v", copyItem)
	}
	data, ok := copyItem.Recipe["data"].(map[string]any)
	if !ok || len(data["items"].([]any)) != 1 || data["items"].([]any)[0].(map[string]any)["name"] != "新数据" {
		t.Fatalf("duplicate data override failed: %#v", copyItem.Recipe["data"])
	}
	if base.Recipe["title"] != "基准榜单" {
		t.Fatal("duplicate mutated source recipe")
	}
}

func TestProjectDuplicateCanSwitchTemplateAndStyle(t *testing.T) {
	workspace := t.TempDir()
	created, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"switch-base","template_id":"leaderboard-tech","style_id":"tech-blue"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	base := created.(map[string]any)["project"].(project)
	duplicated, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.duplicate",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","source_project_id":"` + base.ID + `","name":"switch-copy","template_id":"photo-narrative","style_id":"minimal-white"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	item := duplicated.(map[string]any)["project"].(project)
	if item.Template != "photo-narrative" || item.Style != "minimal-white" || item.Recipe["adapter"] != "hyperframes" {
		t.Fatalf("template/style switch was not applied: %#v", item)
	}
	if valid, rpcError := handle(himindjsonrpc.Request{Method: "short.video.recipe.validate", Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"switch-copy"}`)}); rpcError != nil || !valid.(map[string]any)["valid"].(bool) {
		t.Fatalf("switched duplicate should remain valid: %#v %v", valid, rpcError)
	}
}

func TestProjectListReturnsCreatedProjects(t *testing.T) {
	workspace := t.TempDir()
	created, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"年度榜单","template_id":"leaderboard-tech","style_id":"tech-blue"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	createdProject := created.(map[string]any)["project"].(project)

	listed, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.list",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	result := listed.(map[string]any)
	if result["workspace_root"] != workspace || result["total"] != 1 {
		t.Fatalf("unexpected project list metadata: %#v", result)
	}
	items := result["items"].([]project)
	if len(items) != 1 || items[0].ID != createdProject.ID || items[0].Name != "年度榜单" {
		t.Fatalf("created project missing from project list: %#v", items)
	}

	// The list must also recover projects persisted by an earlier process.
	recoveredRoot := filepath.Join(workspace, metadataDir, projectDir, "recovered")
	recovered := project{ID: "recovered", Name: "恢复项目", Template: "leaderboard-tech", Style: "tech-blue", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z", Revision: 2, Recipe: defaultRecipe("leaderboard-tech", "tech-blue", nil)}
	if err := writeJSON(filepath.Join(recoveredRoot, "project.json"), recovered); err != nil {
		t.Fatal(err)
	}
	listed, rpcError = handle(himindjsonrpc.Request{
		Method: "short.video.project.list",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	items = listed.(map[string]any)["items"].([]project)
	if len(items) != 2 || items[1].ID != "recovered" {
		t.Fatalf("persisted project was not recovered in stable order: %#v", items)
	}
}

func TestProjectListDiscoversDirectChildWorkspaces(t *testing.T) {
	parent := t.TempDir()
	workspaceA := filepath.Join(parent, "campaign-a")
	workspaceB := filepath.Join(parent, "campaign-b")
	if err := os.MkdirAll(workspaceA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspaceB, 0755); err != nil {
		t.Fatal(err)
	}

	for _, item := range []struct {
		workspace string
		name      string
	}{
		{workspace: workspaceA, name: "A 项目"},
		{workspace: workspaceB, name: "B 项目"},
	} {
		_, rpcError := handle(himindjsonrpc.Request{
			Method: "short.video.project.create",
			Params: []byte(`{"workspace_root":"` + filepath.ToSlash(item.workspace) + `","name":"` + item.name + `","template_id":"leaderboard-tech","style_id":"tech-blue"}`),
		})
		if rpcError != nil {
			t.Fatal(rpcError)
		}
	}

	listed, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.list",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(parent) + `"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	result := listed.(map[string]any)
	items := result["items"].([]project)
	if result["workspace_root"] != parent || result["total"] != 2 || len(items) != 2 {
		t.Fatalf("child workspace projects were not discovered: %#v", result)
	}
	roots := map[string]bool{}
	for _, item := range items {
		roots[item.WorkspaceRoot] = true
	}
	if !roots[workspaceA] || !roots[workspaceB] {
		t.Fatalf("project workspace metadata is incomplete: %#v", items)
	}
	if items[0].Name != "A 项目" || items[1].Name != "B 项目" {
		t.Fatalf("project list ordering is not stable: %#v", items)
	}
}

func TestProjectListDiscoversNestedWorkspacesAndSkipsGeneratedDirectories(t *testing.T) {
	parent := t.TempDir()
	nestedWorkspace := filepath.Join(parent, "test-output", "style-review")
	generatedWorkspace := filepath.Join(parent, "node_modules", "fixture")
	for _, workspace := range []string{nestedWorkspace, generatedWorkspace} {
		if err := os.MkdirAll(workspace, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if _, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(nestedWorkspace) + `","name":"嵌套项目","template_id":"leaderboard-tech","style_id":"tech-blue"}`),
	}); rpcError != nil {
		t.Fatal(rpcError)
	}
	if _, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(generatedWorkspace) + `","name":"依赖目录项目","template_id":"leaderboard-tech","style_id":"tech-blue"}`),
	}); rpcError != nil {
		t.Fatal(rpcError)
	}

	listed, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.list",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(parent) + `"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	result := listed.(map[string]any)
	items := result["items"].([]project)
	if result["total"] != 1 || len(items) != 1 || items[0].Name != "嵌套项目" {
		t.Fatalf("nested project discovery or generated-directory filter failed: %#v", result)
	}
	if items[0].WorkspaceRoot != nestedWorkspace {
		t.Fatalf("nested project workspace metadata is incomplete: %#v", items[0])
	}
}

func TestProjectListReturnsEmptyWorkspaceMetadata(t *testing.T) {
	workspace := t.TempDir()
	listed, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.list",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	result := listed.(map[string]any)
	if result["workspace_root"] != workspace || result["total"] != 0 {
		t.Fatalf("unexpected empty workspace metadata: %#v", result)
	}
	if items, ok := result["items"].([]project); !ok || len(items) != 0 {
		t.Fatalf("empty workspace should return an empty project slice: %#v", result["items"])
	}
}

func TestProjectListUIUsesWorkspaceAwareProjectIdentity(t *testing.T) {
	ui, err := os.ReadFile(filepath.Join("ui", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(ui)
	for _, fragment := range []string{
		"call('short.video.project.list',{workspace_root:root})",
		"const projectKey = project => projectRoot(project)+'|'+project.id",
		"loadProject({id:state.selected,workspace_root:state.selectedWorkspace})",
		"当前目录及子目录暂无项目",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("project manager UI lost workspace-aware behavior: %q", fragment)
		}
	}
}

func TestUnknownRendererIsBlocked(t *testing.T) {
	workspace := t.TempDir()
	result, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.render.start",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","recipe":{"template_id":"leaderboard-tech","style_id":"tech-blue","adapter":"ffmpeg","duration_seconds":12,"canvas":{"width":1080,"height":1920,"fps":30},"safe_area":{"top":96,"right":72,"bottom":180,"left":72},"data":{"items":[{"name":"A","value":80}]}}}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	blocked := result.(map[string]any)
	if blocked["state"] != "blocked" {
		t.Fatalf("unknown renderer must be blocked: %#v", blocked)
	}
}

func TestDefaultRecipesFollowTemplateAndStyle(t *testing.T) {
	tests := []struct {
		name          string
		templateID    string
		styleID       string
		expectedTitle string
		expectedSlot  string
		expectedStyle string
	}{
		{name: "photo narrative", templateID: "photo-narrative", styleID: "minimal-white", expectedTitle: "图片叙事", expectedSlot: "media", expectedStyle: "1.0.0"},
		{name: "map story", templateID: "gdp-map-story", styleID: "tech-blue", expectedTitle: "地图数据故事", expectedSlot: "map_data", expectedStyle: "1.1.0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			result, rpcError := handle(himindjsonrpc.Request{
				Method: "short.video.project.create",
				Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"default-` + test.templateID + `","template_id":"` + test.templateID + `","style_id":"` + test.styleID + `"}`),
			})
			if rpcError != nil {
				t.Fatal(rpcError)
			}
			project := result.(map[string]any)["project"].(project)
			if project.Recipe["title"] != test.expectedTitle {
				t.Fatalf("unexpected title: %#v", project.Recipe["title"])
			}
			if project.Recipe["template_version"] != "1.0.0" || project.Recipe["style_version"] != test.expectedStyle {
				t.Fatalf("template/style versions were not resolved: %#v", project.Recipe)
			}
			data, ok := project.Recipe["data"].(map[string]any)
			if !ok || data[test.expectedSlot] == nil {
				t.Fatalf("default slot %q is missing: %#v", test.expectedSlot, project.Recipe)
			}
			validated, rpcError := handle(himindjsonrpc.Request{
				Method: "short.video.recipe.validate",
				Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"` + project.ID + `"}`),
			})
			if rpcError != nil {
				t.Fatal(rpcError)
			}
			if !validated.(map[string]any)["valid"].(bool) {
				t.Fatalf("default recipe should be valid: %#v", validated)
			}
		})
	}
}

func TestLeaderboardDefaultRecipeCarriesVideoFlowBaseline(t *testing.T) {
	recipe := defaultRecipe("leaderboard-tech", "tech-blue", nil)
	if recipe["title"] != "安徽省热门景区\n游客量排行榜" {
		t.Fatalf("unexpected baseline title: %#v", recipe["title"])
	}
	if recipe["period"] != "2025年度" || recipe["unit"] != "万人次" {
		t.Fatalf("baseline editorial metadata missing: %#v", recipe)
	}
	data, ok := recipe["data"].(map[string]any)
	if !ok {
		t.Fatalf("baseline data missing: %#v", recipe["data"])
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) != 10 {
		t.Fatalf("Video Flow baseline should contain ten ranking items: %#v", data["items"])
	}
}

func TestProjectUpdateAndCandidateLoop(t *testing.T) {
	workspace := t.TempDir()
	created, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"ranking","template_id":"leaderboard-tech","style_id":"tech-blue"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	item := created.(map[string]any)["project"].(project)
	updatedRecipe := item.Recipe
	updatedRecipe["title"] = "2026 年度榜单"
	updated, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.update",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"ranking","expected_updated_at":"` + item.UpdatedAt + `","recipe":{"template_id":"leaderboard-tech","template_version":"1.1.0","style_id":"tech-blue","style_version":"1.1.0","adapter":"remotion","title":"2026 年度榜单","duration_seconds":12,"canvas":{"width":1080,"height":1920,"fps":30},"safe_area":{"top":96,"right":72,"bottom":180,"left":72},"data":{"items":[{"name":"A","value":90}]}}}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	updatedItem := updated.(map[string]any)["project"].(project)
	if updatedItem.Revision != 2 || updatedItem.Recipe["title"] != "2026 年度榜单" {
		t.Fatalf("unexpected updated project: %#v", updatedItem)
	}

	_, rpcError = handle(himindjsonrpc.Request{
		Method: "short.video.feedback.record",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"ranking","category":"style","feedback":"标题更醒目","accepted":true}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	feedback, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.feedback.list",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"ranking","accepted_only":true}`),
	})
	if rpcError != nil || len(feedback.(map[string]any)["items"].([]any)) != 1 {
		t.Fatalf("unexpected feedback: %#v %v", feedback, rpcError)
	}
	candidateResult, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.candidate.save",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"ranking","candidate_type":"style_pack","base_template_id":"leaderboard-tech","base_style_id":"tech-blue","changes":{"typography.title_size":80},"evidence":["标题更醒目"]}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	candidateItem := candidateResult.(map[string]any)["candidate"].(candidate)
	listed, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.candidate.list",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"ranking"}`),
	})
	if rpcError != nil || len(listed.(map[string]any)["items"].([]candidate)) != 1 {
		t.Fatalf("unexpected candidates: %#v %v", listed, rpcError)
	}
	got, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.candidate.get",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","candidate_id":"` + candidateItem.ID + `"}`),
	})
	if rpcError != nil || got.(map[string]any)["candidate"].(candidate).ID != candidateItem.ID {
		t.Fatalf("candidate get failed: %#v %v", got, rpcError)
	}
}

func TestRenderCompleteRegistersArtifactAndRejectsOutsidePath(t *testing.T) {
	workspace := t.TempDir()
	created, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.project.create",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","name":"renderable","template_id":"leaderboard-tech","style_id":"tech-blue"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	item := created.(map[string]any)["project"].(project)
	item.Recipe["adapter"] = "hyperframes"
	started, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.render.start",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"renderable","recipe":{"template_id":"leaderboard-tech","template_version":"1.1.0","style_id":"tech-blue","style_version":"1.1.0","adapter":"hyperframes","duration_seconds":12,"canvas":{"width":1080,"height":1920,"fps":30},"safe_area":{"top":96,"right":72,"bottom":180,"left":72},"data":{"items":[{"name":"A","value":90}]}}}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	jobItem := started.(map[string]any)["job"].(job)
	outside := filepath.Join(t.TempDir(), "video.mp4")
	if err := os.WriteFile(outside, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	blocked, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.render.complete",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","job_id":"` + jobItem.ID + `","status":"completed","artifact_path":"` + filepath.ToSlash(outside) + `"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	if blocked.(map[string]any)["state"] != "blocked" {
		t.Fatalf("outside path must be blocked: %#v", blocked)
	}

	videoPath := filepath.Join(workspace, "output.mp4")
	if err := os.WriteFile(videoPath, []byte("fake mp4"), 0644); err != nil {
		t.Fatal(err)
	}
	completed, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.render.complete",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","job_id":"` + jobItem.ID + `","status":"completed","artifact_path":"` + filepath.ToSlash(videoPath) + `","renderer":"remotion"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	completedJob := completed.(map[string]any)["job"].(job)
	if completedJob.Status != "completed" || completedJob.ArtifactID == "" {
		t.Fatalf("render was not completed: %#v", completed)
	}
	if !strings.Contains(filepath.ToSlash(completedJob.ArtifactPath), "/.himind-video/artifacts/") {
		t.Fatalf("external renderer output was not archived: %s", completedJob.ArtifactPath)
	}
	if _, err := os.Stat(completedJob.ArtifactPath); err != nil {
		t.Fatalf("archived video missing: %v", err)
	}
	exported, rpcError := handle(himindjsonrpc.Request{
		Method: "short.video.artifact.export",
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","artifact_id":"` + completedJob.ArtifactID + `"}`),
	})
	if rpcError != nil {
		t.Fatal(rpcError)
	}
	exportedPath := exported.(map[string]any)["path"].(string)
	if filepath.Clean(exportedPath) != filepath.Clean(completedJob.ArtifactPath) {
		t.Fatalf("export returned a different artifact path: %s", exportedPath)
	}
}
