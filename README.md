# Service Unlock v0.3.1 · Linux 独立程序

**无需安装 Node、Python 或 Go。** 提供 Linux amd64 / arm64 静态二进制，以及自动选择架构、校验 SHA256、运行后清理的 POSIX `check.sh`。

固定检测 **7 个平台、8 项**：Netflix、YouTube Premium、TikTok、ChatGPT Web、ChatGPT App、Gemini、NotebookLM、Claude。ChatGPT 两项分列，不生成合并“解锁”结论。不需要账号，不做测速、GeoIP、路由、IP 评分或 GUKO 接入。

下载短链：<https://check.siaoz.si>，固定提供 v0.3.1 的 `check.sh`。旧 Python / Node.js 版本在历史 Release 中保留。

> v0.3.1 恢复原版 Geocheck 的终端输出格式；检测逻辑与 JSON 结构不变。已下载过旧 `check.sh` 的用户请重新执行下面的下载命令，旧脚本不会自动追随新版。

## 一键使用

需要现有 `curl` 或 GNU `wget`、`sha256sum` / `shasum` / `openssl` 任一种，以及常见 POSIX 工具（`sh`、`uname`、`mktemp`、`chmod`、`rm`）。不会自动安装任何工具，不需要 sudo。

先完整下载启动脚本，成功后才运行。`check.sh` 留在当前目录供再次使用；检测程序只放在本次临时目录，结束后清理，不安装常驻服务。

```sh
curl -fLsS https://check.siaoz.si -o check.sh && sh check.sh
```

输出 JSON：`sh check.sh --json`；也可指定 `sh check.sh --timeout 10 --workers 4`。

已有 `check.sh` 时直接运行：

```sh
sh check.sh
sh check.sh --json
sh check.sh --timeout 10 --workers 4
```

启动器识别 `x86_64` / `amd64` 与 `aarch64` / `arm64`，不支持的系统或架构会明确拒绝。固定从以下版本下载，不追随 latest，不执行安装器：

```
https://github.com/shui1iao/service-unlock/releases/download/v0.3.1/
```

下载失败、HTTP 错误、缺失/重复/无效清单条目、SHA256 不一致都拒绝执行。参数及检测退出码原样传递，启动错误仅写 stderr，`--json` 的 stdout 不混入下载日志。Ctrl-C / TERM 会终止被跟踪的下载器或检测进程并清理本次临时目录。清单和程序都依赖同一 GitHub HTTPS 发布源，**哈希校验不是独立发布签名**。

## 直接运行二进制

选与机器对应的资产，校验后执行，例如 amd64：

```sh
chmod +x service-unlock_linux_amd64
./service-unlock_linux_amd64 --help       # 不联网
./service-unlock_linux_amd64 --license    # 查看内嵌MIT许可，不联网
./service-unlock_linux_amd64
./service-unlock_linux_amd64 --json > result.json
```

arm64 对应 `service-unlock_linux_arm64`。程序通过 `CGO_ENABLED=0` 构建，不依赖动态 libc 或语言运行时；仍需要正常的 Linux 网络、DNS 和系统 CA 证书。临时目录若被挂载为 `noexec`，可将 `TMPDIR` 指向自己可写、允许执行的目录；不要因此关闭证书验证或修改系统挂载配置。只支持所列 64 位架构，不承诺过旧 Linux 内核兼容性。

- `--timeout`：默认 **6 秒/GET 总预算**，含 DNS、连接、TLS、重定向与正文读取；解压也在返回前检查同一预算。支持 `(0,2147483.647]` 秒，按毫秒向上取整。Netflix / Claude 可能有两次 GET，并非整轮总时限。
- `--workers`：默认 4，范围 1–8。全部 8 项固定输出，无新增平台。
- 检测本身始终直连，忽略代理环境变量；无 Cookie、无账号。系统选择地址族，没有分别验证 IPv4/IPv6。启动器的 curl/wget 下载传输可遵循它们自己的代理环境配置，这不代表程序检测走代理。
- TLS 验证开启，使用系统 CA；不提供跳过证书验证选项。
- 退出码：`0`＝全部得到结论（包括 restricted / blocked）；`1`＝至少一项 error 或启动/执行失败；`2`＝检测参数错误；`130`＝Ctrl-C；`143`＝TERM。退出 0 **不表示全部解锁**。

## 输出与判据

