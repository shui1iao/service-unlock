'use strict';
/* Offline fixtures only: no live platform requests, npm, or external commands.
 * Cases derived from geocheck v0.3.0 and the v0.1.0 Python port.
 * MIT License — Copyright (c) 2026 Remnawave. See LICENSE.
 * Run: node --test test_service_unlock.js
 */
const { test, before, after } = require('node:test');
const assert = require('node:assert/strict');
const http = require('node:http');
const https = require('node:https');
const dns = require('node:dns');
const { gzipSync } = require('node:zlib');
const { once } = require('node:events');
const { spawn, spawnSync } = require('node:child_process');
const path = require('node:path');
const s = require('./service_unlock.js');
const SCRIPT = path.join(__dirname, 'service_unlock.js');
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const response = (status = 200, body = 'ok', final_url = 'https://example.com/') => ({ status, body, final_url });

// Fail closed if a transport test accidentally tries a non-loopback target.
const originalLookup = dns.lookup;
const originalHTTP = http.request;
const originalHTTPS = https.request;
before(() => {
  dns.lookup = function guardedLookup(host, ...args) {
    assert.ok(['127.0.0.1', '::1', 'localhost'].includes(host), `external DNS forbidden: ${host}`);
    return originalLookup.call(this, host, ...args);
  };
  for (const [transport, original] of [[http, originalHTTP], [https, originalHTTPS]]) {
    transport.request = function guardedRequest(url, ...args) {
      assert.ok(['127.0.0.1', '[::1]', 'localhost'].includes(new URL(url).hostname), `external HTTP forbidden: ${url}`);
      return original.call(this, url, ...args);
    };
  }
});
after(() => { dns.lookup = originalLookup; http.request = originalHTTP; https.request = originalHTTPS; });

class FakeClient {
  async get(url) {
    if (url === s.CLAUDE_TRACE) return response(200, 'fl=1\nloc=FR\n', url);
    if (url.startsWith(s.NETFLIX_BASE + '/title/')) return response(200, 'ok', url);
    const bodies = {
      chatgpt_web: '{"is_cookie_required":false}', chatgpt_app: 'app landing page',
      youtube_premium_access: 'Enjoy AD-FREE viewing', claude_access: '<html><div id="root"></div></html>',
      tiktok_access: '<html>TikTok</html>', gemini_access: ',3,7,200,"FRA"', notebooklm_access: '',
    };
    for (const [id, [endpoint]] of Object.entries(s.ENDPOINTS)) {
      if (endpoint === url) return response(200, bodies[id], id === 'notebooklm_access'
        ? 'https://accounts.google.com/v3/signin/identifier' : url);
    }
    throw new Error(`unexpected fixture URL: ${url}`);
  }
}
const factory = () => new FakeClient();
function assertState(id, reply, expected, region = null) {
  const result = s.classify(id, reply, region);
  assert.equal(result.status, expected, JSON.stringify(result));
  if (expected === 'error') assert.ok(result.error);
  else assert.equal(result.error, null);
  return result;
}

