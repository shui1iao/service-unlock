"""Offline tests: in-memory fixtures and loopback HTTP only; no platform requests.

Decision cases adapted from remnawave/geocheck v0.3.0, MIT,
Copyright (c) 2026 Remnawave. See LICENSE and service_unlock.py.
Run: python3 -m unittest -v test_service_unlock.py
"""
import contextlib
import gzip
import io
import json
import socket
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from unittest.mock import patch

import service_unlock as s


REAL_GETADDRINFO = socket.getaddrinfo


def loopback_only(host, *args, **kwargs):
    if host not in ("127.0.0.1", "::1", "localhost"):
        raise AssertionError("offline test attempted external DNS/network: " + str(host))
    return REAL_GETADDRINFO(host, *args, **kwargs)


def setUpModule():
    global DNS_GUARD
    DNS_GUARD = patch("socket.getaddrinfo", side_effect=loopback_only)
    DNS_GUARD.start()


def tearDownModule():
    DNS_GUARD.stop()


def resp(status=200, body="ok", final="https://example.com/"):
    return s.Response(status, body, final)


class FakeClient:
    def __init__(self, timeout=6):
        self.timeout = timeout

    def get(self, url, headers=None):
        if url == s.CLAUDE_TRACE:
            return resp(body="fl=1\nloc=FR\n", final=url)
        if url.startswith(s.NETFLIX_BASE + "/title/"):
            return resp(final=url)
        bodies = {
            "chatgpt_web": '{"is_cookie_required":false}',
            "chatgpt_app": "app landing page",
            "youtube_premium_access": "Enjoy AD-FREE viewing",
            "claude_access": '<html><div id="root"></div></html>',
            "tiktok_access": "<html>TikTok</html>",
            "gemini_access": ',3,7,200,"FRA"',
            "notebooklm_access": "",
        }
        for check_id, (endpoint, _) in s.ENDPOINTS.items():
            if url == endpoint:
                final = "https://accounts.google.com/v3/signin/identifier" if check_id == "notebooklm_access" else url
                return resp(body=bodies[check_id], final=final)
        raise AssertionError("unexpected test URL: " + url)


