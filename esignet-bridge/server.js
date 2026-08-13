// eSignet <-> WSO2 Identity Server bridge for the custom authenticator extension.
//
//   1. IS POSTs /authenticate      -> reply INCOMPLETE + redirect to eSignet
//   2. user authenticates at eSignet
//   3. eSignet redirects to /callback -> exchange the code (private_key_jwt)
//   4. fetch userinfo (a signed JWT) and verify it against eSignet's JWKS
//   5. redirect to IS /commonauth?flowId=..
//   6. IS POSTs /authenticate again -> reply SUCCESS with mapped claims

import express from 'express';
import { readFileSync } from 'fs';
import { randomUUID, randomBytes, createHash, timingSafeEqual } from 'crypto';
import { importJWK, SignJWT, jwtVerify, createRemoteJWKSet } from 'jose';

const cfg = {
  PORT:        process.env.PORT        || 4000,
  BRIDGE_URL:  process.env.BRIDGE_URL  || 'http://localhost:4000',
  IS_BASE_URL: process.env.IS_BASE_URL || 'https://localhost:9443',
  ESIGNET_UI:  process.env.ESIGNET_UI  || 'http://localhost:3000',
  ESIGNET_API: process.env.ESIGNET_API || 'http://localhost:8088/v1/esignet',
  CLIENT_ID:   process.env.CLIENT_ID   || 'wso2-is-bridge',
  ACR:         process.env.ACR         || 'mosip:idp:acr:generated-code',
  SCOPE:       process.env.SCOPE       || 'openid profile',
  API_KEY:     process.env.BRIDGE_API_KEY || '',
  FLOW_TTL_MS: Number(process.env.FLOW_TTL_MS || 10 * 60 * 1000),
};

const privJwk = JSON.parse(readFileSync('private.jwk.json', 'utf8'));
const privKey = await importJWK(privJwk, 'RS256');
const esignetJwks = createRemoteJWKSet(
  new URL(`${cfg.ESIGNET_API}/oauth/.well-known/jwks.json`));

// ---------- security helpers ----------
const clean = (s, max = 200) => String(s ?? '').replace(/[\r\n\t]/g, ' ').slice(0, max);
const TENANT_RE = /^[A-Za-z0-9._-]{1,120}$/;
const safeTenant = (t) => (TENANT_RE.test(t || '') ? t : 'carbon.super');
const newState = () => randomBytes(32).toString('base64url');

const redact = (o) => {
  try {
    const c = JSON.parse(JSON.stringify(o));
    for (const k of ['access_token','id_token','refresh_token','client_assertion','code'])
      if (k in c) c[k] = '<redacted>';
    return JSON.stringify(c);
  } catch { return '<unserialisable>'; }
};

function keyMatches(expected, got) {
  if (!expected) return true;
  const a = Buffer.from(String(expected)), b = Buffer.from(String(got ?? ''));
  return a.length === b.length && timingSafeEqual(a, b);
}

// bounded, expiring stores so half-finished logins cannot exhaust memory
class TtlMap {
  constructor(ttl, max = 5000) { this.m = new Map(); this.ttl = ttl; this.max = max; }
  set(k, v) {
    this.sweep();
    if (this.m.size >= this.max) this.m.delete(this.m.keys().next().value);
    this.m.set(k, { v, exp: Date.now() + this.ttl }); return this;
  }
  get(k) { const e = this.m.get(k); if (!e) return undefined;
           if (e.exp < Date.now()) { this.m.delete(k); return undefined; } return e.v; }
  delete(k) { return this.m.delete(k); }
  sweep() { const n = Date.now(); for (const [k, e] of this.m) if (e.exp < n) this.m.delete(k); }
}
const flows  = new TtlMap(cfg.FLOW_TTL_MS);
const states = new TtlMap(cfg.FLOW_TTL_MS);

// ---------- app ----------
const app = express();
app.disable('x-powered-by');
app.use(express.json({ limit: '64kb' }));
app.use((_req, res, next) => {
  res.setHeader('X-Content-Type-Options', 'nosniff');
  res.setHeader('X-Frame-Options', 'DENY');
  res.setHeader('Cache-Control', 'no-store');
  next();
});

app.use('/authenticate', (req, res, next) => {
  if (!keyMatches(cfg.API_KEY, req.get('x-api-key'))) {
    return res.status(401).json({
      actionStatus: 'ERROR',
      errorMessage: 'Unauthorized',
      errorDescription: 'Failed to authorize the request.',
    });
  }
  next();
});

