# Service Unlock v0.5.5

> **v0.5.5（2026-11-05）**：停掉宽松的 `geo.youtube_premium`，只保留严格的 `access.youtube_premium_access`。原生目录 **35→34**，`services` **21→20**，global **41→40**，各地区总量减1。**破坏性变化：移除 `services` 数组中 `id=youtube_premium` 的项目**，不是仅隐藏一行；依赖旧ID/数量的消费者须调整。保留项的raw字段/名称/顺序及判据不变。
>
> 终端固定六类用途，同平台不同能力相邻，并明确地区/片库/Web/API等后缀；单圆角框、三列、字体、状态颜色和换行算法不变。Spotify旧匿名端点保守Unknown/精确拒绝判据完全沿用v0.5.4，不代表实际能注册或登录。9个安全Unknown保留，GUKO未部署。
>
> 本版 `check.sh` 固定下载v0.5.5；**旧脚本不会自动升级，须重新下载**。发布时运行短链仍待独立切换；可先从[固定GitHub地址](https://github.com/shui1iao/service-unlock/releases/download/v0.5.5/check.sh)获取。下述旧版实测均为历史记录，不是本版全平台复测。

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
curl -fLsS https://github.com/shui1iao/service-unlock/releases/download/v0.5.5/check.sh -o check.sh && sh check.sh
```

默认无需追加地区参数。仍可人工覆盖；显式group/global完全跳过定位：

```sh
sh check.sh --region hk
sh check.sh --region jp
sh check.sh --region na --json > result.json
```

## v0.5.5目录数量

地区值：`auto / global / tw / hk / jp / na / sa / eu / oc / kr / sea / af`，默认`auto`。下表为v0.5.5 resolved group的结果数：

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

不是每台服务器检测所有地区。auto只用出口国家调度一个已存在的当地组，不从Google/Netflix地区或库存投票猜所在地。旧 `check.sh` 固定旧版本，升级需重新下载。启动器固定下载 v0.5.5，对应架构二进制通过 SHA256 校验后在临时目录运行并清理；需现有 curl/GNU wget、SHA256 校验工具和系统 CA，不自动安装软件，不安装常驻服务。

## 自动调度、安全与数据

- 仅auto请求固定HTTPS端点：优先`https://www.cloudflare.com/cdn-cgi/trace`的严格`h/loc`响应；必要时只fallback一次到`https://api.ip.sb/geoip`的`country_code`。不执行下载代码、不需key/账号/SDK，不安装依赖。
- 和extra使用**同一实际地址族/direct transport/system CA**；不使用默认http.Get、代理或不同族回退。无该族本地路由则明确失败global，不静默换族。native34仍system-selected，不冒标整份报告同族。
- 定位是探测前的串行流程，两个请求共享**一个`--timeout`总预算**（默认6秒），各请求仍继承extra的压缩/解压4MiB/取消保护；定位仅接受16KiB以内的响应，拒绝定位端点重定向（包括自循环）。失败不会终止服务检查或把它们改No/全Unknown；取消仍不输出半份JSON。
- 只保留国家码、固定来源与安全错误码，不展示/保存定位响应的公网IP、ASN、坐标或原文。重复/冲突字段、API错误、非法/私用代码、HTTP/挑战/超限均安全降级。
- auto人类报告只有一行如`Automatic region: JP -> Japan (jp)`；JSON仍纯JSON，machine的JSON和ANSI使用同次selection，不重新定位。

## 检测与展示

- 按用户要求已删除 NotebookLM、ChatGPT App 和 JCOM 有效探针/判据/输出。v0.5.5再移除宽松的 geo YouTube Premium，核心目录现为 34 项（20 项地区/访问、6 项服务可用性、8 项 AI API 可达性）；JSON results 仍6项，services 为20项，其余保留项字段语义保持。
- 增补目录共 134 个 ID：125 项 Go 原生实现，9 项因账户、明文传输、弱 TLS 或非服务 GeoIP 判据而明确不启用，显示 `Unknown` 且不发请求；**不是 134 项全部启用**。
- 严格 access YouTube Premium 保留，旧 geo 宽松实现及其测量已移除，每轮少一次该逻辑请求。Netflix 片库/Fast地区、TikTok Web/地区、Prime Video 地区/访问、AI Web/API/地区等独立能力仍分别测量；不跨族共享或推导结果。
- 不额外搬入旧脚本的定位、CDN、币种等 7 项纯信息/残余观察；不测速、不运行旧脚本广告/计数/主流程。
- 全英文 `Service | Result | Region`：整个报告一个圆角外框、顶部一套列标题，框内固定用途组顺序为 `Video, live & sports / Music & audio / AI services / Gaming / Community / General websites`，空组省略。按源+稳定ID归类、按明确平台键稳定归邻，不合并总状态；地区仍只控制extra选择。同名地区/片库/Web/API能力加终端后缀，DMM标为金融站；原始JSON名称不改。Yes 绿、No 红、Limited 黄、Unknown 默认前景加粗。真实地区用 ES/HK/JP/US 等两字母代码，无地区留空。
- 不再显示各组底部Yes/No/Limited/Unknown统计或解释，不把说明挪入框内。自动选区一行仍保留；不显示每行说明、耗时或总耗时。深浅终端均使用默认正文/边框前景。

`Yes` 是展示归一标签，可表示识别到地区、未发现限制或 API 可达，**不是实际登录、播放或调用模型的保证**。

## 参数与 JSON

以下示例使用重新下载的v0.5.5启动器。machine 外壳仍是 `guko-unified-v1`，report 和 terminal 来自同次测量；terminal 的用途分组与标签已变，按ID/数量消费services的工具须接受批准的删项。共享 renderer 也会影响旁路源码日后构建的ANSI，但本轮未复制/修改cmd旁路，更未改线上GUKO。

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
- `source.port` 保留 v0.4.0 历史实现标记（检测目录已按授权删除NotebookLM与App，JP额外删除JCOM）以维持原字段兼容；发行版本以 `--help` 的 v0.5.5 为准，新增来源见 `selection`。
- 退出码 0＝全部得到结论（含 No/Limited）；1＝至少一项 Unknown 或运行失败；2＝参数错误；130/143＝中断。**0 不代表全部解锁，1 也不代表没有可用结果。** 由于 global 含安全排除项，完整报告仍可能正常返回 1。

9 项未启用：HuluUS、AISPlay、trueID、Kancolle、Karaoke@DAM、DAnimeStore、NeonTV、SD Gundam、Crunchyroll。原因及 Optus/ESPN 移植差异、完整实现与网络限制见 [实现说明](README-unified-native.md)。

## 构建与离线验证

发布前已用Go1.26.10完成全包离线/回环`go test -mod=vendor -count=1 ./...`；首次旧计数和Apple固定索引断言失败后，仅修测试契约并定向、全包复验通过。验证覆盖168项分类/安全Unknown、能力归邻、旧Premium一次严格请求/无旧raw项、JSON/selection/machine、窄屏及launcher拒绝错误hash/清理。Go要求1.26+，保留`godebug default=go1.22`运行时默认值。

源码ZIP包含固定依赖的`vendor/`（不入Git标签），以及依赖许可；其余项目文件逐字节对应v0.5.5标签。使用本机Go工具链，无需网络即可测试/构建：

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
CGO_ENABLED=0 go test -mod=vendor ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -mod=vendor -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOARM64=v8.0 go build -mod=vendor -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_arm64 .
```

Git检出的开发树不含vendor；联网准备依赖时用`go mod download && go mod vendor`（固定`go.mod/go.sum`），然后采用同样命令。依赖的LICENSE/COPYING/NOTICE随vendor保留。

本版验证仅离线/回环，不重新实测全部服务，不请求Spotify真实旧端点，不注册/登录/发邮件。arm64仅交叉构建，未实机运行；Hulu Japan既有真实Go超时仍未解决，不保证任何平台当前线上解锁准确率。旧版的欧洲/新加坡/日本auto实测是历史记录；IPv6及备用定位源没有本版新增实机验证。

本版只发布独立脚本与二进制，不部署GUKO。程序没有PNG导出开关；`--machine`可供外部渲染器生成报告图。

## 来源、许可与对应源码

原 service-unlock / remnawave/geocheck v0.3.0（commit `50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`）的 MIT 版权声明保留。新增原生探针依据 lmc999/RegionRestrictionCheck commit `b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844`，属 AGPL-3.0 派生部分；**组合发行不再是 MIT-only**。修改记录和逐项源函数/hash在 `NOTICE.md`、`internal/extra/NOTICE.md` 和 `catalogue-source.json`。

完整许可见 `LICENSE`、`LICENSES/`、`internal/render/THIRD_PARTY_LICENSES.txt`；`--license` 可离线查看。

**v0.5.5**的完整源码、构建文件、测试、固定数据和许可免费提供在：

- [固定版本源码](https://github.com/shui1iao/service-unlock/tree/v0.5.5)
- [对应源码包](https://github.com/shui1iao/service-unlock/releases/download/v0.5.5/service-unlock-v0.5.5-source.zip)

源码包与同版本二进制在同一 Release 页面提供。网络部署的修改版还须向其交互用户提供实际对应源码；此脚本发布不代表 GUKO 已部署统一报告版。
