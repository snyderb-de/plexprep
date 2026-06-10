import './style.css';
import './app.css';

// --- helpers ---------------------------------------------------------------
const $ = (id) => document.getElementById(id);
const RT = () => window.runtime || (window.parent && window.parent.runtime);
const APP = () => (window.go && window.go.main && window.go.main.App) || null;

function show(id) {
  document.querySelectorAll('.screen').forEach((s) => s.classList.toggle('active', s.id === id));
  $('go-home').hidden = id !== 's-report';
}

// --- settings (persisted) --------------------------------------------------
const settings = {
  crt: localStorage.getItem('pp.crt') || 'on',
  rain: localStorage.getItem('pp.rain') || 'on',
  sound: localStorage.getItem('pp.sound') || 'off',
};
function applySettings() {
  document.body.classList.remove('crt-off', 'crt-low', 'crt-on');
  document.body.classList.add('crt-' + settings.crt);
  document.body.classList.toggle('rain-off', settings.rain === 'off');
  syncSeg('set-crt', settings.crt);
  syncSeg('set-rain', settings.rain);
  syncSeg('set-sound', settings.sound);
}
function syncSeg(id, v) {
  const seg = $(id); if (!seg) return;
  seg.querySelectorAll('button').forEach((b) => b.classList.toggle('on', b.dataset.v === v));
}
function wireSeg(id, key, after) {
  const seg = $(id); if (!seg) return;
  seg.querySelectorAll('button').forEach((b) => b.addEventListener('click', () => {
    settings[key] = b.dataset.v; localStorage.setItem('pp.' + key, b.dataset.v); applySettings(); if (after) after();
  }));
}

// --- sound (optional blips) ------------------------------------------------
let actx = null;
function blip(freq = 660, ms = 40) {
  if (settings.sound !== 'on') return;
  try {
    actx = actx || new (window.AudioContext || window.webkitAudioContext)();
    const o = actx.createOscillator(), g = actx.createGain();
    o.type = 'square'; o.frequency.value = freq; o.connect(g); g.connect(actx.destination);
    g.gain.setValueAtTime(0.04, actx.currentTime);
    g.gain.exponentialRampToValueAtTime(0.0001, actx.currentTime + ms / 1000);
    o.start(); o.stop(actx.currentTime + ms / 1000);
  } catch (e) {}
}

// --- matrix rain -----------------------------------------------------------
function startRain() {
  const c = $('rain'), ctx = c.getContext('2d');
  const chars = '01░▒▓<>/\\|=+*ｱｶｻﾀﾅﾊﾏﾔﾗ'.split('');
  let cols, drops, W, H, fs = 14;
  function resize() {
    W = c.width = window.innerWidth; H = c.height = window.innerHeight;
    cols = Math.floor(W / fs); drops = new Array(cols).fill(0).map(() => Math.random() * -50);
  }
  resize(); window.addEventListener('resize', resize);
  function frame() {
    if (!document.body.classList.contains('rain-off')) {
      ctx.fillStyle = 'rgba(8,10,8,0.10)'; ctx.fillRect(0, 0, W, H);
      ctx.font = fs + 'px JetBrains Mono, monospace';
      for (let i = 0; i < cols; i++) {
        const ch = chars[(Math.random() * chars.length) | 0];
        const x = i * fs, y = drops[i] * fs;
        ctx.fillStyle = Math.random() > 0.975 ? '#a8ffc4' : '#3ad968';
        ctx.fillText(ch, x, y);
        if (y > H && Math.random() > 0.975) drops[i] = 0;
        drops[i] += 0.5;
      }
    }
    requestAnimationFrame(frame);
  }
  frame();
}

// --- boot sequence ---------------------------------------------------------
const LOGO = String.raw`
       _
 _ __ | | _____  ___ __  _ __ ___ _ __
| '_ \| |/ _ \ \/ / '_ \| '__/ _ \ '_ \
| |_) | |  __/>  <| |_) | | |  __/ |_) |
| .__/|_|\___/_/\_\ .__/|_|  \___| .__/
|_|               |_|            |_|          `;

const BOOT = [
  'plexprep v1 // zero-transcode media forge',
  '',
  '[ ok ] webview runtime ........... online',
  '[ ok ] ffmpeg bridge ............. ready',
  '[ ok ] media engine .............. loaded',
  '[ ok ] phosphor display .......... 1280x820',
  '',
  'booting interface_',
];
function boot() {
  const el = $('bootlog'); let out = '';
  let li = 0, ci = 0;
  (function step() {
    if (li >= BOOT.length) { setTimeout(toHome, 450); return; }
    const line = BOOT[li];
    if (ci < line.length) { out += line[ci++]; el.textContent = out; blip(880, 8); setTimeout(step, 8); }
    else { out += '\n'; el.textContent = out; li++; ci = 0; setTimeout(step, line === '' ? 30 : 90); }
  })();
}
function toHome() {
  $('logo').textContent = LOGO;
  renderRecent();
  show('s-home');
}

