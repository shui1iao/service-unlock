# Service Unlock v0.5.8

本版真正精简重复检测：**22个原生主探针（6 access＋12 services＋4 AI）＋134个extra目录，完整raw156**。普通平台只一个主结果Yes/No/Limited/Unknown，不再把Web/API/Geo并列成复杂聚合。不同区域片库/版权样本仍独立，不按品牌合并。

## 辅助国家恢复（v0.5.8）

每平台仍一个项目、一个主结果；ChatGPT(ab initialize)、TikTok(privacy config)、YouTube(直接countryCode，无Google fallback)、Spotify(accounts status)、Reddit(匿名chat widget)、Google(首页/必要Play)恢复官方同平台国家元信息。先保留原主判定及其checked_at/elapsed_ms，再在原worker内顺序做辅助；辅助失败/WAF/坏数据/冲突/未知ISO2只留空，不改主status/value/error/detail/Unknown计数/退出码，不覆盖已有国家。YouTube主响应已有countryCode可复用。Spotify国家不能推出可注册Yes。

同一个direct/system-selected netx.Stack与原UA/CA/正文/重定向安全策略，不保证同一实际IP/固定地址族；每项目辅助总budget不超过--timeout（默认6秒），Google两步共享。实际请求/墙钟耗时会增加，workers默认4全局上限保持。不给其它平台强填出口国家或产品标签，无登录/播放保证。

services仅google_captcha/spotify_signup/reddit_guest可新增可选`region` ISO2，未知省略；access复用已有region字段，顶层四数组/22+134不变。严格JSON消费者需同步本版本；历史原报告不查新、不补贴国家。

## 上版已完成的破坏性JSON目录变化（0.5.7保留）

真正停止以下12项测量并删除注册/专用实现，不是只隐藏行，不提供旧ID别名或复制结果：
- services：`google`、`youtube`、`chatgpt`、`netflix`、`spotify`、`reddit`、`prime`、`tiktok`。
- ai_endpoints：`openai`、`anthropic`、`gemini`、`qwen_cn`。

ChatGPT/Claude/Gemini只保现有Web客户端，Netflix只保片库、PrimeVideo只保访问；Qwen选国际API。DeepSeek/Qwen Intl/Kimi/GLM仍明确(API)，不冒充客户端。其余保留ID/name/platform、raw字段/status/value/detail/time/顺序与原请求判据不变。schema1与历史`source.port`出处标记保留，**该旧标记不是当前检测数量或CLI发行版本**。自动国家识别独立，不用已删geo或其它服务投票。

## 自动选区和显示

默认global＋实际出口对应**一个**地区，不是all-region扫描。extra134原目录、global6与十地区成员/顺序完整保留；其中125项有原生请求实现，9项安全Unknown仍不发请求。原来并没有物理删掉所有地区；v0.5.6丢失的是独立区域栏目/部分原样本名称。

恢复`Regional media — <English region>`：将本次所选地区extra的Video/Music项目按原Selected顺序逐项列出，B-Global SEA/TH/ID/VN、Sky各市场、Hulu JP、Rakuten JP及Bilibili版权项目保原名和独立状态；global媒体仍在用途组。游戏、Meta AI、DMM金融站等不混入媒体区。六用途组加有内容的Regional组，空组省略。

全英文单圆角外框，`Service | Result | Region`三列；字体/颜色/宽度/无损换行不升级，不显示统计/耗时。Region只来自这一保留探针的真实已知国码，没有就空；仅下面六主项目增加官方本平台辅助国家查询。国家观察项目名明确(region)/(store region)；Google Search captcha按原blocked反转，不代表Google全产品解锁。

Spotify保严格`Legacy signup endpoint`：旧匿名不完整POST不是实际注册/登录，不存在匿名Yes路径；只有原精确typed status320代理拒绝为限定No，其余证据不足Unknown。删除独立spotify地区观察不改变请求或判据，不用country/metadata推断注册成功。

