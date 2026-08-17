package gitlab

import "testing"

func TestParseRepositoryIdentityNormalizesGitLabCloneAddresses(t *testing.T) {
	addresses := []string{
		"git@GitLab.Example.com:team/platform/api.git",
		"ssh://git@gitlab.example.com/team/platform/api.git",
		"https://gitlab.example.com/team/platform/api.git",
		"http://GITLAB.EXAMPLE.COM//team/platform/api.git/",
	}
	for _, address := range addresses {
		identity, err := ParseRepositoryIdentity(address)
		if err != nil {
			t.Fatalf("ParseRepositoryIdentity(%q) error = %v", address, err)
		}
		if identity.Host != "gitlab.example.com" || identity.Path != "team/platform/api" {
			t.Fatalf("ParseRepositoryIdentity(%q) = %#v", address, identity)
		}
	}
	for index := 1; index < len(addresses); index++ {
		if !SameRepository(addresses[0], addresses[index]) {
			t.Fatalf("SameRepository(%q, %q) = false", addresses[0], addresses[index])
		}
	}
}

func TestRepositoryIdentityKeepsDifferentHostsAndPathsDistinct(t *testing.T) {
	base := "git@gitlab-a.example.com:team/api.git"
	if SameRepository(base, "https://gitlab-b.example.com/team/api.git") {
		t.Fatal("不同主机的相同路径被识别为重复")
	}
	if SameRepository(base, "https://gitlab-a.example.com/team/web.git") {
		t.Fatal("同主机的不同路径被识别为重复")
	}
	if SameRepository("ssh://git@gitlab.example.com:2222/team/api.git", "ssh://git@gitlab.example.com/team/api.git") {
		t.Fatal("不同端口的仓库被识别为重复")
	}
}

func TestParseRepositoryIdentityRejectsUnsupportedAddresses(t *testing.T) {
	addresses := []string{
		"",
		"team/api",
		"file:///tmp/api.git",
		"https:///team/api.git",
		"https://gitlab.example.com",
		"https://user:secret@gitlab.example.com/team/api.git",
		"https://gitlab.example.com/team/api.git?token=secret",
	}
	for _, address := range addresses {
		if _, err := ParseRepositoryIdentity(address); err == nil {
			t.Fatalf("ParseRepositoryIdentity(%q) error = nil", address)
		}
	}
	if SameRepository("not-a-repository", "not-a-repository") {
		t.Fatal("无法解析的相同字符串被识别为重复")
	}
}
