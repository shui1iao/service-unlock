# 原版渲染参考（全部为离线 mock）

`input.json` 是固定的八项示例结果，不代表任何服务器的真实检测。

三个 `upstream-*.golden` 来自独立参考程序，直接提取 remnawave/geocheck
v0.3.0、commit `50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc` 的以下原始函数，
不是用候选渲染包生成预期值：

- `internal/render/report.go`：`line`、`accessTable`、`accessStateCell`、
  `accessTimeCell`、`footer`、`section`、`table`、`rttCell`、`clip`、`formatRTT`、`maxInt`
- 原始 `internal/render/theme.go`

参考程序只调用 `accessTable` 和 `footer`，输入为 `input.json`，宽度 100、深色主题。
使用与上游相同版本的依赖；access 的结果类型/枚举不变，不调用任何检测。

- `plain`：colorprofile.NoTTY（普通重定向，去除 ANSI）
- `ansi`：colorprofile.TrueColor（保留完整原始真彩 SGR）
- `nocolor`：colorprofile.Ascii（原版 NO_COLOR 行为，去颜色但可保留加粗/复位）

本轮用户明确要求改善深/浅背景可读性，旧 `ansi/nocolor` 文件只保留历史参考，**不是当前颜色预期**。
`TestAccessLayoutUnchangedWithReadablePalette` 将当前输出去除 ANSI 后，与原版 plain 文件完整比较。
正文/地区/详情/耗时改为默认前景，状态使用 ANSI 语义色、error 默认前景加粗；新增三组测试另行验证
37行、目录顺序、分组含义、NO_COLOR、真实 PTY 和深/浅/未提供背景的可读性。原始 `clip` 按字节操作；
原始 `accessTimeCell` 仅因非 nil Err 或零耗时隐藏时间，未将所有 StateError 擅自改写。

来源许可：MIT，Copyright (c) 2026 Remnawave。
