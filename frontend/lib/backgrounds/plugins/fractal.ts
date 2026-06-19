import type { BackgroundInitFn } from '../types';

/**
 * Plugin "fractal" — fractale Mandelbrot↔Julia hybride, zoom en plongée, via shader WebGL.
 *
 * Concept (v2 — "zoom qui rebuild", pas une vraie plongée géométrique) :
 * - Itération z = z² + c calculée PAR PIXEL dans un fragment shader (escape-time).
 * - Ensemble de Julia (z₀=pixel, c=graine). La GRAINE parcourt le bord de la cardioïde de
 *   Mandelbrot (tirée légèrement vers l'intérieur) → la Julia reste TOUJOURS connexe (jamais
 *   de "dust"/mess) et plein cadre. Sa métamorphose continue EST le "rebuild" : la structure
 *   entière se régénère sans cesse, au lieu de magnifier les mêmes pixels.
 * - Zoom : respiration bornée (in/out doux) — jamais profond → reste net (pas de mur de
 *   précision float) et ne sort jamais vers une zone noire/vide.
 * - Souris : pan doux et borné autour du centre → on dirige un peu la vue sans tomber dans le vide.
 * - Palette vive (cosine palette), theme-aware. Intérieur de l'ensemble = transparent (les
 *   blobs aurora restent visibles), seuls les filaments colorés du bord sont rendus.
 * - Le morph continu Mandelbrot↔Julia de la v1 est abandonné : ses états intermédiaires
 *   produisaient un "mess" visuel. uMorph reste câblé (=1) pour réintroduire un hybride propre
 *   plus tard (crossfade entre états nets plutôt qu'interpolation continue).
 *
 * Perf : c'est le fond le plus lourd (per-pixel itératif). On rend en résolution interne
 * réduite (RES_SCALE) + DPR plafonné + itérations bornées. Pour un fond flou c'est invisible.
 *
 * Contrat coquille : pas de requestAnimationFrame ici (frame(dt) appelé par BackgroundHost).
 */

// ----------------------------------------------------------------------------
// Constantes de réglage
// ----------------------------------------------------------------------------
const RES_SCALE = 0.6; // résolution interne (fraction du viewport × dpr) — fond flou OK
const DPR_CAP = 1.5; // plafond dpr pour ce shader lourd
// Zoom respirant BORNÉ (pas de plongée profonde → reste net, jamais de zone vide).
const SCALE_MAX = 2.8; // dézoomé : l'ensemble de Julia tient en entier
const SCALE_MIN = 1.1; // zoomé : détail central, encore plein de structure
const BREATH_MS = 26000; // période de la respiration du zoom
// Graine Julia parcourant le bord de la cardioïde → Julia toujours connexe + métamorphose.
const SEED_PERIOD_MS = 60000; // tour complet du bord en 60s (rebuild lent et continu)
const SEED_INSET = 0.06; // fraction tirée vers l'intérieur de M → garantit la connexité
const CENTER_EASE = 0.02; // douceur du pan vers la cible souris
const PAN_MAX = 0.45; // amplitude max du pan (fraction de scale) → reste sur la structure
const BASE_ITER = 180; // une Julia connexe a besoin d'assez d'itérations
const EXTRA_ITER = 60; // itérations supplémentaires quand on zoome
const MAX_ITER = 256; // borne dure (doit matcher le #define du shader)
const BASE_ALPHA_DARK = 0.8;
const BASE_ALPHA_LIGHT = 0.62;
const COLOR_SPEED = 0.025; // vitesse de défilement de la palette

const VERT_SRC = `
attribute vec2 aPos;
void main() { gl_Position = vec4(aPos, 0.0, 1.0); }
`;