// --- recent paths ----------------------------------------------------------
function recents() { try { return JSON.parse(localStorage.getItem('pp.recent') || '[]'); } catch (e) { return []; } }
function pushRecent(p) {
  let r = recents().filter((x) => x !== p); r.unshift(p); r = r.slice(0, 6);
  localStorage.setItem('pp.recent', JSON.stringify(r)); renderRecent();
}
function renderRecent() {
  const box = $('recent'); const r = recents();
  box.innerHTML = r.length ? '<div class="dimx" style="margin-bottom:4px">recent:</div>' : '';
  r.forEach((p) => {
    const a = document.createElement('a'); a.textContent = '▸ ' + p;
    a.onclick = () => scanFolder(p);
    box.appendChild(a);
  });
}

// --- scanning --------------------------------------------------------------
let spinT = null;
function spin(on) {
  const el = $('scan-spin'); const frames = ['⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏']; let i = 0;
  if (spinT) clearInterval(spinT);
  if (on) spinT = setInterval(() => { el.textContent = frames[i++ % frames.length] + ' working'; }, 90);
  else el.textContent = '';
}
function resetScan(label) {
  $('scan-path').textContent = label;
  $('scan-done').textContent = '0'; $('scan-total').textContent = '?';
  $('scan-bar').style.width = '0%'; $('scan-name').textContent = '';
  $('scan-err').hidden = true; spin(true); show('s-scan');
}
function onScan(ev) {
  if (ev.t === 'begin') { $('scan-total').textContent = ev.total || '?'; }
  else if (ev.t === 'probe') {
    $('scan-done').textContent = ev.done;
    if (ev.total) $('scan-bar').style.width = Math.round(ev.done / ev.total * 100) + '%';
    $('scan-name').textContent = ev.name || ''; blip(520, 6);
  } else if (ev.t === 'error') { scanError(ev.msg); }
  else if (ev.t === 'done') { $('scan-bar').style.width = '100%'; }
}
function scanError(msg) {
  spin(false);
  const e = $('scan-err'); e.hidden = false;
  e.innerHTML = 'scan error: ' + msg + ' &nbsp; <a id="back-home">&larr; back</a>';
  $('back-home').onclick = () => show('s-home');
}
let scanCancelled = false;
async function scanFolder(path) {
  const app = APP(); if (!app) return;
  scanCancelled = false;
  resetScan(path);
  try {
    const html = await app.Scan(path, $('recursive').checked);
    if (scanCancelled) return;
    spin(false); pushRecent(path); openReport(html);
  } catch (e) { if (!scanCancelled) scanError(String(e)); }
}
async function scanFiles(paths) {
  const app = APP(); if (!app) return;
  scanCancelled = false;
  resetScan('(' + paths.length + ' files)');
  try {
    const html = await app.ScanFiles(paths);
    if (scanCancelled) return;
    spin(false); openReport(html);
  } catch (e) { if (!scanCancelled) scanError(String(e)); }
}
function cancelScan() {
  scanCancelled = true;
  spin(false);
  const app = APP(); if (app && app.AbortScan) app.AbortScan();
  show('s-home');
}
function openReport(html) {
  const f = $('rframe'); f.srcdoc = html; show('s-report');
}

// host hook the iframe calls when a convert run finishes ("done — close")
window.ppDone = function () { show('s-home'); blip(440, 80); };

// --- wire up ---------------------------------------------------------------
function wire() {
  // window controls
  $('win-close').onclick = () => RT() && RT().Quit();
  $('win-min').onclick = () => RT() && RT().WindowMinimise();
  $('win-max').onclick = () => RT() && RT().WindowToggleMaximise();

  // settings
  $('open-settings').onclick = () => { $('settings').hidden = false; };
  $('close-settings').onclick = () => { $('settings').hidden = true; };
  wireSeg('set-crt', 'crt'); wireSeg('set-rain', 'rain'); wireSeg('set-sound', 'sound', () => blip(660, 60));

  // home
  $('pick-folder').onclick = async () => {
    const app = APP(); if (!app) return;
    const p = await app.Browse(); if (p) scanFolder(p);
  };
  $('pick-files').onclick = async () => {
    const app = APP(); if (!app) return;
    const fs = await app.BrowseFiles(); if (fs && fs.length) scanFiles(fs);
  };
  $('scan-cancel').onclick = cancelScan;
  $('go-home').onclick = () => show('s-home');

  // backend events
  if (RT() && RT().EventsOn) RT().EventsOn('pp:scan', onScan);
}

applySettings();
wire();
startRain();
boot();