| 选择 | 当前测量/逻辑行总数 |
|---|---:|
| global | 28 |
| tw | 39 |
| hk | 36 |
| jp | 54 |
| na | 65 |
| sa | 31 |
| eu | 58 |
| oc | 47 |
| kr | 36 |
| sea | 39 |
| af | 31 |

## 使用

```sh
curl -fLsS https://github.com/shui1iao/service-unlock/releases/download/v0.5.8/check.sh -o check.sh && sh check.sh
./service-unlock_linux_amd64 --help
./service-unlock_linux_amd64 --machine > report-envelope.json
```

旧check.sh固定旧版本，需重新下载；短链切换由运营方另行完成，**本版发布不代表短链或SG GUKO已切换**。现有curl/GNU wget、SHA256与CA为前置，不自动安装依赖/常驻服务。

`--region auto|global|tw|hk|jp|na|sa|eu|oc|kr|sea|af`，默认auto；显式选区不定位。仍沿同direct/system-CA/实际extra family顺序读取Cloudflare trace、必要时ip.sb，仅保安全国家码及错误码；失败或未映射国家只global，不保存IP/ASN原文。`--extra-family auto|4|6`只影响extra，原生22仍system-selected；不支持该族的项Unknown，不把整报告冒标IPv4。`--timeout`/并发原语义不变。

`--json`为原schema1；`--machine`为同一次测量/selection的JSON＋ANSI `guko-unified-v1`，不重复探测；二者互斥。退出1可为完整Unknown报告，不是访问No或程序崩溃；取消不交部分JSON。GUKO v0.7.1严格同步此新契约，旧历史不重测不改标签。

## 准确源、离线构建与限制

