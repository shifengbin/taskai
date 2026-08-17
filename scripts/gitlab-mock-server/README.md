# GitLab 本地模拟服务

该服务只用于 GitLab 批量导入集成测试，提供当前用户接口和两页项目列表，不应部署到生产环境。

## 正常模式

```sh
go run ./scripts/gitlab-mock-server --listen 127.0.0.1:18080
```

- GitLab 地址：`http://127.0.0.1:18080/private/gitlab`
- 用户名：`integration-user`
- Personal Access Token：`integration-token`

项目数据包含不同命名空间中的两个 `api`、一个可作为已导入样本的 `team/existing`、一个归档项目，以及 public、internal、private 三种可见性。HTTP clone 地址带 `/private/gitlab` 相对路径前缀，SSH clone 地址使用独立的 `2424` 端口，用于覆盖私有部署兼容性。

## 第二页失败模式

```sh
go run ./scripts/gitlab-mock-server --listen 127.0.0.1:18080 --fail-page 2
```

该模式让项目第二页返回 HTTP 500，用于验证应用丢弃第一页结果且不展示部分列表。固定令牌只用于本机模拟服务，不得复用于真实 GitLab。
