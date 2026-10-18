# Service Unlock v0.5.0 — unified native implementation

This release extends the newer Service Unlock CLI itself. It is not a re-skin of a whole RRC run and not the previous original-image + 17-supplement design. Running probes contacts real services; this work used only offline fixtures/loopback tests. No managed server, real service measurement or Telegram message was tested.

## Usage after building locally

```sh
./service-unlock_linux_amd64 --help
./service-unlock_linux_amd64 --region global --extra-family auto
./service-unlock_linux_amd64 --region jp --extra-family 4 --json
./service-unlock_linux_amd64 --region na --extra-family auto --machine
```

No Bash/curl/Node/Python/Go runtime is used for probing. The download launcher uses existing curl or GNU wget and a SHA256 tool; it downloads the fixed v0.5.0 static binary. The previous GUKO-only 17-check side entry is not part of this release.

Original 8 access + 21 services + 8 AI checks and their JSON mappings remain unchanged. New `extra_checks` and `selection` are additive schema-v1 fields. `source.port` remains unchanged solely for old-field compatibility; `selection.extra_source_commit` identifies the RRC-derived addition. `--machine` returns `guko-unified-v1` containing the report and its ANSI rendering from ONE measurement. Normal terminal and --json remain available. Exit1 means at least one Unknown (including explicit unsupported IDs), not a negative unlock verdict. All-Unknown is a valid complete report; cancellation does not emit partial JSON.

| region | total checks |
|---|---:|
| global |43|
| tw |54|
| hk |51|
| jp |70|
| na |80|
| sa |46|
| eu |73|
| oc |62|
| kr |51|
| sea |54|
| af |46|

Only global + ONE region is selected. No India/all/sport/Instagram menu expansion; missing/unmapped inventory country means global without GeoIP fallback. Original37 duplicates (Spotify signup GET included) are not rerun as old RRC functions. No OneTrust/iQyi/GooglePlay-country/Steam-currency/YouTubeCDN/NetflixCDN/Bing-Risky information additions; original37's country checks/fallbacks are untouched.

## Native probes, provenance and safety

135 exact extra IDs remain catalogued, not 135 per server. **126 IDs have native request/parser implementations with per-ID success/refusal fixtures; 9 IDs remain explicit unsupported Unknown without any request**. Safety exclusions are not unfinished Go placeholders:

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

DAZN preserves its IP-ban branch and GeolocatedCountry; Prime access is a separate named row from original prime-country. Abema Oversea Only and Disney Available Soon become Limited. Mora HTTP500, empty/invalid JSON, 429/5xx, network failure and challenge pages become Unknown, not upstream's occasional false No. Catchplay is bound to its stable ID despite upstream's wrong error label. Unknown/No is never upgraded by a country code. New country values use pinned ISO validation; Radiko city, Niconico LiveID and arbitrary destination names are not Region. Wikipedia requires an edit-form marker before positive absence-of-ban is accepted. These are access/edit/game-API observations, not playback/login guarantees.

Every extra ID has a source function/line/hash in `internal/extra/catalogue-source.json`; native HTTP steps are readable fixed data in requests-source.json; scalar/flow format guards are explicit pinned metadata, not a new provider/inference framework. Tests exercise actual native execution paths, not just catalogue names. No source/check.sh, shell/curl process, remote include, JS execution, advertisement, run counter or public IP identity service is invoked.

## Network and limits

Original37: unchanged direct/system-selected. Extra: `--extra-family auto|4|6`; auto performs local UDP socket route selection (no Write/datagram), prefers IPv4 route, otherwise IPv6. JSON states the actual selected extra family; the whole report is NOT labelled IPv4. Source IPv6-unsupported checks explicitly remain Unknown if selected IPv6; no false v4 flag. GUKO passes an existing 4/6 mode when present, otherwise auto; country selection uses inventory only.

Extra transport ignores proxy environment, uses system CA and strict HTTPS including redirects. No InsecureSkipVerify, -k, weak TLS, system install or account credential reads. Each request has a context budget including redirect/body, 4MiB compressed and decoded body limits, max10-redirect bound, bounded worker pool; groups run consecutively under one global workers budget. Cancellation releases bodies/connections. Some endpoint algorithms/opaque anonymous client constants may now be stale: offline correctness is not proof of current live service behavior.

## Report and GUKO integration boundary

One English Service | Result | Region layout with common widths/rounded borders: original three groups plus Global services and, if selected, Regional services. Yes green, No red, Limited yellow, Unknown bold default foreground. ISO2 or blank; no per-row time/detail. Detailed group summaries distinguish observed criteria from playback/login.

The --machine envelope supports one complete measurement and its matching JSON/ANSI report for downstream tools. GUKO integration is a separate deployment, not included in this script release. An offline integration candidate has been tested, but the running GUKO instance is not changed by publishing this CLI. Consumers must validate expected IDs/counts and keep rendering/delivery failures distinct from probe results.

No GUKO image, compose switch or deployment is included/performed. The combined executable is not MIT-only: original MIT notices remain and AGPL-derived additions carry their full license. Exact Corresponding Source is published beside binaries and at the matching v0.5.0 source tag; see LICENSE, LICENSES and NOTICE.md. Real platform unlock and Telegram delivery remain untested.