// Platform decisions: the same rules/markers and improvements as Python v0.1.0.
test('YouTube: positive evidence, explicit refusal, missing wording and anomalous HTTP', () => {
  for (const [body, expected] of [['AD-FREE', 'available'],
    ['YouTube Premium is not available in your country', 'blocked'], ['consent page', 'error']]) {
    assertState('youtube_premium_access', response(200, body), expected);
  }
  assertState('youtube_premium_access', response(403, 'ad-free'), 'error');
  assertState('youtube_premium_access', response(403, 'YouTube Premium is not available in your country'), 'blocked');
});
test('ChatGPT Web: unsupported country on 403 is blocked, bare 401/403/404 are unknown', () => {
  assertState('chatgpt_web', response(200, '{}'), 'available');
  assertState('chatgpt_web', response(403, '{"error":"UNSUPPORTED_COUNTRY"}'), 'blocked');
  for (const status of [401, 403, 404]) assertState('chatgpt_web', response(status, 'unknown failure'), 'error');
});
test('ChatGPT App: both refusal markers retained; challenge precedes refusal', () => {
  assertState('chatgpt_app', response(), 'available');
  for (const body of ['Disallowed ISP', 'you have been blocked']) assertState('chatgpt_app', response(403, body), 'blocked');
  assertState('chatgpt_app', response(403, 'Forbidden'), 'error');
  assertState('chatgpt_app', response(403, 'just a moment; disallowed isp'), 'error');
});
test('Claude: upstream decision cases and all escaped refusal markers', () => {
  for (const [status, body, expected] of [[403, '_cf_chl_opt /app-unavailable-in-region', 'error'],
    [403, 'Forbidden', 'error'], [502, 'Bad Gateway', 'error'], [200, '<div id="root"></div>', 'available']]) {
    assert.equal(assertState('claude_access', response(status, body), expected, 'FR').region, 'FR');
  }
  for (const body of s.CLAUDE_MARKERS) {
    for (const status of [200, 403]) assertState('claude_access', response(status, body.toUpperCase()), 'blocked');
  }
});
test('Cloudflare challenge markers only apply to upstream 403/429/503 statuses', () => {
  for (const marker of s.CHALLENGE_MARKERS) {
    assert.equal(s.isChallenge(200, marker), false);
    for (const status of [403, 429, 503]) assert.equal(s.isChallenge(status, marker), true);
  }
});
test('TikTok: all four original refusal markers and a bare 403', () => {
  assertState('tiktok_access', response(), 'available');
  for (const marker of s.TIKTOK_MARKERS) {
    for (const status of [200, 403]) assertState('tiktok_access', response(status, marker.toUpperCase()), 'blocked');
  }
  assertState('tiktok_access', response(403, 'Forbidden'), 'error');
});
test('Gemini: served region, numeric shifts and exact upstream country list', () => {
  assert.equal(assertState('gemini_access', response(200, ',3,7,200,"USA"'), 'available').region, 'USA');
  assert.deepEqual([...s.GEMINI_UNSUPPORTED].sort(), ['RUS', 'BLR', 'CHN', 'PRK', 'IRN', 'CUB', 'SYR'].sort());
  for (const code of s.GEMINI_UNSUPPORTED) {
    assert.equal(assertState('gemini_access', response(200, `,2,1,200,"${code}"`), 'blocked').region, code);
  }
});
test('Gemini: empty, absent, lowercase and malformed region never produce a verdict', () => {
  for (const body of ['', '<html></html>', '{"error":"nope"}', ',3,7,200,"usa"', ',3,7,200,"US"']) {
    assertState('gemini_access', response(200, body), 'error');
  }
});
test('NotebookLM: upstream login destinations and refusal wins over login host', () => {
  for (const url of ['https://accounts.google.com/v3/signin/identifier',
    'https://notebook.google.com/login?continue=https://notebook.google.com/']) {
    const result = assertState('notebooklm_access', response(200, '', url), 'available');
    assert.equal(result.detail, 'sign-in required');
    assert.equal(result.region, null);
  }
  for (const url of ['https://notebook.google/?location=unsupported',
    'https://notebook.google.com/login?location=unsupported', 'https://accounts.google.com/signin?location=unsupported']) {
    assertState('notebooklm_access', response(200, '', url), 'blocked');
  }
});
test('NotebookLM: invalid/unrecognised destinations and login-address 403 are unknown', () => {
  for (const url of ['', 'https://[', '/relative', 'https://example.com/',
    'https://accounts.google.com.evil.test/login', 'https://user@accounts.google.com/login',
    'https:///accounts.google.com/login', 'https://accounts.google.com:99999/login']) {
    assertState('notebooklm_access', response(200, 'ok', url), 'error');
  }
  assertState('notebooklm_access', response(403, 'Forbidden', 'https://accounts.google.com/login'), 'error');
  // Preserve Python's raw host (including explicit ports) for classifier input.
  assertState('notebooklm_access', response(200, '', 'https://accounts.google.com:443/login'), 'error');
});
test('Every single-response check: 429/5xx override positive and negative markers', () => {
  const body = 'ad-free unsupported_country disallowed isp been blocked app unavailable in region '
    + 'not available in your region youtube premium is not available in your country ,3,7,200,"CHN"';
  for (const id of Object.keys(s.ENDPOINTS)) {
    for (const status of [429, 500, 502, 503, 599]) {
      for (const url of ['https://accounts.google.com/login', 'https://notebook.google/?location=unsupported']) {
        assertState(id, response(status, body, url), 'error');
      }
    }
  }
});
test('Empty text and challenge-echoed refusal cannot masquerade as a result', () => {
  for (const id of Object.keys(s.ENDPOINTS)) {
    if (id !== 'notebooklm_access') {
      for (const body of ['', ' \n\t']) assertState(id, response(200, body), 'error');
    }
    const body = 'cf_chl_opt unsupported_country disallowed isp app unavailable in region '
      + 'not available in your region youtube premium is not available in your country';
    assertState(id, response(403, body, 'https://notebook.google/?location=unsupported'), 'error');
  }
});

