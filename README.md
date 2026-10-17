# Service Unlock v0.2.0 · 独立 Node.js 脚本

**7 个平台、8 项检查**：Netflix、YouTube Premium、TikTok、ChatGPT Web、ChatGPT App、Gemini、NotebookLM、Claude。ChatGPT 的两项结论分列，既不合并，也不生成全局“解锁”结论。

从 remnawave/geocheck **v0.3.0** 提取，沿用此前 v0.1.0 Python 移植版的判定和保守错误处理。现在直接运行 **`service_unlock.js`**，不需要 Python、npm、Docker、Go 或 Remnawave。仅使用 Node 内置模块，不登录、不调用收费模型、不测速度、GeoIP、路由或信誉，不接入 GUKO。

## 使用

要求 **Node.js 22 LTS 或更新版本**；本轮实际验证环境是 **Node.js v24.19.0 / Linux**。建议使用仍受支持的 LTS 版本。单独下载 `service_unlock.js` 即可执行，完整 MIT 许可已嵌入单文件。

```sh
# 不联网
node service_unlock.js --help

# 以下命令会访问真实平台，请在要检测的出口机器上自行决定执行
node service_unlock.js
node service_unlock.js --timeout 10 --workers 4
node service_unlock.js --json > result.json
```

- 默认跑全部 8 项，顺序与上游一致；`--workers` 默认 4，范围 1–8。
- `--timeout` 默认 6 秒：**每次 GET 的总预算，含 DNS 等待、连接、TLS、所有重定向、正文接收及解压**。Netflix、Claude 一项可能包含两次 GET，因此不是整项或整轮的总预算。支持正数秒，按毫秒向上取整，最大 `2147483.647` 秒（Node 定时器上限）。计时依赖事件循环调度，不是实时系统硬保证。
- 超时会中止请求，关闭活动 socket/正文/解压流并清理计时器；操作系统 DNS 查询可能无法在底层立即取消，不会继续发起目标 HTTP 请求。**Ctrl-C** 中止活动请求、停止新检查、退出 130，不输出半份 JSON；CLI 在清理后显式退出，避免不可取消的底层句柄拖住进程。
- 直连，**不使用** `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` 等代理环境变量；无 Cookie、无账号和 API Key。系统 DNS/系统选择地址族，没有分别检测 IPv4/IPv6，不保证所有平台使用同一地址族或出口。
- HTTP/1.1；TLS **明确开启证书验证**，信任库依 Node 配置，不用 `NODE_TLS_REJECT_UNAUTHORIZED=0` 绕过验证。本脚本也没有关闭验证的选项。
- 退出码：`0`＝每项均得出结论（含 restricted / blocked）；`1`＝至少一项 error 或执行失败；`2`＝参数错误；`130`＝用户中断。**退出码 0 不等于全部解锁。**

## 结果与 JSON

| `status` | 含义 |
|---|---|
| `available` | 符合该项可用性判据，不保证播放、付费或登录后功能 |
| `restricted` | Netflix 仅自制剧判据成立 |
| `blocked` | 明确服务拒绝标记，或 Netflix 两个片名均返回 404 |
| `error` | 未能测定；请求/HTTP/解析/反机器人挑战等故障，**不等于未解锁** |

`--json` 的 stdout 仅有一个 JSON 对象，无进度日志。保持 v0.1.0 的 `schema_version: 1` 与字段结构：

- `source`：原始仓库、tag、精确 commit，以及当前 Node 移植说明；`generated_at`：UTC 生成时间。
- `network`：`mode`、`address_family`、`timeout_seconds`、`timeout_scope`、`workers`。本版 `timeout_scope` 为 `request including redirects and body`，如需比较旧结果，应读取此值，不假定超时语义相同。
- `results`：固定 8 项；每项为 `id`、`platform`、`name`、`status`、`detail`、`region`、`error`、`checked_at`、`elapsed_ms`。
- `checked_at` 是该检查开始时的 UTC 时间；`elapsed_ms` 是整项耗时（含子请求，不含排队时间），不是纯 RTT。
- `region` 仅填上游真实提取到的区域：Netflix 二字母片库区域、Claude trace 的二字母地区、Gemini 三字母地区；其他或缺失为 `null`，不做 GeoIP 推断。
- 非 error 的 `error` 为 `null`；error 给出未能判定的原因。`detail` 可以为空字符串。错误消息可随运行时改变，不应当作枚举；程序按 `id` / `status` 读取。

