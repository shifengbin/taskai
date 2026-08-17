package gitlab

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	projectsPerPage = 100
	maxResponseSize = 10 << 20
	maxProjectPages = 10000
)

type ErrorKind string

const (
	ErrorInvalidURL       ErrorKind = "invalid_url"
	ErrorTimeout          ErrorKind = "timeout"
	ErrorCertificate      ErrorKind = "certificate"
	ErrorAuthentication   ErrorKind = "authentication"
	ErrorForbidden        ErrorKind = "forbidden"
	ErrorNotFound         ErrorKind = "not_found"
	ErrorRateLimited      ErrorKind = "rate_limited"
	ErrorUsernameMismatch ErrorKind = "username_mismatch"
	ErrorPagination       ErrorKind = "pagination"
	ErrorResponse         ErrorKind = "response"
)

type RequestError struct {
	Kind       ErrorKind
	Host       string
	Stage      string
	StatusCode int
	Page       int
	cause      error
}

func (current *RequestError) Error() string {
	stage := current.Stage
	if current.Page > 0 {
		stage = fmt.Sprintf("%s第 %d 页", stage, current.Page)
	}
	prefix := "GitLab"
	if current.Host != "" {
		prefix += " " + current.Host
	}
	if stage != "" {
		prefix += " " + stage
	}
	switch current.Kind {
	case ErrorInvalidURL:
		return "GitLab 地址无效，请输入包含主机的 HTTP 或 HTTPS 地址"
	case ErrorTimeout:
		return prefix + "连接超时，请重试"
	case ErrorCertificate:
		return prefix + "证书不受信任，请检查 GitLab 证书"
	case ErrorAuthentication:
		return prefix + "认证失败，请检查个人访问令牌"
	case ErrorForbidden:
		return prefix + "权限不足，请检查令牌的 API 读取权限"
	case ErrorNotFound:
		return prefix + "API 不存在，请检查 GitLab 地址和部署子路径"
	case ErrorRateLimited:
		return prefix + "请求频率受限，请稍后重试"
	case ErrorUsernameMismatch:
		return prefix + "令牌对应用户与输入用户名不一致"
	case ErrorPagination:
		return prefix + "分页响应无效，请重试"
	default:
		if current.StatusCode > 0 {
			return fmt.Sprintf("%s请求失败（HTTP %d），请重试", prefix, current.StatusCode)
		}
		return prefix + "请求失败，请重试"
	}
}

func (current *RequestError) Unwrap() error {
	return current.cause
}

