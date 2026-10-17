#!/usr/bin/env python3
"""Seven platforms / eight access probes, ported from geocheck v0.3.0.

Source commit: 50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc
https://github.com/remnawave/geocheck
Conservative error handling differs from upstream; see README.md.

MIT License

Copyright (c) 2026 Remnawave

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
"""

import argparse
import concurrent.futures
from dataclasses import dataclass
from datetime import datetime, timezone
import gzip
import io
import json
import math
import re
import ssl
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

SOURCE = {
    "repository": "https://github.com/remnawave/geocheck",
    "tag": "v0.3.0",
    "commit": "50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc",
    "port": "Python standard-library port with conservative error handling",
}
UA = ("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
      "(KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36")
MAX_BODY = 4 << 20
HTML_HEADERS = {
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    "Accept-Language": "en-US,en;q=0.9",
}
CLAUDE_HEADERS = dict(HTML_HEADERS, **{
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
    "Sec-Fetch-Dest": "document", "Sec-Fetch-Mode": "navigate",
    "Sec-Fetch-Site": "none", "Sec-Fetch-User": "?1",
    "Upgrade-Insecure-Requests": "1",
})
CHALLENGE_MARKERS = (
    "cf_chl_opt", "_cf_chl", "just a moment", "cf-browser-verification",
    "enable javascript and cookies to continue",
)
CLAUDE_MARKERS = (
    "app unavailable in region", "/app-unavailable-in-region",
    "unfortunately, claude isn't available here.",
    "unfortunately, claude isn&apos;t available here.",
    "unfortunately, claude isn&#39;t available here.",
)
TIKTOK_MARKERS = (
    "service is currently unavailable in your region",
    "tiktok is not available in your country",
    "tiktok is unavailable in your country", "not available in your region",
)
GEMINI_UNSUPPORTED = frozenset(("RUS", "BLR", "CHN", "PRK", "IRN", "CUB", "SYR"))
GEMINI_REGION = re.compile(r',[0-9]+,[0-9]+,200,"([A-Z]{3})"')
CHECKS = (
    ("chatgpt_web", "ChatGPT", "ChatGPT (web)"),
    ("chatgpt_app", "ChatGPT", "ChatGPT (app)"),
    ("youtube_premium_access", "YouTube Premium", "YouTube Premium"),
    ("netflix_access", "Netflix", "Netflix"),
    ("claude_access", "Claude", "Claude"),
    ("tiktok_access", "TikTok", "TikTok"),
    ("gemini_access", "Gemini", "Gemini"),
    ("notebooklm_access", "NotebookLM", "NotebookLM"),
)
ENDPOINTS = {
    "chatgpt_web": ("https://api.openai.com/compliance/cookie_requirements", {}),
    "chatgpt_app": ("https://ios.chat.openai.com", {}),
    "youtube_premium_access": ("https://www.youtube.com/premium", {"Accept-Language": "en-US,en;q=0.9"}),
    "claude_access": ("https://claude.ai/", CLAUDE_HEADERS),
    "tiktok_access": ("https://www.tiktok.com/", HTML_HEADERS),
    "gemini_access": ("https://gemini.google.com/app", HTML_HEADERS),
    "notebooklm_access": ("https://notebooklm.google.com/", HTML_HEADERS),
}
NETFLIX_BASE = "https://www.netflix.com"
LICENSED_TITLE = "70143836"
ORIGINAL_TITLE = "80197526"
CLAUDE_TRACE = "https://claude.ai/cdn-cgi/trace"


class ProbeError(Exception):
    """An inconclusive request, response, or parse failure (never a block)."""


@dataclass(frozen=True)
class Response:
    status: int
    body: str
    final_url: str


