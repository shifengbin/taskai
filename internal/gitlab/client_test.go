package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClientValidatesPrivateGitLabAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		kind    ErrorKind
	}{
		{name: "empty", address: "", kind: ErrorInvalidURL},
		{name: "missing host", address: "https:///gitlab", kind: ErrorInvalidURL},
		{name: "missing hostname", address: "http://:8080", kind: ErrorInvalidURL},
		{name: "unsupported scheme", address: "ftp://gitlab.example.com", kind: ErrorInvalidURL},
		{name: "userinfo", address: "https://user:secret@gitlab.example.com", kind: ErrorInvalidURL},
		{name: "query", address: "https://gitlab.example.com?token=secret", kind: ErrorInvalidURL},
		{name: "fragment", address: "https://gitlab.example.com/#secret", kind: ErrorInvalidURL},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewClient(test.address, time.Second)
			assertRequestErrorKind(t, err, test.kind)
		})
	}

	secure, err := NewClient("https://gitlab.example.com/root/", time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if secure.UsesPlainHTTP() {
		t.Fatal("UsesPlainHTTP() = true，期望 HTTPS 返回 false")
	}
	insecure, err := NewClient("http://gitlab.example.com/root", time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if !insecure.UsesPlainHTTP() {
		t.Fatal("UsesPlainHTTP() = false，期望 HTTP 返回 true")
	}
}

func TestClientChecksIdentityAndPreservesPrivateDeploymentPath(t *testing.T) {
	const token = "temporary-token"
	requests := make([]*http.Request, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Clone(request.Context()))
		if request.Header.Get("PRIVATE-TOKEN") != token {
			t.Errorf("PRIVATE-TOKEN = %q", request.Header.Get("PRIVATE-TOKEN"))
		}
		switch request.URL.Path {
		case "/private/gitlab/api/v4/user":
			writeJSON(writer, http.StatusOK, `{"username":"Integration-User"}`)
		case "/private/gitlab/api/v4/projects":
			writeJSON(writer, http.StatusOK, `[{"id":7,"name":"api","path_with_namespace":"team/api","ssh_url_to_repo":"git@gitlab.example.com:team/api.git","http_url_to_repo":"https://gitlab.example.com/team/api.git","archived":false,"visibility":"private"}]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/private/gitlab/", time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	projects, err := client.ListAccessibleProjects(context.Background(), "integration-user", token)
	if err != nil {
		t.Fatalf("ListAccessibleProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != 7 || projects[0].PathWithNamespace != "team/api" || projects[0].Visibility != "private" {
		t.Fatalf("ListAccessibleProjects() = %#v", projects)
	}
	if len(requests) != 2 {
		t.Fatalf("请求数量 = %d，期望 2", len(requests))
	}
}

func TestClientRejectsMismatchedUsernameBeforeListingProjects(t *testing.T) {
	projectsRequested := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "/user") {
			writeJSON(writer, http.StatusOK, `{"username":"other-user"}`)
			return
		}
		projectsRequested = true
		writeJSON(writer, http.StatusOK, `[]`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	projects, err := client.ListAccessibleProjects(context.Background(), "integration-user", "secret-token")
	if projects != nil {
		t.Fatalf("ListAccessibleProjects() projects = %#v，期望 nil", projects)
	}
	assertRequestErrorKind(t, err, ErrorUsernameMismatch)
	if projectsRequested {
		t.Fatal("用户名不匹配后仍请求了项目列表")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("错误泄露令牌: %v", err)
	}
}

func TestClientListsEveryAccessibleProjectPageWithoutMembershipFilter(t *testing.T) {
	projectQueries := make([]url.Values, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v4/user":
			writeJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
		case "/api/v4/projects":
			projectQueries = append(projectQueries, request.URL.Query())
			switch request.URL.Query().Get("page") {
			case "1":
				writer.Header().Set("X-Next-Page", "2")
				writeJSON(writer, http.StatusOK, `[{"id":1,"name":"public","path_with_namespace":"team/public","ssh_url_to_repo":"git@example.com:team/public.git","http_url_to_repo":"https://example.com/team/public.git","visibility":"public"},{"id":2,"name":"internal","path_with_namespace":"team/internal","ssh_url_to_repo":"git@example.com:team/internal.git","http_url_to_repo":"https://example.com/team/internal.git","visibility":"internal"}]`)
			case "2":
				writeJSON(writer, http.StatusOK, `[{"id":3,"name":"private","path_with_namespace":"team/private","ssh_url_to_repo":"git@example.com:team/private.git","http_url_to_repo":"https://example.com/team/private.git","visibility":"private","archived":true}]`)
			default:
				t.Fatalf("意外页码 %q", request.URL.Query().Get("page"))
			}
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	projects, err := client.ListAccessibleProjects(context.Background(), "integration-user", "token")
	if err != nil {
		t.Fatalf("ListAccessibleProjects() error = %v", err)
	}
	if len(projects) != 3 || !projects[2].Archived {
		t.Fatalf("ListAccessibleProjects() = %#v", projects)
	}
	for _, query := range projectQueries {
		if query.Get("membership") != "" {
			t.Fatalf("projects query 包含 membership: %v", query)
		}
		if query.Get("per_page") != "100" {
			t.Fatalf("per_page = %q，期望 100", query.Get("per_page"))
		}
	}
}

func TestClientDiscardsProjectsWhenLaterPageFails(t *testing.T) {
	const token = "do-not-leak"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "/user") {
			writeJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
			return
		}
		if request.URL.Query().Get("page") == "1" {
			writer.Header().Set("X-Next-Page", "2")
			writeJSON(writer, http.StatusOK, `[{"id":1,"name":"api","path_with_namespace":"team/api","ssh_url_to_repo":"git@example.com:team/api.git","http_url_to_repo":"https://example.com/team/api.git"}]`)
			return
		}
		writeJSON(writer, http.StatusInternalServerError, `{"message":"do-not-leak"}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	projects, err := client.ListAccessibleProjects(context.Background(), "integration-user", token)
	if projects != nil {
		t.Fatalf("失败后返回部分项目: %#v", projects)
	}
	assertRequestErrorKind(t, err, ErrorResponse)
	if !strings.Contains(err.Error(), "第 2 页") || strings.Contains(err.Error(), token) {
		t.Fatalf("分页错误 = %q", err)
	}
}

func TestClientAllowsSameHostRedirectAndRejectsCrossHostRedirect(t *testing.T) {
	t.Run("same host", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/api/v4/user":
				http.Redirect(writer, request, "/redirected-user", http.StatusTemporaryRedirect)
			case "/redirected-user":
				if request.Header.Get("PRIVATE-TOKEN") != "token" {
					t.Fatalf("重定向请求缺少 PRIVATE-TOKEN")
				}
				writeJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
			case "/api/v4/projects":
				writeJSON(writer, http.StatusOK, `[]`)
			default:
				http.NotFound(writer, request)
			}
		}))
		defer server.Close()
		client, err := NewClient(server.URL, time.Second)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if _, err := client.ListAccessibleProjects(context.Background(), "integration-user", "token"); err != nil {
			t.Fatalf("ListAccessibleProjects() error = %v", err)
		}
	})

	t.Run("cross host", func(t *testing.T) {
		target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			t.Fatal("不应向跨主机目标发送请求")
		}))
		defer target.Close()
		source := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
		}))
		defer source.Close()
		client, err := NewClient(source.URL, time.Second)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		projects, err := client.ListAccessibleProjects(context.Background(), "integration-user", "token")
		if projects != nil {
			t.Fatalf("ListAccessibleProjects() projects = %#v", projects)
		}
		assertRequestErrorKind(t, err, ErrorResponse)
	})

	t.Run("https downgrade", func(t *testing.T) {
		client, err := NewClient("https://gitlab.example.com", time.Second)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		redirect, err := http.NewRequest(http.MethodGet, "http://gitlab.example.com/redirected-user", nil)
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		previous, err := http.NewRequest(http.MethodGet, "https://gitlab.example.com/api/v4/user", nil)
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		if err := client.httpClient.CheckRedirect(redirect, []*http.Request{previous}); err == nil {
			t.Fatal("HTTPS 降级到 HTTP 的重定向未被拒绝")
		}
	})
}

