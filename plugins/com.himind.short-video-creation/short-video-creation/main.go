package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"himind-plugin/short-video-creation/internal/himindjsonrpc"
)

const (
	metadataDir = ".himind-video"
	jobDir      = "jobs"
	projectDir  = "projects"
	artifactDir = "artifacts"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,63}$`)

type requestInput struct {
	WorkspaceRoot       string         `json:"workspace_root"`
	ProjectID           string         `json:"project_id"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	ExpectedUpdatedAt   string         `json:"expected_updated_at"`
	CreativeBrief       string         `json:"creative_brief"`
	SourceProjectID     string         `json:"source_project_id"`
	SourceWorkspaceRoot string         `json:"source_workspace_root"`
	TemplateID          string         `json:"template_id"`
	StyleID             string         `json:"style_id"`
	Family              string         `json:"family"`
	Version             string         `json:"version"`
	Title               string         `json:"title"`
	Subtitle            string         `json:"subtitle"`
	Unit                string         `json:"unit"`
	Period              string         `json:"period"`
	Source              string         `json:"source"`
	Total               any            `json:"total"`
	Average             any            `json:"average"`
	Highlight           any            `json:"highlight"`
	Recipe              map[string]any `json:"recipe"`
	Data                map[string]any `json:"data"`
	JobID               string         `json:"job_id"`
	ArtifactID          string         `json:"artifact_id"`
	Category            string         `json:"category"`
	Feedback            string         `json:"feedback"`
	Accepted            *bool          `json:"accepted"`
	AcceptedOnly        bool           `json:"accepted_only"`
	PreviewJobID        string         `json:"preview_job_id"`
	CandidateID         string         `json:"candidate_id"`
	CandidateType       string         `json:"candidate_type"`
	BaseTemplateID      string         `json:"base_template_id"`
	BaseTemplateVersion string         `json:"base_template_version"`
	BaseStyleID         string         `json:"base_style_id"`
	BaseStyleVersion    string         `json:"base_style_version"`
	Changes             map[string]any `json:"changes"`
	Evidence            []any          `json:"evidence"`
	SourceFeedbackIDs   []string       `json:"source_feedback_ids"`
	Status              string         `json:"status"`
	ArtifactPath        string         `json:"artifact_path"`
	ArtifactKind        string         `json:"artifact_kind"`
	Renderer            string         `json:"renderer"`
	Message             string         `json:"message"`
	OpenMode            string         `json:"mode"`
}

type style struct {
	ID          string         `json:"id"`
	Version     string         `json:"version"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Canvas      map[string]any `json:"canvas"`
	Palette     map[string]any `json:"palette"`
	Typography  map[string]any `json:"typography"`
	Motion      map[string]any `json:"motion"`
	Quality     []string       `json:"quality_rules"`
}

type videoTemplate struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	Family          string   `json:"family"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Adapter         string   `json:"adapter"`
	Composition     string   `json:"composition"`
	Slots           []string `json:"slots"`
	CompatibleStyle []string `json:"compatible_styles"`
	DurationSeconds int      `json:"duration_seconds"`
}

type project struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	CreativeBrief string `json:"creative_brief,omitempty"`
	Template      string `json:"template_id"`
	Style         string `json:"style_id"`
	// WorkspaceRoot is response metadata populated by project.list. It is
	// intentionally omitted from project.json so projects stay portable.
	WorkspaceRoot string         `json:"workspace_root,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	Revision      int            `json:"revision"`
	Recipe        map[string]any `json:"recipe"`
}

type job struct {
	ID           string `json:"job_id"`
	ProjectID    string `json:"project_id"`
	Stage        string `json:"stage"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
	ArtifactID   string `json:"artifact_id,omitempty"`
	ArtifactPath string `json:"artifact_path,omitempty"`
	RecipePath   string `json:"recipe_path,omitempty"`
	RecipeSHA256 string `json:"recipe_sha256,omitempty"`
	Message      string `json:"message,omitempty"`
	WorkerPID    int    `json:"worker_pid,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type candidate struct {
	ID                  string         `json:"candidate_id"`
	ProjectID           string         `json:"project_id"`
	CandidateType       string         `json:"candidate_type"`
	BaseTemplateID      string         `json:"base_template_id,omitempty"`
	BaseTemplateVersion string         `json:"base_template_version,omitempty"`
	BaseStyleID         string         `json:"base_style_id,omitempty"`
	BaseStyleVersion    string         `json:"base_style_version,omitempty"`
	Changes             map[string]any `json:"changes"`
	Evidence            []any          `json:"evidence"`
	SourceFeedbackIDs   []string       `json:"source_feedback_ids,omitempty"`
	Status              string         `json:"status"`
	CreatedAt           string         `json:"created_at"`
}

type artifact struct {
	ID        string `json:"artifact_id"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	ProjectID string `json:"project_id"`
	CreatedAt string `json:"created_at"`
	Renderer  string `json:"renderer,omitempty"`
	Ready     bool   `json:"ready"`
}

type previewPage struct {
	ProjectName string
	Template    videoTemplate
	Style       style
	Recipe      map[string]any
	GeneratedAt string
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--render-job" {
		if err := runRemotionWorker(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "short video render worker stopped:", err)
			os.Exit(1)
		}
		return
	}
	if err := himindjsonrpc.Serve(os.Stdin, os.Stdout, handle); err != nil {
		fmt.Fprintln(os.Stderr, "short video plugin stopped:", err)
	}
}

func handle(request himindjsonrpc.Request) (any, *himindjsonrpc.Error) {
	var in requestInput
	if rpcError := himindjsonrpc.DecodeParams(request, &in); rpcError != nil {
		return nil, rpcError
	}
	switch request.Method {
	case "short.video.catalog.list":
		return catalogList(in)
	case "short.video.runtime.status":
		return runtimeStatus(in)
	case "short.video.style.list":
		return styleList()
	case "short.video.style.describe":
		return styleDescribe(in)
	case "short.video.template.list":
		return templateList(in)
	case "short.video.template.describe":
		return templateDescribe(in)
	case "short.video.project.create":
		return projectCreate(in)
	case "short.video.project.duplicate":
		return projectDuplicate(in)
	case "short.video.project.list":
		return projectList(in)
	case "short.video.project.get":
		return projectGet(in)
	case "short.video.project.update":
		return projectUpdate(in)
	case "short.video.recipe.validate":
		return recipeValidate(in)
	case "short.video.preview.start":
		return previewStart(in)
	case "short.video.preview.status":
		return jobStatus(in, "preview")
	case "short.video.preview.cancel":
		return jobCancel(in, "preview")
	case "short.video.feedback.record":
		return feedbackRecord(in)
	case "short.video.feedback.list":
		return feedbackList(in)
	case "short.video.candidate.save":
		return candidateSave(in)
	case "short.video.candidate.list":
		return candidateList(in)
	case "short.video.candidate.get":
		return candidateGet(in)
	case "short.video.quality.check":
		return qualityCheck(in)
	case "short.video.render.start":
		return renderStart(in)
	case "short.video.render.status":
		return jobStatus(in, "render")
	case "short.video.render.cancel":
		return jobCancel(in, "render")
	case "short.video.render.complete":
		return renderComplete(in)
	case "short.video.artifact.export":
		return artifactExport(in)
	case "short.video.artifact.open":
		return artifactOpen(in)
	case "short.video.artifact.read":
		return artifactRead(in)
	default:
		return nil, himindjsonrpc.InvalidParams("unsupported short video capability")
	}
}

func catalogList(in requestInput) (any, *himindjsonrpc.Error) {
	templates := loadTemplates()
	styles := loadStyles()
	if in.Family != "" {
		filtered := templates[:0]
		for _, item := range templates {
			if item.Family == in.Family {
				filtered = append(filtered, item)
			}
		}
		templates = filtered
	}
	return map[string]any{
		"catalog_version": "1.1.0",
		"templates":       templates,
		"styles":          styles,
		"adapters":        adaptersWithRuntimeStatus(),
		"next":            "使用 template.list 和 style.list 选择资产，再调用 project.create",
	}, nil
}