function sequenceClient(replies) {
  return { urls: [], async get(url) {
    this.urls.push(url);
    assert.ok(replies.length, 'unexpected extra request');
    const value = replies.shift();
    if (value instanceof Error) throw value;
    return value;
  } };
}
test('Netflix: full catalogue, locale/US extraction and no unnecessary second request', async () => {
  for (const [urlPath, region] of [['/fr-en/title/70143836', 'FR'], ['/title/70143836', 'US'], ['/jp/title/70143836', 'JP']]) {
    const client = sequenceClient([response(200, 'ok', s.NETFLIX_BASE + urlPath)]);
    const result = await s.checkNetflix(client);
    assert.deepEqual([result.status, result.region], ['available', region]);
    assert.deepEqual(client.urls, [s.NETFLIX_BASE + '/title/70143836']);
  }
});
test('Netflix: licensed 404 + original 200 = restricted; two 404s = blocked', async () => {
  const client = sequenceClient([response(404), response(200, 'ok', s.NETFLIX_BASE + '/fr-en/title/80197526')]);
  const result = await s.checkNetflix(client);
  assert.deepEqual([result.status, result.region], ['restricted', 'FR']);
  assert.equal(client.urls.length, 2);
  assert.equal((await s.checkNetflix(sequenceClient([response(404), response(404)]))).status, 'blocked');
});
test('Netflix: 403/429/5xx are not evidence of a withheld title', async () => {
  for (const status of [403, 429, 500, 502, 503]) {
    for (const replies of [[response(status)], [response(404), response(status)]]) {
      await assert.rejects(s.checkNetflix(sequenceClient(replies)), s.ProbeError);
    }
  }
});
test('Netflix: title, region, hostname and URL parse failures never claim available', async () => {
  for (const url of ['', 'https://[', s.NETFLIX_BASE + '/login', s.NETFLIX_BASE + '/fr-en/login',
    s.NETFLIX_BASE + '/ab/anything', s.NETFLIX_BASE + '/title/123', 'https://example.com/title/70143836']) {
    await assert.rejects(s.checkNetflix(sequenceClient([response(200, 'ok', url)])), s.ProbeError);
  }
});

