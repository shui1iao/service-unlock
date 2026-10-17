#!/usr/bin/env node
'use strict';
/*
 * Service Unlock v0.2.0 — standalone Node.js port, seven platforms / eight checks.
 * Derived from remnawave/geocheck v0.3.0, commit
 * 50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc, and the v0.1.0 Python port.
 * https://github.com/remnawave/geocheck
 * Conservative error handling and transport differences: see README.md.
 *
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
*/

const http = require('node:http');
const https = require('node:https');
const { createGunzip } = require('node:zlib');
const { performance } = require('node:perf_hooks');

const SOURCE = Object.freeze({
  repository: 'https://github.com/remnawave/geocheck',
  tag: 'v0.3.0',
  commit: '50e084bb3ed34b55fc6839fe0dc4bafd9fe275fc',
  port: 'Node.js v0.2.0 standard-library port with conservative error handling',
});
const UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 '
  + '(KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36';
const MAX_BODY = 4 << 20;
const MAX_TIMEOUT = 2147483.647; // Node's maximum setTimeout delay, in seconds.
const HTML_HEADERS = {
  Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
  'Accept-Language': 'en-US,en;q=0.9',
};
const CLAUDE_HEADERS = {
  ...HTML_HEADERS,
  Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8',
  'Sec-Fetch-Dest': 'document', 'Sec-Fetch-Mode': 'navigate',
  'Sec-Fetch-Site': 'none', 'Sec-Fetch-User': '?1', 'Upgrade-Insecure-Requests': '1',
};
const CHALLENGE_MARKERS = [
  'cf_chl_opt', '_cf_chl', 'just a moment', 'cf-browser-verification',
  'enable javascript and cookies to continue',
];
const CLAUDE_MARKERS = [
  'app unavailable in region', '/app-unavailable-in-region',
  "unfortunately, claude isn't available here.",
  'unfortunately, claude isn&apos;t available here.',
  'unfortunately, claude isn&#39;t available here.',
];
const TIKTOK_MARKERS = [
  'service is currently unavailable in your region',
  'tiktok is not available in your country', 'tiktok is unavailable in your country',
  'not available in your region',
];
const GEMINI_UNSUPPORTED = new Set(['RUS', 'BLR', 'CHN', 'PRK', 'IRN', 'CUB', 'SYR']);
const GEMINI_REGION = /,[0-9]+,[0-9]+,200,"([A-Z]{3})"/;
const CHECKS = [
  ['chatgpt_web', 'ChatGPT', 'ChatGPT (web)'],
  ['chatgpt_app', 'ChatGPT', 'ChatGPT (app)'],
  ['youtube_premium_access', 'YouTube Premium', 'YouTube Premium'],
  ['netflix_access', 'Netflix', 'Netflix'],
  ['claude_access', 'Claude', 'Claude'],
  ['tiktok_access', 'TikTok', 'TikTok'],
  ['gemini_access', 'Gemini', 'Gemini'],
  ['notebooklm_access', 'NotebookLM', 'NotebookLM'],
];
const ENDPOINTS = {
  chatgpt_web: ['https://api.openai.com/compliance/cookie_requirements', {}],
  chatgpt_app: ['https://ios.chat.openai.com', {}],
  youtube_premium_access: ['https://www.youtube.com/premium', { 'Accept-Language': 'en-US,en;q=0.9' }],
  claude_access: ['https://claude.ai/', CLAUDE_HEADERS],
  tiktok_access: ['https://www.tiktok.com/', HTML_HEADERS],
  gemini_access: ['https://gemini.google.com/app', HTML_HEADERS],
  notebooklm_access: ['https://notebooklm.google.com/', HTML_HEADERS],
};
const NETFLIX_BASE = 'https://www.netflix.com';
const LICENSED_TITLE = '70143836';
const ORIGINAL_TITLE = '80197526';
const CLAUDE_TRACE = 'https://claude.ai/cdn-cgi/trace';

class ProbeError extends Error {
  constructor(message) { super(message); this.name = 'ProbeError'; }
}

function validateTimeout(timeout) {
  if (!Number.isFinite(timeout) || timeout <= 0 || timeout > MAX_TIMEOUT) {
    throw new ProbeError(`timeout must be > 0 and <= ${MAX_TIMEOUT} seconds`);
  }
}

