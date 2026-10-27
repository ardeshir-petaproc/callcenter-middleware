/**
 * Programmatic icon generation.
 *
 * Instead of shipping binary image blobs, the app draws its icons at runtime
 * (tray, window, Windows overlay badge) and at build time (build/icon.png for
 * electron-builder) using a tiny anti-aliased shape rasterizer and a minimal
 * PNG encoder built on node:zlib. Pure Node — no Electron imports — so the
 * build script can use it too.
 */
import zlib from 'node:zlib';

export type RGBA = readonly [number, number, number, number];

/** Status colors used for the dot drawn on the tray icon. */
export const STATUS_COLORS: Record<string, RGBA> = {
  available: [22, 163, 74, 255], // green
  ringing: [37, 99, 235, 255], // blue
  busy: [220, 38, 38, 255], // red
  wrapup: [147, 51, 234, 255], // purple
  paused: [217, 119, 6, 255], // amber
  offline: [113, 113, 122, 255], // grey
};

const BRAND: RGBA = [37, 99, 235, 255];
const BRAND_DARK: RGBA = [29, 78, 216, 255];
const WHITE: RGBA = [255, 255, 255, 255];

// ---------------------------------------------------------------------------
// PNG encoding
// ---------------------------------------------------------------------------

const CRC_TABLE = (() => {
  const t = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    t[n] = c >>> 0;
  }
  return t;
})();

function crc32(buf: Buffer): number {
  let c = 0xffffffff;
  for (let i = 0; i < buf.length; i++) c = CRC_TABLE[(c ^ buf[i]) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}

function chunk(type: string, data: Buffer): Buffer {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length, 0);
  const td = Buffer.concat([Buffer.from(type, 'ascii'), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(td), 0);
  return Buffer.concat([len, td, crc]);
}

/** Encodes non-premultiplied RGBA pixels as an 8-bit truecolor+alpha PNG. */
export function encodePng(width: number, height: number, rgba: Buffer): Buffer {
  if (rgba.length !== width * height * 4) throw new Error('encodePng: buffer size mismatch');
  const stride = width * 4;
  const raw = Buffer.alloc((stride + 1) * height);
  for (let y = 0; y < height; y++) {
    raw[y * (stride + 1)] = 0; // filter: none
    rgba.copy(raw, y * (stride + 1) + 1, y * stride, (y + 1) * stride);
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8; // bit depth
  ihdr[9] = 6; // color type RGBA
  ihdr[10] = 0;
  ihdr[11] = 0;
  ihdr[12] = 0;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk('IHDR', ihdr),
    chunk('IDAT', zlib.deflateSync(raw, { level: 9 })),
    chunk('IEND', Buffer.alloc(0)),
  ]);
}

// ---------------------------------------------------------------------------
// Shape rasterizer (unit coordinates: 0..1 on both axes)
// ---------------------------------------------------------------------------

type Inside = (x: number, y: number) => boolean;
interface Layer {
  inside: Inside;
  color: RGBA | ((x: number, y: number) => RGBA);
}

const circle =
  (cx: number, cy: number, r: number): Inside =>
  (x, y) =>
    (x - cx) ** 2 + (y - cy) ** 2 <= r * r;

const upperArc =
  (cx: number, cy: number, r: number, thickness: number): Inside =>
  (x, y) =>
    y <= cy && Math.abs(Math.hypot(x - cx, y - cy) - r) <= thickness / 2;

const roundRect =
  (cx: number, cy: number, w: number, h: number, r: number): Inside =>
  (x, y) => {
    const qx = Math.abs(x - cx) - (w / 2 - r);
    const qy = Math.abs(y - cy) - (h / 2 - r);
    const outside = Math.hypot(Math.max(qx, 0), Math.max(qy, 0));
    return outside + Math.min(Math.max(qx, qy), 0) - r <= 0;
  };

