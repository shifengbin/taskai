package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitLabMockHandlerAuthenticatesAndReturnsTwoProjectPages(t *testing.T) {
	server := httptest.NewServer(newGitLabMockHandler(mockOptions{}))
	defer server.Close()

	userRequest, err := http.NewRequest(http.MethodGet, server.URL+"/private/gitlab/api/v4/user", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	userRequest.Header.Set("PRIVATE-TOKEN", integrationToken)
	userResponse, err := http.DefaultClient.Do(userRequest)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer userResponse.Body.Close()
	var user map[string]any
	if err := json.NewDecoder(userResponse.Body).Decode(&user); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if userResponse.StatusCode != http.StatusOK || user["username"] != integrationUser {
		t.Fatalf("user response = %d %#v", userResponse.StatusCode, user)
	}

	page1 := getMockProjectsPage(t, server.URL, "1")
	if page1.response.Header.Get("X-Next-Page") != "2" || len(page1.projects) < 3 {
		t.Fatalf("page 1 = header %q projects %#v", page1.response.Header.Get("X-Next-Page"), page1.projects)
	}
	if page1.projects[0].PathWithNamespace != "team-a/api" || page1.projects[1].PathWithNamespace != "team/existing" {
		t.Fatalf("page 1 projects = %#v", page1.projects)
	}
	if page1.projects[0].HTTPURL != server.URL+"/private/gitlab/team-a/api.git" || page1.projects[0].SSHURL != "ssh://git@127.0.0.1:2424/team-a/api.git" {
		t.Fatalf("page 1 clone URLs = %#v", page1.projects[0])
	}
	page2 := getMockProjectsPage(t, server.URL, "2")
	if page2.response.Header.Get("X-Next-Page") != "" || len(page2.projects) < 3 {
		t.Fatalf("page 2 = header %q projects %#v", page2.response.Header.Get("X-Next-Page"), page2.projects)
	}
	if page2.projects[0].PathWithNamespace != "team-b/api" || !page2.projects[1].Archived {
		t.Fatalf("page 2 projects = %#v", page2.projects)
	}
}

func TestGitLabMockHandlerRejectsInvalidToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/private/gitlab/api/v4/user", nil)
	response := httptest.NewRecorder()

	newGitLabMockHandler(mockOptions{}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d，期望 401", response.Code)
	}
}

func TestGitLabMockHandlerCanFailSecondProjectPage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/private/gitlab/api/v4/projects?page=2", nil)
	request.Header.Set("PRIVATE-TOKEN", integrationToken)
	response := httptest.NewRecorder()

	newGitLabMockHandler(mockOptions{failPage: 2}).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d，期望 500", response.Code)
	}
}

type mockProjectPage struct {
	response *http.Response
	projects []mockProject
}

type mockProject struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	SSHURL            string `json:"ssh_url_to_repo"`
	HTTPURL           string `json:"http_url_to_repo"`
	Archived          bool   `json:"archived"`
	Visibility        string `json:"visibility"`
}

func getMockProjectsPage(t *testing.T, serverURL, page string) mockProjectPage {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, serverURL+"/private/gitlab/api/v4/projects?page="+page+"&per_page=100", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("PRIVATE-TOKEN", integrationToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()
	var projects []mockProject
	if err := json.NewDecoder(response.Body).Decode(&projects); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	return mockProjectPage{response: response, projects: projects}
}