type Project struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"pathWithNamespace"`
	SSHURL            string `json:"sshUrl"`
	HTTPURL           string `json:"httpUrl"`
	Archived          bool   `json:"archived"`
	Visibility        string `json:"visibility"`
	Imported          bool   `json:"imported"`
}

type ConnectionDefaults struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func NormalizeConnectionDefaults(current ConnectionDefaults) (ConnectionDefaults, error) {
	client, err := NewClient(current.Address, 0)
	if err != nil {
		return ConnectionDefaults{}, err
	}
	username := strings.TrimSpace(current.Username)
	if username == "" {
		return ConnectionDefaults{}, fmt.Errorf("GitLab 用户名不能为空")
	}
	if strings.TrimSpace(current.Token) == "" {
		return ConnectionDefaults{}, fmt.Errorf("GitLab 个人访问令牌不能为空")
	}
	return ConnectionDefaults{
		Address:  client.baseURL.String(),
		Username: username,
		Token:    current.Token,
	}, nil
}

type CloneURLMode string

const (
	CloneURLSSH   CloneURLMode = "ssh"
	CloneURLHTTPS CloneURLMode = "https"
)

func (project Project) CloneURL(mode CloneURLMode) (string, error) {
	switch mode {
	case CloneURLSSH:
		return strings.TrimSpace(project.SSHURL), nil
	case CloneURLHTTPS:
		return strings.TrimSpace(project.HTTPURL), nil
	default:
		return "", fmt.Errorf("不支持的 Git 仓库地址格式 %q", mode)
	}
}

type Client struct {
	baseURL    url.URL
	httpClient *http.Client
}

var (
	errCrossHostRedirect = errors.New("cross-host GitLab redirect rejected")
	errInsecureRedirect  = errors.New("GitLab HTTPS downgrade redirect rejected")
)

func NewClient(address string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(address))
	if err != nil || parsed.Host == "" || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, &RequestError{Kind: ErrorInvalidURL}
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	client.CheckRedirect = func(request *http.Request, previous []*http.Request) error {
		if !strings.EqualFold(request.URL.Host, parsed.Host) {
			return errCrossHostRedirect
		}
		previousScheme := parsed.Scheme
		if len(previous) > 0 {
			previousScheme = previous[len(previous)-1].URL.Scheme
		}
		if previousScheme == "https" && request.URL.Scheme != "https" {
			return errInsecureRedirect
		}
		if len(previous) >= 10 {
			return errors.New("too many GitLab redirects")
		}
		return nil
	}
	return &Client{baseURL: *parsed, httpClient: client}, nil
}

func (client *Client) UsesPlainHTTP() bool {
	return client.baseURL.Scheme == "http"
}

func (client *Client) ListAccessibleProjects(ctx context.Context, username, token string) ([]Project, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, &RequestError{Kind: ErrorUsernameMismatch, Host: client.baseURL.Host, Stage: "用户校验"}
	}
	if strings.TrimSpace(token) == "" {
		return nil, &RequestError{Kind: ErrorAuthentication, Host: client.baseURL.Host, Stage: "用户校验"}
	}
	var user struct {
		Username string `json:"username"`
	}
	if err := client.getJSON(ctx, "user", nil, token, "用户校验", 0, &user); err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(user.Username), username) {
		return nil, &RequestError{Kind: ErrorUsernameMismatch, Host: client.baseURL.Host, Stage: "用户校验"}
	}

	projects := make([]Project, 0)
	for page := 1; page <= maxProjectPages; page++ {
		query := url.Values{
			"page":     []string{strconv.Itoa(page)},
			"per_page": []string{strconv.Itoa(projectsPerPage)},
		}
		var current []struct {
			ID                int64  `json:"id"`
			Name              string `json:"name"`
			PathWithNamespace string `json:"path_with_namespace"`
			SSHURL            string `json:"ssh_url_to_repo"`
			HTTPURL           string `json:"http_url_to_repo"`
			Archived          bool   `json:"archived"`
			Visibility        string `json:"visibility"`
		}
		nextPage, err := client.getProjectPage(ctx, query, token, page, &current)
		if err != nil {
			return nil, err
		}
		if current == nil {
			return nil, &RequestError{Kind: ErrorResponse, Host: client.baseURL.Host, Stage: "项目列表", Page: page}
		}
		for _, project := range current {
			if project.ID <= 0 || strings.TrimSpace(project.Name) == "" || strings.TrimSpace(project.PathWithNamespace) == "" || strings.TrimSpace(project.SSHURL) == "" || strings.TrimSpace(project.HTTPURL) == "" {
				return nil, &RequestError{Kind: ErrorResponse, Host: client.baseURL.Host, Stage: "项目列表", Page: page}
			}
			projects = append(projects, Project{
				ID:                project.ID,
				Name:              strings.TrimSpace(project.Name),
				PathWithNamespace: strings.Trim(strings.TrimSpace(project.PathWithNamespace), "/"),
				SSHURL:            strings.TrimSpace(project.SSHURL),
				HTTPURL:           strings.TrimSpace(project.HTTPURL),
				Archived:          project.Archived,
				Visibility:        strings.TrimSpace(project.Visibility),
			})
		}
		if nextPage == 0 {
			return projects, nil
		}
		if nextPage != page+1 || nextPage > maxProjectPages {
			return nil, &RequestError{Kind: ErrorPagination, Host: client.baseURL.Host, Stage: "项目列表", Page: page}
		}
		page = nextPage - 1
	}
	return nil, &RequestError{Kind: ErrorPagination, Host: client.baseURL.Host, Stage: "项目列表"}
}

func (client *Client) getProjectPage(ctx context.Context, query url.Values, token string, page int, destination any) (int, error) {
	response, err := client.do(ctx, "projects", query, token, "项目列表", page)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if err := decodeJSON(response.Body, destination); err != nil {
		return 0, &RequestError{Kind: ErrorResponse, Host: client.baseURL.Host, Stage: "项目列表", Page: page, cause: err}
	}
	next := strings.TrimSpace(response.Header.Get("X-Next-Page"))
	if next == "" {
		return 0, nil
	}
	nextPage, err := strconv.Atoi(next)
	if err != nil || nextPage <= 0 {
		return 0, &RequestError{Kind: ErrorPagination, Host: client.baseURL.Host, Stage: "项目列表", Page: page, cause: err}
	}
	return nextPage, nil
}

func (client *Client) getJSON(ctx context.Context, endpoint string, query url.Values, token, stage string, page int, destination any) error {
	response, err := client.do(ctx, endpoint, query, token, stage, page)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := decodeJSON(response.Body, destination); err != nil {
		return &RequestError{Kind: ErrorResponse, Host: client.baseURL.Host, Stage: stage, Page: page, cause: err}
	}
	return nil
}

func (client *Client) do(ctx context.Context, endpoint string, query url.Values, token, stage string, page int) (*http.Response, error) {
	requestURL := client.baseURL
	requestURL.Path = strings.TrimRight(client.baseURL.Path, "/") + "/api/v4/" + endpoint
	requestURL.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, &RequestError{Kind: ErrorResponse, Host: client.baseURL.Host, Stage: stage, Page: page, cause: err}
	}
	request.Header.Set("PRIVATE-TOKEN", token)
	request.Header.Set("Accept", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, classifyTransportError(err, client.baseURL.Host, stage, page)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		response.Body.Close()
		return nil, &RequestError{Kind: classifyStatus(response.StatusCode), Host: client.baseURL.Host, Stage: stage, StatusCode: response.StatusCode, Page: page}
	}
	return response, nil
}

func classifyStatus(status int) ErrorKind {
	switch status {
	case http.StatusUnauthorized:
		return ErrorAuthentication
	case http.StatusForbidden:
		return ErrorForbidden
	case http.StatusNotFound:
		return ErrorNotFound
	case http.StatusTooManyRequests:
		return ErrorRateLimited
	default:
		return ErrorResponse
	}
}

func classifyTransportError(err error, host, stage string, page int) error {
	requestError := &RequestError{Kind: ErrorResponse, Host: host, Stage: stage, Page: page, cause: err}
	if errors.Is(err, errCrossHostRedirect) {
		return requestError
	}
	if errors.Is(err, context.DeadlineExceeded) {
		requestError.Kind = ErrorTimeout
		return requestError
	}
	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		requestError.Kind = ErrorTimeout
		return requestError
	}
	var verificationError *tls.CertificateVerificationError
	var unknownAuthorityError x509.UnknownAuthorityError
	var hostnameError x509.HostnameError
	var certificateInvalidError x509.CertificateInvalidError
	if errors.As(err, &verificationError) || errors.As(err, &unknownAuthorityError) || errors.As(err, &hostnameError) || errors.As(err, &certificateInvalidError) {
		requestError.Kind = ErrorCertificate
	}
	return requestError
}

func decodeJSON(reader io.Reader, destination any) error {
	limited := &io.LimitedReader{R: reader, N: maxResponseSize + 1}
	decoder := json.NewDecoder(limited)
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if limited.N <= 0 {
		return fmt.Errorf("GitLab response exceeds size limit")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("GitLab response contains trailing data")
		}
		return err
	}
	return nil
}
