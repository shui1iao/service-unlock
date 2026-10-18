# Service Unlock v0.5.3

> v0.5.3 将完整报告合并为一个圆角框，框内按检测组分区，仅顶部一套三列表头；删除所有组统计与解释。自动选区、检测逻辑和 JSON 内容保持不变。

> v0.5.2 默认自动识别出口国家，执行全球＋对应区域服务，不再要求手动传地区参数。欧洲 DE→eu（71项）、新加坡 SG→sea（52项）、日本 JP→jp（67项）已各一轮真实默认参数验证通过。

> v0.5.1 删除 **NotebookLM、ChatGPT App、JCOM**，并发布 WOWOW、Spotify signup、Abema、DeepSeek API 的已验证修复。Hulu Japan 虽有受控 HTTP/1.1 回退，但 JP 真机仍超时，继续显示 Unknown，**尚未解决**。本次仅发布脚本，不部署 GUKO。

Linux amd64 / arm64 独立检测程序。以新版统一格式补齐缺失项目，按 **全球＋所选地区** 检测；无需在运行端安装 Go、Node、Python 或 Bash 探测后端。

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

不传`--region`即`auto`：先选择新增检查实际地址族，再识别这个出口国家，自动执行global＋对应当地组。例如JP→jp（67项）、US→na（78项）、ES/DE→eu（71项）、SG→sea（52项）。国家只是调度依据，不是物理位置保证，也不是内容平台提供的地区。

显式参数始终覆盖且**不请求定位**：`--region global`只跑全球41项，`--region jp`固定全球＋日本。定位失败或国家无已支持组（如CN/IN）则只跑global，并给出简短英文原因；不新增印度/体育/全地区目录。

## 一键下载运行

```sh
curl -fLsS https://check.siaoz.si -o check.sh && sh check.sh
```

默认无需追加地区参数。仍可人工覆盖；显式group/global完全跳过定位：

```sh
sh check.sh --region hk
sh check.sh --region jp
sh check.sh --region na --json > result.json
```

地区值：`auto / global / tw / hk / jp / na / sa / eu / oc / kr / sea / af`，默认`auto`。下表为resolved group的固定结果数：

| 地区 | 结果数 |
|---|---:|
| global | 41 |
| tw | 52 |
| hk | 49 |
| jp | 67 |
| na | 78 |
| sa | 44 |
| eu | 71 |
| oc | 60 |
| kr | 49 |
| sea | 52 |
| af | 44 |

不是每台服务器检测所有地区。auto只用出口国家调度一个已存在的当地组，不从Google/Netflix地区或库存投票猜所在地。旧 `check.sh` 固定旧版本，升级需重新下载。启动器固定下载 v0.5.3，对应架构二进制通过 SHA256 校验后在临时目录运行并清理；需现有 curl/GNU wget、SHA256 校验工具和系统 CA，不自动安装软件，不安装常驻服务。

## 自动调度、安全与数据

- 仅auto请求固定HTTPS端点：优先`https://www.cloudflare.com/cdn-cgi/trace`的严格`h/loc`响应；必要时只fallback一次到`https://api.ip.sb/geoip`的`country_code`。不执行下载代码、不需key/账号/SDK，不安装依赖。
- 和extra使用**同一实际地址族/direct transport/system CA**；不使用默认http.Get、代理或不同族回退。无该族本地路由则明确失败global，不静默换族。native35仍system-selected，不冒标整份报告同族。
- 定位是探测前的串行流程，两个请求共享**一个`--timeout`总预算**（默认6秒），各请求仍继承extra的压缩/解压4MiB/取消保护；定位仅接受16KiB以内的响应，拒绝定位端点重定向（包括自循环）。失败不会终止服务检查或把它们改No/全Unknown；取消仍不输出半份JSON。
- 只保留国家码、固定来源与安全错误码，不展示/保存定位响应的公网IP、ASN、坐标或原文。重复/冲突字段、API错误、非法/私用代码、HTTP/挑战/超限均安全降级。
- auto人类报告只有一行如`Automatic region: JP -> Japan (jp)`；JSON仍纯JSON，machine的JSON和ANSI使用同次selection，不重新定位。

## 检测与展示

- 按用户要求已删除 NotebookLM、ChatGPT App 和 JCOM 有效探针/判据/输出。核心目录现为 35 项（21 项地区/访问、6 项服务可用性、8 项 AI API 可达性）；JSON results 现6项，其余字段语义保持。
- 增补目录共 134 个 ID：125 项 Go 原生实现，9 项因账户、明文传输、弱 TLS 或非服务 GeoIP 判据而明确不启用，显示 `Unknown` 且不发请求；**不是 134 项全部启用**。
- 同功能不重复跑；Prime Video 的地区与访问限制是不同检查，分别保留。
- 不额外搬入旧脚本的定位、CDN、币种等 7 项纯信息/残余观察；不测速、不运行旧脚本广告/计数/主流程。
- 全英文 `Service | Result | Region`：整个报告一个圆角外框、顶部一套列标题，框内按原组顺序分区：Yes 绿、No 红、Limited 黄、Unknown 默认前景加粗。真实地区用 ES/HK/JP/US 等两字母代码，无地区留空。
- 不再显示各组底部Yes/No/Limited/Unknown统计或解释，不把说明挪入框内。自动选区一行仍保留；不显示每行说明、耗时或总耗时。深浅终端均使用默认正文/边框前景。

