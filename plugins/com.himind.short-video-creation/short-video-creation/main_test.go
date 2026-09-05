package main

import (
	"himind-plugin/short-video-creation/internal/himindjsonrpc"
	"os"
	"path/filepath"
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
	}{
		{name: "photo narrative", templateID: "photo-narrative", styleID: "minimal-white", expectedTitle: "图片叙事", expectedSlot: "media"},
		{name: "map story", templateID: "gdp-map-story", styleID: "tech-blue", expectedTitle: "地图数据故事", expectedSlot: "map_data"},
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
			if project.Recipe["template_version"] != "1.0.0" || project.Recipe["style_version"] != "1.0.0" {
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
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"ranking","expected_updated_at":"` + item.UpdatedAt + `","recipe":{"template_id":"leaderboard-tech","template_version":"1.0.0","style_id":"tech-blue","style_version":"1.0.0","adapter":"remotion","title":"2026 年度榜单","duration_seconds":12,"canvas":{"width":1080,"height":1920,"fps":30},"safe_area":{"top":96,"right":72,"bottom":180,"left":72},"data":{"items":[{"name":"A","value":90}]}}}`),
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
		Params: []byte(`{"workspace_root":"` + filepath.ToSlash(workspace) + `","project_id":"renderable","recipe":{"template_id":"leaderboard-tech","template_version":"1.0.0","style_id":"tech-blue","style_version":"1.0.0","adapter":"hyperframes","duration_seconds":12,"canvas":{"width":1080,"height":1920,"fps":30},"safe_area":{"top":96,"right":72,"bottom":180,"left":72},"data":{"items":[{"name":"A","value":90}]}}}`),
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
}