// ---------- 1 & 6: the endpoint WSO2 IS calls ----------
app.post('/authenticate', (req, res) => {
  const { flowId, event } = req.body || {};
  if (!flowId || typeof flowId !== 'string') {
    return res.status(400).json({
      actionStatus: 'ERROR', errorMessage: 'missingFlowId',
      errorDescription: 'flowId is required.' });
  }

  const existing = flows.get(flowId);
  if (existing && existing.status) {                       // second invocation
    flows.delete(flowId);
    if (existing.status === 'SUCCESS') {
      return res.json({ actionStatus: 'SUCCESS', data: { user: existing.user } });
    }
    return res.json({
      actionStatus: 'FAILED',
      failureReason: 'authenticationFailed',
      failureDescription: clean(existing.reason || 'eSignet authentication failed.'),
    });
  }

  const state     = newState();
  const nonce     = randomUUID();
  const verifier  = randomBytes(32).toString('base64url');
  const challenge = createHash('sha256').update(verifier).digest('base64url');

  flows.set(flowId, { tenant: safeTenant(event?.tenant?.name), verifier });
  states.set(state, flowId);

  const u = new URL('/authorize', cfg.ESIGNET_UI);
  u.search = new URLSearchParams({
    response_type: 'code',
    client_id: cfg.CLIENT_ID,
    redirect_uri: `${cfg.BRIDGE_URL}/callback`,
    scope: cfg.SCOPE,
    acr_values: cfg.ACR,
    state, nonce,
    code_challenge: challenge,
    code_challenge_method: 'S256',
    ui_locales: 'en',
    claims_locales: 'en',
  }).toString();

  res.json({ actionStatus: 'INCOMPLETE',
             operations: [{ op: 'redirect', url: u.toString() }] });
});

// ---------- 3, 4, 5: eSignet sends the browser here ----------
app.get('/callback', async (req, res) => {
  const { code, state, error, error_description } = req.query;

  const flowId = states.get(state);
  states.delete(state);                        // single use — blocks replay
  if (!flowId) return res.status(400).type('text/plain').send('Invalid or expired request.');
  const flow = flows.get(flowId);
  if (!flow) return res.status(400).type('text/plain').send('Invalid or expired request.');

  // redirect target is built only from config plus a validated tenant, never user input
  const finish = () => res.redirect(
    `${cfg.IS_BASE_URL}/t/${encodeURIComponent(flow.tenant)}/commonauth` +
    `?flowId=${encodeURIComponent(flowId)}`);

  if (error || !code) {
    flow.status = 'FAILED';
    flow.reason = clean(error_description || error || 'no authorization code');
    console.warn('[auth-failed] flow=%s reason=%s', clean(flowId, 64), flow.reason);
    return finish();
  }

  try {
    const tokenUrl = `${cfg.ESIGNET_API}/oauth/v2/token`;

    const assertion = await new SignJWT({})
      .setProtectedHeader({ alg: 'RS256', typ: 'JWT', kid: privJwk.kid })
      .setIssuer(cfg.CLIENT_ID)        // iss MUST equal client_id
      .setSubject(cfg.CLIENT_ID)       // sub MUST equal client_id
      .setAudience(tokenUrl)           // aud MUST be the token endpoint URL
      .setIssuedAt()
      .setExpirationTime('5m')
      .setJti(randomUUID())            // eSignet rejects a reused jti
      .sign(privKey);

    const tr = await fetch(tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        grant_type: 'authorization_code',
        code: String(code),
        client_id: cfg.CLIENT_ID,
        redirect_uri: `${cfg.BRIDGE_URL}/callback`,
        code_verifier: flow.verifier,
        client_assertion_type: 'urn:ietf:params:oauth:client-assertion-type:jwt-bearer',
        client_assertion: assertion,
      }),
    });
    const tokens = await tr.json();
    if (!tr.ok || !tokens.access_token) {
      throw new Error(`token endpoint returned ${tr.status}: ${redact(tokens)}`);
    }

    const ur = await fetch(`${cfg.ESIGNET_API}/oidc/userinfo`, {
      headers: { Authorization: `Bearer ${tokens.access_token}` },
    });
    const jws = (await ur.text()).trim();
    if (!ur.ok) throw new Error(`userinfo endpoint returned ${ur.status}`);

    // eSignet always returns a signed JWT here, never JSON.
    const { payload: claims } = await jwtVerify(jws, esignetJwks);

    flow.status = 'SUCCESS';
    flow.user = toWso2User(claims);
    console.log('[login-ok] flow=%s sub=%s claims=%d',
      clean(flowId, 64), clean(claims.sub, 64), flow.user.claims.length);
  } catch (e) {
    flow.status = 'FAILED';
    flow.reason = clean(e.message);
    console.error('[login-error] flow=%s %s', clean(flowId, 64), clean(e.message));
  }
  finish();
});

function toWso2User(c) {
  const claims = [];
  const add = (uri, v) => { if (v !== undefined && v !== null && v !== '')
                              claims.push({ uri, value: String(v) }); };
  add('http://wso2.org/claims/username',     c.sub);
  add('http://wso2.org/claims/emailaddress', c.email);
  add('http://wso2.org/claims/givenname',    c.given_name || c.name);
  add('http://wso2.org/claims/lastname',     c.family_name);
  add('http://wso2.org/claims/mobile',       c.phone_number || c.phone);
  add('http://wso2.org/claims/dob',          c.birthdate);
  add('http://wso2.org/claims/gender',       c.gender);
  return { id: c.sub, claims };
}

app.get('/health', (_q, r) => r.json({ status: 'ok' }));

app.listen(cfg.PORT, () => {
  console.log(`bridge listening on ${cfg.BRIDGE_URL}`);
  console.log(`  eSignet UI  : ${cfg.ESIGNET_UI}`);
  console.log(`  eSignet API : ${cfg.ESIGNET_API}`);
  console.log(`  WSO2 IS     : ${cfg.IS_BASE_URL}`);
  console.log(`  client_id   : ${cfg.CLIENT_ID}`);
  console.log(`  api key     : ${cfg.API_KEY ? 'enabled' : 'DISABLED (dev only)'}`);
});
