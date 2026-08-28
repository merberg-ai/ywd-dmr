const ui = {
  health: document.querySelector('#health'),
  sessionPill: document.querySelector('#session-pill'),
  coreState: document.querySelector('#core-state'),
  networkState: document.querySelector('#network-state'),
  vocoderState: document.querySelector('#vocoder-state'),
  version: document.querySelector('#version'),
  setupStage: document.querySelector('#setup-stage'),
  setupNext: document.querySelector('#setup-next'),
  configState: document.querySelector('#config-state'),
  configRevision: document.querySelector('#config-revision'),
  operationState: document.querySelector('#operation-state'),
  result: document.querySelector('#test-result'),
  claimForm: document.querySelector('#claim-form'),
  loginForm: document.querySelector('#login-form'),
  identityForm: document.querySelector('#identity-form'),
  networkForm: document.querySelector('#network-form'),
  logoutButton: document.querySelector('#logout-button'),
  refreshButton: document.querySelector('#refresh-all'),
  validateButton: document.querySelector('#network-validate'),
  testButton: document.querySelector('#network-test'),
  commitButton: document.querySelector('#network-commit'),
  clearResult: document.querySelector('#clear-result')
};

let setupStatus = null;
let sessionStatus = { authenticated: false };

async function readResponse(response) {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch (_) {
    return { raw: text };
  }
}

async function api(path, options = {}) {
  const headers = new Headers(options.headers || {});
  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const response = await fetch(path, {
    credentials: 'same-origin',
    ...options,
    headers
  });
  return {
    ok: response.ok,
    status: response.status,
    statusText: response.statusText,
    body: await readResponse(response)
  };
}

function redact(value) {
  if (Array.isArray(value)) return value.map(redact);
  if (!value || typeof value !== 'object') return value;

  const secretNames = new Set([
    'password',
    'claim_code',
    'token',
    'session',
    'salt',
    'hash',
    'digest',
    'challenge'
  ]);

  const clean = {};
  for (const [key, item] of Object.entries(value)) {
    clean[key] = secretNames.has(key.toLowerCase()) ? '[REDACTED]' : redact(item);
  }
  return clean;
}

function responseSucceeded(result) {
  if (!result.ok) return false;
  const body = result.body;
  if (!body || typeof body !== 'object') return true;
  if (body.ok === false || body.valid === false || body.committed === false) return false;
  return true;
}

function showResult(label, result) {
  const payload = {
    operation: label,
    http_status: result.status,
    response: redact(result.body)
  };
  ui.result.textContent = JSON.stringify(payload, null, 2);
  const succeeded = responseSucceeded(result);
  setOperation(
    succeeded ? `${label}: success` : `${label}: did not complete successfully`,
    succeeded ? 'ok' : 'error'
  );
}

function showClientError(label, error) {
  ui.result.textContent = JSON.stringify({
    operation: label,
    client_error: String(error && error.message ? error.message : error)
  }, null, 2);
  setOperation(`${label} failed before completion.`, 'error');
}

function setOperation(message, state = '') {
  ui.operationState.textContent = message;
  ui.operationState.classList.remove('state-ok', 'state-error', 'state-running');
  if (state) ui.operationState.classList.add(`state-${state}`);
}

function setFormEnabled(form, enabled) {
  if (!form) return;
  form.classList.toggle('is-disabled', !enabled);
  for (const element of form.querySelectorAll('input, button, select, textarea')) {
    element.disabled = !enabled;
  }
}

function applyAccessState() {
  const claimed = Boolean(setupStatus && setupStatus.claimed);
  const authenticated = Boolean(sessionStatus && sessionStatus.authenticated);
  const isAdmin = authenticated && String(sessionStatus.role).toLowerCase() === 'admin';

  setFormEnabled(ui.claimForm, !claimed);
  setFormEnabled(ui.loginForm, claimed && !authenticated);
  setFormEnabled(ui.identityForm, isAdmin);
  setFormEnabled(ui.networkForm, isAdmin);
  ui.logoutButton.disabled = !authenticated;

  ui.sessionPill.classList.toggle('subdued', !authenticated);
  if (authenticated) {
    ui.sessionPill.textContent = `${String(sessionStatus.role || 'user').toUpperCase()} · ${sessionStatus.username || 'SIGNED IN'}`;
  } else {
    ui.sessionPill.textContent = 'SIGNED OUT';
  }
}

async function loadDashboard() {
  ui.refreshButton.disabled = true;
  try {
    const [healthResult, statusResult, systemResult, setupResult, sessionResult] = await Promise.all([
      api('/api/v1/health'),
      api('/api/v1/status'),
      api('/api/v1/system'),
      api('/api/v1/setup/status'),
      api('/api/v1/auth/session')
    ]);

    if (!healthResult.ok || !statusResult.ok || !systemResult.ok || !setupResult.ok || !sessionResult.ok) {
      throw new Error('one or more status endpoints returned an error');
    }

    const health = healthResult.body || {};
    const status = statusResult.body || {};
    const system = systemResult.body || {};
    setupStatus = setupResult.body || {};
    sessionStatus = sessionResult.body || { authenticated: false };

    const networkConfigured = Boolean(
      setupStatus.configuration && setupStatus.configuration.network_configured
    );
    const networkConnected = Boolean(status.network && status.network.connected);

    ui.health.textContent = health.ok ? 'CORE ONLINE' : 'ERROR';
    ui.coreState.textContent = String(status.state || 'unknown').toUpperCase();
    ui.networkState.textContent = networkConnected ? 'CONNECTED' : (networkConfigured ? 'CONFIGURED' : 'OFFLINE');
    ui.vocoderState.textContent = status.vocoder && status.vocoder.available ? 'READY' : 'NONE';
    ui.version.textContent = `${system.version || 'unknown'} · ${system.goarch || '?'} / ${system.goos || '?'}`;

    ui.setupStage.textContent = setupStatus.stage || '—';
    ui.setupNext.textContent = setupStatus.next_step || '—';
    ui.configState.textContent = setupStatus.configuration && setupStatus.configuration.state ? setupStatus.configuration.state : '—';
    ui.configRevision.textContent = setupStatus.configuration && setupStatus.configuration.revision ? setupStatus.configuration.revision : '—';

    applyAccessState();
  } catch (error) {
    ui.health.textContent = 'CORE OFFLINE';
    setOperation('Could not refresh daemon status.', 'error');
    console.error(error);
  } finally {
    ui.refreshButton.disabled = false;
  }
}