test('Runner: exact ordered eight IDs, seven platforms, unchanged JSON keys and timing', async () => {
  const report = await s.runAll({ clientFactory: factory });
  assert.deepEqual(report.results.map((r) => r.id), ['chatgpt_web', 'chatgpt_app', 'youtube_premium_access',
    'netflix_access', 'claude_access', 'tiktok_access', 'gemini_access', 'notebooklm_access']);
  assert.deepEqual([...new Set(report.results.map((r) => r.platform))].sort(),
    ['Netflix', 'YouTube Premium', 'TikTok', 'ChatGPT', 'Gemini', 'NotebookLM', 'Claude'].sort());
  for (const result of report.results) {
    assert.equal(result.status, 'available');
    assert.deepEqual(Object.keys(result).sort(), ['id', 'platform', 'name', 'status', 'detail', 'region', 'error', 'checked_at', 'elapsed_ms'].sort());
    assert.ok(result.elapsed_ms >= 0);
    assert.match(result.checked_at, /^\d{4}-\d\d-\d\dT.*Z$/);
  }
  assert.equal(report.results[4].region, 'FR');
  assert.equal(report.schema_version, 1);
  assert.equal(report.network.timeout_scope, 'request including redirects and body');
  assert.equal(Object.hasOwn(report, 'status'), false);
});
test('Runner: timeout, network and parse failures yield error on all eight checks', async () => {
  for (const error of [new s.ProbeError('request timed out'), new Error('network unreachable'), new SyntaxError('bad response')]) {
    const report = await s.runAll({ clientFactory: () => ({ async get() { throw error; } }) });
    assert.equal(report.results.length, 8);
    for (const result of report.results) {
      assert.equal(result.status, 'error');
      assert.ok(result.error.includes(error.message));
    }
  }
});
test('Runner: Claude trace failure is optional; main failure retains trace region', async () => {
  class NoTrace extends FakeClient {
    async get(url) { if (url === s.CLAUDE_TRACE) throw new Error('trace timeout'); return super.get(url); }
  }
  let result = await s.runOne(s.CHECKS[4], 6, () => new NoTrace());
  assert.deepEqual([result.status, result.region], ['available', null]);
  class MainFailure extends FakeClient {
    async get(url) { if (url === s.ENDPOINTS.claude_access[0]) throw new Error('main timeout'); return super.get(url); }
  }
  result = await s.runOne(s.CHECKS[4], 6, () => new MainFailure());
  assert.deepEqual([result.status, result.region], ['error', 'FR']);
});
test('Runner: concurrency bound of two is enforced', async () => {
  let active = 0;
  let peak = 0;
  class Slow extends FakeClient {
    async get(url) {
      active += 1; peak = Math.max(peak, active);
      try { await sleep(10); return await super.get(url); } finally { active -= 1; }
    }
  }
  await s.runAll({ workers: 2, clientFactory: () => new Slow() });
  assert.equal(peak, 2);
  assert.equal(active, 0);
});
test('CLI: --json is a single valid JSON object; success/error exit values', async () => {
  for (const [clientFactory, expected] of [[factory, 0], [() => null, 1]]) {
    let output = '';
    const code = await s.main(['--json', '--workers', '2', '--timeout', '.1'], {
      clientFactory, stdout: (text) => { output += text; }, stderr: () => assert.fail('unexpected stderr'),
    });
    const report = JSON.parse(output);
    assert.equal(code, expected);
    assert.equal(report.results.length, 8);
    assert.equal(report.network.workers, 2);
    assert.equal(report.network.timeout_seconds, 0.1);
  }
});
test('CLI: human-readable output distinguishes ChatGPT Web/App and error meaning', async () => {
  let output = '';
  assert.equal(await s.main([], { clientFactory: factory, stdout: (text) => { output += text; } }), 0);
  for (const marker of ['7个平台 / 8项检查', 'ChatGPT (web)', 'ChatGPT (app)', 'error = 未知']) assert.ok(output.includes(marker));
});
test('CLI: four states serialize faithfully, blocked/restricted alone do not exit 1', async () => {
  class Refusals extends FakeClient {
    async get(url) {
      if (url.includes('/title/' + s.LICENSED_TITLE)) return response(404);
      if (url === s.ENDPOINTS.chatgpt_web[0]) return response(403, 'unsupported_country', url);
      return super.get(url);
    }
  }
  let output = '';
  const code = await s.main(['--json'], { clientFactory: () => new Refusals(), stdout: (text) => { output += text; } });
  assert.equal(code, 0);
  const statuses = new Set(JSON.parse(output).results.map((r) => r.status));
  assert.deepEqual(statuses, new Set(['blocked', 'restricted', 'available']));
});
test('CLI: help, unknown flags and invalid arguments never construct a client', async () => {
  const deps = { clientFactory: () => assert.fail('help/validation attempted network'), stdout: () => {}, stderr: () => {} };
  for (const args of [['--help'], ['-h']]) assert.equal(await s.main(args, deps), 0);
  for (const args of [['--workers', '0'], ['--workers=9'], ['--workers', '1.5'], ['--workers'],
    ['--timeout', '0'], ['--timeout', '-1'], ['--timeout', 'nan'], ['--timeout', 'Infinity'],
    ['--timeout', '9999999999'], ['--timeout='], ['--unknown'], ['--json=true']]) {
    assert.equal(await s.main(args, deps), 2, JSON.stringify(args));
  }
});
test('CLI: actual help/usage child-process exit codes and no network', () => {
  const help = spawnSync(process.execPath, [SCRIPT, '--help'], { encoding: 'utf8', timeout: 2000 });
  assert.equal(help.status, 0);
  assert.match(help.stdout, /7个平台/);
  const invalid = spawnSync(process.execPath, [SCRIPT, '--json', '--workers', '0'], { encoding: 'utf8', timeout: 2000 });
  assert.equal(invalid.status, 2);
  assert.equal(invalid.stdout, '');
});
test('CLI: actual process produces JSON and correct exit 0/1 using only injected fixtures', () => {
  for (const failed of [false, true]) {
    const program = `const s=require(${JSON.stringify(SCRIPT)}); const response=${response.toString()};\n`
      + FakeClient.toString() + '\n'
      + `s.runCLI(['--json'], {clientFactory:()=>${failed ? 'null' : 'new FakeClient()'}}).then(c=>{process.exitCode=c});`;
    const child = spawnSync(process.execPath, ['-e', program], { encoding: 'utf8', timeout: 2000 });
    assert.equal(child.status, failed ? 1 : 0, child.stderr);
    assert.equal(JSON.parse(child.stdout).results.length, 8);
  }
});

