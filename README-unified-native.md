# Service Unlock v0.5.4 — unified native implementation

> v0.5.4 narrows Spotify signup conclusions to the available evidence: no unverified positive eligibility verdict, Unknown when inconclusive, and a scoped No only for an explicit observed legacy-endpoint refusal. This audit read public official sources and replayed existing responses; no new signup endpoint request, account creation or login was performed. Renderer/font/layout/GUKO are unchanged.

> v0.5.3 presents the whole report in one rounded outer frame, with internal group sections and one shared column header. All per-group status counts and explanations are removed. Probe, JSON and default-auto semantics are unchanged.

> v0.5.2 defaults to automatic exit-country selection. One unoverridden run each on Europe (DE -> eu, 71 checks), Singapore (SG -> sea, 52) and Japan (JP -> jp, 67) successfully identified and executed the mapped regional checks. These live tests used IPv4 and the primary country source; fallback and IPv6 behavior have offline coverage only. GUKO is not deployed by this release.

> v0.5.1 publishes the NotebookLM/App/JCOM removals and retained JP fixes. Scoped FXTRANSIT-JP retests verified Spotify signup, WOWOW, Abema and DeepSeek; Hulu Japan still times out, including a separate 15-second diagnostic. No full platform scan or GUKO deployment is claimed.

This release extends the newer Service Unlock CLI itself. It is not a re-skin of a whole RRC run and not the previous original-image + 17-supplement design. Running probes contacts real services. Offline/loopback regressions remain; a separately authorized four-check live retest confirmed three explicit service refusals. Hulu's actual HTTP/1.1 fallback still timed out under the original6-second budget, so it remains Unknown. No Telegram/GUKO deployment or other machine was used.

## Repairs and remaining diagnostic limits

ChatGPT App (access ID) and JCOM (JP extra ID) have now been removed by user request: no active probe/helper/parser, endpoint, registration, selection or output remains. The dedicated edge-denial and JCOM header-protection tests/helpers are recoverably archived outside the repository, not left as dead checks. This does not remove ChatGPT Web, the ChatGPT country observation or OpenAI API, and does not remove shared challenge protection used by other checks.

The DeepSeek owner's precise official-/models/401/Authentication Fails (governor) fix is retained unchanged; reachable proves an unauthenticated authentication response, not model calls. Gemini remains an anonymous-page/region observation (JPN Yes), not a signed-in/redirect/model-success test. No old feature marker or country-list speculation is added. DAZN still uses its own explicit Startup business allow fields; curl's different edge reply is not a reason to change the Go verdict.

The earlier App/JCOM retest is historical only; neither is supported by this release. A single final Hulu phase diagnostic propagated ClientTrace into the task-copy fallback: TCP and TLS succeed, ALPN=http/1.1, request is written, but no response byte before the original6s deadline. Same-parameter curlh1 reaches restrict.html. No candidate cloning/connection/setup failure was demonstrated, and the real Go Hulu timeout remains unresolved/Unknown; no global h2 disabling, TLS/UA changes, proxy or budget increase is applied. The trace change is observational task-copy-only, not a production probe change.

## Usage after building locally

```sh
./service-unlock_linux_amd64 --help
./service-unlock_linux_amd64   # default --region auto: exit country -> global + mapped region
./service-unlock_linux_amd64 --json > result.json
./service-unlock_linux_amd64 --region global  # explicit override: NO country requests
./service-unlock_linux_amd64 --region jp --extra-family 4 --json
./service-unlock_linux_amd64 --region na --extra-family auto --machine
```

No Bash/curl/Node/Python/Go runtime is used for probing. The download launcher uses existing curl or GNU wget and a SHA256 tool; it downloads the fixed v0.5.4 static binary. The previous GUKO-only side entry is not part of this release; its local source now selects16 checks (1 access +7 services +8 AI), without changing the deployed old17 binary.

NotebookLM and ChatGPT App have been removed from the active access implementation/predicates/catalogue/fixtures/output by user request; JCOM is removed only from the JP extra group. Core checks are now 6 access + 21 services + 8 AI =35; results contains6 rows, without changing other field semantics. Removed dedicated tests are archived in the task artifact, not executed. New `extra_checks` and `selection` are additive schema-v1 fields. `source.port` remains a historical unchanged marker despite this authorized catalogue deletion, solely for old-field compatibility; `selection.extra_source_commit` identifies the RRC-derived addition. `--machine` returns `guko-unified-v1` containing the report and its ANSI rendering from ONE measurement. Normal terminal and --json remain available. Exit1 means at least one Unknown (including explicit unsupported IDs), not a negative unlock verdict. All-Unknown is a valid complete report; cancellation does not emit partial JSON.

| region | total checks |
|---|---:|
| global |41|
| tw |52|
| hk |49|
| jp |67|
| na |78|
| sa |44|
| eu |71|
| oc |60|
| kr |49|
| sea |52|
| af |44|