class ClassificationTests(unittest.TestCase):
    def assertState(self, check, response, expected, region=None):
        result = s.classify(check, response, region)
        self.assertEqual(result["status"], expected, result)
        if expected == "error":
            self.assertTrue(result["error"])
        else:
            self.assertIsNone(result["error"])
        return result

    def test_youtube_positive_refusal_and_missing_wording(self):
        for body, expected in (("AD-FREE", "available"),
                               ("YouTube Premium is not available in your country", "blocked"),
                               ("consent page", "error")):
            with self.subTest(body=body):
                self.assertState("youtube_premium_access", resp(body=body), expected)
        self.assertState("youtube_premium_access", resp(403, "ad-free"), "error")

    def test_chatgpt_web(self):
        self.assertState("chatgpt_web", resp(body="{}"), "available")
        self.assertState("chatgpt_web", resp(403, '{"error":"UNSUPPORTED_COUNTRY"}'), "blocked")
        for status in (401, 403, 404):
            self.assertState("chatgpt_web", resp(status, "unknown failure"), "error")

    def test_chatgpt_app(self):
        self.assertState("chatgpt_app", resp(), "available")
        for marker in ("Disallowed ISP", "you have been blocked"):
            self.assertState("chatgpt_app", resp(403, marker), "blocked")
        self.assertState("chatgpt_app", resp(403, "Forbidden"), "error")
        self.assertState("chatgpt_app", resp(403, "just a moment; disallowed isp"), "error")

    def test_claude_upstream_cases(self):
        cases = ((403, "_cf_chl_opt /app-unavailable-in-region", "error"),
                 (403, "Forbidden", "error"), (502, "Bad Gateway", "error"),
                 (200, '<html><div id="root"></div></html>', "available"))
        for status, body, expected in cases:
            result = self.assertState("claude_access", resp(status, body), expected, "FR")
            self.assertEqual(result["region"], "FR")
        for marker in s.CLAUDE_MARKERS:
            for status in (200, 403):
                with self.subTest(marker=marker, status=status):
                    self.assertState("claude_access", resp(status, marker.upper()), "blocked")

    def test_challenge_needs_refusal_status(self):
        for marker in s.CHALLENGE_MARKERS:
            self.assertFalse(s.is_challenge(200, marker))
            for status in (403, 429, 503):
                self.assertTrue(s.is_challenge(status, marker))

    def test_tiktok_all_refusal_markers(self):
        self.assertState("tiktok_access", resp(), "available")
        for marker in s.TIKTOK_MARKERS:
            for status in (200, 403):
                self.assertState("tiktok_access", resp(status, marker.upper()), "blocked")
        self.assertState("tiktok_access", resp(403, "Forbidden"), "error")

    def test_gemini_region_pattern_and_blocklist(self):
        result = self.assertState("gemini_access", resp(body=',3,7,200,"USA"'), "available")
        self.assertEqual(result["region"], "USA")
        self.assertEqual(s.GEMINI_UNSUPPORTED, {"RUS", "BLR", "CHN", "PRK", "IRN", "CUB", "SYR"})
        for code in s.GEMINI_UNSUPPORTED:
            result = self.assertState("gemini_access", resp(body=',2,1,200,"' + code + '"'), "blocked")
            self.assertEqual(result["region"], code)

    def test_gemini_missing_or_malformed_region_is_error(self):
        for body in ("", "<html></html>", '{"error":"nope"}', ',3,7,200,"usa"', ',3,7,200,"US"'):
            self.assertState("gemini_access", resp(body=body), "error")

    def test_notebooklm_upstream_destinations(self):
        for final in ("https://accounts.google.com/v3/signin/identifier",
                      "https://notebook.google.com/login?continue=https://notebook.google.com/"):
            result = self.assertState("notebooklm_access", resp(body="", final=final), "available")
            self.assertEqual(result["detail"], "sign-in required")
            self.assertIsNone(result["region"])
        for final in ("https://notebook.google/?location=unsupported",
                      "https://notebook.google.com/login?location=unsupported",
                      "https://accounts.google.com/signin?location=unsupported"):
            self.assertState("notebooklm_access", resp(body="", final=final), "blocked")

    def test_notebooklm_parse_and_unrecognised_destination(self):
        for final in ("", "https://[", "/relative", "https://example.com/", "https://accounts.google.com.evil.test/login"):
            self.assertState("notebooklm_access", resp(final=final), "error")
        self.assertState("notebooklm_access", resp(403, "Forbidden", "https://accounts.google.com/login"), "error")

    def test_server_errors_and_rate_limit_override_all_markers(self):
        # Even positive/refusal markers echoed in an error page are inconclusive.
        body = ('ad-free unsupported_country disallowed isp been blocked '
                'app unavailable in region not available in your region ,3,7,200,"CHN"')
        for check_id in s.ENDPOINTS:
            for status in (429, 500, 502, 503, 599):
                for final in ("https://accounts.google.com/login", "https://notebook.google/?location=unsupported"):
                    with self.subTest(check=check_id, status=status, final=final):
                        self.assertState(check_id, resp(status, body, final), "error")

    def test_empty_text_response_is_not_available(self):
        for check_id in s.ENDPOINTS:
            if check_id != "notebooklm_access":
                for body in ("", " \n\t"):
                    self.assertState(check_id, resp(body=body), "error")

    def test_challenge_echo_cannot_claim_block(self):
        for check_id in s.ENDPOINTS:
            body = ('cf_chl_opt unsupported_country disallowed isp '
                    'app unavailable in region not available in your region '
                    'youtube premium is not available in your country')
            self.assertState(check_id, resp(403, body, "https://notebook.google/?location=unsupported"), "error")


class NetflixTests(unittest.TestCase):
    def client(self, replies):
        class SequenceClient:
            def __init__(self):
                self.urls = []
                self.replies = iter(replies)

            def get(self, url, headers=None):
                self.urls.append(url)
                value = next(self.replies)
                if isinstance(value, Exception):
                    raise value
                return value
        return SequenceClient()

    def test_full_catalogue_region_and_short_circuit(self):
        for path, region in (("/fr-en/title/70143836", "FR"), ("/title/70143836", "US"), ("/jp/title/70143836", "JP")):
            client = self.client([resp(final=s.NETFLIX_BASE + path)])
            result = s.check_netflix(client)
            self.assertEqual((result["status"], result["region"]), ("available", region))
            self.assertEqual(client.urls, [s.NETFLIX_BASE + "/title/70143836"])

    def test_originals_only(self):
        client = self.client([resp(404), resp(final=s.NETFLIX_BASE + "/fr-en/title/80197526")])
        result = s.check_netflix(client)
        self.assertEqual((result["status"], result["region"]), ("restricted", "FR"))
        self.assertEqual(len(client.urls), 2)

    def test_two_404s_are_blocked(self):
        self.assertEqual(s.check_netflix(self.client([resp(404), resp(404)]))["status"], "blocked")

    def test_http_error_cannot_be_a_withheld_title(self):
        for status in (403, 429, 500, 502, 503):
            for replies in ([resp(status)], [resp(404), resp(status)]):
                with self.subTest(status=status, replies=replies):
                    with self.assertRaises(s.ProbeError):
                        s.check_netflix(self.client(replies))

    def test_invalid_title_destination_is_error(self):
        for final in ("", "https://[", s.NETFLIX_BASE + "/login", s.NETFLIX_BASE + "/fr-en/login",
                      s.NETFLIX_BASE + "/ab/anything", s.NETFLIX_BASE + "/title/123",
                      "https://example.com/title/70143836"):
            with self.subTest(final=final), self.assertRaises(s.ProbeError):
                s.check_netflix(self.client([resp(final=final)]))