const capsule =
  (ax: number, ay: number, bx: number, by: number, r: number): Inside =>
  (x, y) => {
    const px = x - ax;
    const py = y - ay;
    const dx = bx - ax;
    const dy = by - ay;
    const t = Math.max(0, Math.min(1, (px * dx + py * dy) / (dx * dx + dy * dy)));
    return Math.hypot(px - dx * t, py - dy * t) <= r;
  };

function rasterize(size: number, layers: Layer[], samples = 4): Buffer {
  const out = Buffer.alloc(size * size * 4);
  const n = samples * samples;
  for (let py = 0; py < size; py++) {
    for (let px = 0; px < size; px++) {
      let r = 0;
      let g = 0;
      let b = 0;
      let a = 0;
      for (let sy = 0; sy < samples; sy++) {
        for (let sx = 0; sx < samples; sx++) {
          const x = (px + (sx + 0.5) / samples) / size;
          const y = (py + (sy + 0.5) / samples) / size;
          // Painter's algorithm: the top-most layer containing the sample wins.
          for (let i = layers.length - 1; i >= 0; i--) {
            const layer = layers[i];
            if (!layer.inside(x, y)) continue;
            const c = typeof layer.color === 'function' ? layer.color(x, y) : layer.color;
            const ca = c[3] / 255;
            r += c[0] * ca;
            g += c[1] * ca;
            b += c[2] * ca;
            a += ca;
            break;
          }
        }
      }
      const o = (py * size + px) * 4;
      if (a > 0) {
        out[o] = Math.round(r / a);
        out[o + 1] = Math.round(g / a);
        out[o + 2] = Math.round(b / a);
        out[o + 3] = Math.round((a / n) * 255);
      }
    }
  }
  return out;
}

function mix(c1: RGBA, c2: RGBA, t: number): RGBA {
  const k = Math.max(0, Math.min(1, t));
  return [
    Math.round(c1[0] + (c2[0] - c1[0]) * k),
    Math.round(c1[1] + (c2[1] - c1[1]) * k),
    Math.round(c1[2] + (c2[2] - c1[2]) * k),
    Math.round(c1[3] + (c2[3] - c1[3]) * k),
  ];
}

export interface IconOptions {
  /** Optional status dot in the lower-right corner. */
  dot?: RGBA | null;
}

/** RGBA pixels of the app icon: a blue disc with a white headset. */
export function drawAppIcon(size: number, opts: IconOptions = {}): Buffer {
  const layers: Layer[] = [
    { inside: circle(0.5, 0.5, 0.47), color: (_x, y) => mix(BRAND, BRAND_DARK, y) },
    // headband
    { inside: upperArc(0.5, 0.54, 0.235, 0.075), color: WHITE },
    // ear cups
    { inside: roundRect(0.265, 0.585, 0.13, 0.22, 0.055), color: WHITE },
    { inside: roundRect(0.735, 0.585, 0.13, 0.22, 0.055), color: WHITE },
    // microphone boom + tip
    { inside: capsule(0.7, 0.68, 0.56, 0.77, 0.028), color: WHITE },
    { inside: circle(0.53, 0.775, 0.05), color: WHITE },
  ];
  if (opts.dot) {
    layers.push({ inside: circle(0.79, 0.79, 0.205), color: WHITE });
    layers.push({ inside: circle(0.79, 0.79, 0.15), color: opts.dot });
  }
  return rasterize(size, layers, size >= 128 ? 3 : 4);
}

/** PNG of the app icon at the given pixel size. */
export function appIconPng(size: number, opts: IconOptions = {}): Buffer {
  return encodePng(size, size, drawAppIcon(size, opts));
}

/** PNG of a small solid badge disc (used as Windows taskbar overlay). */
export function badgePng(size = 16, color: RGBA = [220, 38, 38, 255]): Buffer {
  const px = rasterize(size, [
    { inside: circle(0.5, 0.5, 0.5), color: WHITE },
    { inside: circle(0.5, 0.5, 0.38), color },
  ]);
  return encodePng(size, size, px);
}