func runtimeStatus(in requestInput) (any, *himindjsonrpc.Error) {
	nodePath, nodeErr := exec.LookPath("node")
	npmPath, npmErr := lookPathAny("npm.cmd", "npm")
	available := nodeErr == nil && npmErr == nil
	state := map[bool]string{true: "available", false: "missing_runtime"}[available]
	initialized := false
	if strings.TrimSpace(in.WorkspaceRoot) != "" {
		if root, err := workspace(in.WorkspaceRoot); err == nil {
			initialized = remotionRuntimeReady(root)
			if available && initialized {
				state = "ready"
			}
		}
	}
	return map[string]any{
		"state": map[bool]string{true: "ready", false: "degraded"}[available],
		"renderers": []any{map[string]any{
			"id": "remotion", "role": "primary", "state": state, "available": available,
			"initialized": initialized, "node_path": nodePath, "npm_path": npmPath, "version": "4.0.521",
		}},
		"adapters": []any{
			map[string]any{"id": "remotion", "role": "primary", "state": state},
			map[string]any{"id": "hyperframes", "role": "pluggable", "state": "external"},
		},
		"limits": map[string]any{"max_duration_seconds": 300, "background_job": true},
	}, nil
}

func adaptersWithRuntimeStatus() []map[string]any {
	items := loadAdapters()
	_, nodeErr := exec.LookPath("node")
	_, npmErr := lookPathAny("npm.cmd", "npm")
	remotionAvailable := nodeErr == nil && npmErr == nil
	for _, item := range items {
		switch asString(item["id"]) {
		case "remotion":
			item["status"] = "built_in"
			item["state"] = map[bool]string{true: "available", false: "missing_runtime"}[remotionAvailable]
			item["available"] = remotionAvailable
			item["version"] = "4.0.521"
		default:
			item["state"] = "external"
		}
	}
	return items
}

func lookPathAny(names ...string) (string, error) {
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("runtime not found")
}

func styleList() (any, *himindjsonrpc.Error) {
	items := loadStyles()
	return map[string]any{"items": items, "total": len(items)}, nil
}

func styleDescribe(in requestInput) (any, *himindjsonrpc.Error) {
	for _, item := range loadStyles() {
		if item.ID == in.StyleID && (in.Version == "" || in.Version == item.Version) {
			return map[string]any{"style": item, "immutable": true}, nil
		}
	}
	return nil, himindjsonrpc.InvalidParams("style_id or version not found")
}

func templateList(in requestInput) (any, *himindjsonrpc.Error) {
	items := loadTemplates()
	filtered := make([]videoTemplate, 0, len(items))
	for _, item := range items {
		if in.Family != "" && item.Family != in.Family {
			continue
		}
		if in.StyleID != "" && !contains(item.CompatibleStyle, in.StyleID) {
			continue
		}
		filtered = append(filtered, item)
	}
	return map[string]any{"items": filtered, "total": len(filtered)}, nil
}

func templateDescribe(in requestInput) (any, *himindjsonrpc.Error) {
	for _, item := range loadTemplates() {
		if item.ID == in.TemplateID && (in.Version == "" || in.Version == item.Version) {
			return map[string]any{"template": item, "immutable": true}, nil
		}
	}
	return nil, himindjsonrpc.InvalidParams("template_id or version not found")
}

func projectCreate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.TemplateID) == "" || strings.TrimSpace(in.StyleID) == "" {
		return nil, himindjsonrpc.InvalidParams("name, template_id and style_id are required")
	}
	if _, ok := findTemplate(in.TemplateID); !ok {
		return nil, himindjsonrpc.InvalidParams("unknown template_id")
	}
	if _, ok := findStyle(in.StyleID); !ok {
		return nil, himindjsonrpc.InvalidParams("unknown style_id")
	}
	id := slug(in.Name)
	if !idPattern.MatchString(id) {
		// Keep the user-facing name intact while giving non-ASCII names a
		// deterministic-safe fallback ID for the on-disk project directory.
		id = fmt.Sprintf("video-project-%d", time.Now().UTC().UnixNano())
	}
	projectRoot := filepath.Join(root, metadataDir, projectDir, id)
	if _, statErr := os.Stat(projectRoot); statErr == nil {
		return nil, himindjsonrpc.InvalidParams("project already exists; use its project_id")
	}
	recipe := defaultRecipe(in.TemplateID, in.StyleID, in.Data)
	applyCreativeFields(recipe, in)
	nowValue := now()
	item := project{ID: id, Name: in.Name, Description: in.Description, CreativeBrief: strings.TrimSpace(in.CreativeBrief), Template: in.TemplateID, Style: in.StyleID, CreatedAt: nowValue, UpdatedAt: nowValue, Revision: 1, Recipe: recipe}
	persisted := item
	persisted.WorkspaceRoot = ""
	if err := writeJSON(filepath.Join(projectRoot, "project.json"), persisted); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := writeJSON(filepath.Join(projectRoot, "recipe.json"), recipe); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if item.CreativeBrief != "" {
		if err := os.WriteFile(filepath.Join(projectRoot, "brief.md"), []byte(item.CreativeBrief+"\n"), 0644); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	return map[string]any{
		"project":        item,
		"workspace_root": root,
		"project_path":   projectRoot,
		"next":           []string{"调用 recipe.validate", "调用 preview.start", "根据反馈修改 Recipe 后再次预览", "调用 quality.check 和 render.start"},
	}, nil
}

// projectDuplicate creates a new project from a proven project/Recipe. It is
// intentionally a capability rather than a UI-only shortcut so an AI tool can
// repeat the same template-first production workflow without copying files.
func projectDuplicate(in requestInput) (any, *himindjsonrpc.Error) {
	targetRoot, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.SourceProjectID) {
		return nil, himindjsonrpc.InvalidParams("source_project_id is required")
	}
	sourceRoot := targetRoot
	if strings.TrimSpace(in.SourceWorkspaceRoot) != "" {
		sourceRoot, err = workspace(in.SourceWorkspaceRoot)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams(err.Error())
		}
	}
	source, err := loadProject(sourceRoot, in.SourceProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = source.Name + " 副本"
	}
	id := slug(name)
	if id == "" || !idPattern.MatchString(id) {
		id = source.ID + "-copy"
	}
	if id == source.ID && filepath.Clean(targetRoot) == filepath.Clean(sourceRoot) {
		id = source.ID + "-copy"
	}
	if len(id) > 64 {
		id = id[:64]
	}
	projectPath := filepath.Join(targetRoot, metadataDir, projectDir, id)
	if _, statErr := os.Stat(projectPath); statErr == nil {
		return nil, himindjsonrpc.InvalidParams("duplicated project already exists; choose another name")
	}

	templateID := source.Template
	if strings.TrimSpace(in.TemplateID) != "" {
		templateID = in.TemplateID
	}
	styleID := source.Style
	if strings.TrimSpace(in.StyleID) != "" {
		styleID = in.StyleID
	}
	templateItem, ok := findTemplate(templateID)
	if !ok {
		return nil, himindjsonrpc.InvalidParams("unknown template_id")
	}
	styleItem, ok := findStyle(styleID)
	if !ok {
		return nil, himindjsonrpc.InvalidParams("unknown style_id")
	}
	recipe, err := cloneMap(source.Recipe)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	// A template/style switch keeps editorial data but refreshes the renderer,
	// canvas and immutable asset versions from the selected catalog entries.
	base := defaultRecipe(templateID, styleID, nil)
	recipe["template_id"] = templateID
	recipe["template_version"] = templateItem.Version
	recipe["style_id"] = styleID
	recipe["style_version"] = styleItem.Version
	if templateID != source.Template {
		for _, key := range []string{"adapter", "duration_seconds"} {
			recipe[key] = base[key]
		}
		if in.Data == nil {
			recipe["data"] = base["data"]
		}
	}
	if styleID != source.Style {
		recipe["canvas"] = base["canvas"]
		recipe["safe_area"] = base["safe_area"]
	}
	if in.Data != nil {
		recipe["data"] = in.Data
	}
	applyCreativeFields(recipe, in)
	brief := source.CreativeBrief
	if strings.TrimSpace(in.CreativeBrief) != "" {
		brief = strings.TrimSpace(in.CreativeBrief)
	}
	nowValue := now()
	item := project{ID: id, Name: name, Description: source.Description, CreativeBrief: brief, Template: templateID, Style: styleID, CreatedAt: nowValue, UpdatedAt: nowValue, Revision: 1, Recipe: recipe}
	if strings.TrimSpace(in.Description) != "" {
		item.Description = in.Description
	}
	if err := writeJSON(filepath.Join(projectPath, "project.json"), item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := writeJSON(filepath.Join(projectPath, "recipe.json"), recipe); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if brief != "" {
		if err := os.WriteFile(filepath.Join(projectPath, "brief.md"), []byte(brief+"\n"), 0644); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	return map[string]any{
		"project": item, "workspace_root": targetRoot, "project_path": projectPath,
		"source_project_id": source.ID, "source_workspace_root": sourceRoot,
		"next": []string{"调用 recipe.validate", "调用 preview.start", "根据反馈更新 Recipe 后再次预览", "调用 quality.check 和 render.start"},
	}, nil
}

// applyCreativeFields keeps the project API ergonomic: callers can provide
// common editorial fields without reconstructing the full versioned Recipe.
// Advanced callers can still use project.update with an explicit Recipe.
func applyCreativeFields(recipe map[string]any, in requestInput) {
	for key, value := range map[string]any{
		"title": in.Title, "subtitle": in.Subtitle, "unit": in.Unit,
		"period": in.Period, "source": in.Source, "total": in.Total,
		"average": in.Average, "highlight": in.Highlight,
	} {
		switch value := value.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				recipe[key] = value
			}
		case nil:
			continue
		default:
			recipe[key] = value
		}
	}
}

