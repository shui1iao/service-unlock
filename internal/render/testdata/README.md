# Offline display fixtures

`input.json` is an eight-check access fixture, not a real measurement.
`upstream-*.golden` preserves geocheck v0.3.0's old five-column output for historical provenance only.

The current user-approved presentation is English `Service / Result / Region`, with
Yes / No / Limited / Unknown and blank missing regions. It has no detail/time columns or duration footer.

`english-positive.golden` and `english-negative.golden` are deterministic 37-row
regression snapshots, reviewed against explicit category/state/colour assertions:
- positive: captcha raw `no` -> Yes; signup raw `yes` -> Yes;
- negative: captcha raw `yes` -> No; signup raw `no` -> No;
- both retain Netflix Limited, access/API refusals, Unknown and observed English regions.

`english_results_test.go` verifies the complete golden output, whole-word ANSI colouring,
blank regions, and known regions not overriding negative/unknown results.
`three_columns_test.go` exercises the complete CLDR English region catalogue, ISO2/ISO3,
shared widths, lossless 14–160-column wrapping (including headers, long region names,
synthetic CJK/combining text), and background/NO_COLOR behavior.
`theme_linux_test.go` uses a real local PTY. CLI tests preserve all original JSON values.

MIT License; Copyright (c) 2026 Remnawave.