const FRAG_SRC = `
precision highp float;
uniform vec2 uRes;
uniform vec2 uCenter;
uniform float uScale;
uniform vec2 uSeed;
uniform float uMorph;
uniform int uIter;
uniform float uTime;
uniform float uAlpha;
uniform float uDark;
#define MAX_ITER 256

// Cosine palette (Inigo Quilez) — vive mais cohérente, dominante bleu/teal/violet.
vec3 pal(float t) {
  return 0.5 + 0.5 * cos(6.28318 * (vec3(1.0) * t + vec3(0.0, 0.15, 0.30)));
}

void main() {
  // pixel → plan complexe (normalisé en y, centré, aspect-correct)
  vec2 uv = (gl_FragCoord.xy - 0.5 * uRes) / uRes.y;
  vec2 p = uCenter + uv * uScale;

  // Hybride : morph=0 → Mandelbrot (z0=0, c=p) ; morph=1 → Julia (z0=p, c=graine)
  vec2 z = mix(vec2(0.0), p, uMorph);
  vec2 c = mix(p, uSeed, uMorph);

  float n = 0.0;
  bool esc = false;
  float mag = 0.0;
  for (int i = 0; i < MAX_ITER; i++) {
    if (i >= uIter) break;
    z = vec2(z.x * z.x - z.y * z.y, 2.0 * z.x * z.y) + c;
    mag = dot(z, z);
    if (mag > 256.0) { esc = true; break; }
    n += 1.0;
  }

  // Intérieur de l'ensemble → transparent (laisse voir l'aurora dessous).
  if (!esc) { gl_FragColor = vec4(0.0); return; }

  // Coloration lisse continue (smooth iteration count) → pas de banding.
  float mu = n + 1.0 - log2(log2(mag));
  float t = fract(mu * 0.045 + uTime * ${COLOR_SPEED.toFixed(3)});
  vec3 col = pal(t);
  // Adoucir légèrement sur thème clair (sinon trop saturé sur fond blanc).
  col = mix(col * 0.85 + 0.08, col, uDark);
  gl_FragColor = vec4(col, uAlpha);
}
`;