let server;
let base;
const sockets = new Set();
const pendingTimers = new Set();
function later(callback, ms) {
  const timer = setTimeout(() => { pendingTimers.delete(timer); callback(); }, ms);
  pendingTimers.add(timer);
  return timer;
}
before(async () => {
  server = http.createServer((req, res) => {
    res.on('error', () => {});
    const url = new URL(req.url, 'http://127.0.0.1');
    const route = url.pathname;
    if (route === '/hang') return; // Only our test process will connect; abort must close this socket.
    if (route === '/slow') { later(() => res.end('ok'), 150); return; }
    if (route === '/trickle') {
      res.writeHead(200); res.write('x');
      const tick = () => { if (!res.destroyed) { res.write('x'); later(tick, 10); } };
      later(tick, 10); return;
    }
    if (route === '/slow-hop') {
      later(() => { res.writeHead(302, { Location: '/slow-hop?n=' + (Number(url.searchParams.get('n') || 0) + 1) }); res.end(); }, 25);
      return;
    }
    if (route === '/redirect') { res.writeHead(302, { Location: '/final' }); res.end(); return; }
    if (route === '/gzip-redirect') { res.writeHead(307, { Location: '/gzip' }); res.end(); return; }
    if (route.startsWith('/hop/')) { res.writeHead(302, { Location: '/hop/' + (Number(route.split('/')[2]) + 1) }); res.end(); return; }
    if (route === '/loop') { res.writeHead(302, { Location: '/loop' }); res.end(); return; }
    if (route === '/bad-redirect') { res.writeHead(302, { Location: 'file:///etc/passwd' }); res.end(); return; }
    if (route === '/missing-location') { res.writeHead(302); res.end(); return; }
    if (route === '/forbidden') { res.writeHead(403); res.end('unsupported_country'); return; }
    if (route === '/server-error') { res.writeHead(500); res.end('ad-free'); return; }
    if (route === '/headers') { res.end(JSON.stringify(req.headers)); return; }
    if (route === '/gzip') { res.writeHead(200, { 'Content-Encoding': 'gzip' }); res.end(gzipSync('ad-free')); return; }
    if (route === '/bad-gzip') { res.writeHead(200, { 'Content-Encoding': 'gzip' }); res.end('not gzip'); return; }
    if (route === '/gzip-bomb') { res.writeHead(200, { 'Content-Encoding': 'gzip' }); res.end(gzipSync(Buffer.alloc(s.MAX_BODY + 1, 120))); return; }
    if (route === '/large') { res.end(Buffer.alloc(s.MAX_BODY + 1, 120)); return; }
    if (route === '/chunked-large') { res.writeHead(200); res.write(Buffer.alloc(s.MAX_BODY, 120)); res.end('x'); return; }
    if (route === '/encoding') { res.writeHead(200, { 'Content-Encoding': 'br' }); res.end('ok'); return; }
    if (route === '/truncated') { res.writeHead(200, { 'Content-Length': '100', Connection: 'close' }); res.end('short'); return; }
    if (route === '/no-content') { res.writeHead(204); res.end(); return; }
    res.end('ok');
  });
  server.on('connection', (socket) => { sockets.add(socket); socket.on('close', () => sockets.delete(socket)); });
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  base = `http://127.0.0.1:${server.address().port}`;
});
after(async () => {
  for (const timer of pendingTimers) clearTimeout(timer);
  for (const socket of sockets) socket.destroy();
  if (server) await new Promise((resolve) => server.close(resolve));
});