## 端点及保留的判定

| id | 端点 / 主要判据 |
|---|---|
| `chatgpt_web` | `https://api.openai.com/compliance/cookie_requirements`；`unsupported_country` 表示拒绝 |
| `chatgpt_app` | `https://ios.chat.openai.com`；`disallowed isp` / `been blocked` 表示拒绝，挑战页优先排除 |
| `youtube_premium_access` | `https://www.youtube.com/premium`；地区拒绝句优先，否则需 `ad-free` 正向词 |
| `netflix_access` | `https://www.netflix.com/title/70143836`，必要时查 `80197526`；授权片可用＝完整片库，授权片 404 而自制片可用＝仅自制，两个 404＝blocked；地区取最终片名路径，裸 `/title/…` 为 US |
| `claude_access` | 先从 `https://claude.ai/cdn-cgi/trace` 尽力读 `loc`，再查 `https://claude.ai/`；保留全部区域拒绝文案及 HTML 转义形式，挑战页优先 |
| `tiktok_access` | `https://www.tiktok.com/`；保留原始 4 条地区拒绝标记，否则按正常 HTTP 范围判断 |
| `gemini_access` | `https://gemini.google.com/app`；匹配 `,数字,数字,200,"三字母地区"`，保留不支持列表 RUS、BLR、CHN、PRK、IRN、CUB、SYR；缺失地区＝error |
| `notebooklm_access` | `https://notebooklm.google.com/`；最终 URL 的 `location=unsupported` 优先表示拒绝；Google 账号页或 `.google.com` 的 `/login` 路径表示可用但需登录；其他目的地未知 |

端点、UA、请求头、标题 ID、关键标记和国家列表均沿用现有移植，未新增平台或自行更新探针算法。

## 与上游及 Python 版的差异

**沿用 v0.1.0 的保守修正，不冒充与 Go 上游逐字等价：**

1. 429、5xx、异常 <200、网络失败、超时及解析失败为 error；即使错误响应回显成功/拒绝词，也不视为解锁或封锁。所有检查先排除上游的 Cloudflare 挑战标记（仍限 403/429/503）。
2. ChatGPT 的无拒绝词响应、YouTube 正向词、NotebookLM 登录目的地必须满足正常 HTTP 范围；正文型检查空白正文为 error。服务特定 403 拒绝词仍有效：例如 403 + `unsupported_country` 是 blocked，裸 403 是 error。
3. Netflix 仅把 404 当作片名未提供；其他非 2xx 为 error。成功目的地必须是 Netflix 域名下正确片名的 `/title/<id>` 或 `/<两字母地区[-语言]>/title/<id>`，不能把登录页或任意两字母路径误认作片库。
4. NotebookLM 要有可解析的绝对 HTTP(S) 目的地；Claude trace 仅从 2xx 取地区，失败不决定主检查状态。原始和解压后正文各限 4 MiB，超限、截断、损坏 gzip 或不支持的压缩格式均为 error，而不是截取后继续判定。

**v0.2.0 的必要运行时差异：**

