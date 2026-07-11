# HTTP API

`serve` 命令启动一个本地 HTTP API，用于管理合并规则、下载合并后的订阅，并编辑合并规则引用的模板 YAML。所有管理类接口都挂载在 `/api/` 前缀下，根路径与未匹配 `/api/` 的路径会被作为前端 SPA 处理（详见 [Web UI](#web-ui)）。

## 启动

```bash
go run . serve -config-dir <dir> [-addr 127.0.0.1:8080] [-token <token>]
```

- `-config-dir` 是 API 管理的配置目录，必填；目录不存在时会自动创建。
- `-addr` 是监听地址，默认 `127.0.0.1:8080`。
- `-token` 是 API token；如果未传，会读取 `CLASH_COMPOSER_TOKEN`。
- 如果 `-token` 和 `CLASH_COMPOSER_TOKEN` 都为空，服务会启动失败。
- merge rule 的 `template` 和本地配置源 `path` 在 serve 模式下必须位于 `config-dir` 内；相对路径会基于 `config-dir` 解析。
- merge rule 中的 `cmd` 配置源会在 `config-dir` 内执行。

## Web UI

默认构建会通过 `//go:embed` 把 `webapp/dist` 嵌入二进制，访问 `http://127.0.0.1:8080/` 会得到一个 React 单页应用：

- 根路径 `/` 与未匹配 `/api/` 前缀的路径会回落到 SPA 的 `index.html`，便于前端路由（例如 `/configs/demo`）。
- 静态资源（`/assets/...`）由 `embed.FS` 直接提供。
- 前端使用 Bearer token 鉴权管理类接口；登录页输入的 token 会保存到浏览器 `localStorage`。
- 订阅下载链接由前端按 `http://<host>/api/subscriptions/<id>.yaml?token=<token>` 拼出，可直接复制给 Mihomo / Clash。
- 使用 `go build -tags noembed`（或 `make build-slim`）构建时不嵌入前端，访问根路径会返回 404 提示，仅 `/api/` 可用。

## 鉴权

管理类接口使用 Bearer token：

```http
Authorization: Bearer <token>
```

订阅下载接口使用 query string token，便于 Mihomo/Clash 客户端订阅：

```http
GET /api/subscriptions/{id}.yaml?token=<token>
```

## 通用响应

管理类接口返回 JSON。错误响应格式：

```json
{
  "error": "message"
}
```

常见状态码：

- `400 Bad Request`: 请求路径、JSON 或参数不合法。
- `401 Unauthorized`: token 缺失或错误。
- `404 Not Found`: 配置、模板项、规则分组或规则不存在。
- `405 Method Not Allowed`: HTTP 方法不支持。
- `409 Conflict`: 创建重复资源，或存在重复规则分组导致无法按名称修改。
- `500 Internal Server Error`: 读取、写入、合并或序列化失败。

## 配置 CRUD

配置 ID 会映射为 `config-dir` 下的 `<id>.json`。ID 只能包含字母、数字、`.`、`-`、`_`，并且不能包含 `..`。

### 列出配置

```http
GET /api/configs
Authorization: Bearer <token>
```

响应：

```json
{
  "configs": ["demo"]
}
```

### 创建配置

```http
POST /api/configs/{id}
Authorization: Bearer <token>
Content-Type: application/json
```

请求体是 merge rule JSON：

```json
{
  "template": "template.yaml",
  "configurations": {
    "High": {
      "sources": [
        {
          "path": "high.yaml"
        }
      ],
      "includeDirect": true,
      "enableUrlTest": true,
      "includeGroups": ["Common"]
    },
    "Common": {
      "sources": [
        {
          "url": "https://example.com/subscription.yaml"
        }
      ],
      "includeDirect": true
    }
  },
  "cacheDurationSeconds": 0,
  "rulesetStrategy": "url-ruleset"
}
```

每个配置分组的 `sources` 支持 `path`、`url`、`cmd` 三种形式，每项必须且只能设置一种。`path` 必须解析到 `config-dir` 内，`cmd` 会在 `config-dir` 内执行。

`enableUrlTest` 缺省或为 `true` 时会生成 `<分组名>-UrlTest`；设为 `false` 时只生成 `<分组名>` select，节点和插入项直接进入该 select。`includeDirect` 为 `true` 时会把 `DIRECT` 插入该分组的 select 代理组；`includeGroups` 可以插入其他配置分组名、模板中已有的 proxy group 名称，或内置的 `DIRECT` / `REJECT`。旧版数组结构仍可读取，API 返回时会规范化为对象结构。

创建成功返回 `201 Created` 和保存后的 JSON。若配置已存在，返回 `409 Conflict`。

### 获取配置

```http
GET /api/configs/{id}
Authorization: Bearer <token>
```

响应为 merge rule JSON。

### 替换配置

```http
PUT /api/configs/{id}
Authorization: Bearer <token>
Content-Type: application/json
```

请求体同创建配置。配置不存在时返回 `404 Not Found`。

### 删除配置

```http
DELETE /api/configs/{id}
Authorization: Bearer <token>
```

删除成功返回 `204 No Content`。

## 下载合并后的订阅

```http
GET /api/subscriptions/{id}.yaml?token=<token>
```

- `{id}` 对应 `config-dir` 下的 `<id>.json`。
- 服务会读取 merge rule，合并模板和配置源，并返回 YAML。
- 当 merge rule 的 `cacheDurationSeconds` 大于 `0` 时，服务会把生成后的 YAML 缓存在 `config-dir/.cache/subscriptions/`，未过期请求会直接返回缓存。
- `cacheDurationSeconds` 为 `0` 或缺省时不缓存。
- 响应 `Content-Type` 为 `application/yaml; charset=utf-8`。
- 响应头 `X-Clash-Composer-Cache` 表示缓存状态：`hit`、`miss` 或 `disabled`。
- 该接口不使用 Bearer token，只校验 query string 中的 `token`。

示例：

```bash
curl 'http://127.0.0.1:8080/api/subscriptions/demo.yaml?token=secret'
```

## 上传 config-dir 文件

该接口用于把模板或代理来源 YAML 文件保存到 `config-dir` 内。

```http
POST /api/files?path=templates/base.yaml&overwrite=false
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

表单字段：

- `file`: 要上传的 YAML 文件。

响应：

```json
{
  "path": "templates/base.yaml",
  "size": 1234
}
```

- `path` 必填，必须是相对路径，且必须解析到 `config-dir` 内。
- 只允许 `.yaml` / `.yml` 文件。
- 子目录会自动创建。
- `overwrite` 默认为 `false`；目标文件已存在且未启用覆盖时返回 `409 Conflict`。

## 模板 rule-providers CRUD

这些接口操作配置 `{id}` 引用的模板 YAML 中的 `rule-providers` 字段。

### 列出 rule-providers

```http
GET /api/configs/{id}/template/rule-providers
Authorization: Bearer <token>
```

响应为 `rule-providers` 对象；如果模板中没有该字段，返回空对象：

```json
{}
```

### 创建 rule-provider

```http
POST /api/configs/{id}/template/rule-providers/{name}
Authorization: Bearer <token>
Content-Type: application/json
```

请求体示例：

```json
{
  "type": "http",
  "behavior": "domain",
  "url": "https://example.com/google.txt",
  "path": "./ruleset/google.yaml",
  "interval": 86400
}
```

创建成功返回 `201 Created`。如果 `{name}` 已存在，返回 `409 Conflict`。

### 获取 rule-provider

```http
GET /api/configs/{id}/template/rule-providers/{name}
Authorization: Bearer <token>
```

### 替换 rule-provider

```http
PUT /api/configs/{id}/template/rule-providers/{name}
Authorization: Bearer <token>
Content-Type: application/json
```

请求体同创建。目标不存在时返回 `404 Not Found`。

### 删除 rule-provider

```http
DELETE /api/configs/{id}/template/rule-providers/{name}
Authorization: Bearer <token>
```

删除成功返回 `204 No Content`。

## 模板 rules 分组 CRUD

这些接口操作配置 `{id}` 引用的模板 YAML 中的 `rules` 字段。

模板中的注释会被解释为规则分组：

```yaml
rules:
  - RULE-SET,private,DIRECT

  # steam
  - DOMAIN-SUFFIX,steampowered.com,DIRECT

  # openai
  - DOMAIN-SUFFIX,openai.com,High
```

- 首个分类注释前的规则属于固定分组 `default`。
- 非 `default` 分组名来自规则项前的注释文本，例如 `# steam` 对应 `steam`。
- 写回 YAML 时，非 `default` 分组会写为第一条规则前的注释。
- `default` 分组可以编辑规则，但不能创建、重命名或删除。
- 空分组不会被保留；删除某分组最后一条规则后，该分组会从 YAML 中消失。

### 列出规则分组

```http
GET /api/configs/{id}/template/rule-groups
Authorization: Bearer <token>
```

响应：

```json
[
  {
    "name": "default",
    "rules": ["RULE-SET,private,DIRECT"]
  },
  {
    "name": "steam",
    "rules": ["DOMAIN-SUFFIX,steampowered.com,DIRECT"]
  }
]
```

### 创建规则分组

```http
POST /api/configs/{id}/template/rule-groups
Authorization: Bearer <token>
Content-Type: application/json
```

请求体：

```json
{
  "name": "google",
  "index": 2,
  "rules": ["RULE-SET,google,High"]
}
```

- `index` 必填，表示新分组插入到当前分组列表中的位置。
- `index` 必须大于 `0`，因此不能插入到 `default` 前面。
- `rules` 至少包含一条规则，因为空分组无法稳定写回 YAML。
- 分组已存在时返回 `409 Conflict`。

### 获取规则分组

```http
GET /api/configs/{id}/template/rule-groups/{name}
Authorization: Bearer <token>
```

### 替换规则分组

```http
PUT /api/configs/{id}/template/rule-groups/{name}
Authorization: Bearer <token>
Content-Type: application/json
```

请求体至少包含 `name` 或 `rules` 之一：

```json
{
  "name": "search",
  "rules": [
    "RULE-SET,google,High",
    "DOMAIN-SUFFIX,google.com,High"
  ]
}
```

- 传 `name` 时会重命名分组。
- 传 `rules` 时会替换该分组内的全部规则。
- `default` 分组不能重命名。

### 删除规则分组

```http
DELETE /api/configs/{id}/template/rule-groups/{name}
Authorization: Bearer <token>
```

删除成功返回 `204 No Content`。删除分组会同时删除该分组中的全部规则。`default` 分组不能删除。

### 添加分组内规则

```http
POST /api/configs/{id}/template/rule-groups/{name}/rules
Authorization: Bearer <token>
Content-Type: application/json
```

请求体：

```json
{
  "rule": "DOMAIN-SUFFIX,google.com,High",
  "index": 1
}
```

- `index` 可选；未传时追加到分组末尾。
- `index` 表示分组内规则位置，从 `0` 开始。

成功返回更新后的分组。

### 获取分组内单条规则

```http
GET /api/configs/{id}/template/rule-groups/{name}/rules/{index}
Authorization: Bearer <token>
```

响应：

```json
{
  "rule": "DOMAIN-SUFFIX,google.com,High"
}
```

### 替换分组内单条规则

```http
PUT /api/configs/{id}/template/rule-groups/{name}/rules/{index}
Authorization: Bearer <token>
Content-Type: application/json
```

请求体：

```json
{
  "rule": "DOMAIN-SUFFIX,google.com,High"
}
```

成功返回更新后的分组。

### 删除分组内单条规则

```http
DELETE /api/configs/{id}/template/rule-groups/{name}/rules/{index}
Authorization: Bearer <token>
```

成功返回更新后的分组。如果删除的是非 `default` 分组的最后一条规则，该分组会从 YAML 中消失。