function validDestination(value) {
  try {
    // Do not let WHATWG URL turn relative, credential-bearing, or malformed
    // absolute destinations into evidence of availability.
    const authority = typeof value === 'string' && value.match(/^https?:\/\/([^/?#]+)/i);
    if (!authority || authority[1].includes('@') || /[\s\\]/u.test(value)) throw new Error();
    const url = new URL(value);
    if (!['http:', 'https:'].includes(url.protocol) || !url.hostname) throw new Error();
    // Preserve raw authority/path for Python's netloc/path classifier semantics
    // (notably explicit ports and case); URL is used to validate and to connect.
    const tail = value.slice(authority[0].length);
    const path = tail.startsWith('/') ? tail.split(/[?#]/, 1)[0] : '';
    return { url, hostname: url.hostname, host: authority[1], path };
  } catch {
    throw new ProbeError('invalid or missing final URL');
  }
}

class HTTPClient {
  constructor(timeout = 6, signal = undefined) {
    validateTimeout(timeout);
    this.timeout = timeout;
    this.signal = signal;
  }

  async get(url, headers = {}) {
    const controller = new AbortController();
    const abort = () => controller.abort(this.signal.reason || new ProbeError('request aborted'));
    if (this.signal?.aborted) abort();
    else this.signal?.addEventListener('abort', abort, { once: true });
    const timer = setTimeout(() => controller.abort(new ProbeError('request timed out')),
      Math.max(1, Math.ceil(this.timeout * 1000)));
    try {
      let current = url;
      let redirects = 0;
      while (true) {
        controller.signal.throwIfAborted();
        const destination = validDestination(current);
        const response = await this.requestOnce(destination.url, headers, controller.signal);
        if (response.status >= 300 && response.status < 400) {
          if (![301, 302, 303, 307, 308].includes(response.status) || !response.location) {
            throw new ProbeError('redirect failed');
          }
          redirects += 1;
          if (redirects >= 10) throw new ProbeError('stopped after 10 redirects');
          current = new URL(response.location, destination.url).href;
          continue;
        }
        return { status: response.status, body: response.body, final_url: destination.url.href };
      }
    } finally {
      clearTimeout(timer);
      this.signal?.removeEventListener('abort', abort);
    }
  }

  requestOnce(url, headers, signal) {
    return new Promise((resolve, reject) => {
      let request;
      let response;
      let decoder;
      let settled = false;
      let rawBytes = 0;
      let decodedBytes = 0;
      let expectedLength = null;
      const chunks = [];
      const cleanup = () => signal.removeEventListener('abort', onAbort);
      const fail = (error) => {
        if (settled) return;
        settled = true;
        cleanup();
        if (decoder && decoder !== response) decoder.destroy();
        response?.destroy();
        request?.destroy();
        reject(error);
      };
      const onAbort = () => fail(signal.reason || new ProbeError('request aborted'));
      if (signal.aborted) { onAbort(); return; }
      signal.addEventListener('abort', onAbort, { once: true });
      try {
        const transport = url.protocol === 'https:' ? https : http;
        request = transport.request(url, {
          method: 'GET', agent: false, rejectUnauthorized: true,
          headers: { 'User-Agent': UA, 'Accept-Encoding': 'gzip', ...headers },
        }, (incoming) => {
          if (settled) { incoming.destroy(); return; }
          response = incoming;
          response.on('error', fail);
          response.on('aborted', () => fail(new ProbeError('incomplete response body')));
          const length = response.headers['content-length'];
          if (length !== undefined) {
            if (!/^[0-9]+$/.test(length) || !Number.isSafeInteger(Number(length))) {
              fail(new ProbeError('invalid Content-Length')); return;
            }
            expectedLength = Number(length);
            if (expectedLength > MAX_BODY) {
              fail(new ProbeError('response exceeds 4 MiB; not classified')); return;
            }
          }
          const encoding = (response.headers['content-encoding'] || 'identity').trim().toLowerCase();
          if (!['', 'identity', 'gzip'].includes(encoding)) {
            fail(new ProbeError(`unsupported Content-Encoding: ${encoding}`)); return;
          }
          decoder = encoding === 'gzip' ? createGunzip() : response;
          decoder.on('error', fail);
          response.on('data', (chunk) => {
            rawBytes += chunk.length;
            if (rawBytes > MAX_BODY) fail(new ProbeError('response exceeds 4 MiB; not classified'));
          });
          decoder.on('data', (chunk) => {
            if (settled) return;
            decodedBytes += chunk.length;
            if (decodedBytes > MAX_BODY) {
              fail(new ProbeError('decoded response exceeds 4 MiB; not classified')); return;
            }
            chunks.push(chunk);
          });
          decoder.on('end', () => {
            if (settled) return;
            if (signal.aborted) { onAbort(); return; }
            if (!response.complete || (expectedLength !== null && rawBytes !== expectedLength)) {
              fail(new ProbeError('incomplete response body')); return;
            }
            settled = true;
            cleanup();
            resolve({ status: response.statusCode, body: Buffer.concat(chunks).toString('utf8'),
              location: response.headers.location });
          });
          if (decoder !== response) response.pipe(decoder);
        });
        request.on('error', fail);
        request.end();
      } catch (error) { fail(error); }
    });
  }
}

function verdict(status, detail = '', region = null) {
  return { status, detail, region: region || null, error: status === 'error' ? detail : null };
}
function isChallenge(status, lowerBody) {
  return [403, 429, 503].includes(status) && CHALLENGE_MARKERS.some((marker) => lowerBody.includes(marker));
}
function responseProblem(response) {
  if (isChallenge(response.status, response.body.toLowerCase())) {
    return 'Cloudflare challenged the request, so availability was never tested';
  }
  if (response.status === 429 || response.status >= 500 || response.status < 200) {
    return `unexpected HTTP ${response.status}`;
  }
  return null;
}

function classify(checkId, response, region = null) {
  const { status, body, final_url: final } = response;
  const problem = responseProblem(response);
  if (problem) return verdict('error', problem, region);
  const lower = body.toLowerCase();
  if (checkId === 'notebooklm_access') {
    let destination;
    try { destination = validDestination(final); }
    catch (error) { return verdict('error', error.message); }
    if (final.includes('location=unsupported')) return verdict('blocked', 'not offered in this country');
    if (!(status >= 200 && status < 400)) return verdict('error', `unexpected HTTP ${status}`);
    const { host, path } = destination;
    if (host === 'accounts.google.com' || (host.endsWith('.google.com') && path.includes('/login'))) {
      return verdict('available', 'sign-in required');
    }
    return verdict('error', `unexpected destination ${host}`);
  }
  if (!body.trim()) return verdict('error', 'empty response; no availability evidence', region);
  if (checkId === 'youtube_premium_access') {
    if (lower.includes('youtube premium is not available in your country')) {
      return verdict('blocked', 'not offered in this country');
    }
    if (!(status >= 200 && status < 400)) return verdict('error', `unexpected HTTP ${status}`);
    if (lower.includes('ad-free')) return verdict('available');
    return verdict('error', 'neither the offer nor the refusal wording was present');
  }
  if (checkId === 'chatgpt_web') {
    if (lower.includes('unsupported_country')) return verdict('blocked', 'unsupported country');
    if (!(status >= 200 && status < 400)) return verdict('error', `unexpected HTTP ${status}`);
    return verdict('available');
  }
  if (checkId === 'chatgpt_app') {
    if (lower.includes('disallowed isp')) {
      return verdict('blocked', 'disallowed ISP; the exit address is classed as hosting or proxy space');
    }
    if (lower.includes('been blocked')) return verdict('blocked', 'blocked');
    if (!(status >= 200 && status < 400)) return verdict('error', `unexpected HTTP ${status}`);
    return verdict('available');
  }
  if (checkId === 'claude_access') {
    if (CLAUDE_MARKERS.some((marker) => lower.includes(marker))) {
      return verdict('blocked', 'not available in this region', region);
    }
    if (status === 403) {
      return verdict('error', 'HTTP 403 without the region page, so the cause is unknown', region);
    }
    if (status >= 200 && status < 400) return verdict('available', '', region);
    return verdict('error', `unexpected HTTP ${status}`, region);
  }
  if (checkId === 'tiktok_access') {
    if (TIKTOK_MARKERS.some((marker) => lower.includes(marker))) return verdict('blocked', 'blocked by region');
    if (status >= 200 && status < 400) return verdict('available');
    return verdict('error', `unexpected HTTP ${status}`);
  }
  if (checkId === 'gemini_access') {
    if (!(status >= 200 && status < 400)) return verdict('error', `unexpected HTTP ${status}`);
    const match = body.match(GEMINI_REGION);
    if (!match) return verdict('error', 'could not read the served region from the page');
    region = match[1];
    if (GEMINI_UNSUPPORTED.has(region)) return verdict('blocked', 'not offered in this country', region);
    return verdict('available', '', region);
  }
  throw new ProbeError(`unknown classifier: ${checkId}`);
}

async function netflixTitle(client, title) {
  const response = await client.get(`${NETFLIX_BASE}/title/${title}`);
  const problem = responseProblem(response);
  if (problem) throw new ProbeError(problem);
  if (response.status === 404) return [null, false];
  if (!(response.status >= 200 && response.status < 300)) throw new ProbeError(`unexpected HTTP ${response.status}`);
  const destination = validDestination(response.final_url);
  if (!['www.netflix.com', 'netflix.com'].includes(destination.hostname)) {
    throw new ProbeError('unexpected Netflix destination host');
  }
  const match = destination.path.match(new RegExp(`^/(?:([A-Za-z]{2})(?:-[A-Za-z]+)?/)?title/${title}/?$`));
  if (!match) throw new ProbeError('could not read the served Netflix title/region from the final URL');
  return [(match[1] || 'US').toUpperCase(), true];
}
async function checkNetflix(client) {
  let [region, served] = await netflixTitle(client, LICENSED_TITLE);
  if (served) return verdict('available', 'full catalogue', region);
  [region, served] = await netflixTitle(client, ORIGINAL_TITLE);
  if (served) return verdict('restricted', 'originals only; licensed titles are withheld', region);
  return verdict('blocked', 'not available');
}
async function cloudflareLoc(client) {
  try {
    const response = await client.get(CLAUDE_TRACE, { Accept: 'text/plain,*/*;q=0.8' });
    if (!(response.status >= 200 && response.status < 300)) return null;
    return response.body.match(/loc=([A-Z]{2})/)?.[1] || null;
  } catch { return null; } // Best effort; this does not determine service availability.
}

async function runOne(check, timeout, clientFactory = (t, signal) => new HTTPClient(t, signal), signal) {
  const [id, platform, name] = check;
  const started = performance.now();
  const checkedAt = new Date().toISOString();
  let region = null;
  let result;
  try {
    signal?.throwIfAborted();
    const client = clientFactory(timeout, signal);
    if (id === 'netflix_access') result = await checkNetflix(client);
    else {
      if (id === 'claude_access') region = await cloudflareLoc(client);
      const [url, headers] = ENDPOINTS[id];
      result = classify(id, await client.get(url, headers), region);
    }
  } catch (error) {
    signal?.throwIfAborted(); // User cancellation is not a completed report.
    result = verdict('error', 'request or response processing failed', region);
    result.error = `${error.name || 'Error'}: ${error.message || String(error)}`;
  }
  return { ...result, id, platform, name, checked_at: checkedAt,
    elapsed_ms: Math.round((performance.now() - started) * 1000) / 1000 };
}

async function runAll({ timeout = 6, workers = 4, clientFactory, signal } = {}) {
  validateTimeout(timeout);
  if (!Number.isInteger(workers) || workers < 1 || workers > 8) throw new ProbeError('workers must be between 1 and 8');
  const results = new Array(CHECKS.length);
  let next = 0;
  async function worker() {
    while (next < CHECKS.length) {
      signal?.throwIfAborted();
      const index = next++;
      results[index] = await runOne(CHECKS[index], timeout, clientFactory, signal);
    }
  }
  await Promise.all(Array.from({ length: workers }, worker));
  signal?.throwIfAborted();
  return { schema_version: 1, source: SOURCE, generated_at: new Date().toISOString(),
    network: { mode: 'direct', address_family: 'system-selected', timeout_seconds: timeout,
      timeout_scope: 'request including redirects and body', workers }, results };
}

const HELP = `Usage: node service_unlock.js [--json] [--timeout SECONDS] [--workers N]

7个平台、8项access检查；执行检测会访问真实平台，--help不联网。
Node.js 22+；无需 npm 依赖。

  --json          只输出 JSON（error = 未知，不代表未解锁）
  --timeout N     每次请求总预算秒数，含重定向及正文，默认6
  --workers N     并发检查数1–8，默认4
  --help, -h      显示帮助，不联网
`;

function parseArgs(argv) {
  if (argv.includes('--help') || argv.includes('-h')) return { help: true };
  const options = { json: false, timeout: 6, workers: 4 };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--json') { options.json = true; continue; }
    const match = arg.match(/^(--timeout|--workers)(?:=(.*))?$/);
    if (!match) throw new ProbeError(`unknown argument: ${arg}`);
    const value = match[2] === undefined ? argv[++i] : match[2];
    if (value === undefined || !/^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$/.test(value)) {
      throw new ProbeError(`invalid value for ${match[1]}`);
    }
    if (match[1] === '--timeout') {
      options.timeout = Number(value);
      validateTimeout(options.timeout);
    } else {
      if (!/^[+]?[0-9]+$/.test(value)) throw new ProbeError('workers must be between 1 and 8');
      options.workers = Number(value);
      if (!Number.isInteger(options.workers) || options.workers < 1 || options.workers > 8) {
        throw new ProbeError('workers must be between 1 and 8');
      }
    }
  }
  return options;
}

function formatTable(report) {
  const lines = ['7个平台 / 8项检查 · geocheck v0.3.0 Node.js移植改进版',
    '直连，系统选择地址族；error = 未知，不代表未解锁。',
    '服务               状态         地区   耗时(ms)  详情 / 错误'];
  for (const result of report.results) {
    let detail = result.detail || '-';
    if (result.error && result.error !== result.detail) detail += ` | ${result.error}`;
    detail = detail.replace(/[\p{Cc}\p{Cf}\p{Zl}\p{Zp}]/gu, ' ');
    lines.push(`${result.name.padEnd(18)} ${result.status.padEnd(12)} ${(result.region || '-').padEnd(5)} `
      + `${result.elapsed_ms.toFixed(1).padStart(9)}  ${detail}`);
  }
  return lines.join('\n') + '\n';
}

async function main(argv = process.argv.slice(2), dependencies = {}) {
  const { stdout = (text) => process.stdout.write(text), stderr = (text) => process.stderr.write(text) } = dependencies;
  let args;
  try { args = parseArgs(argv); }
  catch (error) { stderr(`参数错误: ${error.message}\n`); return 2; }
  if (args.help) { stdout(HELP); return 0; }
  const report = await runAll({ ...args, clientFactory: dependencies.clientFactory, signal: dependencies.signal });
  stdout(args.json ? JSON.stringify(report, null, 2) + '\n' : formatTable(report));
  return report.results.some((result) => result.status === 'error') ? 1 : 0;
}

async function runCLI(argv = process.argv.slice(2), dependencies = {}) {
  const controller = new AbortController();
  const interrupt = () => controller.abort(new ProbeError('interrupted'));
  process.on('SIGINT', interrupt);
  try { return await main(argv, { ...dependencies, signal: controller.signal }); }
  catch (error) {
    const stderr = dependencies.stderr || ((text) => process.stderr.write(text));
    if (controller.signal.aborted) { stderr('检测已中断\n'); return 130; }
    stderr(`检测失败: ${error.message || String(error)}\n`);
    return 1;
  } finally { process.removeListener('SIGINT', interrupt); }
}

// Importing the file for offline tests performs no requests and installs no handlers.
module.exports = { SOURCE, UA, MAX_BODY, MAX_TIMEOUT, CHECKS, ENDPOINTS, NETFLIX_BASE,
  LICENSED_TITLE, ORIGINAL_TITLE, CLAUDE_TRACE, CHALLENGE_MARKERS, CLAUDE_MARKERS,
  TIKTOK_MARKERS, GEMINI_UNSUPPORTED, ProbeError, HTTPClient, verdict, validDestination,
  isChallenge, responseProblem, classify, netflixTitle, checkNetflix, cloudflareLoc,
  runOne, runAll, parseArgs, formatTable, main, runCLI };

if (require.main === module) {
  runCLI().then((code) => {
    process.exitCode = code;
    // Requests/streams/timers are already aborted. Some OS DNS lookups cannot
    // be cancelled by Node; explicit user cancellation must still exit promptly.
    if (code === 130) process.stderr.write('', () => process.exit(130));
  }, (error) => {
    process.stderr.write(`检测失败: ${error.message || String(error)}\n`);
    process.exitCode = 1;
  });
}