func projectList(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	items := make([]project, 0)
	projectRoots := discoverProjectRoots(root)
	for _, projectRoot := range projectRoots {
		entries, readErr := os.ReadDir(filepath.Join(projectRoot, metadataDir, projectDir))
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return nil, himindjsonrpc.InternalError(readErr.Error())
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			var item project
			if err := readJSON(filepath.Join(projectRoot, metadataDir, projectDir, entry.Name(), "project.json"), &item); err == nil {
				if item.ID == "" {
					item.ID = entry.Name()
				}
				item.WorkspaceRoot = projectRoot
				items = append(items, item)
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].UpdatedAt > items[j].UpdatedAt ||
			(items[i].UpdatedAt == items[j].UpdatedAt && (items[i].Name < items[j].Name ||
				(items[i].Name == items[j].Name && (items[i].WorkspaceRoot < items[j].WorkspaceRoot ||
					(items[i].WorkspaceRoot == items[j].WorkspaceRoot && items[i].ID < items[j].ID)))))
	})
	workspaceRoots := make([]string, 0, len(projectRoots))
	for _, projectRoot := range projectRoots {
		workspaceRoots = append(workspaceRoots, projectRoot)
	}
	return map[string]any{"items": items, "total": len(items), "workspace_root": root, "workspace_roots": workspaceRoots}, nil
}

const projectDiscoveryMaxDepth = 4

var projectDiscoveryIgnoredDirs = map[string]bool{
	".git":          true,
	".himind-video": true,
	".cache":        true,
	".remotion":     true,
	"node_modules":  true,
	"dist":          true,
	"target":        true,
	"build":         true,
	"out":           true,
}

// discoverProjectRoots supports aggregate extension repositories where a
// project may live a few levels below the selected root (for example
// test-output/style-review). The depth limit and generated-directory filter
// keep project.list deterministic and cheap without searching the whole disk.
func discoverProjectRoots(root string) []string {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		absoluteRoot = root
	}
	roots := []string{absoluteRoot}
	seen := map[string]bool{absoluteRoot: true}
	var visit func(string, int)
	visit = func(current string, depth int) {
		if depth >= projectDiscoveryMaxDepth {
			return
		}
		entries, readErr := os.ReadDir(current)
		if readErr != nil {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || projectDiscoveryIgnoredDirs[entry.Name()] {
				continue
			}
			candidate := filepath.Join(current, entry.Name())
			candidate, err = filepath.Abs(candidate)
			if err != nil || seen[candidate] {
				continue
			}
			seen[candidate] = true
			if info, statErr := os.Stat(filepath.Join(candidate, metadataDir, projectDir)); statErr == nil && info.IsDir() {
				roots = append(roots, candidate)
			}
			visit(candidate, depth+1)
		}
	}
	visit(absoluteRoot, 0)
	sort.Strings(roots[1:])
	return roots
}

func projectGet(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ProjectID) {
		return nil, himindjsonrpc.InvalidParams("project_id is required")
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	jobs := projectJobs(root, item.ID)
	artifacts := projectArtifacts(root, item.ID)
	return map[string]any{
		"project":        item,
		"workspace_root": root,
		"project_path":   filepath.Join(root, metadataDir, projectDir, item.ID),
		"jobs":           jobs,
		"artifacts":      artifacts,
		"latest_job":     latestJob(jobs),
		"latest_video":   latestArtifact(artifacts, "video/"),
		"latest_preview": latestArtifact(artifacts, "preview/"),
	}, nil
}

