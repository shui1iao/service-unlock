# Offline display fixtures

`input.json` is a six-check access fixture, not a real measurement.
`upstream-*.golden` preserves geocheck v0.3.0's old five-column output for historical provenance only.

The current user-approved presentation is English `Service / Result / Region`, with
Yes / No / Limited / Unknown and blank missing regions. Known regions are canonical uppercase ISO alpha-2 codes (including conversion from alpha-3), not names. It has no detail/time columns or duration footer. A single rounded frame contains one header and spanning group sections; there are no group status summaries/explanations.

`english-positive.golden` and `english-negative.golden` are deterministic 35-row
regression snapshots, reviewed against explicit category/state/colour assertions:
- positive: captcha raw `no` -> Yes; signup raw `yes` -> Yes;
- negative: captcha raw `yes` -> No; signup raw `no` -> No;
- both retain Netflix Limited, access/API refusals, Unknown and observed region codes; per-row semantics and order are unchanged.

`english_results_test.go` verifies the complete golden output, whole-word ANSI colouring,
blank regions, and known regions not overriding negative/unknown results.
`summaries_test.go` retains region-column assertions only; retired summary generators are removed. `single_box_test.go` verifies one frame, lossless internal titles across14–160 columns and absent explanations; root single-box tests verify complete41/67/71/78-row reports and byte-identical reportJSON before/after rendering.
`three_columns_test.go` exercises the pinned library's region catalogue, ISO2/ISO3-to-alpha-2,
shared widths, lossless 14–160-column wrapping (including headers, long region names,
synthetic CJK/combining text), and background/NO_COLOR behavior.
`theme_linux_test.go` uses a real local PTY. CLI tests preserve all original JSON values.

MIT License; Copyright (c) 2026 Remnawave.
