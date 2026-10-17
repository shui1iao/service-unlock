# Service Unlock v0.4.1 · 三列中文展示

Linux amd64 / arm64 独立程序，**运行端不需要 Go、Node 或 Python**。v0.4.1 只修改终端展示：三组统一为 **服务｜结果｜地区**，删除每行耗时、说明及总耗时页脚。37 项检测、网络、并发、退出码和全部 JSON 字段/值保持 v0.4.0 不变。

> 下载短链 `https://check.siaoz.si` 提供固定 **v0.4.1（三列中文展示）** 的启动脚本。旧版本保留；请重新下载 `check.sh`，旧脚本不会自动跟随新版。

## 运行

直接运行发行程序（执行检测会访问真实平台，由使用者自行决定）：

```sh
chmod +x service-unlock_linux_amd64
./service-unlock_linux_amd64 --help   # 不联网
./service-unlock_linux_amd64
./service-unlock_linux_amd64 --json > result.json
./service-unlock_linux_amd64 --timeout 10 --workers 4
```

arm64 使用对应的 `service-unlock_linux_arm64`。`--license` 显示内嵌许可，不检测。

**一键下载运行新版：**

```sh
curl -fLsS https://check.siaoz.si -o check.sh && sh check.sh
```

启动器机制不变：现有 curl/GNU wget + sha256sum/shasum/openssl 任一，校验成功才运行、临时二进制结束即清理；不 sudo、不安装运行时、无常驻服务。它固定指向 `https://github.com/shui1iao/service-unlock/releases/download/v0.4.1/`。清单和程序来自同一 HTTPS 发布源，哈希校验不等于独立签名。

## 三组范围与含义

这是 **37 项检查，不是 37 个互不重叠的平台**，按原目录顺序分组展示。

### Popular services：21 项地区/服务观察

Google、Google Search captcha、YouTube、YouTube Premium、YouTube Music、Twitch、ChatGPT、Netflix、Spotify、Spotify signup、Deezer、Reddit、Reddit guest access、Amazon Prime Video、Apple、Steam、PlayStation、TikTok、Ookla Speedtest、JetBrains、Microsoft (Bing)。

- `country`：服务报告的两字母地区，**不是解锁结果或物理定位**。
- `availability`：YouTube Premium / Music、Spotify signup、Reddit guest 的 `yes/no` 观察，不等于登录、付费或播放测试。
- `captcha/block`（JSON 类型 `blocked`）：Google Search 的拒绝/验证码标记观察；`yes` 是发现标记，`no` 是未发现标记，不是“已解锁”。
- 原版 Google→Play、YouTube→Google、Bing→Microsoft account 回退保留，并在 JSON 的 detail 标明来源；回退值不冒充该站点直接提供的地区。请求失败、429/5xx 不被回退结果掩盖。
- Ookla 仅请求配置里的地区，Netflix 仅请求 Fast 的地区元数据；**不会开始测速，也不会下载它们返回的测速目标**。

### Stash checks：原有 8 项不变

ChatGPT Web、ChatGPT App、YouTube Premium、Netflix、Claude、TikTok、Gemini、NotebookLM。

`available` / `restricted` / `blocked` / `error` 语义和原端点保留；ChatGPT Web/App 独立。Netflix `restricted` 是仅自制片判据；`error` 是未知，不等于未解锁。原有 JSON `results` 始终只放这 8 项。

### AI endpoints：8 项无凭据 API 可达性

OpenAI、Anthropic、Google Gemini、DeepSeek、Qwen (Singapore)、Qwen (Beijing)、Kimi / Moonshot、Zhipu GLM。

使用原版 `/models` 等只读端点，不带用户 API Key，不发模型生成请求。`reachable` 只证明 API 回应，不等于可调用、账户可用或解锁。按各端点原规则，401 可以是 reachable；Google Gemini 的特定规则也接受 403，因此该端点**不能据此判断地区被封锁**。明确地区拒绝标记为 blocked；异常 HTTP、挑战页、空白/畸形非 JSON 正向响应为 error。