- 改用 Node `http` / `https` / `zlib`，无需 npm 依赖；仍为 HTTP/1.1、直连、不保留 Cookie、不复用连接。TLS/HTTP 指纹、CA 信任来源、DNS/地址选择可能与 Python 或 Go 不同。
- 超时从 Python 的“每次 socket 操作”改为包含重定向和正文解压的**每次 GET 总预算**，慢速滴流不能无限刷新超时。中断使用 AbortController，显式销毁流并清理计时器。
- 跟随 301/302/303/307/308，最多跟随 9 次，第 10 次为 error；循环也受此限制，Python urllib 对重复地址可能更早报错。无 Location、非 HTTP(S) 或带凭据的目的地拒绝。中间重定向响应也执行大小、编码及完整性校验。
- URL 连接与重定向使用 Node WHATWG URL 规范化，拒绝空白、反斜杠、无效端口/主机等畸形地址；与 Python URL 容错和规范化行为在非常规 URL 上可能不同。分类器保留原始 authority/path 来避免把显式端口或异常路径当作新的成功证据。
- JSON 结构、检查 ID、平台判据和状态语义保持不变；仅 `source.port`、`network.timeout_scope` 反映新实现。运行时异常的名称/描述不承诺与 Python 文本一致。

**原判据的限制仍然存在：** ChatGPT 正常非空响应主要凭“无拒绝词”，TikTok/Claude 部分分支依赖成功 HTTP，不等于实际登录/播放/模型调用验证。Netflix 片名版权、Gemini 地区列表、网页结构、文案、重定向及反机器人策略可能变化。本脚本不是浏览器模拟器，也不保证付费、账户或实际内容访问。

## 离线测试与已验证范围

```sh
node --check service_unlock.js
node --check test_service_unlock.js
node service_unlock.js --help
node --test test_service_unlock.js
```

**Node.js v24.19.0 / Linux：40 项测试全部通过**；语法和帮助验证通过。覆盖：

- 四状态、7 平台/8 项、全部关键标记、403/404 服务语义、429/5xx、挑战页、解析失败、可选地区；JSON、终端输出、真实子进程退出码和并发上限。
- 回环 HTTP 重定向/循环、gzip、损坏编码、4 MiB 原始/解压限额、截断、慢响应/滴流/重定向总预算、取消、代理环境变量忽略。
- 回环自签名 HTTPS 拒绝（即使环境设为 `NODE_TLS_REJECT_UNAUTHORIZED=0`，代码仍强制验证）；该用例会产生 Node 的预期环境变量安全警告，测试后恢复环境。测试内的私钥/证书是公开、一次性 localhost 夹具，**不是部署凭据，不加入信任库**。
- 超时子进程自然退出、Ctrl-C 关闭活动连接并以 130 退出，无半份 JSON；真实 CLI 入口还验证了底层不可取消句柄模拟。

另外开发阶段对现有 Python v0.1.0 做了 **5005 组纯内存分类差分**，`status/detail/region/error` 全部一致。差分不代表任意 URL 或网络行为完全等价；运行本包及其随附测试均不需要 Python。

所有测试只用内存 mock 和回环 HTTP/HTTPS，不安装依赖、不连接真实平台或管理服务器。**未实测线上解锁、真实地区、Node 22、其他操作系统、真实代理出口或分别双栈行为。** 无真实解锁结果可报告。

## 文件与许可

`service-unlock-v0.2.0.zip` 内含同名目录，只有：

- `service_unlock.js`：单文件入口（含完整 MIT 许可）
- `test_service_unlock.js`：Node 内置测试器离线用例
- `README.md`
- `LICENSE`：上游 MIT 原文

本包不覆盖已有 Python v0.1.0 文件，也不包含安装器、自动更新、发布或短链修改逻辑。

来源：<https://github.com/remnawave/geocheck>，tag **v0.3.0**；annotated tag 对象 `ff684cfb377c91fd845976d71d8036b2ee40d317`，解引用 commit **`50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`**。

判定来自该 commit 的 [internal/access/checks.go](https://github.com/remnawave/geocheck/blob/50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc/internal/access/checks.go)、`access.go`；网络参考 `internal/netx/request.go`、`netx.go`。测试沿用该版本与 Python v0.1.0 的离线用例。上游 access 注释说明其探针来自 Stash collapsed-tile panel scripts；本包直接根据 geocheck 与既有移植编写，未引入其他第三方代码。

**MIT License — Copyright (c) 2026 Remnawave**。JS 移植与随附测试按同一 MIT 许可提供；保留原始署名和许可，修改说明见上节。
