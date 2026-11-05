# Offline display fixtures

`input.json` is a six-check access fixture, not a real measurement.
`upstream-*.golden` preserves geocheck v0.3.0's old five-column output for historical provenance only.

The current user-approved presentation is English `Service / Result / Region`, with
Yes / No / Limited / Unknown and blank missing regions. Known regions are canonical uppercase ISO alpha-2 codes (including conversion from alpha-3), not names. It has no detail/time columns or duration footer. A single rounded frame contains one header and spanning group sections; there are no group status summaries/explanations.

`english-positive.golden` and `english-negative.golden` are statically updated
34-row fixtures for the v0.5.5 six-purpose layout:
- positive: captcha raw `no` -> Yes; legacy signup inconclusive -> Unknown;
- negative: captcha raw `yes` -> No; legacy signup explicit-refusal display -> No;
- both retain Netflix Limited, access/API refusals, Unknown and observed region codes;
  strict access YouTube Premium remains, while the retired geo row is absent.

`english_results_test.go` asserts complete golden output, whole-word ANSI colouring,
blank regions, and known regions not overriding negative/unknown results.
`catalogue_test.go` asserts all168 active IDs map to one purpose, nine safety Unknowns
stay visible, known capability labels/platform adjacency and independent B-Global samples.
`summaries_test.go` retains region-column assertions only; retired summary generators are removed.
`single_box_test.go` asserts one frame and lossless titles across14–160 columns; root
single-box tests cover complete40/66/70/77-row reports and unchanged reportJSON on rendering.
`three_columns_test.go` exercises the pinned library's region catalogue, ISO2/ISO3-to-alpha-2,
shared widths, lossless 14–160-column wrapping (including headers, long region names,
synthetic CJK/combining text), and background/NO_COLOR behavior.
`theme_linux_test.go` uses a real local PTY. CLI assertions preserve retained raw JSON values; the approved services.youtube_premium removal is the sole catalogue exception.

MIT License; Copyright (c) 2026 Remnawave.