不包含 GeoIP 数据库、CDN 边缘、IP 身份发现、信誉/评分、路由、吞吐测速、其他平台或 GUKO 接入。

## 网络、配色与退出码

- 继续 **direct / system-selected**；不分别遍历双栈、不绑定源 IP、不添加代理或 DoH。不同请求可能由系统选择不同地址族，所以表格不冒标 IPv4/IPv6。
- TLS 验证开启，使用系统 CA。忽略检测进程的代理环境变量；下载器自身的代理行为不代表探针出口。
- 不读取用户凭据。只使用上游公开客户端常量，以及原版 Google 同意 Cookie；无持久 Cookie jar、不登录、不创建账户（Spotify 仅 signup validate 请求）。
- `--timeout` 默认 6 秒，是每次 HTTP 请求总预算，含连接/TLS/重定向/正文；多请求回退可能使整项更长。`--workers` 默认 4、范围 1–8，是**整个运行的并发上限**：三组顺序执行，组内有界并发。支持 Ctrl-C/TERM 取消，不输出半份 JSON。
- 原始及 gzip 解压后正文各最多 4 MiB；损坏、截断、超限为未知。保持最多跟随 9 次重定向。
- 保留圆角框、三组分区及中文分类数量统计，统一三列 **服务｜结果｜地区**。三张表按全部结果统一计算列宽，外框及表头对齐；按终端显示单元宽度处理中文，窄屏完整换行、不用省略号截断国家名。正常布局最低需要 14 个终端列（3 个完整中文表头加外框），建议至少 40 列；14–19 列会移除内边距。**不显示说明、每行耗时、HTTP/Type 列或总耗时页脚**，需要完整数据请用 JSON。
- 服务名保留品牌名称；正文、地区、表头和边框使用终端默认前景。状态保持 ANSI 语义色与中文文字/符号，`○ 未知` 默认前景加粗；不猜背景、不强制白/黑字、不发送 OSC 查询。
- 未设置 `COLORFGBG`、浅色或深色都使用同一自适应方案。NO_COLOR 优先于强制彩色；普通非 TTY 输出去除 ANSI。NO_COLOR 按原行为可能保留加粗/复位；JSON 完全绕过渲染。
- `0`：所有检查得到观察/判据结论（包括 no/blocked/restricted），**不是全部解锁**；`1`：任一组至少一项未知或执行失败；`2`：参数错误；`130`：Ctrl-C；`143`：TERM。表后数量统计不是新的总体解锁状态。

## 中文结果与地区

| 原观察/状态 | 终端结果 | 地区 |
|---|---|---|
| country 有可靠名称 | 已识别地区（不是解锁） | 中文国家/地区名 |
| availability yes / no | 可访问 / 不可访问 | — |
| captcha/block yes / no | 发现限制 / 未发现限制 | — |
| access available / blocked | 可用 / 不可用 | 仅显示检测实际提供的地区 |
| Netflix restricted | 仅自制（其他受限结果为“受限”） | 同上 |
| AI reachable / blocked | API可达 / 被拒绝（不等于可调用） | — |
| error / 无法判定 | 未知（不改写为封锁） | 无可靠地区则 — |

地区名使用固定 `golang.org/x/text v0.28.0` 的 `language.ParseRegion` 正规 ISO2/ISO3 映射及 `display.Regions(language.SimplifiedChinese)` 简体中文数据（该版本标注 CLDR 32）。例如 US/USA 均为美国、FR/FRA 均为法国；遵循库提供的名称，不用零散手写表或在线翻译。缺失、非法、私用、WW/未知及无可用中文名称的代码显示 `—`，不推断物理位置。表后按本组中文结果计数，不合并为总体解锁结论。

## JSON 兼容性

