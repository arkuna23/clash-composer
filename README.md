# Clash Composer

`clash-composer` 是一个用于管理和组合 Mihomo/Clash 配置的小工具。  

当前工具适合把不同来源的代理节点整理到统一模板里，再生成一份 `merged.yaml`。

## 构建

```bash
make build
```

构建产物位于：

```bash
build/clash-composer
```

清理构建产物：

```bash
make clean
```

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
  "configurations": {
    "High": [
      { "path": "example/high.yaml" }
    ],
    "Common": [
      { "path": "example/common.yaml" }
    ]
  },
  "rulesetStrategy": "url-ruleset"
}
```

配置来源支持三种形式，且每项只能设置一种：

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

## 示例文件

- `example/template.yaml`: 基础模板配置
- `example/high.yaml`: High 组示例代理
- `example/common.yaml`: Common 组示例代理
- `example/merge.json`: 合并示例规则

## 测试

```bash
go test ./...
```