class LimitedRedirect(urllib.request.HTTPRedirectHandler):
    """Match Go's stop at the tenth redirect; reject non-HTTP(S) targets."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        count = getattr(req, "_redirect_count", 0) + 1
        if count >= 10:
            raise ProbeError("stopped after 10 redirects")
        if urllib.parse.urlsplit(newurl).scheme not in ("http", "https"):
            raise ProbeError("unsupported redirect scheme")
        redirected = super().redirect_request(req, fp, code, msg, headers, newurl)
        if redirected is not None:
            redirected._redirect_count = count
        return redirected


class HTTPClient:
    """Direct HTTP/1.1, system DNS/address selection, verified TLS, no cookies."""

    def __init__(self, timeout=6.0):
        self.timeout = timeout
        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}), LimitedRedirect(),
            urllib.request.HTTPSHandler(context=ssl.create_default_context()),
        )

    def get(self, url, headers=None):
        request_headers = {"User-Agent": UA, "Accept-Encoding": "gzip"}
        request_headers.update(headers or {})
        req = urllib.request.Request(url, headers=request_headers)
        try:
            response = self.opener.open(req, timeout=self.timeout)
        except urllib.error.HTTPError as exc:
            # Service-specific 403/404 wording must remain available to classifiers.
            # urllib redirect-loop errors are not meaningful final responses.
            if 300 <= exc.code < 400:
                exc.close()
                raise ProbeError("redirect failed") from exc
            response = exc
        with response:
            raw = response.read(MAX_BODY + 1)
            if len(raw) > MAX_BODY:
                raise ProbeError("response exceeds 4 MiB; not classified")
            length = response.headers.get("Content-Length")
            if length is not None:
                try:
                    expected = int(length)
                except ValueError as exc:
                    raise ProbeError("invalid Content-Length") from exc
                if expected < 0 or len(raw) != expected:
                    raise ProbeError("incomplete response body")
            encoding = response.headers.get("Content-Encoding", "identity").lower().strip()
            if encoding == "gzip":
                with gzip.GzipFile(fileobj=io.BytesIO(raw)) as compressed:
                    raw = compressed.read(MAX_BODY + 1)
                if len(raw) > MAX_BODY:
                    raise ProbeError("decoded response exceeds 4 MiB; not classified")
            elif encoding not in ("", "identity"):
                raise ProbeError("unsupported Content-Encoding: " + encoding)
            return Response(response.status, raw.decode("utf-8", errors="replace"), response.geturl())


def verdict(status, detail="", region=None):
    return {"status": status, "detail": detail, "region": region or None,
            "error": detail if status == "error" else None}


def is_challenge(status, lower_body):
    return status in (403, 429, 503) and any(m in lower_body for m in CHALLENGE_MARKERS)


def response_problem(response):
    """Error guards precede both positive and refusal markers."""
    if is_challenge(response.status, response.body.lower()):
        return "Cloudflare challenged the request, so availability was never tested"
    if response.status == 429 or response.status >= 500 or response.status < 200:
        return "unexpected HTTP " + str(response.status)
    return None


def valid_destination(url):
    try:
        parsed = urllib.parse.urlsplit(url)
        # Force validation of bracketed hosts and ports; no origin is invented.
        port = parsed.port
        if parsed.scheme not in ("http", "https") or not parsed.hostname or parsed.username is not None:
            raise ValueError("not an absolute HTTP(S) destination")
        return parsed
    except ValueError as exc:
        raise ProbeError("invalid or missing final URL") from exc


def classify(check_id, response, region=None):
    """Port of checks.go single-response classifiers; no I/O."""
    status, body, final = response.status, response.body, response.final_url
    problem = response_problem(response)
    if problem:
        return verdict("error", problem, region)
    lower = body.lower()
    if check_id == "notebooklm_access":
        try:
            destination = valid_destination(final)
        except ProbeError as exc:
            return verdict("error", str(exc))
        if "location=unsupported" in final:
            return verdict("blocked", "not offered in this country")
        if not 200 <= status < 400:
            return verdict("error", "unexpected HTTP " + str(status))
        host = destination.netloc
        if host == "accounts.google.com" or (host.endswith(".google.com") and "/login" in destination.path):
            return verdict("available", "sign-in required")
        return verdict("error", "unexpected destination " + host)

    if not body.strip():
        return verdict("error", "empty response; no availability evidence", region)
    if check_id == "youtube_premium_access":
        if "youtube premium is not available in your country" in lower:
            return verdict("blocked", "not offered in this country")
        if not 200 <= status < 400:
            return verdict("error", "unexpected HTTP " + str(status))
        if "ad-free" in lower:
            return verdict("available")
        return verdict("error", "neither the offer nor the refusal wording was present")
    if check_id == "chatgpt_web":
        if "unsupported_country" in lower:
            return verdict("blocked", "unsupported country")
        if not 200 <= status < 400:
            return verdict("error", "unexpected HTTP " + str(status))
        return verdict("available")
    if check_id == "chatgpt_app":
        if "disallowed isp" in lower:
            return verdict("blocked", "disallowed ISP; the exit address is classed as hosting or proxy space")
        if "been blocked" in lower:
            return verdict("blocked", "blocked")
        if not 200 <= status < 400:
            return verdict("error", "unexpected HTTP " + str(status))
        return verdict("available")
    if check_id == "claude_access":
        if any(m in lower for m in CLAUDE_MARKERS):
            return verdict("blocked", "not available in this region", region)
        if status == 403:
            return verdict("error", "HTTP 403 without the region page, so the cause is unknown", region)
        if 200 <= status < 400:
            return verdict("available", region=region)
        return verdict("error", "unexpected HTTP " + str(status), region)
    if check_id == "tiktok_access":
        if any(m in lower for m in TIKTOK_MARKERS):
            return verdict("blocked", "blocked by region")
        if 200 <= status < 400:
            return verdict("available")
        return verdict("error", "unexpected HTTP " + str(status))
    if check_id == "gemini_access":
        if not 200 <= status < 400:
            return verdict("error", "unexpected HTTP " + str(status))
        match = GEMINI_REGION.search(body)
        if match is None:
            return verdict("error", "could not read the served region from the page")
        region = match[1]
        if region in GEMINI_UNSUPPORTED:
            return verdict("blocked", "not offered in this country", region)
        return verdict("available", region=region)
    raise ValueError("unknown classifier: " + check_id)


def netflix_title(client, title):
    response = client.get(NETFLIX_BASE + "/title/" + title)
    problem = response_problem(response)
    if problem:
        raise ProbeError(problem)
    if response.status == 404:
        return None, False
    if not 200 <= response.status < 300:
        raise ProbeError("unexpected HTTP " + str(response.status))
    destination = valid_destination(response.final_url)
    # Unlike upstream's loose first-segment parsing, do not treat a login or
    # unrelated two-letter path as a successfully served title.
    if destination.hostname not in ("www.netflix.com", "netflix.com"):
        raise ProbeError("unexpected Netflix destination host")
    match = re.fullmatch(r"/(?:([A-Za-z]{2})(?:-[A-Za-z]+)?/)?title/" + title + r"/?", destination.path)
    if match is None:
        raise ProbeError("could not read the served Netflix title/region from the final URL")
    return (match[1] or "US").upper(), True


def check_netflix(client):
    region, served = netflix_title(client, LICENSED_TITLE)
    if served:
        return verdict("available", "full catalogue", region)
    region, served = netflix_title(client, ORIGINAL_TITLE)
    if served:
        return verdict("restricted", "originals only; licensed titles are withheld", region)
    return verdict("blocked", "not available")


def cloudflare_loc(client):
    # Best effort, exactly as upstream: absence/failure does not decide access.
    try:
        response = client.get(CLAUDE_TRACE, {"Accept": "text/plain,*/*;q=0.8"})
        if not 200 <= response.status < 300:
            return None
        match = re.search(r"loc=([A-Z]{2})", response.body)
        return match[1] if match else None
    except Exception:
        return None


def utc_now():
    return datetime.now(timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z")


def run_one(check, timeout, client_factory=HTTPClient):
    check_id, platform, name = check
    started, checked_at = time.monotonic(), utc_now()
    region = None
    try:
        client = client_factory(timeout)
        if check_id == "netflix_access":
            result = check_netflix(client)
        else:
            if check_id == "claude_access":
                region = cloudflare_loc(client)
            url, headers = ENDPOINTS[check_id]
            result = classify(check_id, client.get(url, headers), region)
    except Exception as exc:
        result = verdict("error", "request or response processing failed", region)
        result["error"] = type(exc).__name__ + ": " + str(exc)
    result.update(id=check_id, platform=platform, name=name, checked_at=checked_at,
                  elapsed_ms=round((time.monotonic() - started) * 1000, 3))
    return result


def run_all(timeout=6.0, workers=4, client_factory=HTTPClient):
    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        # map preserves upstream display order even when requests finish out of order.
        results = list(pool.map(lambda check: run_one(check, timeout, client_factory), CHECKS))
    return {"schema_version": 1, "source": SOURCE, "generated_at": utc_now(),
            "network": {"mode": "direct", "address_family": "system-selected",
                        "timeout_seconds": timeout, "timeout_scope": "socket operation",
                        "workers": workers}, "results": results}


def positive_timeout(value):
    try:
        result = float(value)
        if not math.isfinite(result) or result <= 0:
            raise ValueError
        return result
    except ValueError as exc:
        raise argparse.ArgumentTypeError("timeout must be a finite positive number") from exc


def worker_count(value):
    try:
        result = int(value)
        if not 1 <= result <= 8:
            raise ValueError
        return result
    except ValueError as exc:
        raise argparse.ArgumentTypeError("workers must be between 1 and 8") from exc


def print_table(report):
    print("7个平台 / 8项检查 · geocheck v0.3.0 Python移植改进版")
    print("直连，系统选择地址族；error = 未知，不代表未解锁。")
    print("服务               状态         地区   耗时(ms)  详情 / 错误")
    for result in report["results"]:
        detail = result["detail"] or "-"
        if result["error"] and result["error"] != result["detail"]:
            detail += " | " + result["error"]
        # Escape controls from remote response-derived diagnostics in a terminal.
        detail = "".join(c if c.isprintable() else " " for c in detail)
        print("{:<18} {:<12} {:<5} {:>9.1f}  {}".format(
            result["name"], result["status"], result["region"] or "-", result["elapsed_ms"], detail))


def main(argv=None, client_factory=HTTPClient):
    parser = argparse.ArgumentParser(description="7个平台、8项access检查；执行检测会访问真实平台，--help不联网。")
    parser.add_argument("--json", action="store_true", help="只输出JSON（error仍为未知）")
    parser.add_argument("--timeout", type=positive_timeout, default=6.0,
                        help="单次socket操作超时秒数，默认6；不是全程硬截止")
    parser.add_argument("--workers", type=worker_count, default=4, help="并发检查数1–8，默认4")
    args = parser.parse_args(argv)
    report = run_all(args.timeout, args.workers, client_factory)
    if args.json:
        print(json.dumps(report, ensure_ascii=False, indent=2, allow_nan=False))
    else:
        print_table(report)
    return 1 if any(r["status"] == "error" for r in report["results"]) else 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("检测已中断", file=sys.stderr)
        sys.exit(130)