本版Release附check.sh、Linux amd64v1/arm64v8.0程序、SHA256SUMS和完整`service-unlock-v0.5.8-source.zip`（固定vendor、构建/测试/数据及全许可；vendor不入Git）。从该源解包：

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0
go test -mod=vendor -count=1 ./...
GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -mod=vendor -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_amd64 .
GOOS=linux GOARCH=arm64 GOARM64=v8.0 go build -mod=vendor -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_arm64 .
```

固定go.mod/go.sum与go1.22运行defaults不变。实际发布验证与首次失败/修复见匹配Release；只做离线/loopback及help/license，无真实全平台、Spotify请求、注册登录/邮件或跑分；arm64仅交编，不冒实机。Hulu Japan继承真实Go超时/Unknown仍未修，不改变HTTP/TLS/UA/预算掩盖。Yes不是播放/登录/模型调用保证。

原Remnawave/geocheck v0.3.0 (`50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`) MIT保留；RRC (`b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844`)派生为AGPL-3.0-only；**组合发行非MIT-only**。见LICENSE、LICENSES、NOTICE及逐函数来源/hash。准确对应源：
- https://github.com/shui1iao/service-unlock/tree/v0.5.8
- https://github.com/shui1iao/service-unlock/releases/download/v0.5.8/service-unlock-v0.5.8-source.zip

网络提供修改版须向交互用户提供实际对应源码，不把旧公开tag冒称修改源。

<details><summary>历史v0.5.6及更早的说明/证据（非当前目录与显示契约）</summary>

# Service Unlock v0.5.6

> **v0.5.6（2026-11-05）平台单行聚合**：纠正v0.5.5只是归邻的问题。现在**每个平台一个逻辑表格行**，例如`ChatGPT | Web ● Yes · API ● No | JP`（规则示意，不是真测）。每项能力保持独立状态与颜色，不取最好值、不投票、不合成总Yes。窄屏允许单元格自然换行，但平台只占一个displayRow。
>
> 16家族明确合并（完整名单见实现说明），全目录**168测量ID→145平台**，不是新增all-region运行模式。各机器仍只执行global+一个当地组。四数组/selection/raw字段、顺序、状态、计时及网络完全沿用v0.5.5；**本版没有新raw删项，也不减少请求**。地区进入本平台Region，Netflix保`Catalogue JP · Fast US`、Prime保`Access JP · Geo US`，不跨来源/网络族补位；缺失地区观察明确Unknown，extra可选空字段不制造失败。Qwen API SG/CN是部署标签，不是出口国家。
>
> 六类、单圆角框、三列、默认字体/状态色及宽度预算不变。9安全Unknown、Spotify严格旧匿名端点判据和地区观察保留，不保证实际能注册/登录。GUKO/cmd旁路未改未部署；共享renderer仅影响未来构建。
>
> 启动器固定v0.5.6；**旧check.sh须重新下载**。[固定GitHub启动器](https://github.com/shui1iao/service-unlock/releases/download/v0.5.6/check.sh)。运行短链**待主控独立切换**，发布本仓库不表示短链已更新。验证范围见下方及Release，不把历史实测当本版全平台实测。

> **历史发布v0.5.5（2026-11-05）**：停掉宽松的 `geo.youtube_premium`，只保留严格的 `access.youtube_premium_access`。原生目录 **35→34**，`services` **21→20**，global **41→40**，各地区总量减1。**破坏性变化：移除 `services` 数组中 `id=youtube_premium` 的项目**，不是仅隐藏一行；依赖旧ID/数量的消费者须调整。保留项的raw字段/名称/顺序及判据不变。
>
> 终端固定六类用途，同平台不同能力相邻，并明确地区/片库/Web/API等后缀；单圆角框、三列、字体、状态颜色和换行算法不变。Spotify旧匿名端点保守Unknown/精确拒绝判据完全沿用v0.5.4，不代表实际能注册或登录。9个安全Unknown保留，GUKO未部署。
>
> 历史v0.5.5启动器固定v0.5.5；该版只归邻，不含v0.5.6的平台单行聚合。

> v0.5.4 收窄 Spotify signup 判据：不再把未经证实的响应判为可注册；证据不足为 Unknown，仅明确的旧端点拒绝保留限定 No。字体、单框布局和其他检测逻辑不变。本轮没有账号注册/登录，不声称验证了实际注册成功。

> v0.5.3 将完整报告合并为一个圆角框，框内按检测组分区，仅顶部一套三列表头；删除所有组统计与解释。自动选区、检测逻辑和 JSON 内容保持不变。

> v0.5.2 默认自动识别出口国家，执行全球＋对应区域服务，不再要求手动传地区参数。欧洲 DE→eu（71项）、新加坡 SG→sea（52项）、日本 JP→jp（67项）已各一轮真实默认参数验证通过。

> v0.5.1 删除 **NotebookLM、ChatGPT App、JCOM**，并发布 WOWOW、Spotify signup、Abema、DeepSeek API 的已验证修复。Hulu Japan 虽有受控 HTTP/1.1 回退，但 JP 真机仍超时，继续显示 Unknown，**尚未解决**。本次仅发布脚本，不部署 GUKO。

Linux amd64 / arm64 独立检测程序。以新版统一格式补齐缺失项目，按 **全球＋所选地区** 检测；无需在运行端安装 Go、Node、Python 或 Bash 探测后端。

## v0.5.4：Spotify signup准确性边界

- 此项发的是无email/password/电话/登录令牌的**不完整匿名旧注册端点POST**，不是官方当前完整注册流程，也不是带`validate=1`的操作。它不能可靠证明“可注册”。
- 去掉`311＋country＋is_country_launched=true→Yes`的无实证规则：此前“成功”夹具是人工构造，不是真实注册成功证据；现在这类响应为Unknown。也不再用裸120/320或launched=false推导No。
- 只有已捕获的`status320`＋有效JSON中完全匹配的明确proxy拒绝，才为范围受限No，Detail说明**此旧端点的这次匿名请求被拒绝，实际注册/登录未测试**。裸状态、未知文案、配置/launch标志、类型/重复字段错误或挑战/HTTP错误均Unknown。
- 旧JP320明确拒绝是真实证据，但不是所有注册渠道都不可用的证明。原Spotify地区观察仍独立，不给signup投票。固定HK creation_point/生日/同意字段未猜测修改，也不拿它们推断用户资格。
- 项目ID/名称保留；不存在可信匿名正向判据，不能为了显示Yes去补邮箱/密码/验证码或创建账号。若要进一步判断真实注册结果，须用户在官方流程自行实际尝试，不能由本探针替代。

## v0.5.1 修复与已知限制

ChatGPT App与JCOM已从有效检测目录/判据/请求/输出删除，不再执行其端点；此前NotebookLM删除保留。ChatGPT Web、ChatGPT地区观察和OpenAI API检查仍在。历史检测报告未改，本次删项不做新实测。

- DeepSeek API：已整合官方`/models`的特定401纯文本`Authentication Fails (governor)`窄识别；只证明无凭据认证入口可达，不是模型可调用。另一授权单项实测已验证，未重复请求。
- Gemini仍是匿名页面/JPN地区观察Yes，不是登录或模型调用成功；DAZN保留实际Startup业务允许字段，不为了追齐curl的边缘403改判。
- Hulu专属诊断确认Go的h1回退TCP/TLS、http/1.1协商和写请求均成功，但没有首字节即6秒预算截止；同参数curlh1到restrict页面。尚无可证明的Go请求代码错误，继续Unknown，**未解决真机兼容**；未全局禁h2、换UA/出口或扩预算。

## 默认自动选择：无需地区参数

```sh
./service-unlock_linux_amd64
./service-unlock_linux_amd64 --json > result.json
./service-unlock_linux_amd64 --machine > report-envelope.json
```

不传`--region`即`auto`：先选择新增检查实际地址族，再识别这个出口国家，自动执行global＋对应当地组。例如JP→jp（66项）、US→na（77项）、ES/DE→eu（70项）、SG→sea（51项）。国家只是调度依据，不是物理位置保证，也不是内容平台提供的地区。

显式参数始终覆盖且**不请求定位**：`--region global`只跑全球40项，`--region jp`固定全球＋日本。定位失败或国家无已支持组（如CN/IN）则只跑global，并给出简短英文原因；不新增印度/体育/全地区目录。

## 一键下载运行

```sh
curl -fLsS https://github.com/shui1iao/service-unlock/releases/download/v0.5.6/check.sh -o check.sh && sh check.sh
```

默认无需追加地区参数。仍可人工覆盖；显式group/global完全跳过定位：

```sh
sh check.sh --region hk
sh check.sh --region jp
sh check.sh --region na --json > result.json
```

## v0.5.6原始测量数量（与v0.5.5相同）

地区值：`auto / global / tw / hk / jp / na / sa / eu / oc / kr / sea / af`，默认`auto`。下表为v0.5.6 resolved group 的**原始测量数**，不是聚合后的终端平台行数：

| 地区 | 结果数 |
|---|---:|
| global | 40 |
| tw | 51 |
| hk | 48 |
| jp | 66 |
| na | 77 |
| sa | 43 |
| eu | 70 |
| oc | 59 |
| kr | 48 |
| sea | 51 |
| af | 43 |

不是每台服务器检测所有地区。auto只用出口国家调度一个已存在的当地组，不从Google/Netflix地区或库存投票猜所在地。旧 `check.sh` 固定旧版本，升级需重新下载。启动器固定下载 v0.5.6，对应架构二进制通过 SHA256 校验后在临时目录运行并清理；需现有 curl/GNU wget、SHA256 校验工具和系统 CA，不自动安装软件，不安装常驻服务。

## 自动调度、安全与数据

- 仅auto请求固定HTTPS端点：优先`https://www.cloudflare.com/cdn-cgi/trace`的严格`h/loc`响应；必要时只fallback一次到`https://api.ip.sb/geoip`的`country_code`。不执行下载代码、不需key/账号/SDK，不安装依赖。
- 和extra使用**同一实际地址族/direct transport/system CA**；不使用默认http.Get、代理或不同族回退。无该族本地路由则明确失败global，不静默换族。native34仍system-selected，不冒标整份报告同族。
- 定位是探测前的串行流程，两个请求共享**一个`--timeout`总预算**（默认6秒），各请求仍继承extra的压缩/解压4MiB/取消保护；定位仅接受16KiB以内的响应，拒绝定位端点重定向（包括自循环）。失败不会终止服务检查或把它们改No/全Unknown；取消仍不输出半份JSON。
- 只保留国家码、固定来源与安全错误码，不展示/保存定位响应的公网IP、ASN、坐标或原文。重复/冲突字段、API错误、非法/私用代码、HTTP/挑战/超限均安全降级。
- auto人类报告只有一行如`Automatic region: JP -> Japan (jp)`；JSON仍纯JSON，machine的JSON和ANSI使用同次selection，不重新定位。