**本轮不修改 JSON，包括 source.port。** `schema_version: 1`、`source/generated_at/network` 及以下三个数组的所有既有字段与值均保留；终端删列不删除 JSON 的说明、耗时、HTTP 状态或地区代码，也不把 JSON 枚举翻译成中文：

| 字段 | 数量 | 每项字段与状态 |
|---|---:|---|
| `results` | 8 | 原 `id/platform/name/status/detail/region/error/checked_at/elapsed_ms`，完全保留 access 语义 |
| `services` | 21 | `id/name/type/status/value/detail/error/checked_at/elapsed_ms`；type 为 country/availability/blocked；status 为 observed/error；value 是地区码或 yes/no，未知为 null |
| `ai_endpoints` | 8 | `id/name/vendor/status/http_status/detail/error/checked_at/elapsed_ms`；status 为 reachable/blocked/error；没收到 HTTP 响应时 http_status 为 0 |

所有时间为 UTC 开始时间和整项毫秒耗时（含子请求，不含排队）；非错误的 error 为 null。地区缺失不推断。ID 按所属数组解释；不要解析彩色终端文本或把 error 当 blocked。`source.port` 保留 v0.4.0 检测实现标记以维持 JSON 不变；发行版本以 `--help` 的 v0.4.1 为准。

## 来源、保守处理与限制

源为 remnawave/geocheck **v0.3.0**，commit `50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`：直接复用 `internal/access`、`internal/geo/services.go`、`internal/ai` 及最小 `countries/jsonx`；没有拷入 GeoIP/CDN/IP 发现执行代码。服务组运行器只保留依赖顺序和单一系统出口结果，不伪造公共 IP。

在原算法上保留/增加明确的未知保护：网络/超时、429/5xx、挑战页、解析缺失不作成功；Google captcha 的 429 不再直接当发现封锁，Reddit guest 网络失败不再当 no，Spotify signup 异常字段为未知，非法/无法识别地区或 WW 不冒充国家。服务特定 403 拒绝词仍有效。AI 在原认证码规则下要求有效 JSON 才给正向结论，已明确的地区拒绝词仍优先；服务器错误不得回显拒绝词就变 blocked。

**仍有上游弱判据**：部分检查依赖“没有拒绝词”或正常 HTTP，地区回退并非站点直接证据；页面、文案、片名版权、国家列表、反机器人策略可能变化。探针不是浏览器、播放或账户验证，不能承诺登录/播放/模型可调用。本轮仅离线验证，没有真实平台测量。

## 开发与验证

运行程序不需要 Go。开发使用 Go 1.26+（实际 Go 1.27.2），仅新增固定 x/text v0.28.0 供终端地区显示，未升级现有依赖。保留 `godebug default=go1.22`，不改变探测行为或运行时网络默认值。

```sh
GOTOOLCHAIN=local GOPROXY=https://proxy.golang.org go mod download
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOARM64=v8.0 go build -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_arm64 .
```

原 37 项调用路径、端点/算法、错误与回退、并发/取消、HTTP/TLS 和启动器离线回归继续保留。本轮新增：三组中文结果映射、CLDR 完整地区目录及 ISO2/ISO3 一致性、相同三列宽/CJK 对齐、14–160 列的无损换行、删除全部终端说明/耗时但 JSON 字段和值保持、深/浅/无背景、NO_COLOR、真实本地 PTY 宽度检测。旧五列表格 golden 只作历史参考，不作为本轮展示预期。

amd64 实跑帮助及离线测试/演示；arm64 仅交叉构建与静态检查，未实跑。所有演示为 mock 非实测，未访问真实平台；发行下载和短链另行验收，不等于真实解锁验证。

**MIT License — Copyright (c) 2026 Remnawave**。原 LICENSE 保留并内嵌；渲染依赖许可见 `internal/render/THIRD_PARTY_LICENSES.txt`，随 `--license` 输出。上游来源：<https://github.com/remnawave/geocheck>。