test('Transport: follows redirects, preserves final URL, UA/language and gzip header', async () => {
  const client = new s.HTTPClient(1);
  const result = await client.get(base + '/redirect');
  assert.equal(result.final_url, base + '/final');
  assert.equal(result.status, 200);
  const headers = JSON.parse((await client.get(base + '/headers', { 'Accept-Language': 'en-US,en;q=0.9' })).body);
  assert.equal(headers['user-agent'], s.UA);
  assert.equal(headers['accept-encoding'], 'gzip');
  assert.equal(headers['accept-language'], 'en-US,en;q=0.9');
});
test('Transport: HTTP error body preserved for service-specific 403, 500 stays unknown', async () => {
  const client = new s.HTTPClient(1);
  assertState('chatgpt_web', await client.get(base + '/forbidden'), 'blocked');
  assertState('youtube_premium_access', await client.get(base + '/server-error'), 'error');
});
test('Transport: gzip body, including after redirect, is decoded', async () => {
  for (const route of ['/gzip', '/gzip-redirect']) assert.equal((await new s.HTTPClient(1).get(base + route)).body, 'ad-free');
});
test('Transport: malformed gzip and unsupported encoding reject', async () => {
  for (const route of ['/bad-gzip', '/encoding']) await assert.rejects(new s.HTTPClient(1).get(base + route));
});
test('Transport: 4 MiB bound covers content-length, chunked and decoded bodies', async () => {
  for (const route of ['/large', '/chunked-large', '/gzip-bomb']) {
    await assert.rejects(new s.HTTPClient(2).get(base + route), /exceeds 4 MiB/);
  }
});
test('Transport: truncated Content-Length rejects; empty 204 stays non-evidence', async () => {
  await assert.rejects(new s.HTTPClient(1).get(base + '/truncated'), /incomplete|aborted/i);
  assertState('chatgpt_web', await new s.HTTPClient(1).get(base + '/no-content'), 'error');
});
test('Transport: redirect limit, loops, forbidden scheme and missing Location reject', async () => {
  for (const route of ['/hop/0', '/loop']) await assert.rejects(new s.HTTPClient(1).get(base + route), /10 redirects/);
  for (const route of ['/bad-redirect', '/missing-location']) await assert.rejects(new s.HTTPClient(1).get(base + route), s.ProbeError);
});
test('Transport: total budget cancels stalled headers, trickle body and redirect chain', async () => {
  for (const route of ['/slow', '/trickle', '/slow-hop']) {
    const started = Date.now();
    await assert.rejects(new s.HTTPClient(0.06).get(base + route), /timed out/);
    assert.ok(Date.now() - started < 800, 'timeout did not cancel promptly');
  }
});
test('Transport: explicit abort closes active request and rejects pre-aborted requests', async () => {
  const controller = new AbortController();
  const requestSeen = once(server, 'request');
  const pending = new s.HTTPClient(60, controller.signal).get(base + '/hang');
  await requestSeen;
  controller.abort(new s.ProbeError('test cancellation'));
  await assert.rejects(pending, /test cancellation/);
  await assert.rejects(new s.HTTPClient(60, controller.signal).get(base + '/final'), /test cancellation/);
});
test('Transport: direct requests ignore proxy environment variables', async () => {
  const names = ['HTTP_PROXY', 'http_proxy', 'HTTPS_PROXY', 'https_proxy', 'NO_PROXY', 'no_proxy', 'NODE_USE_ENV_PROXY'];
  const saved = Object.fromEntries(names.map((name) => [name, process.env[name]]));
  try {
    for (const name of names) process.env[name] = name === 'NODE_USE_ENV_PROXY' ? '1' : name.toLowerCase() === 'no_proxy' ? '' : 'http://127.0.0.1:1';
    assert.equal((await new s.HTTPClient(1).get(base + '/final')).status, 200);
  } finally {
    for (const name of names) { if (saved[name] === undefined) delete process.env[name]; else process.env[name] = saved[name]; }
  }
});

