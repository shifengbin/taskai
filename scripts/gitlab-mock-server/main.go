package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	integrationUser  = "integration-user"
	integrationToken = "integration-token"
	mockBasePath     = "/private/gitlab"
	mockSSHPort      = "2424"
)

type mockOptions struct {
	failPage int
}

type project struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	SSHURL            string `json:"ssh_url_to_repo"`
	HTTPURL           string `json:"http_url_to_repo"`
	Archived          bool   `json:"archived"`
	Visibility        string `json:"visibility"`
}

func main() {
	listen := flag.String("listen", "127.0.0.1:18080", "监听地址")
	failPage := flag.Int("fail-page", 0, "让指定项目页返回 500，0 表示关闭")
	flag.Parse()
	server := &http.Server{
		Addr:              *listen,
		Handler:           newGitLabMockHandler(mockOptions{failPage: *failPage}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("GitLab 模拟服务监听 http://%s%s，用户 %s，失败页 %d", *listen, mockBasePath, integrationUser, *failPage)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func newGitLabMockHandler(options mockOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(mockBasePath+"/api/v4/user", func(writer http.ResponseWriter, request *http.Request) {
		if !authenticate(writer, request) {
			return
		}
		writeMockJSON(writer, http.StatusOK, map[string]any{
			"id": 1, "username": integrationUser, "name": "TaskAI Integration User",
		})
	})
	mux.HandleFunc(mockBasePath+"/api/v4/projects", func(writer http.ResponseWriter, request *http.Request) {
		if !authenticate(writer, request) {
			return
		}
		page := 1
		if rawPage := request.URL.Query().Get("page"); rawPage != "" {
			parsed, err := strconv.Atoi(rawPage)
			if err != nil || parsed <= 0 {
				writeMockJSON(writer, http.StatusBadRequest, map[string]string{"message": "invalid page"})
				return
			}
			page = parsed
		}
		if options.failPage == page {
			writeMockJSON(writer, http.StatusInternalServerError, map[string]string{"message": "controlled page failure"})
			return
		}
		projects, nextPage := mockProjects(request.Host, page)
		if nextPage > 0 {
			writer.Header().Set("X-Next-Page", strconv.Itoa(nextPage))
		}
		writeMockJSON(writer, http.StatusOK, projects)
	})
	return mux
}

func authenticate(writer http.ResponseWriter, request *http.Request) bool {
	if request.Header.Get("PRIVATE-TOKEN") == integrationToken {
		return true
	}
	writeMockJSON(writer, http.StatusUnauthorized, map[string]string{"message": "401 Unauthorized"})
	return false
}

func mockProjects(host string, page int) ([]project, int) {
	newProject := func(id int64, name, path, visibility string, archived bool) project {
		hostname := requestHostname(host)
		return project{
			ID:                id,
			Name:              name,
			PathWithNamespace: path,
			SSHURL:            fmt.Sprintf("ssh://git@%s/%s.git", net.JoinHostPort(hostname, mockSSHPort), path),
			HTTPURL:           fmt.Sprintf("http://%s%s/%s.git", host, mockBasePath, path),
			Archived:          archived,
			Visibility:        visibility,
		}
	}
	switch page {
	case 1:
		return []project{
			newProject(101, "api", "team-a/api", "private", false),
			newProject(102, "existing", "team/existing", "private", false),
			newProject(103, "docs", "public/docs", "public", false),
		}, 2
	case 2:
		return []project{
			newProject(201, "api", "team-b/api", "internal", false),
			newProject(202, "legacy", "archive/legacy", "private", true),
			newProject(203, "worker", "platform/worker", "private", false),
		}, 0
	default:
		return []project{}, 0
	}
}

func requestHostname(host string) string {
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		return hostname
	}
	return host
}

func writeMockJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
