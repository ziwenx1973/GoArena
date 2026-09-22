# 验证记录

本文件记录实际执行结果，不把测试替身、环境跳过或配置文件等同于完整部署成功。

## Windows 开发机

验证日期：2026-09-22。Go 1.27.1 / windows-amd64，另尝试 Go 1.26.0。

| 检查 | 结果 |
| --- | --- |
| go mod tidy | 通过；默认代理超时后使用 goproxy.cn 下载依赖 |
| gofmt | 已格式化全部 Go 文件 |
| go vet ./... | 通过 |
| go test ./... | 通过；真实依赖集成测试明确 SKIP |
| go build ./cmd/server | 通过，生成 server.exe（已忽略） |
| go test -race ./... | Windows 测试进程启动失败，退出码 0xc0000139；Go 1.27.1 与 1.26.0 均出现，不能宣称通过 |
| 本地真实 MySQL + Redis 链路 | 未执行；本机有 MySQL 服务，但未发现 Redis 服务/可执行文件，也没有提供应用数据库凭据 |
| Docker Compose | 未执行；开发机未安装 Docker。仅人工检查服务名、环境变量、依赖、健康检查、volume 和网络配置 |

普通测试覆盖 JWT、九种出拳组合、平局与两胜结束、重复/旧回合出拳、断线、数据库保存失败、房间退出及慢客户端。数据库保存失败测试故意使用返回错误的 RecordStore，因此测试日志中的 `Database error` 是预期路径。

## GitHub Actions

工作流配置了 Ubuntu、MySQL 8.4、Redis 7.4，以及普通检查和真实端到端测试。运行状态以 [Go CI](https://github.com/ziwenx1973/GoArena/actions/workflows/ci.yml) 中具体提交的结果为准。

真实集成测试由 `ARENA_INTEGRATION=1` 启用，不允许使用假数据库冒充通过。它验证真实 HTTP/JWT/WebSocket、取消与重复匹配、平局与两胜结束、MySQL 保存、战绩查询和断线判负。

CI 中运行数据库服务容器不代表项目 Dockerfile 或整个 Compose 已被验证；后者仍需安装 Docker 后单独执行。