`Yes` 是展示归一标签，可表示识别到地区、未发现限制或 API 可达，**不是实际登录、播放或调用模型的保证**。

## 参数与 JSON

```sh
sh check.sh --help
sh check.sh --region jp --extra-family 4 --timeout 10 --workers 4
sh check.sh --region na --machine > report-envelope.json
```

- `--extra-family auto|4|6`：只影响新增检查。auto 通过本地无发包 UDP socket 路由选择优先 IPv4，否则 IPv6；核心 35 项仍是系统选择地址族。不支持所选族的检查为 Unknown，整份报告不冒标为同一地址族。
- `--timeout` 默认每请求 6 秒，含连接、重定向和正文；不是整轮截止。`--workers` 默认 4，范围 1–8，全局并发有界。
- TLS 验证开启，检测直连、不使用代理环境变量。原始/解压正文限额、重定向限额、取消与资源回收保留。
- `--json` 保留原 `schema_version/source/generated_at/network/results/services/ai_endpoints` 字段意义（删项与已证实判据修正对应的行值会改变），增加 `extra_checks`、`selection`。不把 JSON 原枚举翻译成 Yes/No；说明、耗时和 HTTP 诊断仍保留。
- `--machine` 输出同一次测量的 JSON＋ANSI envelope（`guko-unified-v1`），不额外再跑一次检测或定位；与 `--json` 互斥。
- `selection.region`仍是resolved group；新增`requested_region`（auto或显式组）、`detected_country`、`detection_source`、`detection_error`、`resolution_reason`。显式覆盖不声称定位过；失败原因安全化，定位信息不生成额外检查行。
- `source.port` 保留 v0.4.0 历史实现标记（检测目录已按授权删除NotebookLM与App，JP额外删除JCOM）以维持原字段兼容；发行版本以 `--help` 的 v0.5.3 为准，新增来源见 `selection`。
- 退出码 0＝全部得到结论（含 No/Limited）；1＝至少一项 Unknown 或运行失败；2＝参数错误；130/143＝中断。**0 不代表全部解锁，1 也不代表没有可用结果。** 由于 global 含安全排除项，完整报告仍可能正常返回 1。

9 项未启用：HuluUS、AISPlay、trueID、Kancolle、Karaoke@DAM、DAnimeStore、NeonTV、SD Gundam、Crunchyroll。原因及 Optus/ESPN 移植差异、完整实现与网络限制见 [实现说明](README-unified-native.md)。

## 构建与离线验证

开发需要 Go 1.26+，实际使用 Go 1.27.2。运行发行程序无需这些开发步骤。

```sh
go mod download
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOARM64=v8.0 go build -buildvcs=false -trimpath -ldflags='-s -w -buildid=' -o service-unlock_linux_arm64 .
```

固定依赖由 go.mod/go.sum 记录，`godebug default=go1.22` 保持既有运行时默认值。auto真实解析/调度请求路径及逐项请求/parser、错误保护、多阶段令牌流程、地区目录、JSON兼容、并发/TLS/取消及统一图片流程均以 mock/回环验证；amd64 已运行帮助和离线测试，arm64 仅交叉构建/静态检查，未实机运行。**默认auto已在欧洲、东南亚、日本三台各跑完整对应目录，全部正确选择并执行。此次均为IPv4和主定位源；IPv6及备用源仅完成离线验证，不保证所有平台的线上解锁准确率。**

此版本只发布独立脚本与二进制，不部署 GUKO。程序自身没有 PNG 导出开关；`--machine` 可供外部现有渲染器生成统一报告图。

## 来源、许可与对应源码

原 service-unlock / remnawave/geocheck v0.3.0（commit `50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`）的 MIT 版权声明保留。新增原生探针依据 lmc999/RegionRestrictionCheck commit `b6d4a6f9a87fc6eae6d3e62d0092ececcec8e844`，属 AGPL-3.0 派生部分；**组合发行不再是 MIT-only**。修改记录和逐项源函数/hash在 `NOTICE.md`、`internal/extra/NOTICE.md` 和 `catalogue-source.json`。

完整许可见 `LICENSE`、`LICENSES/`、`internal/render/THIRD_PARTY_LICENSES.txt`；`--license` 可离线查看。

v0.5.3 对应完整源码、构建文件、测试、固定数据和许可免费提供在：

- [固定版本源码](https://github.com/shui1iao/service-unlock/tree/v0.5.3)
- [对应源码包](https://github.com/shui1iao/service-unlock/releases/download/v0.5.3/service-unlock-v0.5.3-source.zip)

源码包与同版本二进制在同一 Release 页面提供。网络部署的修改版还须向其交互用户提供实际对应源码；此脚本发布不代表 GUKO 已部署统一报告版。
