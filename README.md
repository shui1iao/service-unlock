# 服务可用性检测（独立 Python 脚本）

从 **remnawave/geocheck v0.3.0** 移植的 access 检测改进版：**7 个平台、8 项检查**。ChatGPT Web / App 检测不同限制，分列保留，不合并结论。

仅需 **Python 3.8+ 标准库**，无需 pip、Go、Docker 或 Remnawave。单独复制 `service_unlock.py` 即可运行（脚本内已含完整 MIT 许可）；完整下载包同时提供说明、LICENSE 和离线测试。不接入 GUKO，不测速度、GeoIP、路由或信誉，不登录账号。

## 使用

```sh
# 只看帮助，不联网
python3 service_unlock.py --help

# 以下命令将真实访问上述平台；请在你希望检测的出口机器上自行执行
python3 service_unlock.py
python3 service_unlock.py --timeout 10 --workers 4
python3 service_unlock.py --json > result.json
```

- 默认全部 8 项，顺序与上游一致；并发默认 4，`--workers` 接受 1–8。
- `--timeout` 默认 6 秒，是**每次 socket 阻塞操作**超时，不是整轮或单请求的严格总时限。DNS、重定向、持续缓慢传输及 Netflix/Claude 的多次请求可使实际耗时更长。
- 直接连接目标，忽略 `HTTP_PROXY` / `HTTPS_PROXY` 等代理环境变量；系统 DNS、系统自动选择 IPv4/IPv6。**没有分别验证双栈，也不保证各平台走同一地址族。** 如需特定出口，应由使用者在运行环境中明确安排，不把本结果解释成未经选择的代理或其他出口的结果。
- TLS 使用系统默认可信 CA 并验证证书；不禁用验证，不携带账号、Cookie 或 API Key。
- 退出码：`0` = 所有检查得出结论（包括 restricted / blocked），`1` = 至少一项 error，`2` = 参数错误，`130` = 中断。因此退出码 0 **不表示全部解锁**。

## 结果含义与 JSON

| status | 含义 |
|---|---|
| `available` | 符合该项上游可用性判据；不保证播放、付费或登录后功能 |
| `restricted` | Netflix 仅自制剧判据成立，不是完整片库 |
| `blocked` | 明确拒绝标记，或 Netflix 两个片名均返回 404 |
| `error` | 未知：请求、HTTP、解析或验证失败；**不等于未解锁** |

`--json` 的标准输出仅包含一个 JSON 对象，没有进度日志。字段：

- `schema_version: 1`、`source`（仓库、tag、精确 commit）、`generated_at`（UTC）。
- `network`：直连模式、系统选地址族、超时数值与语义、并发数。
- `results`：固定 8 项；每项含 `id`、`platform`、`name`、`status`、`detail`、`region`、`error`、`checked_at`、`elapsed_ms`。
- `region` 仅有真实上游提取依据才填：Netflix 的二字母片库区域、Claude trace 的二字母地区、Gemini 页面中的三字母地区；其余或缺失为 `null`，不做 GeoIP 推断。
- `error` 对非 error 状态为 `null`；对 error 状态给出异常或未能判定的原因。`detail` 可为空字符串。`checked_at` 为该检查开始时间，`elapsed_ms` 是整项耗时（含子请求，不是单纯 RTT，不含并发排队时间）。
- 没有 ChatGPT 合并状态，也没有全局“解锁/未解锁”状态。后续程序应按 `id` 和 `status` 读取，而非解析终端表格；本任务未实现任何系统接入。

## 检测与上游对应

| id / 平台 | 请求目标及保留的主要判据 |
|---|---|
| `chatgpt_web` / ChatGPT Web | `https://api.openai.com/compliance/cookie_requirements`；`unsupported_country` 表示拒绝 |
| `chatgpt_app` / ChatGPT App | `https://ios.chat.openai.com`；`disallowed isp` / `been blocked` 表示拒绝，先排除 Cloudflare 挑战 |
| `youtube_premium_access` / YouTube Premium | `https://www.youtube.com/premium`；国家拒绝语句优先，否则需 `ad-free` 正向标记 |
| `netflix_access` / Netflix | `https://www.netflix.com/title/70143836`（授权作品），必要时再查 `80197526`（自制作品）；根据可达片名区分完整片库/仅自制/不可用；区域取最终重定向路径，裸 `/title/…` 为 US |
| `claude_access` / Claude | `https://claude.ai/cdn-cgi/trace` 尽力提取 `loc`，然后 `https://claude.ai/`；挑战页优先，区域拒绝的全部原始文案（含 HTML 转义）均保留 |
| `tiktok_access` / TikTok | `https://www.tiktok.com/`；保留 4 条地区拒绝标记，否则按上游成功 HTTP 范围判定 |
| `gemini_access` / Gemini | `https://gemini.google.com/app`；匹配 `,数字,数字,200,"三字母区域"`；保留上游不支持列表 RUS、BLR、CHN、PRK、IRN、CUB、SYR；提取失败为未知 |
| `notebooklm_access` / NotebookLM | `https://notebooklm.google.com/`；最终 URL 的 `location=unsupported` 为拒绝，Google 账号页或 `.google.com` 的 `/login` 路径为可用（需要登录）；其他目的地未知 |

端点、标题 ID、UA、请求头、上述关键标记及主要判定分支来自上游，不自行更新区域列表、不添加额外 AI 接口。

## 与上游的明确差异

