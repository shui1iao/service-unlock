#!/bin/sh
# Service Unlock v0.4.3 launcher. MIT; Copyright (c) 2026 Remnawave.
# The release URLs below are fixed. No installation and no persistent binary.

BASE=https://github.com/shui1iao/service-unlock/releases/download/v0.4.3
work=
child=
interrupted=0
umask 077

fail() { printf '%s\n' "service-unlock: $*" >&2; exit 1; }
cleanup() {
    if [ -n "$child" ]; then
        kill -TERM "$child" 2>/dev/null || :
        wait "$child" 2>/dev/null || :
    fi
    if [ -n "$work" ]; then rm -rf -- "$work"; fi
}
on_signal() {
    interrupted=$1
    if [ -n "$child" ]; then kill -TERM "$child" 2>/dev/null || :; fi
}
trap cleanup 0
trap 'on_signal 129' HUP
trap 'on_signal 130' INT
trap 'on_signal 143' TERM

# Track both downloaders and the probe. A signal arriving between launch and
# PID assignment is recorded and applied immediately after PID assignment.
run_child() {
    if [ "$interrupted" -ne 0 ]; then exit "$interrupted"; fi
    "$@" &
    child=$!
    if [ "$interrupted" -ne 0 ]; then kill -TERM "$child" 2>/dev/null || :; fi
    wait "$child"
    result=$?
    if [ "$interrupted" -ne 0 ]; then
        kill -TERM "$child" 2>/dev/null || :
        wait "$child" 2>/dev/null || :
        child=
        exit "$interrupted"
    fi
    child=
    return "$result"
}

os=$(uname -s) || fail 'Cannot identify the operating system'
[ "$os" = Linux ] || fail "Unsupported operating system: $os (Linux only)"
arch=$(uname -m) || fail 'Cannot identify the architecture'
case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) fail "Unsupported architecture: $arch (amd64/arm64 only)" ;;
esac
asset=service-unlock_linux_$arch

if command -v curl >/dev/null 2>&1; then downloader=curl
elif command -v wget >/dev/null 2>&1; then downloader=wget
else fail 'curl or wget is required (nothing will be installed)'; fi
if command -v sha256sum >/dev/null 2>&1; then verifier=sha256sum
elif command -v shasum >/dev/null 2>&1; then verifier=shasum
elif command -v openssl >/dev/null 2>&1; then verifier=openssl
else fail 'sha256sum, shasum, or openssl is required (nothing will be installed)'; fi

work=$(mktemp -d "${TMPDIR:-/tmp}/service-unlock.XXXXXXXX") || fail 'Cannot create a secure temporary directory'
[ -n "$work" ] && [ -d "$work" ] || fail 'Invalid temporary directory'
download() {
    if [ "$downloader" = curl ]; then
        run_child curl -fLsS --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 180 -o "$2" "$1"
    else
        run_child wget -q --https-only --timeout=30 --tries=2 -O "$2" "$1"
    fi
}

download "$BASE/SHA256SUMS" "$work/SHA256SUMS" || fail 'Checksum manifest download failed; refusing to execute'
expected=
while read -r digest filename extra; do
    [ "$filename" = "$asset" ] || continue
    [ -z "$extra" ] && [ -z "$expected" ] || fail 'Invalid or duplicate checksum manifest entry'
    [ "${#digest}" -eq 64 ] || fail 'Invalid SHA256 length'
    case "$digest" in *[!0123456789abcdef]*) fail 'Invalid SHA256 format' ;; esac
    expected=$digest
done < "$work/SHA256SUMS"
[ -n "$expected" ] || fail 'Architecture missing from checksum manifest; refusing to execute'
download "$BASE/$asset" "$work/$asset" || fail 'Program download failed; refusing to execute'
case "$verifier" in
    sha256sum) sum=$(sha256sum "$work/$asset") || fail 'Cannot compute SHA256' ;;
    shasum) sum=$(shasum -a 256 "$work/$asset") || fail 'Cannot compute SHA256' ;;
    openssl) sum=$(openssl dgst -sha256 -r "$work/$asset") || fail 'Cannot compute SHA256' ;;
esac
actual=${sum%% *}
[ "$actual" = "$expected" ] || fail 'SHA256 mismatch; refusing to execute'
chmod 700 "$work/$asset" || fail 'Cannot set executable permissions'
run_child "$work/$asset" "$@"
result=$?
[ "$interrupted" -eq 0 ] || exit "$interrupted"
exit "$result"
