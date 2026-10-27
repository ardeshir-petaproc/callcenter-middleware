// Synthesised ringtone (Web Audio, no binary assets) + browser notifications.
let ctx: AudioContext | null = null;
let ringTimer: number | undefined;

function audio(): AudioContext | null {
  if (ctx) return ctx;
  const Ctor = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctor) return null;
  ctx = new Ctor();
  return ctx;
}

/** Call from a user gesture so later playback is allowed by autoplay policy. */
export function unlockAudio() {
  const a = audio();
  if (a && a.state === 'suspended') void a.resume();
}

function burst(a: AudioContext, start: number, dur: number) {
  // Classic dual-tone ring (440 Hz + 480 Hz), amplitude-enveloped.
  const gain = a.createGain();
  gain.gain.setValueAtTime(0, start);
  gain.gain.linearRampToValueAtTime(0.12, start + 0.03);
  gain.gain.setValueAtTime(0.12, start + dur - 0.05);
  gain.gain.linearRampToValueAtTime(0, start + dur);
  gain.connect(a.destination);
  for (const f of [440, 480]) {
    const o = a.createOscillator();
    o.type = 'sine';
    o.frequency.value = f;
    o.connect(gain);
    o.start(start);
    o.stop(start + dur + 0.02);
  }
}

export function startRinging() {
  const a = audio();
  if (!a || ringTimer !== undefined) return;
  if (a.state === 'suspended') void a.resume();
  const ring = () => {
    const t = a.currentTime + 0.02;
    burst(a, t, 0.4);
    burst(a, t + 0.6, 0.4);
  };
  ring();
  ringTimer = window.setInterval(ring, 3000);
}

export function stopRinging() {
  if (ringTimer !== undefined) { clearInterval(ringTimer); ringTimer = undefined; }
}

export function playBeep(freq = 880, dur = 0.12) {
  const a = audio();
  if (!a) return;
  const t = a.currentTime + 0.01;
  const g = a.createGain();
  g.gain.setValueAtTime(0.08, t);
  g.gain.exponentialRampToValueAtTime(0.0001, t + dur);
  g.connect(a.destination);
  const o = a.createOscillator();
  o.frequency.value = freq;
  o.connect(g);
  o.start(t);
  o.stop(t + dur);
}

export function notificationsSupported(): boolean { return typeof Notification !== 'undefined'; }

export async function requestNotifications(): Promise<NotificationPermission | 'unsupported'> {
  if (!notificationsSupported()) return 'unsupported';
  if (Notification.permission === 'default') return Notification.requestPermission();
  return Notification.permission;
}

export function notify(title: string, body: string, onClick?: () => void) {
  if (!notificationsSupported() || Notification.permission !== 'granted') return;
  try {
    const n = new Notification(title, { body, tag: 'cc-screenpop', requireInteraction: false });
    n.onclick = () => { window.focus(); onClick?.(); n.close(); };
  } catch { /* some browsers only allow via service worker */ }
}