// Public, disposable localhost-only self-signed key/cert fixture. NOT a real credential.
const TLS_KEY = "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDEwtZiny9A4ets\ned6w84VKVzKeA3uVIFa2FGl/K0FemCYqJWhP2WD4eDeGDFgPHv06/rNwPcpTnLuz\nKpYm4BsbR+xCJyVHkLDuFTbJzaapPDGzAe8wUkSsZj0v7DRiiFhM7+QsIot992UO\nTGNyPuLpQHhTaIlN9x8zt07MBvS0oXybpVwgQpXZ1/RDgKoHtDC8Fxty5xGKnsd7\nVl+7gudPUetc+Q1j3B9qRHk1ItDCVZURIQMCoEvDbWTHDezTMQgM/vTPaHxONvHp\nLhn1hrrYr3/y/Lh4a7QySVNsH7i3JRKv/Xq3iB0lSOrPe3KjEML6Uuq80Gmt6wdP\nqGPDsVVBAgMBAAECggEAKiHDWMZXzrehmu6EqBTZFYhS6yF5SL8uOhpoVaX8PyZq\nw6sTaUe1mzbYwtApWgGLfWK1JyNgC+PbkFud39CslTyjIy4xLiAWNsDdpLwvw/zC\n8p8HKexvzo+s6fJU7EVgTQcwL2AtSOmc3sEuKTQ8j/AxY8DWZ9ay/HazvX3+z5+E\nyQe/0Zxa1BDJ4zF7eg2TDKSzZ0gmGiMxavV6/ebCWO+1WXMWuiZzImz2A///csT2\ndMYHHVtuSGHMPTjmBpo+eiy6O1SfwKG4t771gNxjTyNFWSGgdaT7FH2cdeAYk57a\neySpfQ9stzWfoNF1UE9s396F+KsTq/57l0EjTiFs/QKBgQD+aBFKU0uuTMMOkMec\nvgxU7yft9Fb2pbbaP2p3LhyJcDl+AzITfdU8b6PX236V1K+Zhie7ws78Hcdf5fnZ\n3Q/0E5Vg0T8gRtJxprbNiIjxajFy/JY66xFXdcbuZhC6CO3kyetIDWnB5peo5X16\nzi6gBcE+2GDqtgwbx8wLR57w5QKBgQDF/lZc44zI+nJ6HS2+0Bi7RAxHLOi14FOI\ncS1ucKCGAEupBfILEJjLDm/cPerbRicnH8p3QG2LplJvThCnVfzwOtNtPF29sadh\nXw2cfFp6VjJV+rzpfx2qbTl1N/WoM/4pLbX9Riy3nzTqbwg/9QNQ6o5KV5ukEmyg\nGBcKGMY5LQKBgGj9PZV0xpjJ6P2L1jTjYWdzwwC6g2EKhmmPHjrM9s3pWHCelP3i\nQivMzsr4NEdVHJq4+JDOFmmJSIxU9kVoq3rI0i7Aa9LAJ3zrlilAHXOzpXCUuNLb\ny65SVqRvzptXsAVpJwX3MxvoM9Jp0tzpVeiIN3XeCjESq/CTBavqBLfZAoGALZm0\nXlRyOl+kOge6X3YiHA9CHKp1Nl5iHsyiQAtkjThINLeP1YiPCEVc8ga4rqPrpJEn\n2mtWT5GQ7t217DRbuyBxTXlXxbbC7CQuSFpDpGVx1meXVtfWB0h8sUigJYtEAEvs\n3FZrOidK0qub5M8TYnGDMBAZYqJzrjOeKs8l6EkCgYBHDg6zj4aIf7Z8S4bm5lAW\nxdfjWC/yJlqngqfQzVP3MySCylM2YLRK51PAjOX1HOLYT3vZ/Nm6pCzc3n//bmV1\ndoKKQl7nkwQRfsC/EFCeBEALdDk4WGdc1q4rZyYFvrhNvKjExby1nnpAnpiWZ1gd\nbAD0l93K/ZR7cvIs0K1YtA==\n-----END PRIVATE KEY-----\n";
const TLS_CERT = "-----BEGIN CERTIFICATE-----\nMIIDCTCCAfGgAwIBAgIUA1CNL5TZWDphfJBYyi8FGjp3HxUwDQYJKoZIhvcNAQEL\nBQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI2MTAxNzA1NDExN1oXDTM2MTAx\nNDA1NDExN1owFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF\nAAOCAQ8AMIIBCgKCAQEAxMLWYp8vQOHrbHnesPOFSlcyngN7lSBWthRpfytBXpgm\nKiVoT9lg+Hg3hgxYDx79Ov6zcD3KU5y7syqWJuAbG0fsQiclR5Cw7hU2yc2mqTwx\nswHvMFJErGY9L+w0YohYTO/kLCKLffdlDkxjcj7i6UB4U2iJTfcfM7dOzAb0tKF8\nm6VcIEKV2df0Q4CqB7QwvBcbcucRip7He1Zfu4LnT1HrXPkNY9wfakR5NSLQwlWV\nESEDAqBLw21kxw3s0zEIDP70z2h8Tjbx6S4Z9Ya62K9/8vy4eGu0MklTbB+4tyUS\nr/16t4gdJUjqz3tyoxDC+lLqvNBpresHT6hjw7FVQQIDAQABo1MwUTAdBgNVHQ4E\nFgQUVOkOi3M8xAZmR1NQEtktXteM58kwHwYDVR0jBBgwFoAUVOkOi3M8xAZmR1NQ\nEtktXteM58kwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEARl56\nDEkYLKoD3kJGm6ZzRpGpn1A8uVwOWNrh3dGFbFrr3EiNWemtyiUcHshIuU9p8d34\nm20abt+XOxBp5/DU/oa69AYUqrvyqsOJlMVPQ0inf4GxTcQtGRIGY7u2e3ZEw+qT\nWb24zHU8XaJqZwcO0FcJ05D3Z4h5vRmwSmW609dgFzM2lLFW3kVnX/XJZl9Fo1J9\nkzT1dL3sS+cP0A0yspW/7eQ09jYyxXIl4dinLDAOmx+xE5MH1IjGRvNr0wN5C0Du\nakx1Z3NzJjYQcpbrnJg75ZGnwewtv7MB0G7ysTtwTyL41vzAqZkHZWO2HI/3px2h\nwnYDqEOwzob3SGFDOA==\n-----END CERTIFICATE-----\n";
test('Transport: self-signed TLS is rejected even with NODE_TLS_REJECT_UNAUTHORIZED=0', async () => {
  const tlsServer = https.createServer({ key: TLS_KEY, cert: TLS_CERT }, (req, res) => res.end('ok'));
  tlsServer.on('tlsClientError', () => {});
  tlsServer.listen(0, '127.0.0.1');
  await once(tlsServer, 'listening');
  const saved = process.env.NODE_TLS_REJECT_UNAUTHORIZED;
  try {
    process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
    await assert.rejects(new s.HTTPClient(1).get(`https://127.0.0.1:${tlsServer.address().port}/`),
      (error) => ['DEPTH_ZERO_SELF_SIGNED_CERT', 'CERT_HAS_EXPIRED', 'CERT_NOT_YET_VALID'].includes(error.code));
  } finally {
    if (saved === undefined) delete process.env.NODE_TLS_REJECT_UNAUTHORIZED;
    else process.env.NODE_TLS_REJECT_UNAUTHORIZED = saved;
    await new Promise((resolve) => tlsServer.close(resolve));
  }
});