func projectJobs(root, projectID string) []job {
	entries, err := os.ReadDir(filepath.Join(root, metadataDir, jobDir))
	if err != nil {
		return []job{}
	}
	items := make([]job, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || strings.HasSuffix(entry.Name(), ".recipe.json") {
			continue
		}
		var item job
		if readJSON(filepath.Join(root, metadataDir, jobDir, entry.Name()), &item) == nil && item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
	return items
}

func projectArtifacts(root, projectID string) []artifact {
	entries, err := os.ReadDir(filepath.Join(root, metadataDir, artifactDir))
	if err != nil {
		return []artifact{}
	}
	items := make([]artifact, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var item artifact
		if readJSON(filepath.Join(root, metadataDir, artifactDir, entry.Name()), &item) == nil && item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	return items
}

func latestJob(items []job) any {
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func latestArtifact(items []artifact, prefix string) any {
	for _, item := range items {
		if strings.HasPrefix(item.Kind, prefix) && item.Ready {
			return item
		}
	}
	return nil
}

func projectUpdate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ProjectID) {
		return nil, himindjsonrpc.InvalidParams("project_id is required")
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if in.ExpectedUpdatedAt != "" && in.ExpectedUpdatedAt != item.UpdatedAt {
		return map[string]any{
			"state":     "blocked",
			"blockers":  []map[string]any{issue("project_stale", "project", "项目已被其他会话更新", "重新调用 project.get 获取最新项目后重试")},
			"retryable": true,
			"project":   item,
		}, nil
	}
	if in.Name != "" {
		item.Name = in.Name
	}
	if in.Description != "" {
		item.Description = in.Description
	}
	if strings.TrimSpace(in.CreativeBrief) != "" {
		item.CreativeBrief = strings.TrimSpace(in.CreativeBrief)
	}
	if in.Recipe != nil {
		issues := validateRecipe(in.Recipe, &item)
		if len(issues) > 0 {
			return map[string]any{"state": "blocked", "blockers": issues, "next_steps": []string{"修复 Recipe 后重新调用 project.update"}}, nil
		}
		item.Recipe = in.Recipe
		item.Template = asString(in.Recipe["template_id"])
		item.Style = asString(in.Recipe["style_id"])
	}
	if in.Name == "" && in.Description == "" && in.CreativeBrief == "" && in.Recipe == nil {
		return nil, himindjsonrpc.InvalidParams("至少提供 name、description、creative_brief 或 recipe")
	}
	item.Revision++
	item.UpdatedAt = now()
	projectPath := filepath.Join(root, metadataDir, projectDir, item.ID)
	if err := writeJSON(filepath.Join(projectPath, "project.json"), item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if in.Recipe != nil {
		if err := writeJSON(filepath.Join(projectPath, "recipe.json"), item.Recipe); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	if item.CreativeBrief != "" {
		if err := os.WriteFile(filepath.Join(projectPath, "brief.md"), []byte(item.CreativeBrief+"\n"), 0644); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	return map[string]any{"project": item, "workspace_root": root, "updated": true}, nil
}

func recipeValidate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	recipe, item, err := resolveRecipe(root, in)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validateRecipe(recipe, item)
	result := map[string]any{"valid": len(issues) == 0, "issues": issues, "recipe": recipe}
	if item != nil {
		result["project_id"] = item.ID
	}
	return result, nil
}

func previewStart(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	recipe, item, err := resolveRecipe(root, in)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validateRecipe(recipe, item)
	if len(issues) > 0 {
		return map[string]any{"state": "blocked", "blockers": issues, "next_steps": []string{"修复 Recipe 后重新调用 preview.start"}}, nil
	}
	templateItem, _ := findTemplate(asString(recipe["template_id"]))
	styleItem, _ := findStyle(asString(recipe["style_id"]))
	jobID := newID("preview")
	jobRoot := filepath.Join(root, metadataDir, jobDir)
	artifactRoot := filepath.Join(root, metadataDir, artifactDir)
	if err := os.MkdirAll(jobRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if err := os.MkdirAll(artifactRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	artifactID := newID("preview-artifact")
	artifactPath := filepath.Join(artifactRoot, artifactID+".html")
	page := previewPage{ProjectName: item.Name, Template: templateItem, Style: styleItem, Recipe: recipe, GeneratedAt: now()}
	if err := writePreview(artifactPath, page); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	artifactItem, err := makeArtifact(artifactID, "preview/html", artifactPath, item.ID, "storyboard", true)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	jobItem := job{ID: jobID, ProjectID: item.ID, Stage: "preview", Status: "completed", Progress: 100, ArtifactID: artifactItem.ID, ArtifactPath: artifactPath, Message: "故事板预览已生成，可在浏览器中打开 path", CreatedAt: now(), UpdatedAt: now()}
	if err := writeJSON(filepath.Join(jobRoot, jobID+".json"), jobItem); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"job": jobItem, "artifact": artifactItem, "preview": map[string]any{"adapter": "storyboard", "path": artifactPath, "openable": true}, "next": "记录反馈后可再次调用 preview.start"}, nil
}

func jobStatus(in requestInput, stage string) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.JobID) {
		return nil, himindjsonrpc.InvalidParams("job_id is required")
	}
	var item job
	if err := readJSON(filepath.Join(root, metadataDir, jobDir, in.JobID+".json"), &item); err != nil {
		return nil, himindjsonrpc.InvalidParams("job not found")
	}
	if item.Stage != stage {
		return nil, himindjsonrpc.InvalidParams("job stage mismatch")
	}
	return map[string]any{"job": item}, nil
}

func jobCancel(in requestInput, stage string) (any, *himindjsonrpc.Error) {
	result, rpcError := jobStatus(in, stage)
	if rpcError != nil {
		return nil, rpcError
	}
	item := result.(map[string]any)["job"].(job)
	if item.Status == "completed" || item.Status == "failed" {
		return map[string]any{"job": item, "cancelled": false, "message": "任务已结束，无需取消"}, nil
	}
	item.Status = "cancelled"
	item.UpdatedAt = now()
	root, _ := workspace(in.WorkspaceRoot)
	if err := writeJSON(filepath.Join(root, metadataDir, jobDir, in.JobID+".json"), item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"job": item, "cancelled": true}, nil
}

func feedbackRecord(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ProjectID) || strings.TrimSpace(in.Category) == "" || strings.TrimSpace(in.Feedback) == "" {
		return nil, himindjsonrpc.InvalidParams("project_id, category and feedback are required")
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !contains([]string{"style", "layout", "motion", "data", "narrative", "quality"}, in.Category) {
		return nil, himindjsonrpc.InvalidParams("category must be one of style, layout, motion, data, narrative, quality")
	}
	if in.PreviewJobID != "" {
		var preview job
		if err := readJSON(filepath.Join(root, metadataDir, jobDir, in.PreviewJobID+".json"), &preview); err != nil || preview.Stage != "preview" || preview.ProjectID != item.ID {
			return nil, himindjsonrpc.InvalidParams("preview_job_id does not belong to project")
		}
	}
	accepted := false
	if in.Accepted != nil {
		accepted = *in.Accepted
	}
	entry := map[string]any{"id": newID("feedback"), "project_id": item.ID, "category": in.Category, "feedback": in.Feedback, "accepted": accepted, "preview_job_id": in.PreviewJobID, "created_at": now()}
	feedbackPath := filepath.Join(root, metadataDir, projectDir, item.ID, "feedback.jsonl")
	if err := appendJSONLine(feedbackPath, entry); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"recorded": true, "feedback": entry, "promotion": map[string]any{"eligible": accepted, "next": "重复反馈确认后交由 video-style-curator 生成模板变体候选"}}, nil
}

func feedbackList(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ProjectID) {
		return nil, himindjsonrpc.InvalidParams("project_id is required")
	}
	if _, err := loadProject(root, in.ProjectID); err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path := filepath.Join(root, metadataDir, projectDir, in.ProjectID, "feedback.jsonl")
	data, readErr := os.ReadFile(path)
	if os.IsNotExist(readErr) {
		return map[string]any{"items": []any{}, "total": 0, "project_id": in.ProjectID}, nil
	}
	if readErr != nil {
		return nil, himindjsonrpc.InternalError(readErr.Error())
	}
	items := make([]any, 0)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return map[string]any{
				"state":     "blocked",
				"blockers":  []map[string]any{issue("feedback_corrupt", "feedback", "反馈记录包含无效 JSON", "修复 feedback.jsonl 后重试")},
				"retryable": false,
			}, nil
		}
		if in.AcceptedOnly && entry["accepted"] != true {
			continue
		}
		items = append(items, entry)
	}
	return map[string]any{"items": items, "total": len(items), "project_id": in.ProjectID}, nil
}

func candidateSave(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ProjectID) || strings.TrimSpace(in.CandidateType) == "" {
		return nil, himindjsonrpc.InvalidParams("project_id and candidate_type are required")
	}
	if _, err := loadProject(root, in.ProjectID); err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !contains([]string{"template_variant", "style_pack", "quality_rule"}, in.CandidateType) {
		return nil, himindjsonrpc.InvalidParams("candidate_type must be template_variant, style_pack or quality_rule")
	}
	if len(in.Changes) == 0 {
		return nil, himindjsonrpc.InvalidParams("changes is required")
	}
	if len(in.Evidence) == 0 {
		return nil, himindjsonrpc.InvalidParams("evidence is required")
	}
	if in.BaseTemplateID != "" {
		templateItem, ok := findTemplate(in.BaseTemplateID)
		if !ok || (in.BaseTemplateVersion != "" && in.BaseTemplateVersion != templateItem.Version) {
			return nil, himindjsonrpc.InvalidParams("base_template_id or version not found")
		}
	}
	if in.BaseStyleID != "" {
		styleItem, ok := findStyle(in.BaseStyleID)
		if !ok || (in.BaseStyleVersion != "" && in.BaseStyleVersion != styleItem.Version) {
			return nil, himindjsonrpc.InvalidParams("base_style_id or version not found")
		}
	}
	item := candidate{
		ID:                  newID("candidate"),
		ProjectID:           in.ProjectID,
		CandidateType:       in.CandidateType,
		BaseTemplateID:      in.BaseTemplateID,
		BaseTemplateVersion: in.BaseTemplateVersion,
		BaseStyleID:         in.BaseStyleID,
		BaseStyleVersion:    in.BaseStyleVersion,
		Changes:             in.Changes,
		Evidence:            in.Evidence,
		SourceFeedbackIDs:   in.SourceFeedbackIDs,
		Status:              "proposed",
		CreatedAt:           now(),
	}
	path := filepath.Join(root, metadataDir, "candidates", item.ID+".json")
	if err := writeJSON(path, item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"candidate":      item,
		"candidate_path": path,
		"next":           []string{"调用 candidate.get 或 candidate.list 复核", "生成 Golden Preview 并通过 quality.check 后，再把候选变更提交到扩展源仓库"},
	}, nil
}

