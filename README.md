# Clash Composer

`clash-composer` 用于组合 Mihomo/Clash 配置：从本地文件、订阅 URL 或命令读取代理，合并到模板 YAML，并输出订阅。它同时提供 CLI、HTTP API 和内嵌 Web UI。

## 功能

- 按配置分组合并代理来源，支持 `DIRECT`、分组插入和 URLTest。
- 下载或生成 Mihomo/Clash 可直接使用的订阅。
- 通过 Web UI 管理合并规则、模板规则、规则提供器和配置目录文件。

## 构建

```bash
make build
```

默认构建会编译并嵌入 Web UI，产物为 `build/clash-composer`。只需要 API/CLI 时可使用 `make build-slim`。

## 快速使用

合并示例配置：

```bash
./build/clash-composer merge example/merge.json
```

命令会在当前目录写入 `merged.yaml`。下载原始订阅：

```bash
./build/clash-composer download <subscription-url> > subscription.yaml
```

启动 API 和 Web UI：

```bash
./build/clash-composer serve -config-dir ./configs -token <secret>
```

服务默认监听 `127.0.0.1:8080`，浏览器访问 `http://127.0.0.1:8080/`。`config-dir` 不存在时会自动创建；其中的本地模板、配置源和命令均在该目录范围内解析或执行。

## 配置与 API

合并规则格式和可用配置项可参考 [example/merge.json](example/merge.json)。完整的 HTTP API、Web UI 鉴权、文件上传、规则管理与订阅缓存说明见 [docs/http.md](docs/http.md)。

## 部署

部署脚本从仓库根目录的 `.env` 读取目标主机信息，交叉编译、上传二进制并重启用户级 systemd 服务：

```bash
cp .env.example .env
$EDITOR .env
make deploy
```

`.env` 包含凭据，不能提交到仓库。变量含义见 [.env.example](.env.example)。

## 参与开发

开发环境、前后端调试、验证与提交约定见 [CONTRIBUTING.md](CONTRIBUTING.md)。