func TestClientClassifiesSafeRequestErrors(t *testing.T) {
	statuses := []struct {
		status int
		kind   ErrorKind
	}{
		{status: http.StatusUnauthorized, kind: ErrorAuthentication},
		{status: http.StatusForbidden, kind: ErrorForbidden},
		{status: http.StatusNotFound, kind: ErrorNotFound},
		{status: http.StatusTooManyRequests, kind: ErrorRateLimited},
		{status: http.StatusInternalServerError, kind: ErrorResponse},
	}
	for _, test := range statuses {
		t.Run(fmt.Sprintf("status %d", test.status), func(t *testing.T) {
			const token = "status-secret"
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writeJSON(writer, test.status, `{"message":"status-secret"}`)
			}))
			defer server.Close()
			client, err := NewClient(server.URL, time.Second)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			_, err = client.ListAccessibleProjects(context.Background(), "integration-user", token)
			assertRequestErrorKind(t, err, test.kind)
			if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "PRIVATE-TOKEN") {
				t.Fatalf("错误泄露敏感信息: %v", err)
			}
		})
	}
}

func TestClientClassifiesTimeoutAndCertificateErrors(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			time.Sleep(80 * time.Millisecond)
			writeJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
		}))
		defer server.Close()
		client, err := NewClient(server.URL, 10*time.Millisecond)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		_, err = client.ListAccessibleProjects(context.Background(), "integration-user", "timeout-secret")
		assertRequestErrorKind(t, err, ErrorTimeout)
	})

	t.Run("certificate", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writeJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
		}))
		defer server.Close()
		client, err := NewClient(server.URL, time.Second)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		_, err = client.ListAccessibleProjects(context.Background(), "integration-user", "certificate-secret")
		assertRequestErrorKind(t, err, ErrorCertificate)
	})
}

func TestClientRejectsInvalidPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "/user") {
			writeJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
			return
		}
		writer.Header().Set("X-Next-Page", request.URL.Query().Get("page"))
		writeJSON(writer, http.StatusOK, `[]`)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	projects, err := client.ListAccessibleProjects(context.Background(), "integration-user", "token")
	if projects != nil {
		t.Fatalf("ListAccessibleProjects() projects = %#v", projects)
	}
	assertRequestErrorKind(t, err, ErrorPagination)
}

func TestClientRejectsSkippedPaginationPageWithoutReturningPartialProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "/user") {
			writeJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
			return
		}
		if request.URL.Query().Get("page") == "1" {
			writer.Header().Set("X-Next-Page", "3")
			writeJSON(writer, http.StatusOK, `[{"id":1,"name":"api","path_with_namespace":"team/api","ssh_url_to_repo":"git@example.com:team/api.git","http_url_to_repo":"https://example.com/team/api.git"}]`)
			return
		}
		writeJSON(writer, http.StatusOK, `[]`)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	projects, err := client.ListAccessibleProjects(context.Background(), "integration-user", "token")
	if projects != nil {
		t.Fatalf("ListAccessibleProjects() projects = %#v", projects)
	}
	assertRequestErrorKind(t, err, ErrorPagination)
}

func TestClientRejectsNullProjectPageWithoutReturningEarlierPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "/user") {
			writeJSON(writer, http.StatusOK, `{"username":"integration-user"}`)
			return
		}
		if request.URL.Query().Get("page") == "1" {
			writer.Header().Set("X-Next-Page", "2")
			writeJSON(writer, http.StatusOK, `[{"id":1,"name":"api","path_with_namespace":"team/api","ssh_url_to_repo":"git@example.com:team/api.git","http_url_to_repo":"https://example.com/team/api.git"}]`)
			return
		}
		writeJSON(writer, http.StatusOK, `null`)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	projects, err := client.ListAccessibleProjects(context.Background(), "integration-user", "token")
	if projects != nil {
		t.Fatalf("ListAccessibleProjects() projects = %#v", projects)
	}
	assertRequestErrorKind(t, err, ErrorResponse)
}

func assertRequestErrorKind(t *testing.T, err error, kind ErrorKind) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil，期望 %q", kind)
	}
	var requestError *RequestError
	if !errors.As(err, &requestError) {
		t.Fatalf("error = %T %v，期望 *RequestError", err, err)
	}
	if requestError.Kind != kind {
		t.Fatalf("RequestError.Kind = %q，期望 %q，完整错误: %v", requestError.Kind, kind, err)
	}
}

func writeJSON(writer http.ResponseWriter, status int, body string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write([]byte(body))
}