const initFractal: BackgroundInitFn = async (ctx) => {
  const { mount } = ctx;
  const dpr = Math.min(ctx.dpr, DPR_CAP);

  // --- Canvas WebGL plein écran ---
  const canvas = document.createElement('canvas');
  canvas.style.position = 'absolute';
  canvas.style.inset = '0';
  canvas.style.width = '100%';
  canvas.style.height = '100%';
  canvas.style.display = 'block';
  mount.appendChild(canvas);

  // preserveDrawingBuffer:true → permet la lecture pixels (tests) ; coût négligeable pour 1 quad.
  const gl = (canvas.getContext('webgl', {
    alpha: true,
    premultipliedAlpha: false,
    antialias: false,
    preserveDrawingBuffer: true,
  }) ||
    canvas.getContext('experimental-webgl')) as WebGLRenderingContext | null;
  // Échec franc : pas de fond silencieusement absent qui masquerait un problème.
  if (!gl) throw new Error('[fractal] WebGL indisponible');

  let width = ctx.width;
  let height = ctx.height;

  // --- Compilation shaders + programme ---
  const compile = (type: number, src: string): WebGLShader => {
    const sh = gl.createShader(type)!;
    gl.shaderSource(sh, src);
    gl.compileShader(sh);
    if (!gl.getShaderParameter(sh, gl.COMPILE_STATUS)) {
      const log = gl.getShaderInfoLog(sh);
      gl.deleteShader(sh);
      throw new Error('[fractal] shader compile: ' + log);
    }
    return sh;
  };

  const vert = compile(gl.VERTEX_SHADER, VERT_SRC);
  const frag = compile(gl.FRAGMENT_SHADER, FRAG_SRC);
  const program = gl.createProgram()!;
  gl.attachShader(program, vert);
  gl.attachShader(program, frag);
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    const log = gl.getProgramInfoLog(program);
    throw new Error('[fractal] program link: ' + log);
  }
  gl.useProgram(program);

  // --- Quad plein écran (2 triangles couvrant le clip space) ---
  const buffer = gl.createBuffer()!;
  gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
  gl.bufferData(
    gl.ARRAY_BUFFER,
    new Float32Array([-1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1]),
    gl.STATIC_DRAW
  );
  const aPos = gl.getAttribLocation(program, 'aPos');
  gl.enableVertexAttribArray(aPos);
  gl.vertexAttribPointer(aPos, 2, gl.FLOAT, false, 0, 0);

  // --- Uniform locations ---
  const u = {
    res: gl.getUniformLocation(program, 'uRes'),
    center: gl.getUniformLocation(program, 'uCenter'),
    scale: gl.getUniformLocation(program, 'uScale'),
    seed: gl.getUniformLocation(program, 'uSeed'),
    morph: gl.getUniformLocation(program, 'uMorph'),
    iter: gl.getUniformLocation(program, 'uIter'),
    time: gl.getUniformLocation(program, 'uTime'),
    alpha: gl.getUniformLocation(program, 'uAlpha'),
    dark: gl.getUniformLocation(program, 'uDark'),
  };

  // (Ré)applique la taille du backing buffer en résolution interne réduite.
  const applySize = () => {
    canvas.width = Math.max(1, Math.floor(width * dpr * RES_SCALE));
    canvas.height = Math.max(1, Math.floor(height * dpr * RES_SCALE));
    gl.viewport(0, 0, canvas.width, canvas.height);
  };
  applySize();

  const isDark = () => document.documentElement.classList.contains('dark');

  // État animé
  const TWO_PI = Math.PI * 2;
  let elapsed = 0; // ms cumulées (pas d'horloge absolue)
  let curScale = SCALE_MAX;
  const center = { x: 0, y: 0 }; // Julia est centrée à l'origine et symétrique
  const target = { x: 0, y: 0 }; // cible du centre (pan dirigé par la souris)

  // --- Frame : avance l'animation + dessine ---
  const frame = (dt: number) => {
    elapsed += dt;

    // Zoom respirant borné (cosine SCALE_MAX→SCALE_MIN→SCALE_MAX) : net, sans zone vide.
    const br = 0.5 - 0.5 * Math.cos(TWO_PI * ((elapsed % BREATH_MS) / BREATH_MS));
    curScale = SCALE_MAX + (SCALE_MIN - SCALE_MAX) * br;

    // Graine Julia le long du bord de la cardioïde de M, tirée vers l'intérieur (-0.25,0) →
    // Julia TOUJOURS connexe (pas de dust) et plein cadre. Sa rotation = le "rebuild" continu.
    const th = TWO_PI * ((elapsed % SEED_PERIOD_MS) / SEED_PERIOD_MS);
    const bx = 0.5 * Math.cos(th) - 0.25 * Math.cos(2 * th);
    const by = 0.5 * Math.sin(th) - 0.25 * Math.sin(2 * th);
    const seedX = bx + (-0.25 - bx) * SEED_INSET;
    const seedY = by + (0 - by) * SEED_INSET;

    // Glissement doux du centre vers la cible (pan souris borné).
    center.x += (target.x - center.x) * CENTER_EASE;
    center.y += (target.y - center.y) * CENTER_EASE;

    // Itérations qui montent un peu quand on zoome.
    const iter = Math.min(MAX_ITER, Math.round(BASE_ITER + br * EXTRA_ITER));
    const dark = isDark();
    const alpha = dark ? BASE_ALPHA_DARK : BASE_ALPHA_LIGHT;

    gl.uniform2f(u.res, canvas.width, canvas.height);
    gl.uniform2f(u.center, center.x, center.y);
    gl.uniform1f(u.scale, curScale);
    gl.uniform2f(u.seed, seedX, seedY);
    gl.uniform1f(u.morph, 1.0); // pure Julia (le morph continu produisait du "mess")
    gl.uniform1i(u.iter, iter);
    gl.uniform1f(u.time, elapsed * 0.001);
    gl.uniform1f(u.alpha, alpha);
    gl.uniform1f(u.dark, dark ? 1.0 : 0.0);

    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT);
    gl.drawArrays(gl.TRIANGLES, 0, 6);
  };

  // --- Resize ---
  const resize = (w: number, h: number) => {
    width = w;
    height = h;
    applySize();
  };

  // --- Souris : pan doux et BORNÉ autour du centre (on dirige la vue sans sortir vers le vide) ---
  const onPointerMove = (clientX: number, clientY: number) => {
    // Reproduit le mapping du shader (y normalisé, centré, y vers le haut).
    const uvx = (clientX - width / 2) / height;
    const uvy = (height / 2 - clientY) / height;
    const lim = PAN_MAX * curScale;
    target.x = Math.max(-lim, Math.min(lim, uvx * curScale * 0.7));
    target.y = Math.max(-lim, Math.min(lim, uvy * curScale * 0.7));
  };

  // --- Libération GPU complète ---
  const dispose = () => {
    gl.deleteBuffer(buffer);
    gl.deleteProgram(program);
    gl.deleteShader(vert);
    gl.deleteShader(frag);
    // Force la libération du contexte GPU (sinon le navigateur garde le contexte vivant).
    gl.getExtension('WEBGL_lose_context')?.loseContext();
    if (canvas.parentNode === mount) mount.removeChild(canvas);
  };

  return { frame, resize, onPointerMove, dispose };
};

export default initFractal;
