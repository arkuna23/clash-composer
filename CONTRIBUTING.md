# 贡献指南

本项目由 Go 后端和 React Web UI 组成。提交修改前请保持改动范围聚焦，并完成与改动相关的校验。

## 目录结构

- `main.go`：CLI 入口，提供 `version`、`merge`、`download` 和 `serve` 命令。
- `composer/`：合并、订阅下载、HTTP API 与模板管理流程。
- `config/`：Mihomo 原始配置类型和 YAML 编解码。
- `webapp/`：React、Vite、TypeScript Web UI。
- `example/`：示例模板、代理和合并规则。
- `docs/http.md`：HTTP API 和配置字段完整参考。
- `scripts/`：部署及规则集辅助脚本。

## 环境准备

安装 `go.mod` 声明的 Go 版本，以及 Node.js 和 pnpm。Web UI 使用仓库提交的 `webapp/pnpm-lock.yaml` 锁定依赖。

```bash
cd webapp
pnpm install --frozen-lockfile
```

不要把个人订阅、凭据或 `.env` 提交到仓库；部署变量以 `.env.example` 为准。

## 构建与本地开发

```bash
make build       # 构建并嵌入 Web UI
make build-slim  # 不构建或嵌入 Web UI
make clean       # 删除 build/
make distclean   # 额外删除 Web UI 构建产物和依赖
```

本地联调使用：

```bash
make dev
```

该命令会在空的配置目录中填充 `example/dev/` 样例，启动后端和本地 URL fixture，再启动 Vite。已有配置目录不会被覆盖。默认后端为 `127.0.0.1:8080`，配置目录为 `./configs`，token 为 `dev`；Web UI 位于 `http://127.0.0.1:5173/`。样例 URL 来源使用本机 `127.0.0.1:8091` 的 fixture 服务，不依赖公网。

样例包含两个配置条目、`path`/`url`/`cmd` 三种来源、DIRECT、proxy/flatten 插入、启用和关闭 URLTest、缓存、两种 ruleset 策略、模板 rule-providers、模板 proxy groups 以及规则类型示例。

可覆盖这些变量：

```bash
make dev DEV_CONFIG_DIR=./configs DEV_TOKEN=dev DEV_ADDR=127.0.0.1:8080
```

## 编码约定

- Go 代码遵循标准 Go 风格，编辑后执行 `gofmt`；编排代码放在 `composer/`，配置类型放在 `config/`。
- TypeScript 使用现有的 React、Tailwind 和 shadcn/ui 模式，优先复用现有组件和 API 类型。
- 合并输出必须保留模板 YAML 中原有字段、顺序与样式；不要因反序列化默认值向模板补充不存在的字段。
- HTTP `serve` 模式下，模板和本地来源必须限制在 `config-dir` 中；命令来源也以 `config-dir` 为工作目录执行。

## 验证

按改动范围执行以下检查。修改合并行为、示例或配置序列化后，应同时运行示例合并：

```bash
go test ./...
make build
go run . merge example/merge.json
git diff --check
```

如默认 Go 缓存不可写，可使用仓库内缓存：

```bash
GOCACHE="$(pwd)/build/.gocache" go test ./...
```

`make build` 会执行 Web UI 的类型检查和生产构建。不要将 `pnpm run lint` 作为当前项目的必需校验命令。

## 提交与部署

每次完成代码或文档修改并通过相关测试后，创建一个聚焦的 Git 提交。提交标题使用简短的祈使句，例如 `Document contributor workflow`；不要混入无关格式化或生成文件。

部署前复制 `.env.example` 为本地 `.env` 并填写目标信息，然后执行：

```bash
make deploy
```

部署依赖 `sshpass`、`ssh`、`scp` 和 `make`。脚本会交叉编译、上传目标二进制并重启 `.env` 中配置的用户级 systemd 服务。
