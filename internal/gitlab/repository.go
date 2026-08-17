package gitlab

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

type RepositoryIdentity struct {
	Host string
	Path string
}

func ParseRepositoryIdentity(address string) (RepositoryIdentity, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return RepositoryIdentity{}, fmt.Errorf("Git 仓库地址不能为空")
	}
	if !strings.Contains(address, "://") {
		return parseSCPRepositoryIdentity(address)
	}
	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return RepositoryIdentity{}, fmt.Errorf("无法解析 Git 仓库地址")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "ssh" {
		return RepositoryIdentity{}, fmt.Errorf("不支持的 Git 仓库地址协议")
	}
	if parsed.Scheme != "ssh" && parsed.User != nil {
		return RepositoryIdentity{}, fmt.Errorf("Git 仓库地址不能包含用户凭据")
	}
	host := normalizeRepositoryHost(parsed.Hostname(), parsed.Port())
	path, err := normalizeRepositoryPath(parsed.Path)
	if err != nil {
		return RepositoryIdentity{}, err
	}
	return RepositoryIdentity{Host: host, Path: path}, nil
}

func SameRepository(left, right string) bool {
	leftIdentity, err := ParseRepositoryIdentity(left)
	if err != nil {
		return false
	}
	rightIdentity, err := ParseRepositoryIdentity(right)
	if err != nil {
		return false
	}
	return leftIdentity == rightIdentity
}

func ProjectRepositoryIdentities(project Project) ([]RepositoryIdentity, error) {
	projectPath, err := normalizeRepositoryPath(project.PathWithNamespace)
	if err != nil {
		return nil, fmt.Errorf("GitLab 项目完整路径无效")
	}
	addresses := []struct {
		name  string
		value string
	}{
		{name: "SSH", value: project.SSHURL},
		{name: "HTTPS", value: project.HTTPURL},
	}
	identities := make([]RepositoryIdentity, 0, len(addresses))
	seen := make(map[RepositoryIdentity]struct{}, len(addresses))
	for _, address := range addresses {
		identity, parseErr := ParseRepositoryIdentity(address.value)
		if parseErr != nil {
			return nil, fmt.Errorf("GitLab 项目 %q 的 %s 地址无效", project.PathWithNamespace, address.name)
		}
		if identity.Path != projectPath && !strings.HasSuffix(identity.Path, "/"+projectPath) {
			return nil, fmt.Errorf("GitLab 项目 %q 的 %s 地址与完整路径不一致", project.PathWithNamespace, address.name)
		}
		if _, duplicate := seen[identity]; !duplicate {
			identities = append(identities, identity)
			seen[identity] = struct{}{}
		}
	}
	return identities, nil
}

func parseSCPRepositoryIdentity(address string) (RepositoryIdentity, error) {
	separator := strings.IndexByte(address, ':')
	if separator <= 0 || separator == len(address)-1 {
		return RepositoryIdentity{}, fmt.Errorf("无法解析 Git 仓库地址")
	}
	left := address[:separator]
	if strings.ContainsAny(left, "/\\ ") {
		return RepositoryIdentity{}, fmt.Errorf("无法解析 Git 仓库地址")
	}
	host := left
	if at := strings.LastIndexByte(left, '@'); at >= 0 {
		host = left[at+1:]
	}
	host = normalizeRepositoryHost(host, "")
	if host == "" {
		return RepositoryIdentity{}, fmt.Errorf("Git 仓库地址缺少主机")
	}
	path, err := normalizeRepositoryPath(address[separator+1:])
	if err != nil {
		return RepositoryIdentity{}, err
	}
	return RepositoryIdentity{Host: host, Path: path}, nil
}

func normalizeRepositoryHost(host, port string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	if port != "" {
		return net.JoinHostPort(host, port)
	}
	return host
}

func normalizeRepositoryPath(path string) (string, error) {
	segments := strings.Split(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"), "/")
	normalized := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." {
			continue
		}
		if segment == ".." {
			return "", fmt.Errorf("Git 仓库路径不能包含上级目录")
		}
		normalized = append(normalized, segment)
	}
	if len(normalized) < 2 {
		return "", fmt.Errorf("Git 仓库地址缺少完整项目路径")
	}
	last := normalized[len(normalized)-1]
	if strings.HasSuffix(strings.ToLower(last), ".git") {
		last = last[:len(last)-4]
	}
	if last == "" {
		return "", fmt.Errorf("Git 仓库地址缺少项目名称")
	}
	normalized[len(normalized)-1] = last
	return strings.Join(normalized, "/"), nil
}
