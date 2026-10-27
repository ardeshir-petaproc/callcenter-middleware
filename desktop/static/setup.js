// @ts-check
// Connection screen logic. Talks to the main process only through the
// `window.ccSetup` bridge exposed by dist/preload/setup.js.
'use strict';

(() => {
  const api = window.ccSetup;

  /** @template {HTMLElement} T @param {string} id @returns {T} */
  const $ = (id) => /** @type {T} */ (document.getElementById(id));

  const form = /** @type {HTMLFormElement} */ ($('form'));
  const input = /** @type {HTMLInputElement} */ ($('url'));
  const status = $('status');
  const testBtn = /** @type {HTMLButtonElement} */ ($('test'));
  const connectBtn = /** @type {HTMLButtonElement} */ ($('connect'));
  const cancelBtn = /** @type {HTMLButtonElement} */ ($('cancel'));
  const lead = $('lead');
  const foot = $('foot');

  /** When the last save failed the health check, the next Connect saves anyway. */
  let forceNext = false;

  /** @param {'ok' | 'error' | 'info' | null} kind @param {string} [text] */
  function show(kind, text) {
    if (!kind) {
      status.hidden = true;
      status.textContent = '';
      return;
    }
    status.hidden = false;
    status.className = `status ${kind}`;
    status.textContent = text || '';
  }

  /** @param {boolean} busy */
  function setBusy(busy) {
    testBtn.disabled = busy;
    connectBtn.disabled = busy;
    input.readOnly = busy;
  }

  function resetForce() {
    forceNext = false;
    connectBtn.textContent = 'Connect';
  }

  input.addEventListener('input', () => {
    input.removeAttribute('aria-invalid');
    resetForce();
    show(null);
  });

  testBtn.addEventListener('click', async () => {
    setBusy(true);
    show('info', 'Checking…');
    try {
      const res = await api.check(input.value);
      if (res.ok) {
        if (res.url) input.value = res.url;
        show('ok', `Server reachable at ${res.url}`);
      } else {
        input.setAttribute('aria-invalid', 'true');
        show('error', res.error || 'Connection failed.');
      }
    } catch (err) {
      show('error', String(err));
    } finally {
      setBusy(false);
    }
  });

  form.addEventListener('submit', async (ev) => {
    ev.preventDefault();
    setBusy(true);
    show('info', forceNext ? 'Saving…' : 'Connecting…');
    try {
      const res = await api.save(input.value, forceNext);
      if (res.ok) {
        show('ok', `Connected. Opening ${res.url}…`);
        return; // the main process closes this window
      }
      if (res.url) input.value = res.url;
      input.setAttribute('aria-invalid', 'true');
      if (res.canForce) {
        forceNext = true;
        connectBtn.textContent = 'Connect anyway';
        show('error', `${res.error || 'Connection failed.'} Check the address, or choose “Connect anyway” if the server is temporarily down.`);
      } else {
        show('error', res.error || 'Invalid address.');
      }
    } catch (err) {
      show('error', String(err));
    } finally {
      setBusy(false);
    }
  });

  cancelBtn.addEventListener('click', () => api.cancel());
  document.addEventListener('keydown', (ev) => {
    if (ev.key === 'Escape' && !cancelBtn.hidden) api.cancel();
  });

  api
    .info()
    .then((info) => {
      if (info.serverUrl) input.value = info.serverUrl;
      if (!info.firstRun) {
        cancelBtn.hidden = false;
        lead.textContent = 'Change the call center server this app connects to.';
      }
      foot.textContent = `Call Center desktop ${info.version}`;
      input.select();
    })
    .catch(() => {});
})();
