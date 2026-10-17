# Service Unlock v0.4.3 · 地区短码与详细统计

Linux amd64 / arm64 独立程序，**运行端不需要 Go、Node 或 Python**。v0.4.3 仅将终端 Region 改为大写两位地区码，并细化每组底部分类统计。英文 **Service｜Result｜Region** 三列、Yes / No / Limited / Unknown 及其配色保留，无每行说明/耗时或总耗时页脚；37 项探测和全部 JSON 字段/值不变。

> 下载短链 `https://check.siaoz.si` 提供固定 **v0.4.3（地区短码/详细统计）** 的启动脚本。旧版本保留；请重新下载 `check.sh`，旧脚本不会自动跟随新版。

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

启动器机制不变：现有 curl/GNU wget + sha256sum/shasum/openssl 任一，校验成功才运行、临时二进制结束即清理；不 sudo、不安装运行时、无常驻服务。它固定指向 `https://github.com/shui1iao/service-unlock/releases/download/v0.4.3/`。清单和程序来自同一 HTTPS 发布源，哈希校验不等于独立签名。

## 三组范围与含义

这是 **37 项检查，不是 37 个互不重叠的平台**，按原目录顺序分组展示。

### Service regions & access：21 项地区/服务观察

Google、Google Search captcha、YouTube、YouTube Premium、YouTube Music、Twitch、ChatGPT、Netflix、Spotify、Spotify signup、Deezer、Reddit、Reddit guest access、Amazon Prime Video、Apple、Steam、PlayStation、TikTok、Ookla Speedtest、JetBrains、Microsoft (Bing)。

- `country`：服务报告的两字母地区，**不是解锁结果或物理定位**。
- `availability`：YouTube Premium / Music、Spotify signup、Reddit guest 的 `yes/no` 观察，不等于登录、付费或播放测试。
- `captcha/block`（JSON 类型 `blocked`）：Google Search 的拒绝/验证码标记观察；`yes` 是发现标记，`no` 是未发现标记，不是“已解锁”。
- 原版 Google→Play、YouTube→Google、Bing→Microsoft account 回退保留，并在 JSON 的 detail 标明来源；回退值不冒充该站点直接提供的地区。请求失败、429/5xx 不被回退结果掩盖。
- Ookla 仅请求配置里的地区，Netflix 仅请求 Fast 的地区元数据；**不会开始测速，也不会下载它们返回的测速目标**。

### Service availability：原有 8 项不变

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
- 保留圆角框、三组分区及英文 Yes/No/Limited/Unknown 结果，三列固定 **Service｜Result｜Region**；底部按各组观察类别给出详细非零统计（见下节）。全体结果共享列宽、外框和表头对齐；按终端显示单元宽度处理长英文名及 CJK/组合字符。窄屏表头和单元格完整换行，不截断名称；最低14列，建议至少40列。**不显示说明、每行耗时、HTTP/Type 列或总耗时页脚**，完整数据仍在 JSON。
- 品牌名称保留；正文、地区、表头和边框使用终端默认前景。Result **整个单词**着色：`● Yes` 绿、`● No` 红、`● Limited` 黄，`○ Unknown` 默认前景加粗；不猜深浅背景，不强制白/黑字，不发 OSC 查询。帮助、参数错误、中断和启动器自有提示也改为英文。
- 未设置 `COLORFGBG`、浅色或深色都使用同一自适应方案。NO_COLOR 优先于强制彩色；普通非 TTY 输出去除 ANSI。NO_COLOR 按原行为可能保留加粗/复位；JSON 完全绕过渲染。
- `0`：所有检查得到观察/判据结论（包括 no/blocked/restricted），**不是全部解锁**；`1`：任一组至少一项未知或执行失败；`2`：参数错误；`130`：Ctrl-C；`143`：TERM。表后数量统计不是新的总体解锁状态。

## 英文结果与地区

| 原观察/状态 | Result | Region |
|---|---|---|
| country 有效识别且无错误 | Yes | 真实结果的大写两位地区码 |
| availability yes / no | Yes / No | 留空 |
| captcha/block no / yes | Yes / No（反向映射） | 留空 |
| access available / blocked | Yes / No | 只显示本项实际提供的地区 |
| access restricted（包括 Netflix 仅自制） | Limited | 同上 |
| AI reachable / blocked | Yes / No | 留空，不根据节点名称猜测 |
| error / 超时 / 无法判定 | Unknown | 有已有真实地区则保留，否则留空 |

Yes 是用户定义的**终端展示归一标签**，不是新增的登录/播放保证：地区识别不等于实际解锁，API 可达也不等于可调用。读取到地区不会覆盖原有 No / Limited / Unknown。

地区沿用固定 `golang.org/x/text v0.28.0` 的 `language.ParseRegion` 正规 ISO2/ISO3 映射与合法性检查，**只输出大写 alpha-2 短码**：ESP→ES、HKG→HK、JPN→JP、USA→US、FRA→FR，已有两位码保持。缺失、非法、私用、WW 或库无法可靠识别的地区仍留空，不显示全名、— / N/A / Unknown，不从其他服务、品牌或 API 节点推断地区。No / Limited / Unknown 即使有真实地区也不改成 Yes。

## 每组详细统计