class RunnerTests(unittest.TestCase):
    def test_exact_scope_order_and_fields(self):
        report = s.run_all(client_factory=FakeClient)
        self.assertEqual([r["id"] for r in report["results"]], [c[0] for c in s.CHECKS])
        self.assertEqual(len(report["results"]), 8)
        self.assertEqual({r["platform"] for r in report["results"]},
                         {"Netflix", "YouTube Premium", "TikTok", "ChatGPT", "Gemini", "NotebookLM", "Claude"})
        for result in report["results"]:
            self.assertEqual(result["status"], "available")
            self.assertGreaterEqual(result["elapsed_ms"], 0)
            self.assertTrue(result["checked_at"].endswith("Z"))
            self.assertTrue({"id", "name", "platform", "status", "detail", "region", "error", "elapsed_ms", "checked_at"} <= result.keys())
        self.assertEqual(next(r for r in report["results"] if r["id"] == "claude_access")["region"], "FR")

    def test_timeouts_network_and_parse_failures_are_errors(self):
        for exception in (TimeoutError("timed out"), OSError("network unreachable"), ValueError("bad response")):
            class FailedClient(FakeClient):
                def get(self, url, headers=None):
                    raise exception
            report = s.run_all(client_factory=FailedClient)
            self.assertEqual({r["status"] for r in report["results"]}, {"error"})
            self.assertTrue(all(type(exception).__name__ in r["error"] for r in report["results"]))

    def test_claude_trace_is_best_effort(self):
        class NoTrace(FakeClient):
            def get(self, url, headers=None):
                if url == s.CLAUDE_TRACE:
                    raise TimeoutError("trace timeout")
                return super().get(url, headers)
        result = s.run_one(s.CHECKS[4], 6, NoTrace)
        self.assertEqual(result["status"], "available")
        self.assertIsNone(result["region"])

    def test_claude_keeps_region_on_main_request_failure(self):
        class MainFailure(FakeClient):
            def get(self, url, headers=None):
                if url == s.ENDPOINTS["claude_access"][0]:
                    raise TimeoutError("main request timeout")
                return super().get(url, headers)
        result = s.run_one(s.CHECKS[4], 6, MainFailure)
        self.assertEqual((result["status"], result["region"]), ("error", "FR"))

    def test_json_cli_and_error_exit_status(self):
        for factory, expected_exit in ((FakeClient, 0), (lambda timeout: None, 1)):
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                code = s.main(["--json", "--workers", "2", "--timeout", "0.1"], factory)
            self.assertEqual(code, expected_exit)
            report = json.loads(output.getvalue())
            self.assertEqual(report["schema_version"], 1)
            self.assertEqual(len(report["results"]), 8)
            self.assertEqual(report["network"]["workers"], 2)
            self.assertNotIn("status", report)  # No invented platform-wide aggregate.

    def test_human_cli(self):
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(s.main([], FakeClient), 0)
        text = output.getvalue()
        self.assertIn("7个平台 / 8项检查", text)
        self.assertIn("ChatGPT (web)", text)
        self.assertIn("ChatGPT (app)", text)
        self.assertIn("error = 未知", text)

    def test_help_and_invalid_args_do_not_construct_network_client(self):
        def forbidden(timeout):
            raise AssertionError("CLI help/validation attempted network")
        for args, expected in ((["--help"], 0), (["--workers", "9"], 2),
                               (["--workers", "0"], 2), (["--timeout", "nan"], 2),
                               (["--timeout", "inf"], 2), (["--timeout", "0"], 2),
                               (["--timeout", "-1"], 2)):
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit) as caught:
                    s.main(args, forbidden)
            self.assertEqual(caught.exception.code, expected)

    def test_concurrency_is_bounded(self):
        lock = threading.Lock()
        active = 0
        peak = 0

        class SlowClient(FakeClient):
            def get(self, url, headers=None):
                nonlocal active, peak
                with lock:
                    active += 1
                    peak = max(peak, active)
                try:
                    time.sleep(0.01)
                    return super().get(url, headers)
                finally:
                    with lock:
                        active -= 1
        s.run_all(workers=2, client_factory=SlowClient)
        self.assertEqual(peak, 2)
        self.assertEqual(active, 0)