这是**移植改进版，不是逐字等价复刻**。为避免将故障误报为已解锁或未解锁，做了下列保守修正：

1. 所有检查先排除 HTTP **429、5xx**（及异常 <200），即使错误页包含成功/拒绝文字或最终 URL 是登录页，也返回 error。网络异常、超时、解析异常同样为 error。
2. 将上游已有的 Cloudflare 挑战识别规则应用于所有检查，仍仅在 403/429/503 且含原始挑战标记时成立，先于区域拒绝判断。不会把回显的拒绝路径当作实际地区封锁。
3. ChatGPT Web/App 不再仅因“未找到拒绝词”就接受异常 HTTP；YouTube 的 `ad-free` 也需成功 HTTP 范围。保留服务特定拒绝文字的意义，例如 403 + `unsupported_country` 仍是 blocked，而无拒绝证据的 403 是 error。所有正文型检查的空白正文改为 error；NotebookLM/Netflix 本就以 URL 为依据，不强求正文。
4. Netflix 只有 **404** 作为片名未提供的负向证据；其他非 2xx（包括无明确片库含义的 403）为 error，不再沿用“两次服务器错误也可能 blocked”的行为。成功 URL 必须是 Netflix 域名下当前片名的 `/title/<id>` 或 `/<两字母地区[-语言]>/title/<id>`，缺失、异常主机、登录页或无法解析的路径均为 error，不猜 US。
5. NotebookLM 在有效绝对 HTTP(S) URL 上保留拒绝子串优先；登录页需正常 HTTP 范围才算 available，403 登录地址不算成功；无法解析的最终 URL 为 error。
6. Claude trace 仅从 2xx 响应取地区；失败或无法提取仍不影响主检查结论，`region: null`，没有伪造地区或独立增加检测平台。
7. HTTP 使用 Python urllib 的 HTTP/1.1，而非 Go 的 HTTP/2 尝试；不移植 netx 的指定地址族、网卡绑定、SOCKS5、DoH、连接复用。默认仍为无代理、无 Cookie、验证 TLS；按最终 URL 判定，最多跟随 9 次重定向（第 10 次报错）。urllib 对重复地址循环也可能更早报错。超时改为上述 socket 操作语义，而非 Go 的请求总预算。
8. 保留 gzip 解压支持；原始及解压后正文最多各 4 MiB。超限、截断、损坏 gzip、非支持压缩格式返回 error，上游则会截取前 4 MiB 继续判定。新增固定 JSON 字段、UTC 时间和有界并发 CLI；不新增状态汇总。

### 判据自身的限制

仍保留上游的弱判据：ChatGPT 在正常非空响应中通过“没有拒绝词”判断，TikTok/Claude 部分分支按正常 HTTP 判断；这**不是实际登录、播放、支付或模型请求验证**。Netflix 两个指定片名的地区版权可能变化，Gemini 国家列表与页面结构可能过时，服务的重定向、文案及反机器人策略也可能变化。HTTP/1.1 / TLS 指纹与 Go 或浏览器不同，可能触发不同拦截。遇到 error 应视为未能测定，不凭本脚本承诺实际解锁。

## 离线验证

```sh
python3 -m py_compile service_unlock.py test_service_unlock.py
python3 service_unlock.py --help
python3 -m unittest -v test_service_unlock.py
```

测试只使用内存响应与 `127.0.0.1` 隔离 HTTP 服务；额外拦截非回环 DNS 请求，不安装任何包，不调用真实平台。已在 Python **3.13.5** 执行：**35 个测试全部通过**，并通过语法编译与 CLI 帮助检查。

覆盖四种状态、7 平台/8 项列表、上游 Claude/Gemini/NotebookLM/Netflix 的关键用例、403/404 的服务语义、429/5xx 覆盖拒绝词、空白/畸形内容、超时/网络异常、Netflix URL 解析、Claude 地区可选、有效 JSON、终端输出、退出码、参数校验、并发上限、本地 HTTP 重定向/循环、gzip/损坏/解压大小限制、截断正文、忽略代理环境变量。

**本轮没有向真实平台发起任何检测，没有验证线上可用性、实际地区、双栈或真实代理出口，也没有实测其他 Python 版本或操作系统。** 不附带或宣称真实解锁结果。

## 来源与许可

- 仓库：<https://github.com/remnawave/geocheck>
- 精确源版本：tag **v0.3.0**；annotated tag 对象 `ff684cfb377c91fd845976d71d8036b2ee40d317`。
- tag 解引用后的 **commit：`50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`**（归档目录前缀 `ff684cf` 是 tag 对象前缀，不是 commit）。
- 判定：[internal/access/checks.go](https://github.com/remnawave/geocheck/blob/50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc/internal/access/checks.go)、[access.go](https://github.com/remnawave/geocheck/blob/50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc/internal/access/access.go)。网络参考：`internal/netx/request.go`、`netx.go`。离线用例参考同版本 `access_test.go`、`claude_test.go`、`gemini_test.go`、`notebooklm_test.go`。
- 上游 access 注释说明其探针来自 Stash collapsed-tile panel scripts；本交付直接依据上述 geocheck 版本移植，未另行引入或执行其他来源代码。
- **MIT License，Copyright (c) 2026 Remnawave**。`LICENSE` 保留上游全文，单文件脚本也内嵌全文。此 Python 移植与随附测试按同一 MIT 许可提供，修改内容见上节。
