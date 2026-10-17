# 历史渲染参考与本轮三列测试

`input.json` 是固定的8项 access mock，不代表真实检测。

`upstream-*.golden` 保留的是 remnawave/geocheck v0.3.0（commit
`50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc`）原始五列布局/配色输出，仅作历史来源记录。
用户已明确要求三组统一【服务｜结果｜地区】并删除说明及全部耗时，所以本轮**不以旧golden约束新显示**。

当前验证见 `three_columns_test.go`：37项中文映射、共同列宽与圆角外框、14–160终端列无损换行、
完整CLDR地区目录/ISO2与ISO3名称一致、删除说明和耗时、默认前景和NO_COLOR；
`theme_linux_test.go` 使用真实本地PTY测量窄屏尺寸后渲染。
CLI测试另外验证终端翻译/删列不改JSON原说明、耗时、状态枚举或地区码。

来源许可：MIT，Copyright (c) 2026 Remnawave。