class LocalHandler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_GET(self):
        status, body, headers = 200, b"ok", {}
        if self.path == "/headers":
            body = json.dumps(dict(self.headers)).encode()
        elif self.path == "/redirect":
            status, headers = 302, {"Location": "/final"}
        elif self.path.startswith("/hop/"):
            hop = int(self.path.rsplit("/", 1)[1])
            status, headers = 302, {"Location": "/hop/" + str(hop + 1)}
        elif self.path == "/loop":
            status, headers = 302, {"Location": "/loop"}
        elif self.path == "/forbidden":
            status, body = 403, b"unsupported_country"
        elif self.path == "/server-error":
            status, body = 500, b"ad-free"
        elif self.path == "/gzip":
            body, headers = gzip.compress(b"ad-free"), {"Content-Encoding": "gzip"}
        elif self.path == "/bad-gzip":
            body, headers = b"not gzip", {"Content-Encoding": "gzip"}
        elif self.path == "/gzip-bomb":
            body, headers = gzip.compress(b"x" * 500), {"Content-Encoding": "gzip"}
        elif self.path == "/large":
            body = b"x" * 500
        elif self.path == "/encoding":
            headers = {"Content-Encoding": "br"}
        elif self.path == "/truncated":
            headers = {"Content-Length": "100"}
        elif self.path == "/slow":
            time.sleep(0.15)
        self.send_response(status)
        for name, value in headers.items():
            self.send_header(name, value)
        if "Content-Length" not in headers:
            self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        try:
            self.wfile.write(body)
        except (BrokenPipeError, ConnectionResetError):
            pass


class TransportTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), LocalHandler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base = "http://127.0.0.1:" + str(cls.server.server_port)

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def test_redirect_final_url_and_headers(self):
        client = s.HTTPClient(1)
        response = client.get(self.base + "/redirect")
        self.assertEqual(response.final_url, self.base + "/final")
        self.assertEqual(response.status, 200)
        headers = json.loads(client.get(self.base + "/headers", {"Accept-Language": "en-US,en;q=0.9"}).body)
        self.assertEqual(headers["User-Agent"], s.UA)
        self.assertEqual(headers["Accept-Encoding"], "gzip")
        self.assertEqual(headers["Accept-Language"], "en-US,en;q=0.9")

    def test_http_error_body_is_preserved_for_service_semantics(self):
        client = s.HTTPClient(1)
        response = client.get(self.base + "/forbidden")
        self.assertEqual(response.status, 403)
        self.assertEqual(s.classify("chatgpt_web", response)["status"], "blocked")
        response = client.get(self.base + "/server-error")
        self.assertEqual(s.classify("youtube_premium_access", response)["status"], "error")

    def test_gzip(self):
        self.assertEqual(s.HTTPClient(1).get(self.base + "/gzip").body, "ad-free")

    def test_bad_gzip_and_unknown_encoding_are_errors(self):
        for path in ("/bad-gzip", "/encoding"):
            with self.subTest(path=path), self.assertRaises((s.ProbeError, OSError, EOFError)):
                s.HTTPClient(1).get(self.base + path)

    def test_body_limits_include_decompressed_size(self):
        for path in ("/large", "/gzip-bomb"):
            with self.subTest(path=path), patch.object(s, "MAX_BODY", 100), self.assertRaises(s.ProbeError):
                s.HTTPClient(1).get(self.base + path)

    def test_truncated_body_is_not_classified(self):
        with self.assertRaises(s.ProbeError):
            s.HTTPClient(1).get(self.base + "/truncated")

    def test_redirect_limits_and_loop(self):
        for path in ("/hop/0", "/loop"):
            with self.subTest(path=path), self.assertRaises(s.ProbeError):
                s.HTTPClient(1).get(self.base + path)

    def test_socket_timeout_is_error(self):
        with self.assertRaises((TimeoutError, OSError)):
            s.HTTPClient(0.03).get(self.base + "/slow")

    def test_proxy_environment_is_ignored(self):
        with patch.dict("os.environ", {"HTTP_PROXY": "http://127.0.0.1:1", "http_proxy": "http://127.0.0.1:1", "NO_PROXY": "", "no_proxy": ""}):
            self.assertEqual(s.HTTPClient(1).get(self.base + "/final").status, 200)


if __name__ == "__main__":
    unittest.main()