func candidateList(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	entries, readErr := os.ReadDir(filepath.Join(root, metadataDir, "candidates"))
	if readErr != nil && !os.IsNotExist(readErr) {
		return nil, himindjsonrpc.InternalError(readErr.Error())
	}
	items := make([]candidate, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var item candidate
		if err := readJSON(filepath.Join(root, metadataDir, "candidates", entry.Name()), &item); err != nil {
			continue
		}
		if in.ProjectID != "" && item.ProjectID != in.ProjectID {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	return map[string]any{"items": items, "total": len(items)}, nil
}

func candidateGet(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.CandidateID) {
		return nil, himindjsonrpc.InvalidParams("candidate_id is required")
	}
	var item candidate
	path := filepath.Join(root, metadataDir, "candidates", in.CandidateID+".json")
	if err := readJSON(path, &item); err != nil {
		return nil, himindjsonrpc.InvalidParams("candidate not found")
	}
	return map[string]any{"candidate": item, "candidate_path": path}, nil
}

func qualityCheck(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	recipe, item, err := resolveRecipe(root, in)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validateRecipe(recipe, item)
	checks := []map[string]any{
		{"id": "template", "passed": findTemplateValue(recipe["template_id"])},
		{"id": "style", "passed": findStyleValue(recipe["style_id"])},
		{"id": "duration", "passed": numberValue(recipe["duration_seconds"]) >= 1 && numberValue(recipe["duration_seconds"]) <= 300},
		{"id": "canvas", "passed": canvasValid(recipe)},
		{"id": "data", "passed": dataPresent(recipe)},
		{"id": "safe_area", "passed": safeAreaValid(recipe)},
	}
	passed := len(issues) == 0
	for _, check := range checks {
		if !check["passed"].(bool) {
			passed = false
		}
	}
	return map[string]any{"passed": passed, "checks": checks, "issues": issues, "project_id": item.ID, "next": map[string]any{"render": "passed 时可调用 render.start", "fix": "失败时修改 Recipe 后重新校验"}}, nil
}

func renderStart(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	recipe, item, err := resolveRecipe(root, in)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	issues := validateRecipe(recipe, item)
	if len(issues) > 0 {
		return map[string]any{"state": "blocked", "blockers": issues, "next_steps": []string{"先完成 recipe.validate 和 quality.check"}}, nil
	}
	adapter := asString(recipe["adapter"])
	if adapter == "" {
		adapter = "remotion"
	}
	if adapter == "remotion" {
		return startRemotionRender(root, item, recipe)
	}
	jobID := newID("render")
	jobRoot := filepath.Join(root, metadataDir, jobDir)
	if err := os.MkdirAll(jobRoot, 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	recipePath := filepath.Join(jobRoot, jobID+".recipe.json")
	if err := writeJSON(recipePath, recipe); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	recipeHash, err := fileSHA256(recipePath)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	jobItem := job{ID: jobID, ProjectID: item.ID, Stage: "render", Status: "waiting_adapter", Progress: 0, RecipePath: recipePath, RecipeSHA256: recipeHash, Message: "等待已注册的渲染适配器接管", CreatedAt: now(), UpdatedAt: now()}
	if err := writeJSON(filepath.Join(jobRoot, jobID+".json"), jobItem); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"job": jobItem, "adapter": adapter, "recipe_path": recipePath, "next": "由对应 Renderer Adapter 读取 recipe_path 生成视频，再调用 render.complete 回写产物"}, nil
}

func renderComplete(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.JobID) {
		return nil, himindjsonrpc.InvalidParams("job_id is required")
	}
	var item job
	jobPath := filepath.Join(root, metadataDir, jobDir, in.JobID+".json")
	if err := readJSON(jobPath, &item); err != nil || item.Stage != "render" {
		return nil, himindjsonrpc.InvalidParams("render job not found")
	}
	if item.Status == "completed" || item.Status == "failed" || item.Status == "cancelled" {
		return map[string]any{"job": item, "updated": false, "message": "任务已结束"}, nil
	}
	status := in.Status
	if status == "" {
		status = "completed"
	}
	if status != "completed" && status != "failed" {
		return nil, himindjsonrpc.InvalidParams("status must be completed or failed")
	}
	if status == "failed" {
		if strings.TrimSpace(in.Message) == "" {
			return nil, himindjsonrpc.InvalidParams("message is required for failed render")
		}
		item.Status = status
		item.Progress = 100
		item.Message = in.Message
		item.UpdatedAt = now()
		if err := writeJSON(jobPath, item); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
		return map[string]any{"job": item, "updated": true}, nil
	}
	if strings.TrimSpace(in.ArtifactPath) == "" {
		return nil, himindjsonrpc.InvalidParams("artifact_path is required for completed render")
	}
	artifactPath, err := workspaceFile(root, in.ArtifactPath)
	if err != nil {
		return map[string]any{
			"state":     "blocked",
			"blockers":  []map[string]any{issue("artifact_outside_workspace", "artifact", err.Error(), "将渲染产物写入 workspace_root 内后重试")},
			"retryable": false,
		}, nil
	}
	info, err := os.Stat(artifactPath)
	if err != nil || info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("artifact_path must be an existing file inside workspace_root")
	}
	kind := in.ArtifactKind
	if kind == "" {
		kind = "video/mp4"
	}
	if !strings.HasPrefix(kind, "video/") {
		return nil, himindjsonrpc.InvalidParams("artifact_kind must start with video/")
	}
	artifactID := newID("video-artifact")
	archivedPath, err := archiveVideoArtifact(root, artifactID, kind, artifactPath)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	artifactItem, err := makeArtifact(artifactID, kind, archivedPath, item.ProjectID, in.Renderer, true)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	item.Status = "completed"
	item.Progress = 100
	item.ArtifactID = artifactItem.ID
	item.ArtifactPath = archivedPath
	item.Message = "最终视频已生成"
	item.UpdatedAt = now()
	if err := writeJSON(jobPath, item); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{"job": item, "artifact": artifactItem, "updated": true}, nil
}

func artifactExport(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ArtifactID) {
		return nil, himindjsonrpc.InvalidParams("artifact_id is required")
	}
	entries, _ := os.ReadDir(filepath.Join(root, metadataDir, artifactDir))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".json") || !strings.HasPrefix(entry.Name(), in.ArtifactID+".") {
			continue
		}
		path := filepath.Join(root, metadataDir, artifactDir, entry.Name())
		var item artifact
		if readErr := readJSON(path+".json", &item); readErr == nil {
			return map[string]any{"artifact": item, "path": path, "exported": true}, nil
		}
		return map[string]any{"artifact_id": in.ArtifactID, "path": path, "exported": true}, nil
	}
	return nil, himindjsonrpc.InvalidParams("artifact not found")
}