Only global + ONE region is selected. This independent CLI defaults to --region auto (not inventory or platform-locale inference). It first resolves the actual extra family by the unchanged route policy, then serially reads the exit country using that same direct/system-CA/family-pinned transport. Explicit global or a group always skips country lookup. Valid but unmapped countries (including CN/IN) resolve to global with unsupported_region_country; failed lookup resolves to global with detection_failed, while normal core/extra probes continue. No India/all/sport/Instagram menu expansion is added. Core35 duplicates are not rerun as old RRC functions. The audited implementation retains one anonymous incomplete legacy signup POST for spotify_signup, but no longer claims it establishes registration eligibility; no GET+POST voting or migration to an account-creation workflow. The endpoint receives no email/password/account token/Cookie/authorization/opaque identifier_token, send-email is disabled, and registration cannot be completed by this incomplete form. Fixed public RRC client key and non-identity validation fields are kept for their original validation purpose; a neutral synthetic display label replaces the upstream dummy label. The old status311/country/launched=true positive was an unverified heuristic with a synthetic fixture, NOT observed registration success. There is now NO anonymous Yes path: status1/configuration, status311, bare120/320 and launched=false are Unknown. Only the captured status320 plus its exact typed proxy-refusal message yields a scoped No for this legacy request, explicitly not a guarantee that all signup channels fail. Types, duplicate/conflicting keys, changed endpoints, HTTP/challenge and response-format guards remain conservative. Country/launch metadata do not vote on signup; optional country appears only in the existing error/detail channel, without schema changes. The RRC-derived helper is AGPL-labelled in internal/extra/spotify_signup.go; the geo adapter retains the service ID/type and existing direct/system-selected stack. No OneTrust/iQyi/GooglePlay-country/Steam-currency/YouTubeCDN/NetflixCDN/Bing-Risky information additions; the retained core country checks/fallbacks are untouched.

## Native probes, provenance and safety

134 exact extra IDs remain catalogued, not 134 per server. **125 IDs have native request/parser implementations with per-ID success/refusal fixtures; 9 IDs remain explicit unsupported Unknown without any request**. Safety exclusions are not unfinished Go placeholders:

- HuluUS: hard-coded account/password login form.
- AISPlay: source account/guest credentials and stream authorization.
- trueID: source credentials/basic-auth flow.
- Kancolle: source literal HTTP target, no evidenced same-capability HTTPS replacement.
- Karaoke@DAM: source HTTP media playlist target, no evidenced HTTPS replacement.
- DAnimeStore: source explicitly enables UnsafeLegacyServerConnect.
- **NeonTV (newly discovered during implementation):** fixed cookie template is UpdateAccount with email/password/account token, not anonymous metadata. That template is not copied into runnable source.
- **SD Gundam (newly identified identity boundary):** source explicit user_id + opaque x-token + encrypted consent body cannot be established as anonymous client data; those values are not copied into runnable source.
- **Crunchyroll (newly identified scope boundary):** old function only consults c.evidon.com generic third-party GeoIP and infers US access. No new GeoIP or guessed platform verdict is allowed; no request is made.

Anonymous device/client constants for Disney+/ESPN+ and service API metadata are fixed source data, not system/user credentials. Anonymous Bahamut cookies live only in a per-check memory jar. Optus's validation-path email is changed from the upstream real-looking address to reserved `region-check@example.invalid`; no account login is attempted. This retains the original validation endpoint/HTTP criterion but is not claimed live-equivalent. ESPN's upstream cookie line points at registerDevice instead of the refresh query it expects; this translation uses the fixed anonymous refreshToken GraphQL template with the ESPN client authorization. Both differences have offline parameter/sequence fixtures and require real validation only if separately authorized later.

DAZN preserves its IP-ban branch and GeolocatedCountry; Prime access is a separate named row from original prime-country. Abema Oversea Only and Disney Available Soon become Limited. This repair recognizes only Abema's captured business JSON message anonymous_ip as an explicit refusal, after shared HTTP/challenge guards; missing country/unknown messages remain Unknown. WOWOW now models JSON lineup, HTML programme/content pages and JSON playback authorization separately; empty/invalid stage data or missing session evidence is Unknown, while captured error2055/VPN refusal is No. Hulu Japan alone may clone the existing strict/direct/family-pinned transport for one HTTP/1.1 attempt after typed PROTOCOL_ERROR/INTERNAL_ERROR on its credential-free GET; both attempts, redirects and body share the original context budget. No HTTP-status/generic-error retry or global HTTP2 disabling is added. The fallback is recorded in detail. The scoped live retest confirmed it triggers after INTERNAL_ERROR, but the HTTP/1.1 attempt still timed out; this is not a claimed successful Hulu fix. Mora HTTP500, empty/invalid JSON, 429/5xx, network failure and challenge pages become Unknown, not upstream's occasional false No. Catchplay is bound to its stable ID despite upstream's wrong error label. Unknown/No is never upgraded by a country code. New country values use pinned ISO validation; Radiko city, Niconico LiveID and arbitrary destination names are not Region. Wikipedia requires an edit-form marker before positive absence-of-ban is accepted. These are access/edit/game-API observations, not playback/login guarantees.