终端直接复用 **geocheck v0.3.0 的 Stash checks 渲染函数和主题**，不是自行绘制的类似表格：列顺序为 `Service / Status / Region / Detail / Time`，圆角外框、表头分隔线，无列间/行间分隔线，保持原服务顺序。`● available` 绿、`● restricted` 黄、`● blocked` 红、`○ error` 灰。地区缺失显示 `—`，Detail 使用原版 `clip(..., 52)`（按字节截断），总耗时按真实检查执行区间显示 `completed in %.1fs`。

表后仅统计非零的 `available / restricted / blocked / inconclusive` 数量，以 `·` 分隔；这不是新的总体解锁状态。Time 严格沿用原版：有底层 `Err` 或零耗时显示 `—`；只有 `StateError`、没有底层 `Err` 的未能判定结果仍显示耗时。耗时格式和 30/80/160 ms 配色阈值不变。

TTY、宽度、深浅背景、`NO_COLOR` 均使用原版 `theme.go`：普通重定向输出为无 ANSI 文本；`NO_COLOR` 优先于 `CLICOLOR_FORCE`，关闭颜色但按原版仍可能保留加粗/复位 SGR；`COLORFGBG` 决定深浅背景，不向终端发送背景查询。默认宽度 100，检测到大于 40 列的真实终端时采用实际宽度。`--json` 完全绕过终端渲染，不混入 ANSI、表格、计数或耗时页脚。

| status | 含义 |
|---|---|
| `available` | 符合相应可用性判据，不保证实际播放、付费或登录后功能 |
| `restricted` | Netflix 仅自制剧判据成立 |
| `blocked` | 明确拒绝标记，或 Netflix 两个片名都返回 404 |
| `error` | 未能测定，包括请求、HTTP、挑战页、解析失败；不是“未解锁” |

JSON 保留前两版结构：`schema_version: 1`、`source`、`generated_at`、`network`、`results`。每项有 `id/platform/name/status/detail/region/error/checked_at/elapsed_ms`。没有顶层聚合状态。非 error 的 `error` 为 `null`；缺失地区为 `null`。`checked_at` 是 UTC 开始时间，耗时包含子请求但不含并发排队。程序应读取稳定的 ID / status，不解析错误文本。

| id | 原始端点与关键判断 |
|---|---|
| `chatgpt_web` | `api.openai.com/compliance/cookie_requirements`：`unsupported_country` |
| `chatgpt_app` | `ios.chat.openai.com`：`disallowed isp` / `been blocked` |
| `youtube_premium_access` | `www.youtube.com/premium`：地区拒绝句优先，否则需 `ad-free` |
| `netflix_access` | `www.netflix.com/title/70143836`，必要时 `80197526`；授权片 / 自制片区分完整与受限片库，地区从最终片名 URL 取 |
| `claude_access` | `claude.ai/cdn-cgi/trace` 尽力读地区，再查 `claude.ai/` 的原始区域拒绝文案 |
| `tiktok_access` | `www.tiktok.com/`：保留四条地区拒绝标记 |
| `gemini_access` | `gemini.google.com/app`：提取三字母地区；保留 RUS、BLR、CHN、PRK、IRN、CUB、SYR 不支持列表 |
| `notebooklm_access` | `notebooklm.google.com/`：最终 URL `location=unsupported` 优先；Google 登录页表示需要账号但地区被服务 |

均为 HTTPS。地区只在 Netflix、Claude、Gemini 有真实提取依据时填写，不推断 GeoIP。

### 已保留的改进与必要差异

直接复用上游 Go `internal/access` 检测代码和测试，应用前两版的保守修正：429/5xx、网络/超时/解析失败为 error；先识别 Cloudflare 挑战；正文型检查空白正文不算成功；无拒绝证据的异常 HTTP 不算解锁。服务特定 403 标记仍有效，不一律抹成未知。Netflix 仅用 404 作为片名未提供证据，并严格验证目的地片名/地区；NotebookLM 异常 HTTP 登录地址不算 available；Claude trace 失败不决定主结果。

与 JS v0.2.0 相比，字段与平台判据不变；`source.port` 更新为 Go 独立版，`network.timeout_scope` 仍为 `request including redirects and body`。网络使用 Go 标准库，恢复上游的 HTTP/2 尝试及连接复用（必要时 HTTP/1.1），TLS 指纹、URL 解析、纯 Go DNS 与 Node/系统 NSS 可能不同。没有移植上游的固定地址族、网卡绑定、SOCKS5 或 DoH。

