# Offline display fixtures

`input.json` is a six-check access fixture, not a real measurement.
`upstream-*.golden` preserves geocheck v0.3.0's old five-column output for historical provenance only.

The current user-approved presentation is English `Service / Result / Region`, with
Yes / No / Limited / Unknown and blank missing regions. Known regions are canonical uppercase ISO alpha-2 codes (including conversion from alpha-3), not names. It has no detail/time columns or duration footer. Each group's summary explains the underlying observation categories without changing the per-row labels.

`english-positive.golden` and `english-negative.golden` are deterministic 35-row
regression snapshots, reviewed against explicit category/state/colour assertions:
- positive: captcha raw `no` -> Yes; signup raw `yes` -> Yes;
- negative: captcha raw `yes` -> No; signup raw `no` -> No;
- both retain Netflix Limited, access/API refusals, Unknown and observed region codes; summaries separate identified regions, accessibility, restriction markers, and API reachability.

`english_results_test.go` verifies the complete golden output, whole-word ANSI colouring,
blank regions, and known regions not overriding negative/unknown results.
`summaries_test.go` verifies detailed category counts, captcha inversion, mixed Unknown/negative results with regions, total counts across all 35 rows, and lossless narrow-screen summary wrapping.
`three_columns_test.go` exercises the pinned library's region catalogue, ISO2/ISO3-to-alpha-2,
shared widths, lossless 14–160-column wrapping (including headers, long region names,
synthetic CJK/combining text), and background/NO_COLOR behavior.
`theme_linux_test.go` uses a real local PTY. CLI tests preserve all original JSON values.

MIT License; Copyright (c) 2026 Remnawave.