Every extra ID has a source function/line/hash in `internal/extra/catalogue-source.json`; native HTTP steps are readable fixed data in requests-source.json; scalar/flow format guards are explicit pinned metadata, not a new provider/inference framework. Tests exercise actual native execution paths, not just catalogue names. No source/check.sh, shell/curl process, remote include, JS execution, advertisement or run counter is invoked. Auto scheduling alone now permits the two fixed HTTPS country sources below; it does not add an IP identity/ASN/quality/CDN report row.

## Network and limits

Core35: unchanged direct/system-selected. Extra: `--extra-family auto|4|6`; auto performs local UDP socket route selection (no Write/datagram), prefers IPv4 route, otherwise IPv6. JSON states the actual selected extra family; the whole report is NOT labelled IPv4. Source IPv6-unsupported checks explicitly remain Unknown if selected IPv6; no false v4 flag. The independent CLI now determines its own exit country only in region=auto; it never reads GUKO inventory. A future GUKO consumer can explicitly pass a region to retain inventory scheduling and zero country requests; no GUKO code/deployment is changed here.

Extra transport ignores proxy environment, uses system CA and strict HTTPS including redirects. No InsecureSkipVerify, -k, weak TLS, system install or account credential reads. Each request has a context budget including redirect/body, 4MiB compressed and decoded body limits, max10-redirect bound, bounded worker pool; groups run consecutively under one global workers budget. Cancellation releases bodies/connections. Some endpoint algorithms/opaque anonymous client constants may now be stale: offline correctness is not proof of current live service behavior.

## Automatic scheduling contract

- Primary: https://www.cloudflare.com/cdn-cgi/trace; require the expected h field and one strict ISO loc field. Fallback only if necessary: https://api.ip.sb/geoip; require a typed country_code, reject duplicate/conflicting fields and API errors. UK canonicalizes to GB; map100 established GUKO country keys only, not a new regional catalogue.
- At most two serial requests share ONE --timeout budget (default6s), including redirects/body. The existing extra client imposes strict TLS/no proxy and compressed/decoded4MiB caps; country responses larger than16KiB are rejected. Every country-endpoint redirect, including self loops, is rejected before following. Family failure does not silently switch IPv4/IPv6. Successful primary, even an unsupported country, does not trigger fallback to hunt for another answer.
- No raw country response/IP/ASN/coordinates or arbitrary HTTP error text is stored in report metadata. Only detected ISO2, fixed source URL and sanitized reason codes survive. The lookup source still necessarily sees its own incoming connection's exit IP; lookup is for scheduling, not a physical-location guarantee.
- selection.region retains the RESOLVED group and old fields retain their meanings. New optional metadata: requested_region (auto or explicit group), detected_country, detection_source, detection_error, resolution_reason. Reasons are country_mapped / unsupported_region_country / detection_failed / explicit_region_override. Manual overrides have no fabricated country/source.
- Human/machine terminal output gets one simple English Automatic region line only in auto; the same completed measurements/selection supply both JSON and ANSI. --json stdout remains JSON-only. No additional probe row or request is performed for rendering. Existing exit-code meanings are unchanged; a lookup failure itself is not a service No.

## Report and GUKO integration boundary

One English Service | Result | Region layout in ONE rounded outer frame with one top header and shared column widths. Internal spanning section titles retain the original three groups plus Global services and, if selected, Regional services; no empty local section. Yes green, No red, Limited yellow, Unknown bold default foreground. ISO2 or blank; no per-row time/detail. All per-group status counts/explanations are removed, not relocated. The single automatic-region notice remains outside the frame; observation caveats remain documentation, not report footers.

The --machine envelope supports one complete measurement and its matching JSON/ANSI report for downstream tools. GUKO integration is a separate deployment, not included in this script release. An offline integration candidate has been tested, but the running GUKO instance is not changed by publishing this CLI. Consumers must validate expected IDs/counts and keep rendering/delivery failures distinct from probe results.

No GUKO image, compose switch or deployment is included/performed. The combined executable is not MIT-only: original MIT notices remain and AGPL-derived additions carry their full license. Exact Corresponding Source is published beside binaries and at the matching v0.5.4 source tag; see LICENSE, LICENSES and NOTICE.md. The scoped four-check retest is described above; it does not validate the rest of the catalogue against live platforms, and no Telegram test was sent. The earlier full JP comparison describes the unmodified public release.