## 检测与展示

- 按用户要求已删除 NotebookLM、ChatGPT App 和 JCOM 有效探针/判据/输出。v0.5.5再移除宽松的 geo YouTube Premium，核心目录现为 34 项（20 项地区/访问、6 项服务可用性、8 项 AI API 可达性）；JSON results 仍6项，services 为20项，其余保留项字段语义保持。
- 增补目录共 134 个 ID：125 项 Go 原生实现，9 项因账户、明文传输、弱 TLS 或非服务 GeoIP 判据而明确不启用，显示 `Unknown` 且不发请求；**不是 134 项全部启用**。
- v0.5.5已保留严格 access YouTube Premium、移除旧 geo 宽松测量，因此相对v0.5.4少一次该逻辑请求；v0.5.6不再删测量、不再减少请求。Netflix 片库/Fast地区、TikTok Web/地区、Prime Video 地区/访问、AI Web/API/地区等独立能力仍分别测量；不跨族共享或推导结果。
- 不额外搬入旧脚本的定位、CDN、币种等 7 项纯信息/残余观察；不测速、不运行旧脚本广告/计数/主流程。
- 全英文 `Service | Result | Region`：整个报告一个圆角外框、顶部一套列标题，框内固定用途组顺序为 `Video, live & sports / Music & audio / AI services / Gaming / Community / General websites`，空组省略。按源+稳定ID确定平台与用途，每个平台一个逻辑行；不同能力在Result内独立显示短label与状态，不合成总状态。地区仍只控制extra选择，各来源国家在Region中保留冲突与Unknown；纯地区平台Result明确Region/Store region，不冒称解锁。DMM标为金融站，与DMM TV分开。原始JSON名称不改。Yes绿、No红、Limited黄、Unknown默认前景加粗；成功国码规范为ISO2，不将API部署或市场标签冒充观测国家。
- 不再显示各组底部Yes/No/Limited/Unknown统计或解释，不把说明挪入框内。自动选区一行仍保留；不显示每行说明、耗时或总耗时。深浅终端均使用默认正文/边框前景。