最终响应原始及 gzip 解压后各限 4 MiB；截断、损坏压缩或不支持编码为 error。最多跟随 9 次重定向，第 10 次报错。Go 标准库负责中间重定向正文的有限排空，不像 JS 版对每个中间正文完整解压校验；最终响应及目的地判断不变。异常错误文本不是兼容性枚举。

**原始算法限制仍在：** ChatGPT 正常非空响应主要凭“没有拒绝词”，TikTok/Claude 部分分支依赖正常 HTTP；Netflix 片名版权、Gemini 国家列表、页面结构与反机器人策略都会变化。不是浏览器模拟、播放验证或真实模型调用，不能保证登录、播放及账户权限。

## 构建与离线验证（仅开发者需要 Go）

用户运行发行二进制不需要此步骤。检测逻辑仍用 Go 标准库；原版终端渲染增加上游固定版本的 lipgloss v2.0.6、colorprofile v0.4.3、x/term v0.2.2 及必要传递依赖，均锁定在 go.mod/go.sum。依赖要求 Go 1.26+，实际使用 **Go 1.27.2**。`godebug default=go1.22` 保留已发布 v0.3.0 的 Go 运行时默认值，避免引入无关探测行为变化。首次构建需下载固定模块，之后可离线测试。

```sh
GOTOOLCHAIN=local GOPROXY=https://proxy.golang.org go mod download
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go test ./...
sh -n check.sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOARM64=v8.0 go build -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_arm64 .
```

原有 **39 个顶层离线回归**继续通过：四状态、八项列表、关键标记、异常 HTTP/解析/网络/超时、JSON/退出码/并发、回环 HTTP 重定向/gzip/大小限制/截断、拒绝不可信 TLS；启动器用 mock curl/wget 检查固定 URL、架构、三种哈希工具、失败拒绝执行、参数/返回码、INT/TERM 子进程终止及清理。

新增格式测试以固定 mock 输入与独立提取的上游原始函数生成的 golden 文件进行 **plain / truecolor / NO_COLOR 逐字节比对**，并覆盖表格形状、顺序、状态颜色、缺失地区/错误耗时、截断、计数、真实执行总耗时、本地 PTY 宽度/颜色检测及 JSON 字段隔离。参考文件和输入位于 `internal/render/testdata/`，全部是示例，不代表真实检测。本轮合计 **50 个顶层测试、含子用例 105 项全部通过，无跳过**。

**amd64**：实际运行测试、发行二进制 `--help` 及同渲染包的离线 fixture 演示。**arm64**：仅交叉构建并检查 ELF 架构及静态属性，未在 arm64 主机执行。两份二进制均确认 `CGO_ENABLED=0`，无动态解释器/动态段。离线测试不包含真实平台、真实地区、双栈分别检测或旧系统兼容性，不报告真实解锁结果。发行下载与短链另行验收，不等同于真实平台解锁验证。

## 来源、许可与资产

源：<https://github.com/remnawave/geocheck> **v0.3.0**，commit `50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`（annotated tag 对象 `ff684cfb377c91fd845976d71d8036b2ee40d317`）。直接复用 `internal/access/access.go`、`checks.go` 及对应测试；必要 HTTP 实现依据 `internal/netx` 提取为纯标准库。终端直接复用同一 commit 的 `internal/render/report.go` 中 access 表格、状态/耗时/页脚等必要函数及完整 `theme.go`，仅省去无关的 banner、IP 身份、GeoIP、路由等区块。上游注释提及 Stash collapsed-tile panel scripts；未引入其他来源的检测算法；渲染依赖及其许可见下节。

**MIT License — Copyright (c) 2026 Remnawave**。原文保留在 `LICENSE`，也嵌入二进制，可用 `--license` 查看。本改进版与随附代码按同一 MIT 许可提供。新增链接依赖的完整版权与许可原文保留在 `internal/render/THIRD_PARTY_LICENSES.txt`，并随 `--license` 一起输出；原 MIT 原文不变。

v0.3.1 发行资产共五件（旧版资产保留）：`check.sh`、`service-unlock_linux_amd64`、`service-unlock_linux_arm64`、`service-unlock-v0.3.1-source.zip`、`SHA256SUMS`。source zip 含完整独立构建源码、测试、启动器、README 和 LICENSE，不含 Go 工具链或预编译二进制。SHA256SUMS 校验前四件资产。