func artifactOpen(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ArtifactID) {
		return nil, himindjsonrpc.InvalidParams("artifact_id is required")
	}
	var item artifact
	entries, readErr := os.ReadDir(filepath.Join(root, metadataDir, artifactDir))
	if readErr != nil {
		return nil, himindjsonrpc.InvalidParams("artifact not found")
	}
	found := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var candidateItem artifact
		if readJSON(filepath.Join(root, metadataDir, artifactDir, entry.Name()), &candidateItem) == nil && candidateItem.ID == in.ArtifactID {
			item = candidateItem
			found = true
			break
		}
	}
	if !found || !item.Ready {
		return nil, himindjsonrpc.InvalidParams("artifact not found or not ready")
	}
	artifactPath, err := workspaceFile(root, item.Path)
	if err != nil {
		return map[string]any{
			"state":     "blocked",
			"blockers":  []map[string]any{issue("artifact_outside_workspace", "artifact", err.Error(), "只能打开当前工作区内已登记的产物")},
			"retryable": false,
		}, nil
	}
	if info, statErr := os.Stat(artifactPath); statErr != nil || info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("artifact file is missing")
	}
	mode := strings.ToLower(strings.TrimSpace(in.OpenMode))
	if mode == "" {
		mode = "file"
	}
	if mode != "file" && mode != "folder" {
		return nil, himindjsonrpc.InvalidParams("mode must be file or folder")
	}
	target := artifactPath
	if mode == "folder" {
		target = filepath.Dir(artifactPath)
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("explorer.exe", target)
	case "darwin":
		command = exec.Command("open", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	if err := command.Start(); err != nil {
		return map[string]any{
			"state":     "blocked",
			"blockers":  []map[string]any{issue("artifact_open_failed", "artifact", "系统未能打开产物", "确认当前系统存在文件管理器并手动打开返回的 path")},
			"retryable": true,
			"artifact":  item,
			"path":      artifactPath,
		}, nil
	}
	return map[string]any{"opened": true, "mode": mode, "artifact": item, "path": artifactPath}, nil
}

// artifactRead returns artifact content as base64 for inline preview/playback.
// Videos are streamed as a Blob URL on the web side, so keep a size cap to
// avoid moving unbounded payloads over the JSON-RPC pipe.
func artifactRead(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if !idPattern.MatchString(in.ArtifactID) {
		return nil, himindjsonrpc.InvalidParams("artifact_id is required")
	}
	var item artifact
	entries, readErr := os.ReadDir(filepath.Join(root, metadataDir, artifactDir))
	if readErr != nil {
		return nil, himindjsonrpc.InvalidParams("artifact not found")
	}
	found := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var candidateItem artifact
		if readJSON(filepath.Join(root, metadataDir, artifactDir, entry.Name()), &candidateItem) == nil && candidateItem.ID == in.ArtifactID {
			item = candidateItem
			found = true
			break
		}
	}
	if !found || !item.Ready {
		return nil, himindjsonrpc.InvalidParams("artifact not found or not ready")
	}
	artifactPath, err := workspaceFile(root, item.Path)
	if err != nil {
		return map[string]any{
			"state":     "blocked",
			"blockers":  []map[string]any{issue("artifact_outside_workspace", "artifact", err.Error(), "只能读取当前工作区内已登记的产物")},
			"retryable": false,
		}, nil
	}
	info, statErr := os.Stat(artifactPath)
	if statErr != nil || info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("artifact file is missing")
	}
	const maxInlineBytes = 64 << 20
	if info.Size() > maxInlineBytes {
		return map[string]any{
			"state":     "blocked",
			"blockers":  []map[string]any{issue("artifact_too_large", "artifact", "产物超过 64MB，无法内嵌播放", "使用 artifact.open 在系统应用中打开")},
			"retryable": false,
		}, nil
	}
	content, readErr := os.ReadFile(artifactPath)
	if readErr != nil {
		return map[string]any{
			"state":     "blocked",
			"blockers":  []map[string]any{issue("artifact_read_failed", "artifact", "产物读取失败", "重试或使用 artifact.open 打开")},
			"retryable": true,
		}, nil
	}
	return map[string]any{
		"artifact": item,
		"size":     info.Size(),
		"base64":   base64.StdEncoding.EncodeToString(content),
	}, nil
}

func resolveRecipe(root string, in requestInput) (map[string]any, *project, error) {
	if in.Recipe != nil {
		if in.ProjectID == "" {
			return in.Recipe, &project{ID: "adhoc", Name: "未命名视频", Template: asString(in.Recipe["template_id"]), Style: asString(in.Recipe["style_id"])}, nil
		}
	}
	if !idPattern.MatchString(in.ProjectID) {
		return nil, nil, fmt.Errorf("project_id is required unless recipe is supplied")
	}
	item, err := loadProject(root, in.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if in.Recipe != nil {
		item.Recipe = in.Recipe
	}
	return item.Recipe, &item, nil
}

func validateRecipe(recipe map[string]any, item *project) []map[string]any {
	issues := make([]map[string]any, 0)
	templateID := asString(recipe["template_id"])
	styleID := asString(recipe["style_id"])
	templateItem, templateOK := findTemplate(templateID)
	styleItem, styleOK := findStyle(styleID)
	if !templateOK {
		issues = append(issues, issue("template_not_found", "recipe", "template_id 无效", "先调用 template.list 选择已注册模板"))
	}
	if !styleOK {
		issues = append(issues, issue("style_not_found", "recipe", "style_id 无效", "先调用 style.list 选择已注册风格"))
	}
	if templateOK && styleOK && !contains(templateItem.CompatibleStyle, styleItem.ID) {
		issues = append(issues, issue("style_incompatible", "recipe", "模板不支持该风格", "改用兼容风格或选择其他模板"))
	}
	if templateOK {
		if version := asString(recipe["template_version"]); version != "" && version != templateItem.Version {
			issues = append(issues, issue("template_version_not_found", "recipe", "template_version 不存在或已失效", "调用 template.describe 获取当前版本并更新 Recipe"))
		}
	}
	if styleOK {
		if version := asString(recipe["style_version"]); version != "" && version != styleItem.Version {
			issues = append(issues, issue("style_version_not_found", "recipe", "style_version 不存在或已失效", "调用 style.describe 获取当前版本并更新 Recipe"))
		}
	}
	adapter := asString(recipe["adapter"])
	if adapter == "" {
		adapter = "remotion"
	}
	if !adapterAvailable(adapter) {
		issues = append(issues, issue("adapter_not_found", "renderer", "未声明该渲染适配器", "调用 catalog.list 选择已声明的 adapter"))
	}
	duration := numberValue(recipe["duration_seconds"])
	if duration < 1 || duration > 300 {
		issues = append(issues, issue("duration_out_of_range", "recipe", "时长必须在 1 到 300 秒之间", "调整 duration_seconds"))
	}
	if !canvasValid(recipe) {
		issues = append(issues, issue("canvas_invalid", "recipe", "画布尺寸必须是正整数", "设置 width、height 和 fps"))
	}
	if !dataPresent(recipe) {
		issues = append(issues, issue("data_missing", "recipe", "缺少可渲染数据", "提供 data 对象或 items 数组"))
	}
	if !safeAreaValid(recipe) {
		issues = append(issues, issue("safe_area_invalid", "recipe", "安全区必须为非负数且位于画布内", "调整 safe_area 与 canvas"))
	}
	if templateOK {
		for _, slot := range templateItem.Slots {
			if slot == "items" || slot == "title" || slot == "subtitle" || slot == "voiceover" || slot == "captions" || slot == "media" || slot == "map_data" || slot == "trend" || slot == "logo" || slot == "legend" || slot == "callouts" {
				if !slotPresent(recipe, slot) && slotRequired(templateItem, slot) {
					issues = append(issues, issue("slot_missing", "recipe", fmt.Sprintf("缺少模板插槽 %s", slot), "在 Recipe 顶层或 data 中补齐该插槽"))
				}
			}
		}
	}
	if item == nil {
		issues = append(issues, issue("project_missing", "workspace", "项目不存在", "先调用 project.create"))
	}
	return issues
}

func issue(code, stage, message, remediation string) map[string]any {
	return map[string]any{"code": code, "stage": stage, "message": message, "remediation": remediation, "retryable": true}
}

func writePreview(path string, page previewPage) error {
	const source = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.ProjectName}}</title>