仅在每组表格下解释非零结果计数，不增加单行说明列，不生成总体解锁结论。类别按本次实际行统计，主表与摘要总数一致；窄屏按词换行，不撑破表宽。例如：

- **Service regions & access**：`Yes 18: 15 regions identified, 3 accessible`；`No 2: 1 access denied, 1 with restriction markers`；`Unknown 1: inconclusive`。若 captcha 原始 no，则计入 Yes 的 `without restriction markers`；原始 yes 才计入 No 的 `with restriction markers`。地区识别不会被虚称为可登录。
- **Service availability**：`Yes 4: available`；`No 2: blocked`；`Limited 1: originals-only`；`Unknown 1: inconclusive`。Netflix 仅自制与其他 restricted 可精确区分；混合时分别计数。
- **AI endpoints**：`Yes 6: API reachable`；`No 1: region/request refused`；`Unknown 1: inconclusive`。不声称模型可调用。

这些数字仅是示例，不是真实检测。零类别省略；摘要仅解释已经选定的显示状态，不重新判断网络响应。

## JSON 兼容性

**本轮不修改 JSON，包括 source.port。** `schema_version: 1`、`source/generated_at/network` 及以下三个数组的所有既有字段与值均保留；终端删列不删除 JSON 的说明、耗时、HTTP 状态或地区代码，也不将 JSON 的原枚举归一为终端 Yes/No：

| 字段 | 数量 | 每项字段与状态 |
|---|---:|---|
| `results` | 8 | 原 `id/platform/name/status/detail/region/error/checked_at/elapsed_ms`，完全保留 access 语义 |
| `services` | 21 | `id/name/type/status/value/detail/error/checked_at/elapsed_ms`；type 为 country/availability/blocked；status 为 observed/error；value 是地区码或 yes/no，未知为 null |
| `ai_endpoints` | 8 | `id/name/vendor/status/http_status/detail/error/checked_at/elapsed_ms`；status 为 reachable/blocked/error；没收到 HTTP 响应时 http_status 为 0 |

所有时间为 UTC 开始时间和整项毫秒耗时（含子请求，不含排队）；非错误的 error 为 null。地区缺失不推断。ID 按所属数组解释；不要解析彩色终端文本或把 error 当 blocked。`source.port` 保留 v0.4.0 检测实现标记以维持 JSON 不变；发行版本以 `--help` 的 v0.4.3 为准。

## 来源、保守处理与限制

源为 remnawave/geocheck **v0.3.0**，commit `50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`：直接复用 `internal/access`、`internal/geo/services.go`、`internal/ai` 及最小 `countries/jsonx`；没有拷入 GeoIP/CDN/IP 发现执行代码。服务组运行器只保留依赖顺序和单一系统出口结果，不伪造公共 IP。

在原算法上保留/增加明确的未知保护：网络/超时、429/5xx、挑战页、解析缺失不作成功；Google captcha 的 429 不再直接当发现封锁，Reddit guest 网络失败不再当 no，Spotify signup 异常字段为未知，非法/无法识别地区或 WW 不冒充国家。服务特定 403 拒绝词仍有效。AI 在原认证码规则下要求有效 JSON 才给正向结论，已明确的地区拒绝词仍优先；服务器错误不得回显拒绝词就变 blocked。

**仍有上游弱判据**：部分检查依赖“没有拒绝词”或正常 HTTP，地区回退并非站点直接证据；页面、文案、片名版权、国家列表、反机器人策略可能变化。探针不是浏览器、播放或账户验证，不能承诺登录/播放/模型可调用。本轮仅离线验证，没有真实平台测量。

## 开发与验证

运行程序不需要 Go。开发使用 Go 1.26+（实际 Go 1.27.2），本轮沿用既有 x/text v0.28.0 和全部固定依赖，未新增或升级模块。保留 `godebug default=go1.22`，不改变探测行为或运行时网络默认值。

```sh
GOTOOLCHAIN=local GOPROXY=https://proxy.golang.org go mod download
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOARM64=v8.0 go build -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_arm64 .
```

原37项探测/HTTP/TLS/并发/取消/启动器离线回归继续保留。英文 golden 与独立语义断言覆盖地区ISO2/ISO3到正规两位码、非法/空地区、验证码反向分类、按组详细计数与37行总数核对、Unknown带地区不被覆盖、完整单词颜色、共享列宽及14–160列无损换行、深/浅/未指定背景与NO_COLOR。CLI英语帮助/错误保持原退出码；同37项固定mock输入的v0.4.2基线与本版JSON逐字节对照（只固定动态时间）。

amd64 实跑帮助及离线测试/演示；arm64 仅交叉构建与静态检查，未实跑。演示标注 OFFLINE MOCK - NOT REAL RESULTS，未访问真实平台/用户服务器。发行下载与短链另行验收。本版未新增图片导出功能或 GUKO 接入；现有 GUKO ANSI→PNG 渲染路径已另用离线示例验证可行，后续接入需单独实施。

**MIT License — Copyright (c) 2026 Remnawave**。原 LICENSE 保留并内嵌；渲染依赖许可见 `internal/render/THIRD_PARTY_LICENSES.txt`，随 `--license` 输出。上游来源：<https://github.com/remnawave/geocheck>。