async function childResult(child, limit = 3000) {
  let stdout = '';
  let stderr = '';
  child.stdout.on('data', (data) => { stdout += data; });
  child.stderr.on('data', (data) => { stderr += data; });
  const timer = setTimeout(() => child.kill('SIGKILL'), limit);
  try {
    const [code, signal] = await once(child, 'close');
    assert.notEqual(signal, 'SIGKILL', 'child hung with requests/timers open');
    return { code, signal, stdout, stderr };
  } finally { clearTimeout(timer); }
}
function localChildProgram(route, timeout) {
  return `const s=require(${JSON.stringify(SCRIPT)});\n`
    + `s.runCLI(['--json','--timeout','${timeout}','--workers','2'], {clientFactory:(t,signal)=>{\n`
    + `const c=new s.HTTPClient(t,signal); return {get:()=>c.get(${JSON.stringify(base + route)})};\n`
    + '}}).then(code=>{process.exitCode=code});';
}
test('CLI: timeout child exits naturally with eight errors and no lingering timers', async () => {
  const child = spawn(process.execPath, ['-e', localChildProgram('/hang', 0.04)], { stdio: ['ignore', 'pipe', 'pipe'] });
  const result = await childResult(child);
  assert.equal(result.code, 1, result.stderr);
  assert.equal(JSON.parse(result.stdout).results.length, 8);
  assert.ok(JSON.parse(result.stdout).results.every((r) => r.status === 'error'));
});
test('CLI: Ctrl-C aborts active requests, exits 130 and emits no partial JSON', async () => {
  const requestSeen = once(server, 'request');
  const child = spawn(process.execPath, ['-e', localChildProgram('/hang', 60)], { stdio: ['ignore', 'pipe', 'pipe'] });
  const completion = childResult(child);
  await requestSeen;
  child.kill('SIGINT');
  const result = await completion;
  assert.equal(result.code, 130, result.stderr);
  assert.equal(result.stdout, '');
  assert.match(result.stderr, /检测已中断/);
  await sleep(30);
  assert.equal(sockets.size, 0, 'cancelled client left an open server connection');
});
test('CLI entry: SIGINT exits even if an uninterruptible OS-style handle remains', async (t) => {
  const fs = require('node:fs');
  const os = require('node:os');
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'service-unlock-offline-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const preload = path.join(directory, 'loopback-only.cjs');
  // This test-only preload maps EVERY request to loopback. The extra timer
  // simulates a non-cancellable OS resolver handle; it is never in the product.
  fs.writeFileSync(preload, `const http=require('node:http'),https=require('node:https');\n`
    + `const original=http.request; const local=(url,options,cb)=>original(${JSON.stringify(base + '/hang')},options,cb);\n`
    + 'http.request=local; https.request=local; setInterval(()=>{},60000);\n');
  const requestSeen = once(server, 'request');
  const child = spawn(process.execPath, ['--require', preload, SCRIPT, '--json', '--timeout', '60'],
    { stdio: ['ignore', 'pipe', 'pipe'] });
  const completion = childResult(child);
  await requestSeen;
  child.kill('SIGINT');
  const result = await completion;
  assert.equal(result.code, 130, result.stderr);
  assert.equal(result.stdout, '');
  assert.match(result.stderr, /检测已中断/);
});
