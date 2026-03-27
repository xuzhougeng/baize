# myclaw

`myclaw` 现在是一个最小化的 Go 桌面常驻工具，先只做一件事：通过微信收消息，把需要记住的内容存进本地知识库，然后在回答时读取全部知识后直接回复。

当前版本刻意保持简单：

- 知识库存储在本地 JSON 文件里
- 普通提问时，直接读取完整知识库并返回
- 微信桥接只保留扫码登录、长轮询、文本/语音文字收发
- 没有向量检索、没有模型总结、没有多租户隔离

## 目录

```text
cmd/myclaw            程序入口
internal/app          最小消息处理逻辑
internal/knowledge    本地知识库存储
internal/weixin       iLink 微信桥接最小实现
```

## 运行

### 1. 微信扫码登录

```bash
go run ./cmd/myclaw -weixin-login
```

当前实现不依赖第三方 Go 包，但也没有内置终端二维码渲染。执行登录命令后，程序会输出 `qrcode_img_content`，你需要把这段内容生成二维码后再用微信扫码。

登录成功后，凭证会写到 `data/weixin-bridge/account.json`。

### 2. 启动微信桥接

```bash
go run ./cmd/myclaw -weixin
```

或者：

```bash
MYCLAW_WEIXIN_ENABLED=1 go run ./cmd/myclaw
```

### 3. 常用消息

- `记住：Windows 版本先做微信接口`
- `/remember 未来要支持 macOS`
- `/list`
- `/stats`
- `/clear`
- `现在我记了什么？`

## 编译

### Windows 本机

PowerShell:

```powershell
.\scripts\build-windows.ps1
.\scripts\build-windows.ps1 -All
.\scripts\build-windows.ps1 -Arch arm64 -RunTests
```

默认会输出到 `dist/`：

- `dist/myclaw-windows-amd64.exe`
- `dist/myclaw-windows-arm64.exe`（使用 `-All` 或 `-Arch arm64`）

### Linux 交叉编译

```bash
make build-windows
make build-macos
make build-linux
make release
```

其中：

- `make build-windows` 会构建 `windows/amd64` 和 `windows/arm64`
- `make build-macos` 会构建 `darwin/amd64` 和 `darwin/arm64`
- `make build-linux` 会构建 `linux/amd64` 和 `linux/arm64`
- `make release` 会先跑测试，再把三类平台一起编出来

## Commit 规范

仓库内置了 `commit-msg` hook，提交信息必须使用下面三类前缀之一：

- `feat(scope): summary`
- `docs(scope): summary`
- `chore(scope): summary`

例如：

- `feat(weixin): add basic message loop`
- `docs(readme): explain build targets`
- `chore(hooks): enforce commit format`

### 安装 hook

Linux / macOS:

```bash
make install-hooks
```

Windows PowerShell:

```powershell
.\scripts\install-hooks.ps1
```

安装后，仓库会把 `core.hooksPath` 指向 `.githooks`，提交时会自动校验格式。

## 数据文件

- `data/knowledge/entries.json`: 知识库
- `data/weixin-bridge/account.json`: 微信登录凭证
- `data/weixin-bridge/sync_buf`: 微信长轮询游标

## 微信桥接协议说明

微信接入细节参考：[scAgent 文档](/home/xzg/project/scAgent/docs/weixin-bridge.md)。

当前实现只用了这份文档里的最小子集：

- `get_bot_qrcode`
- `get_qrcode_status`
- `getupdates`
- `sendmessage`

## Windows / macOS

目前代码用纯 Go 写成，没有绑死 Windows API，所以结构上已经为未来 macOS 支持留了空间。现阶段仍然按 Windows 桌面常驻进程来用，后续如果要加 GUI、托盘、模型调用或更复杂的能力，可以在这个骨架上继续扩。
