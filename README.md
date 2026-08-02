# Clash Composer

`clash-composer` 是一个用于管理和组合 Mihomo/Clash 配置的小工具。  

当前工具适合把不同来源的代理节点整理到统一模板里，再生成一份 `merged.yaml`。除了 CLI，还内置了一个 HTTP API 与对应的 React Web UI（默认嵌入二进制）。

## 构建

默认构建会先编译 `webapp/`，再把构建产物 (`webapp/dist`) 嵌入 Go 二进制：

```bash
make build
```

构建产物位于：

```bash
build/clash-composer
```

如果不需要前端，可以使用 slim 构建（添加 `noembed` build tag，跳过 `pnpm install` / `pnpm run build`）：

```bash
make build-slim
```

清理构建产物：

```bash
make clean        # 仅清理 build/
make distclean    # 同时清理 webapp/node_modules 和 webapp/dist 内容
```

## 部署

部署脚本会读取仓库根目录的 `.env`，交叉编译目标机器二进制，上传到远端并重启用户级 systemd 服务。`.env` 包含远程机器密码，已被 `.gitignore` 忽略；仓库只提交 `.env.example`。

```bash
cp .env.example .env
$EDITOR .env
make deploy
```

脚本依赖 `sshpass`、`ssh`、`scp` 和 `make`。当前默认示例面向 `linux/arm64`，可在 `.env` 中调整 `DEPLOY_GOOS`、`DEPLOY_GOARCH` 和安装路径。

## 使用

### 0. 查看版本

```bash
go run . version
```

### 1. 合并配置

```bash
go run . merge example/merge.json
```

执行后会在当前目录生成：

```bash
merged.yaml
```

`merge` 使用一个 JSON 规则文件描述模板和配置来源，例如：

```json
{
  "template": "example/template.yaml",
  "configurations": [
    {
      "name": "High",
      "sources": [
        { "path": "example/high.yaml" }
      ],
      "includeDirect": true,
      "includeGroups": [
        { "name": "Common", "mode": "proxy" }
      ]
    },
    {
      "name": "Common",
      "sources": [
        { "path": "example/common.yaml" }
      ],
      "includeDirect": true
    }
  ],
  "cacheDurationSeconds": 0,
  "rulesetStrategy": "url-ruleset"
}
```

`configurations` 是有序数组，数组顺序决定生成的代理分组顺序。每个配置分组默认会生成 `<分组名>-UrlTest` 和 `<分组名>` 两个 proxy group。`enableUrlTest` 设为 `false` 时只生成 `<分组名>` select，节点、`DIRECT` 和插入项会直接加入该 select。`includeDirect` 会把 `DIRECT` 插入到 select 分组中。

`includeGroups` 是有序插入条目数组。`mode: "proxy"` 会把目标分组作为一个代理节点插入；`mode: "flatten"` 会把目标配置分组来源中的代理展开到当前分组列表。展开模式只能选择其他 `configurations` 分组，模板中的 proxy group 和 `DIRECT` / `REJECT` 只能使用 `proxy` 模式。旧版 `includeGroups: ["Common"]` 会在服务启动时迁移为 `[{"name":"Common","mode":"proxy"}]`。

`cacheDurationSeconds` 控制订阅缓存时长，单位秒；`0` 或缺省表示每次下载订阅时实时生成。

配置来源 `sources` 支持三种形式，且每项只能设置一种：

- `path`: 从本地文件读取
- `url`: 从 HTTP 订阅地址下载
- `cmd`: 从命令的 `stdout` 读取

`cmd` 会在 merge JSON 文件所在目录执行。

### 2. 下载订阅

```bash
go run . download https://your-subscription-url
```

`download` 会将下载内容直接输出到 `stdout`，因此通常配合重定向使用：

```bash
go run . download https://your-subscription-url > subscription.yaml
```

### 3. 启动 HTTP 服务 + Web UI

```bash
./build/clash-composer serve -config-dir ./configs -token <secret>
```

- HTTP API 挂载在 `/api/` 前缀下，详见 [docs/http.md](docs/http.md)。
- 默认构建会同时提供 Web UI；浏览器打开 `http://127.0.0.1:8080/` 即可使用。
- slim 构建（`make build-slim`）只暴露 `/api/`，根路径会返回 404 提示。

## Web UI 开发

Web UI 源码位于 `webapp/`（React + Vite + TypeScript + Tailwind + shadcn/ui），开发流程：

```bash
make dev
```

`make dev` 会先构建后端，再启动测试后端和 Vite 前端。默认后端监听 `127.0.0.1:8080`，配置目录为 `./configs`，登录 token 为 `dev`。打开 `http://127.0.0.1:5173/` 即可。

可以通过 Make 变量覆盖默认值：

```bash
make dev DEV_CONFIG_DIR=./configs DEV_TOKEN=dev DEV_ADDR=127.0.0.1:8080
```

## 示例文件

- `example/template.yaml`: 基础模板配置
- `example/high.yaml`: High 组示例代理
- `example/common.yaml`: Common 组示例代理
- `example/merge.json`: 合并示例规则

## 测试

```bash
go test ./...
```
