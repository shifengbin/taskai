package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taskai/internal/gitlab"
	"taskai/internal/task"
)

func TestAppListsGitLabProjectsMarksExistingRepositoriesAndSavesDefaultsExplicitly(t *testing.T) {
	const token = "temporary-app-token"
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("PRIVATE-TOKEN") != token {
			t.Fatalf("PRIVATE-TOKEN = %q", request.Header.Get("PRIVATE-TOKEN"))
		}
		switch request.URL.Path {
		case "/api/v4/user":
			writeAppGitLabJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
		case "/api/v4/projects":
			host := mustURLHost(t, server.URL)
			writeAppGitLabJSON(writer, http.StatusOK, `[{
				"id":1,"name":"api","path_with_namespace":"team/api",
				"ssh_url_to_repo":"ssh://git@`+host+`/team/api.git",
				"http_url_to_repo":"`+server.URL+`/team/api.git","visibility":"private"
			},{
				"id":2,"name":"web","path_with_namespace":"team/web",
				"ssh_url_to_repo":"ssh://git@`+host+`/team/web.git",
				"http_url_to_repo":"`+server.URL+`/team/web.git","visibility":"internal","archived":true
			}]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	dataDirectory := t.TempDir()
	app := newApp(dataDirectory)
	template := gitTemplateForTest(t, app)
	if _, err := app.SaveExtraInfo(task.ExtraInfo{
		TemplateID: template.ID,
		Catalogue:  template.Catalogue,
		Fields: []task.ExtraInfoField{
			{Key: "name", Value: "api"},
			{Key: "repository", Value: server.URL + "/team/api.git"},
		},
	}); err != nil {
		t.Fatalf("SaveExtraInfo() error = %v", err)
	}

	result, err := app.ListGitLabProjects(server.URL+"/", " integration-user ", token)
	if err != nil {
		t.Fatalf("ListGitLabProjects() error = %v", err)
	}
	if !result.UsesPlainHTTP || len(result.Projects) != 2 {
		t.Fatalf("ListGitLabProjects() = %#v", result)
	}
	if !result.Projects[0].Imported || result.Projects[1].Imported || !result.Projects[1].Archived {
		t.Fatalf("项目导入标记 = %#v", result.Projects)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), token) {
		t.Fatalf("返回结果泄露令牌: %s", encoded)
	}
	defaults, err := app.GetGitLabImportDefaults()
	if err != nil {
		t.Fatalf("GetGitLabImportDefaults() error = %v", err)
	}
	if defaults != (gitlab.ConnectionDefaults{}) {
		t.Fatalf("ListGitLabProjects() 后默认连接 = %#v, want empty", defaults)
	}
	if err := app.SaveGitLabImportDefaults(server.URL+"/", " integration-user ", token); err != nil {
		t.Fatalf("SaveGitLabImportDefaults() error = %v", err)
	}
	defaults, err = app.GetGitLabImportDefaults()
	if err != nil {
		t.Fatalf("GetGitLabImportDefaults() error = %v", err)
	}
	if defaults != (gitlab.ConnectionDefaults{Address: server.URL, Username: "integration-user", Token: token}) {
		t.Fatalf("GetGitLabImportDefaults() = %#v", defaults)
	}
	contents, err := os.ReadFile(filepath.Join(dataDirectory, "tasks.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(contents), token) || !strings.Contains(string(contents), "integration-user") {
		t.Fatalf("持久化数据未明文保存连接凭据: %s", contents)
	}
}

func TestAppDoesNotOverwriteGitLabDefaultsWhenProjectListFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeAppGitLabJSON(writer, http.StatusUnauthorized, `{"message":"new-token"}`)
	}))
	defer server.Close()

	app := newApp(t.TempDir())
	previous := gitlab.ConnectionDefaults{Address: "https://previous.example.com", Username: "previous-user", Token: "previous-token"}
	if _, err := app.repository.SaveGitLabImportDefaults(previous); err != nil {
		t.Fatalf("SaveGitLabImportDefaults() error = %v", err)
	}
	if _, err := app.ListGitLabProjects(server.URL, "new-user", "new-token"); err == nil {
		t.Fatal("ListGitLabProjects() error = nil")
	}
	loaded, err := app.GetGitLabImportDefaults()
	if err != nil {
		t.Fatalf("GetGitLabImportDefaults() error = %v", err)
	}
	if loaded != previous {
		t.Fatalf("GetGitLabImportDefaults() = %#v, want %#v", loaded, previous)
	}
}

func TestAppDoesNotOverwriteDefaultsWhenExplicitSaveFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v4/user":
			writeAppGitLabJSON(writer, http.StatusOK, `{"username":"new-user"}`)
		case "/api/v4/projects":
			writeAppGitLabJSON(writer, http.StatusOK, `[{"id":1,"name":"api","path_with_namespace":"team/api","ssh_url_to_repo":"git@gitlab.example.com:team/api.git","http_url_to_repo":"https://gitlab.example.com/team/api.git"}]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	app := newApp(t.TempDir())
	previous := gitlab.ConnectionDefaults{Address: "https://previous.example.com", Username: "previous-user", Token: "previous-token"}
	if _, err := app.repository.SaveGitLabImportDefaults(previous); err != nil {
		t.Fatalf("SaveGitLabImportDefaults() error = %v", err)
	}
	app.gitLabDefaultsSaver = func(gitlab.ConnectionDefaults) (gitlab.ConnectionDefaults, error) {
		return gitlab.ConnectionDefaults{}, errors.New("受控默认连接保存失败")
	}

	result, err := app.ListGitLabProjects(server.URL, "new-user", "new-token")
	if err != nil || len(result.Projects) != 1 {
		t.Fatalf("ListGitLabProjects() result/error = %#v / %v", result, err)
	}
	if err := app.SaveGitLabImportDefaults(server.URL, "new-user", "new-token"); err == nil || strings.Contains(err.Error(), "new-token") {
		t.Fatalf("SaveGitLabImportDefaults() error = %v", err)
	}
	loaded, loadErr := app.GetGitLabImportDefaults()
	if loadErr != nil {
		t.Fatalf("GetGitLabImportDefaults() error = %v", loadErr)
	}
	if loaded != previous {
		t.Fatalf("GetGitLabImportDefaults() = %#v, want %#v", loaded, previous)
	}
}

func TestAppImportsSelectedGitLabProjectsWithRequestedURLMode(t *testing.T) {
	app := newApp(t.TempDir())
	project := gitlab.Project{
		ID:                7,
		Name:              "api",
		PathWithNamespace: "team/api",
		SSHURL:            "git@gitlab.example.com:team/api.git",
		HTTPURL:           "https://gitlab.example.com/team/api.git",
		Visibility:        "private",
	}
	result, err := app.ImportGitLabProjects([]gitlab.Project{project}, "https")
	if err != nil {
		t.Fatalf("ImportGitLabProjects() error = %v", err)
	}
	if result.Imported != 1 || result.Skipped != 0 || len(result.Infos) != 1 {
		t.Fatalf("ImportGitLabProjects() = %#v", result)
	}
	if got := appExtraInfoField(result.Infos[0], "repository"); got != project.HTTPURL {
		t.Fatalf("repository = %q", got)
	}
	listed, err := app.ListExtraInfos()
	if err != nil {
		t.Fatalf("ListExtraInfos() error = %v", err)
	}
	if len(listed) != 1 || task.ExtraInfoName(listed[0]) != "api" {
		t.Fatalf("ListExtraInfos() = %#v", listed)
	}
}

func TestAppMarksExistingRelativeURLProjectAcrossDifferentClonePorts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/private/gitlab/api/v4/user":
			writeAppGitLabJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
		case "/private/gitlab/api/v4/projects":
			writeAppGitLabJSON(writer, http.StatusOK, `[{"id":1,"name":"api","path_with_namespace":"team/api","ssh_url_to_repo":"ssh://git@gitlab.example.com:2424/team/api.git","http_url_to_repo":"http://gitlab.example.com:8929/private/gitlab/team/api.git"}]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	app := newApp(t.TempDir())
	template := gitTemplateForTest(t, app)
	if _, err := app.SaveExtraInfo(task.ExtraInfo{
		TemplateID: template.ID,
		Catalogue:  template.Catalogue,
		Fields: []task.ExtraInfoField{
			{Key: "name", Value: "api"},
			{Key: "repository", Value: "ssh://git@gitlab.example.com:2424/team/api.git"},
		},
	}); err != nil {
		t.Fatalf("SaveExtraInfo() error = %v", err)
	}
	result, err := app.ListGitLabProjects(server.URL+"/private/gitlab", "integration-user", "token")
	if err != nil {
		t.Fatalf("ListGitLabProjects() error = %v", err)
	}
	if len(result.Projects) != 1 || !result.Projects[0].Imported {
		t.Fatalf("ListGitLabProjects() = %#v", result)
	}
}

func TestAppReturnsSanitizedGitLabAuthenticationError(t *testing.T) {
	const token = "never-return-this-token"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeAppGitLabJSON(writer, http.StatusUnauthorized, `{"message":"never-return-this-token"}`)
	}))
	defer server.Close()

	app := newApp(t.TempDir())
	result, err := app.ListGitLabProjects(server.URL, "integration-user", token)
	if err == nil {
		t.Fatal("ListGitLabProjects() error = nil")
	}
	if result.Projects != nil || strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "PRIVATE-TOKEN") {
		t.Fatalf("ListGitLabProjects() result/error = %#v / %v", result, err)
	}
}

func writeAppGitLabJSON(writer http.ResponseWriter, status int, body string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write([]byte(body))
}

func mustURLHost(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return parsed.Host
}

func appExtraInfoField(info task.ExtraInfo, key string) string {
	for _, field := range info.Fields {
		if field.Key == key {
			return field.Value
		}
	}
	return ""
}