<style>body{margin:0;background:#09111f;color:#eef5ff;font:16px system-ui,"Microsoft YaHei",sans-serif}main{width:min(760px,calc(100% - 32px));margin:32px auto}header{display:flex;justify-content:space-between;gap:16px;align-items:end;border-bottom:1px solid #263854;padding-bottom:20px}h1{margin:0;font-size:30px}p{color:#a9b9d1}.stage{margin:24px auto;max-width:420px;aspect-ratio:9/16;border-radius:14px;background:{{palette "background"}};border:1px solid #385274;padding:28px;box-sizing:border-box;display:flex;flex-direction:column;justify-content:space-between}.eyebrow{color:{{palette "accent"}};font-size:13px;letter-spacing:.08em;text-transform:uppercase}.title{font-size:34px;font-weight:700;line-height:1.1}.bars{display:grid;gap:10px}.bar{display:grid;grid-template-columns:28px 1fr auto;gap:10px;align-items:center}.track{height:13px;border-radius:10px;background:#1d2d45;overflow:hidden}.fill{height:100%;background:{{palette "accent"}}}.meta{font-size:12px;color:#9eb2cc}.facts{display:grid;grid-template-columns:repeat(3,1fr);gap:10px}.fact{padding:12px;background:#102039;border:1px solid #27415f;border-radius:8px}.fact strong{display:block;font-size:20px}.fact span{font-size:12px;color:#9eb2cc}.recipe{background:#0d1b30;border:1px solid #263854;padding:16px;border-radius:8px;overflow:auto;font-size:12px;white-space:pre-wrap}</style></head>
<body><main><header><div><div class="eyebrow">预览 · {{.Template.Name}}</div><h1>{{.ProjectName}}</h1><p>{{.Style.Name}} · {{.Template.Adapter}} · {{.GeneratedAt}}</p></div><div class="meta">{{.Template.DurationSeconds}} 秒</div></header>
<section class="stage"><div><div class="eyebrow">{{.Template.Family}}</div><div class="title">{{value .Recipe "title"}}</div><p>{{value .Recipe "subtitle"}}</p></div><div class="bars">{{range $index, $item := items .Recipe}}<div class="bar"><span>{{$index | plusOne}}</span><div class="track"><div class="fill" style="width:{{percent $item}}%"></div></div><strong>{{value $item "name"}}</strong></div>{{end}}</div><div class="facts"><div class="fact"><strong>{{value .Recipe "duration_seconds"}}</strong><span>秒</span></div><div class="fact"><strong>{{lenItems .Recipe}}</strong><span>数据项</span></div><div class="fact"><strong>{{.Style.Name}}</strong><span>风格</span></div></div></section>
<h2>Recipe 快照</h2><div class="recipe">{{json .Recipe}}</div></main></body></html>`
	funcs := template.FuncMap{
		"palette": func(key string) string {
			if value, ok := page.Style.Palette[key].(string); ok {
				return value
			}
			return "#12233d"
		},
		"value": func(input any, key string) string {
			if object, ok := input.(map[string]any); ok {
				return asString(object[key])
			}
			return ""
		},
		"items": func(recipe map[string]any) []any {
			if values, ok := recipe["items"].([]any); ok {
				return values
			}
			if data, ok := recipe["data"].(map[string]any); ok {
				if values, ok := data["items"].([]any); ok {
					return values
				}
			}
			return []any{}
		},
		"lenItems": func(recipe map[string]any) int { return lenItems(recipe) },
		"plusOne":  func(value int) int { return value + 1 },
		"percent": func(input any) int {
			if object, ok := input.(map[string]any); ok {
				value := numberValue(object["value"])
				if value > 100 {
					value = 100
				}
				return value
			}
			return 0
		},
		"json": func(input any) string { data, _ := json.MarshalIndent(input, "", "  "); return string(data) },
	}
	tmpl, err := template.New("preview").Funcs(funcs).Parse(source)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, page)
}

func workspace(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("workspace_root is required")
	}
	root, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("workspace_root must be an existing directory")
	}
	return root, nil
}

func workspaceFile(root, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("file path is required")
	}
	path := input
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("file path must stay inside workspace_root")
	}
	// A lexical path can still escape through a symlink. Render outputs are
	// existing files, so resolve both sides before accepting the artifact.
	if resolvedRoot, rootErr := filepath.EvalSymlinks(root); rootErr == nil {
		if resolvedPath, pathErr := filepath.EvalSymlinks(path); pathErr == nil {
			resolvedRel, relErr := filepath.Rel(resolvedRoot, resolvedPath)
			if relErr != nil || resolvedRel == ".." || strings.HasPrefix(resolvedRel, ".."+string(filepath.Separator)) || filepath.IsAbs(resolvedRel) {
				return "", fmt.Errorf("file path must stay inside workspace_root")
			}
		}
	}
	return path, nil
}

func loadProject(root, id string) (project, error) {
	if !idPattern.MatchString(id) {
		return project{}, fmt.Errorf("invalid project_id")
	}
	var item project
	if err := readJSON(filepath.Join(root, metadataDir, projectDir, id, "project.json"), &item); err != nil {
		return project{}, fmt.Errorf("project not found: %s", id)
	}
	return item, nil
}

func makeArtifact(id, kind, path, projectID, renderer string, ready bool) (artifact, error) {
	info, err := os.Stat(path)
	if err != nil {
		return artifact{}, err
	}
	hash, err := fileSHA256(path)
	if err != nil {
		return artifact{}, err
	}
	item := artifact{ID: id, Kind: kind, Path: path, Size: info.Size(), SHA256: hash, ProjectID: projectID, CreatedAt: now(), Renderer: renderer, Ready: ready}
	if err := writeJSON(path+".json", item); err != nil {
		return artifact{}, err
	}
	return item, nil
}

// archiveVideoArtifact gives every renderer the same durable artifact contract.
// Adapters may write anywhere inside the selected workspace, but project.get,
// export and open only index files under .himind-video/artifacts.
func archiveVideoArtifact(root, artifactID, kind, sourcePath string) (string, error) {
	artifactRoot := filepath.Join(root, metadataDir, artifactDir)
	if err := os.MkdirAll(artifactRoot, 0755); err != nil {
		return "", err
	}
	extension := strings.ToLower(filepath.Ext(sourcePath))
	if extension == "" || strings.ContainsAny(extension, `/\\`) {
		if kind == "video/webm" {
			extension = ".webm"
		} else {
			extension = ".mp4"
		}
	}
	destination := filepath.Join(artifactRoot, artifactID+extension)
	sourceAbs, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", err
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	if sourceAbs != destinationAbs {
		input, err := os.Open(sourceAbs)
		if err != nil {
			return "", err
		}
		defer input.Close()
		output, err := os.Create(destinationAbs)
		if err != nil {
			return "", err
		}
		if _, err = io.Copy(output, input); err != nil {
			_ = output.Close()
			return "", err
		}
		if err = output.Close(); err != nil {
			return "", err
		}
	}
	return destinationAbs, nil
}

func defaultRecipe(templateID, styleID string, data map[string]any) map[string]any {
	if data == nil {
		switch item, ok := findTemplate(templateID); {
		case ok && item.ID == "leaderboard-tech":
			// Keep the first render useful and visually representative of the
			// Video Flow baseline, even when the caller only supplies IDs.
			data = map[string]any{"items": []any{
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
			}}
		case ok && item.Family == "photo-narrative":
			data = map[string]any{"media": []any{map[string]any{"kind": "placeholder", "label": "待添加图片"}}}
		case ok && item.Family == "map-story":
			data = map[string]any{
				"map_data": map[string]any{"points": []any{}, "region": "待添加地区数据"},
				"legend":   []any{map[string]any{"label": "待添加指标", "value": 0}},
			}
		default:
			data = map[string]any{"items": []any{map[string]any{"name": "项目 A", "value": 92}, map[string]any{"name": "项目 B", "value": 78}, map[string]any{"name": "项目 C", "value": 64}}}
		}
	}
	templateVersion, styleVersion := "", ""
	adapter := "remotion"
	duration := 12
	title := "数据排行榜"
	if item, ok := findTemplate(templateID); ok {
		templateVersion = item.Version
		duration = item.DurationSeconds
		if item.Adapter == "remotion" || item.Adapter == "hyperframes" {
			adapter = item.Adapter
		}
		switch item.Family {
		case "photo-narrative":
			title = "图片叙事"
		case "map-story":
			title = "地图数据故事"
		}
		if templateID == "leaderboard-tech" {
			title = "安徽省热门景区\n游客量排行榜"
		}
	}
	if item, ok := findStyle(styleID); ok {
		styleVersion = item.Version
		canvas := item.Canvas
		width, height, fps := numberValue(canvas["width"]), numberValue(canvas["height"]), numberValue(canvas["fps"])
		if width == 0 {
			width = 1080
		}
		if height == 0 {
			height = 1920
		}
		if fps == 0 {
			fps = 30
		}
		subtitle := "基于已沉淀模板快速生成"
		if templateID == "leaderboard-tech" {
			subtitle = "统计周期：2025年1月 - 2025年12月 · 单位：万人次"
		}
		safeArea := map[string]any{"top": height / 20, "right": width / 15, "bottom": height / 10, "left": width / 15}
		if templateID == "leaderboard-tech" && styleID == "tech-blue" && width == 2160 && height == 3840 {
			// These are the reference safe-area tokens used by Video Flow's
			// DataRanking composition. Keeping them in the Recipe makes the
			// baseline deterministic for both the UI and headless rendering.
			safeArea = map[string]any{"top": 325, "right": 265, "bottom": 450, "left": 265}
		}
		recipe := map[string]any{"template_id": templateID, "template_version": templateVersion, "style_id": styleID, "style_version": styleVersion, "adapter": adapter, "title": title, "subtitle": subtitle, "duration_seconds": duration, "canvas": map[string]any{"width": width, "height": height, "fps": fps}, "safe_area": safeArea, "data": data}
		if templateID == "leaderboard-tech" {
			recipe["period"] = "2025年度"
			recipe["unit"] = "万人次"
			recipe["source"] = "数据来源：安徽省文化和旅游厅（示例数据）"
			recipe["total"] = "2.04 亿"
			recipe["average"] = "+11.2%"
			recipe["highlight"] = "芜湖方特"
		}
		return recipe
	}
	return map[string]any{"template_id": templateID, "template_version": templateVersion, "style_id": styleID, "style_version": styleVersion, "adapter": adapter, "title": title, "subtitle": "基于已沉淀模板快速生成", "duration_seconds": duration, "canvas": map[string]any{"width": 1080, "height": 1920, "fps": 30}, "safe_area": map[string]any{"top": 96, "right": 72, "bottom": 180, "left": 72}, "data": data}
}

func loadStyles() []style {
	var items []style
	if err := readJSON(filepath.Join(pluginRoot(), "catalog", "styles.json"), &items); err == nil && len(items) > 0 {
		return items
	}
	return []style{{ID: "tech-blue", Version: "1.1.0", Name: "科技蓝", Description: "深色科技蓝底、渐变强调和视频流式数据层级。", Canvas: map[string]any{"width": 2160, "height": 3840, "fps": 30}, Palette: map[string]any{"background": "#080c1a", "accent": "#ff6b6b", "accent_secondary": "#4fc3f7", "text": "#ffffff"}, Typography: map[string]any{"font_family": "Microsoft YaHei", "title_weight": 800}, Motion: map[string]any{"entrance": "spring-stagger", "duration_ms": 520}, Quality: []string{"使用 2160×3840 竖屏安全区", "前三名使用奖牌式层级"}}, {ID: "minimal-white", Version: "1.0.0", Name: "极简白", Description: "明亮留白、低噪声排版、突出数据和叙事。", Canvas: map[string]any{"width": 1080, "height": 1920, "fps": 30}, Palette: map[string]any{"background": "#f7f9fc", "accent": "#1769aa", "text": "#10233d"}, Typography: map[string]any{"font_family": "Microsoft YaHei", "title_weight": 700}, Motion: map[string]any{"entrance": "fade", "duration_ms": 360}, Quality: []string{"正文与背景有足够对比度", "保持四周留白"}}}
}

func loadTemplates() []videoTemplate {
	var items []videoTemplate
	if err := readJSON(filepath.Join(pluginRoot(), "catalog", "templates.json"), &items); err == nil && len(items) > 0 {
		return items
	}
	return []videoTemplate{{ID: "leaderboard-tech", Version: "1.1.0", Family: "leaderboard", Name: "排行榜 · 科技蓝", Description: "参考 Video Flow 数据排行榜结构的可复用 Remotion 模板。", Adapter: "remotion", Composition: "LeaderboardTech", Slots: []string{"title", "subtitle", "items", "logo", "voiceover"}, CompatibleStyle: []string{"tech-blue", "minimal-white"}, DurationSeconds: 12}, {ID: "leaderboard-map-trend", Version: "1.0.0", Family: "leaderboard", Name: "排行榜 · 地图趋势", Description: "排行榜与地图、地区标签和趋势箭头组合。", Adapter: "remotion", Composition: "LeaderboardMapTrend", Slots: []string{"title", "items", "map_data", "trend", "voiceover"}, CompatibleStyle: []string{"tech-blue"}, DurationSeconds: 15}, {ID: "photo-narrative", Version: "1.0.0", Family: "photo-narrative", Name: "图片叙事", Description: "图片、标题和旁白组成的节奏化故事模板。", Adapter: "hyperframes", Composition: "PhotoNarrative", Slots: []string{"title", "subtitle", "media", "voiceover", "captions"}, CompatibleStyle: []string{"tech-blue", "minimal-white"}, DurationSeconds: 20}}
}

func loadAdapters() []map[string]any {
	var items []map[string]any
	if err := readJSON(filepath.Join(pluginRoot(), "catalog", "adapters.json"), &items); err == nil && len(items) > 0 {
		return items
	}
	return []map[string]any{
		{"id": "remotion", "preview": true, "render": true, "status": "built_in", "description": "内置主渲染引擎，按固定版本准备运行时并在后台生成 MP4。"},
		{"id": "hyperframes", "preview": true, "render": true, "status": "external", "description": "与 Remotion 同级的可插拔渲染引擎，通过 Renderer Adapter 协议接入。"},
	}
}

func findStyle(id string) (style, bool) {
	for _, item := range loadStyles() {
		if item.ID == id {
			return item, true
		}
	}
	return style{}, false
}
func findTemplate(id string) (videoTemplate, bool) {
	for _, item := range loadTemplates() {
		if item.ID == id {
			return item, true
		}
	}
	return videoTemplate{}, false
}
func findTemplateValue(value any) bool { _, ok := findTemplate(asString(value)); return ok }
func findStyleValue(value any) bool    { _, ok := findStyle(asString(value)); return ok }
func canvasValid(recipe map[string]any) bool {
	canvas, ok := recipe["canvas"].(map[string]any)
	fps := numberValue(canvas["fps"])
	return ok && numberValue(canvas["width"]) > 0 && numberValue(canvas["height"]) > 0 && fps > 0 && fps <= 120
}
func safeAreaValid(recipe map[string]any) bool {
	area, ok := recipe["safe_area"].(map[string]any)
	if !ok {
		return false
	}
	for _, key := range []string{"top", "right", "bottom", "left"} {
		if numberValue(area[key]) < 0 {
			return false
		}
	}
	canvas, ok := recipe["canvas"].(map[string]any)
	if !ok {
		return false
	}
	width, height := numberValue(canvas["width"]), numberValue(canvas["height"])
	return numberValue(area["left"])+numberValue(area["right"]) < width && numberValue(area["top"])+numberValue(area["bottom"]) < height
}

func adapterAvailable(id string) bool {
	for _, item := range loadAdapters() {
		if asString(item["id"]) == id {
			return true
		}
	}
	return false
}

func slotPresent(recipe map[string]any, slot string) bool {
	if value, ok := recipe[slot]; ok && value != nil && asString(value) != "" {
		return true
	}
	if data, ok := recipe["data"].(map[string]any); ok {
		value, exists := data[slot]
		if exists && value != nil {
			if slot == "items" {
				return lenItems(recipe) > 0
			}
			return asString(value) != ""
		}
	}
	return slot == "items" && lenItems(recipe) > 0
}

func slotRequired(item videoTemplate, slot string) bool {
	// These slots carry the primary render payload. Branding, voiceover and
	// captions may be added incrementally during the creative pass.
	if slot == "items" || slot == "media" || slot == "map_data" {
		return true
	}
	return item.Family == "map-story" && slot == "legend"
}
func dataPresent(recipe map[string]any) bool {
	if lenItems(recipe) > 0 {
		return true
	}
	data, ok := recipe["data"].(map[string]any)
	return ok && len(data) > 0
}
func lenItems(recipe map[string]any) int {
	if values, ok := recipe["items"].([]any); ok {
		return len(values)
	}
	if data, ok := recipe["data"].(map[string]any); ok {
		if values, ok := data["items"].([]any); ok {
			return len(values)
		}
	}
	return 0
}
func pluginRoot() string {
	executable, err := os.Executable()
	if err != nil {
		return "."
	}
	root := filepath.Dir(filepath.Dir(executable))
	if _, err := os.Stat(filepath.Join(root, "catalog")); err == nil {
		return root
	}
	// `go test` places the temporary executable outside the source tree. Use
	// the current module directory in that case so tests exercise real assets.
	if workingDir, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(workingDir, "catalog")); err == nil {
			return workingDir
		}
	}
	return root
}
func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
func appendJSONLine(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(data, '\n'))
	return err
}
func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
func now() string                { return time.Now().UTC().Format(time.RFC3339) }
func newID(prefix string) string { return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano()) }
func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
func asString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return fmt.Sprint(value)
}

func cloneMap(input map[string]any) (map[string]any, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var output map[string]any
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}
	return output, nil
}
func numberValue(value any) int {
	switch number := value.(type) {
	case int:
		return number
	case int64:
		return int(number)
	case float64:
		return int(number)
	case json.Number:
		parsed, _ := number.Int64()
		return int(parsed)
	default:
		return 0
	}
}
func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
func sortStrings(values []string) []string { sort.Strings(values); return values }
