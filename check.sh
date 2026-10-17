#!/bin/sh
# Service Unlock v0.4.0 launcher. MIT; Copyright (c) 2026 Remnawave.
# The release URLs below are fixed. No installation and no persistent binary.

BASE=https://github.com/shui1iao/service-unlock/releases/download/v0.4.0
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

os=$(uname -s) || fail '无法识别系统'
[ "$os" = Linux ] || fail "不支持的系统: $os（仅Linux）"
arch=$(uname -m) || fail '无法识别架构'
case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) fail "不支持的架构: $arch（仅amd64/arm64）" ;;
esac
asset=service-unlock_linux_$arch

if command -v curl >/dev/null 2>&1; then downloader=curl
elif command -v wget >/dev/null 2>&1; then downloader=wget
else fail '需要curl或wget（不会自动安装）'; fi
if command -v sha256sum >/dev/null 2>&1; then verifier=sha256sum
elif command -v shasum >/dev/null 2>&1; then verifier=shasum
elif command -v openssl >/dev/null 2>&1; then verifier=openssl
else fail '需要sha256sum、shasum或openssl（不会自动安装）'; fi

work=$(mktemp -d "${TMPDIR:-/tmp}/service-unlock.XXXXXXXX") || fail '无法创建安全临时目录'
[ -n "$work" ] && [ -d "$work" ] || fail '临时目录无效'
download() {
    if [ "$downloader" = curl ]; then
        run_child curl -fLsS --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 180 -o "$2" "$1"
    else
        run_child wget -q --https-only --timeout=30 --tries=2 -O "$2" "$1"
    fi
}

download "$BASE/SHA256SUMS" "$work/SHA256SUMS" || fail '校验清单下载失败，拒绝执行'
expected=
while read -r digest filename extra; do
    [ "$filename" = "$asset" ] || continue
    [ -z "$extra" ] && [ -z "$expected" ] || fail '校验清单条目无效或重复'
    [ "${#digest}" -eq 64 ] || fail 'SHA256长度无效'
    case "$digest" in *[!0123456789abcdef]*) fail 'SHA256格式无效' ;; esac
    expected=$digest
done < "$work/SHA256SUMS"
[ -n "$expected" ] || fail '校验清单没有目标架构，拒绝执行'
download "$BASE/$asset" "$work/$asset" || fail '程序下载失败，拒绝执行'
case "$verifier" in
    sha256sum) sum=$(sha256sum "$work/$asset") || fail '无法计算SHA256' ;;
    shasum) sum=$(shasum -a 256 "$work/$asset") || fail '无法计算SHA256' ;;
    openssl) sum=$(openssl dgst -sha256 -r "$work/$asset") || fail '无法计算SHA256' ;;
esac
actual=${sum%% *}
[ "$actual" = "$expected" ] || fail 'SHA256不匹配，拒绝执行'
chmod 700 "$work/$asset" || fail '无法设置执行权限'
run_child "$work/$asset" "$@"
result=$?
[ "$interrupted" -eq 0 ] || exit "$interrupted"
exit "$result"