`Yes` 是展示归一标签，可表示识别到地区、未发现限制或 API 可达，**不是实际登录、播放或调用模型的保证**。

## 参数与 JSON

选择对应的逻辑平台行数（不是原始测量数或窄屏物理行数）：global28 / tw39 / hk36 / jp54 / na65 / sa31 / eu56 / oc47 / kr36 / sea36 / af31。窄屏物理行数可以更多，不是少测量。

以下示例使用重新下载的v0.5.6启动器。machine外壳仍是`guko-unified-v1`，report和terminal来自同次测量；terminal现在每平台一行，raw相对v0.5.5完全不变。旧v0.5.5的Premium删项是继承历史，不是本版新增破坏性JSON变化。共享renderer也影响旁路源码日后构建的ANSI，本版未修改cmd，更未改线上GUKO。

```sh
sh check.sh --help
sh check.sh --region jp --extra-family 4 --timeout 10 --workers 4
sh check.sh --region na --machine > report-envelope.json
```

- `--extra-family auto|4|6`：只影响新增检查。auto 通过本地无发包 UDP socket 路由选择优先 IPv4，否则 IPv6；核心 34 项仍是系统选择地址族。不支持所选族的检查为 Unknown，整份报告不冒标为同一地址族。
- `--timeout` 默认每请求 6 秒，含连接、重定向和正文；不是整轮截止。`--workers` 默认 4，范围 1–8，全局并发有界。
- TLS 验证开启，检测直连、不使用代理环境变量。原始/解压正文限额、重定向限额、取消与资源回收保留。
- `--json` 保留原 `schema_version/source/generated_at/network/results/services/ai_endpoints` 字段意义（v0.5.5只移除 services 中 id=youtube_premium，其余保留项不改），增加 `extra_checks`、`selection`。不把 JSON 原枚举翻译成 Yes/No；说明、耗时和 HTTP 诊断仍保留。
- `--machine` 输出同一次测量的 JSON＋ANSI envelope（`guko-unified-v1`），不额外再跑一次检测或定位；与 `--json` 互斥。
- `selection.region`仍是resolved group；新增`requested_region`（auto或显式组）、`detected_country`、`detection_source`、`detection_error`、`resolution_reason`。显式覆盖不声称定位过；失败原因安全化，定位信息不生成额外检查行。
- `source.port` 保留 v0.4.0 历史实现标记（检测目录已按授权删除NotebookLM与App，JP额外删除JCOM）以维持原字段兼容；发行版本以 `--help` 的 v0.5.6 为准，新增来源见 `selection`。
- 退出码 0＝全部得到结论（含 No/Limited）；1＝至少一项 Unknown 或运行失败；2＝参数错误；130/143＝中断。**0 不代表全部解锁，1 也不代表没有可用结果。** 由于 global 含安全排除项，完整报告仍可能正常返回 1。

