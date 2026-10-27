// @ts-check
// Shown in the main window when the panel fails to load; retries automatically.
'use strict';

(() => {
  const params = new URLSearchParams(location.search);
  const raw = params.get('url') || '';
  /** @type {string | null} */
  let target = null;
  try {
    const u = new URL(raw);
    if (u.protocol === 'http:' || u.protocol === 'https:') target = u.toString();
  } catch {
    /* invalid */
  }

  const targetEl = /** @type {HTMLElement} */ (document.getElementById('target'));
  const errorEl = /** @type {HTMLElement} */ (document.getElementById('error'));
  const countdownEl = /** @type {HTMLElement} */ (document.getElementById('countdown'));
  const retryBtn = /** @type {HTMLButtonElement} */ (document.getElementById('retry'));

  targetEl.textContent = target || '';
  errorEl.textContent = params.get('error') || '';

  const retry = () => {
    if (target) location.href = target;
  };
  retryBtn.addEventListener('click', retry);
  retryBtn.disabled = !target;

  if (target) {
    let left = 15;
    const tick = () => {
      countdownEl.textContent = `Retrying automatically in ${left} s…`;
      if (left-- <= 0) retry();
      else setTimeout(tick, 1000);
    };
    tick();
  }
})();