ui.claimForm.addEventListener('submit', async event => {
  event.preventDefault();
  const password = document.querySelector('#claim-password');
  setOperation('Claiming appliance…', 'running');
  try {
    const result = await api('/api/v1/setup/claim', {
      method: 'POST',
      body: JSON.stringify({
        claim_code: document.querySelector('#claim-code').value.trim(),
        username: document.querySelector('#claim-username').value.trim(),
        password: password.value
      })
    });
    password.value = '';
    showResult('Claim appliance', result);
    await loadDashboard();
  } catch (error) {
    password.value = '';
    showClientError('Claim appliance', error);
  }
});

ui.loginForm.addEventListener('submit', async event => {
  event.preventDefault();
  const password = document.querySelector('#login-password');
  setOperation('Signing in…', 'running');
  try {
    const result = await api('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({
        username: document.querySelector('#login-username').value.trim(),
        password: password.value
      })
    });
    password.value = '';
    showResult('Admin login', result);
    await loadDashboard();
  } catch (error) {
    password.value = '';
    showClientError('Admin login', error);
  }
});

ui.logoutButton.addEventListener('click', async () => {
  setOperation('Signing out…', 'running');
  try {
    const result = await api('/api/v1/auth/logout', { method: 'POST' });
    showResult('Admin logout', result);
    await loadDashboard();
  } catch (error) {
    showClientError('Admin logout', error);
  }
});

ui.identityForm.addEventListener('submit', async event => {
  event.preventDefault();
  setOperation('Committing station identity…', 'running');
  try {
    const result = await api('/api/v1/setup/identity/commit', {
      method: 'POST',
      body: JSON.stringify({
        callsign: document.querySelector('#identity-callsign').value.trim(),
        dmr_id: Number(document.querySelector('#identity-dmr-id').value),
        essid: Number(document.querySelector('#identity-essid').value)
      })
    });
    showResult('Commit identity', result);
    await loadDashboard();
  } catch (error) {
    showClientError('Commit identity', error);
  }
});

function networkCandidate() {
  return {
    backend: 'brandmeister',
    master_address: document.querySelector('#network-master').value.trim(),
    master_port: Number(document.querySelector('#network-port').value),
    registration_frequency_hz: Number(document.querySelector('#network-frequency').value),
    password: document.querySelector('#network-password').value
  };
}

ui.validateButton.addEventListener('click', async () => {
  setOperation('Validating candidate locally…', 'running');
  try {
    const result = await api('/api/v1/setup/network/validate', {
      method: 'POST',
      body: JSON.stringify(networkCandidate())
    });
    showResult('Validate BrandMeister candidate', result);
  } catch (error) {
    showClientError('Validate BrandMeister candidate', error);
  }
});

ui.testButton.addEventListener('click', async () => {
  const confirmed = window.confirm('Run the real short-lived BrandMeister login/auth/config test now? This sends UDP setup packets to the selected master but does not persist the network candidate or send DMR voice/data.');
  if (!confirmed) return;

  const password = document.querySelector('#network-password');
  const candidate = networkCandidate();
  setOperation('Running live BrandMeister handshake…', 'running');
  try {
    const result = await api('/api/v1/setup/network/test', {
      method: 'POST',
      body: JSON.stringify(candidate)
    });
    password.value = '';
    showResult('Live BrandMeister test', result);
    await loadDashboard();
  } catch (error) {
    password.value = '';
    showClientError('Live BrandMeister test', error);
  }
});

ui.commitButton.addEventListener('click', async () => {
  const confirmed = window.confirm('Test and COMMIT this BrandMeister candidate? YWD-DMR will run a fresh real login/auth/config handshake. Only if BrandMeister accepts that exact candidate will the daemon create a new known-good revision and store the Hotspot Security password in its restricted local secret store.');
  if (!confirmed) return;

  const password = document.querySelector('#network-password');
  const candidate = networkCandidate();
  setOperation('Testing and committing BrandMeister configuration…', 'running');
  try {
    const result = await api('/api/v1/setup/network/test-and-commit', {
      method: 'POST',
      body: JSON.stringify(candidate)
    });
    password.value = '';
    showResult('Test & commit BrandMeister network', result);
    await loadDashboard();
  } catch (error) {
    password.value = '';
    showClientError('Test & commit BrandMeister network', error);
  }
});

ui.refreshButton.addEventListener('click', loadDashboard);
ui.clearResult.addEventListener('click', () => {
  ui.result.textContent = 'No test has run yet.';
  setOperation('Ready.');
});

loadDashboard();