9 项未启用：HuluUS、AISPlay、trueID、Kancolle、Karaoke@DAM、DAnimeStore、NeonTV、SD Gundam、Crunchyroll。原因及 Optus/ESPN 移植差异、完整实现与网络限制见 [实现说明](README-unified-native.md)。

## v0.5.6验证与离线构建

本版发布前验证结果以Release说明为准；仅离线/回环固定响应，不重新实测全部平台。历史v0.5.5的通过证据不代替本版验证。Go要求1.26+，保留`godebug default=go1.22`运行时默认值。

源码ZIP含固定vendor及依赖许可（vendor不入Git），其余项目文件对应v0.5.6标签。安装好Go后，可在空模块缓存/禁网络环境运行：

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
CGO_ENABLED=0 go test -mod=vendor -count=1 ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -mod=vendor -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOARM64=v8.0 go build -mod=vendor -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_arm64 .
```

Git检出的开发树不含vendor；联网准备依赖时用`go mod download && go mod vendor`（固定`go.mod/go.sum`），然后采用同样命令。依赖的LICENSE/COPYING/NOTICE随vendor保留。

本版验证仅离线/回环，不重新实测全部服务，不请求Spotify真实旧端点，不注册/登录/发邮件。arm64仅交叉构建，未实机运行；Hulu Japan既有真实Go超时仍未解决，不保证任何平台当前线上解锁准确率。旧版的欧洲/新加坡/日本auto实测是历史记录；IPv6及备用定位源没有本版新增实机验证。

本版只发布独立脚本与二进制，不部署GUKO。程序没有PNG导出开关；`--machine`可供外部渲染器生成报告图。

## 来源、许可与对应源码

原 service-unlock / remnawave/geocheck v0.3.0（commit `50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`）的 MIT 版权声明保留。新增原生探针依据 lmc999/RegionRestrictionCheck commit `b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844`，属 AGPL-3.0 派生部分；**组合发行不再是 MIT-only**。修改记录和逐项源函数/hash在 `NOTICE.md`、`internal/extra/NOTICE.md` 和 `catalogue-source.json`。

完整许可见 `LICENSE`、`LICENSES/`、`internal/render/THIRD_PARTY_LICENSES.txt`；`--license` 可离线查看。

**v0.5.6**的完整源码、构建文件、测试、固定数据和许可免费提供在：

- [固定版本源码](https://github.com/shui1iao/service-unlock/tree/v0.5.6)
- [对应源码包](https://github.com/shui1iao/service-unlock/releases/download/v0.5.6/service-unlock-v0.5.6-source.zip)

源码包与同版本二进制在同一 Release 页面提供。网络部署的修改版还须向其交互用户提供实际对应源码；此脚本发布不代表 GUKO 已部署统一报告版。

</details>
