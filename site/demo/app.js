/* RTSPanda interactive demo. Simulated cameras and detection, entirely in the browser. No network, no real video. */
(function () {
  'use strict';
  const EMAIL = 'demo@rtspanda.test', PASS = 'DemoAdmin!2026';
  const W = 640, H = 360;
  const $ = (s, r) => (r || document).querySelector(s), $$ = (s, r) => Array.from((r || document).querySelectorAll(s));
  const esc = (v) => String(v).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const rnd = (a, b) => a + Math.random() * (b - a), pick = (a) => a[Math.floor(Math.random() * a.length)];
  const store = { get(k) { try { return sessionStorage.getItem(k); } catch (e) { return null; } }, set(k, v) { try { sessionStorage.setItem(k, v); } catch (e) { /* ignore */ } }, del(k) { try { sessionStorage.removeItem(k); } catch (e) { /* ignore */ } } };
  const app = $('#app');
  const COL = { person: '#22c55e', car: '#38bdf8', animal: '#f59e0b' };

  const cfg = { mode: 'standard', aiMode: 'local', snapshotAI: false, provider: 'claude', discord: 'https://discord.com/api/webhooks/000000/DEMO-ONLY', autoRecord: true, retention: 14 };
  const cams = [
    { id: 'c1', name: 'Driveway', kind: 'driveway', online: true, res: '1920×1080', br: 3.8 },
    { id: 'c2', name: 'Front Door', kind: 'porch', online: true, res: '2560×1440', br: 5.1 },
    { id: 'c3', name: 'Backyard', kind: 'yard', online: true, res: '1920×1080', br: 3.2 },
    { id: 'c4', name: 'Garage', kind: 'garage', online: false, res: '1280×720', br: 1.9 },
  ];
  cams.forEach((c) => { c.cv = document.createElement('canvas'); c.cv.width = W; c.cv.height = H; c.actors = []; c.dets = []; c.zones = []; c.conf = 0.5; c.detect = true; c.classes = { person: true, car: true, animal: true }; c.rec = null; c.nextSpawn = performance.now() + rnd(800, 2500); c.fps = 0; c.seed = Math.random() * 1000; c.rtsp = 'rtsp://demo:••••@192.0.2.' + (10 + cams.indexOf(c)) + ':554/stream1'; });
  const events = [], recordings = [];
  let state = { drawMode: false, showBoxes: true };

  function toast(m) { let b = $('.toast'); if (!b) { b = document.createElement('div'); b.className = 'toast'; b.setAttribute('role', 'status'); document.body.appendChild(b); } const d = document.createElement('div'); d.textContent = m; b.appendChild(d); setTimeout(() => d.remove(), 3600); }
  function modal(html, mount) { const bg = document.createElement('div'); bg.className = 'modal-bg'; bg.innerHTML = `<div class="modal" role="dialog" aria-modal="true">${html}</div>`; bg.addEventListener('click', (e) => { if (e.target === bg) { bg.remove(); if (bg.onclose) bg.onclose(); } }); document.addEventListener('keydown', function k(e) { if (e.key === 'Escape') { bg.remove(); document.removeEventListener('keydown', k); } }); document.body.appendChild(bg); $$('[data-close]', bg).forEach((b) => b.addEventListener('click', () => bg.remove())); if (mount) mount(bg); return bg; }

  /* ---------- scene rendering ---------- */
  function groundY(c) { return { driveway: 0.66, porch: 0.82, yard: 0.72, garage: 0.78 }[c.kind] * H; }
  function drawScene(c, t) {
    const g = c.cv.getContext('2d'), k = c.kind, gy = groundY(c);
    const sky = g.createLinearGradient(0, 0, 0, H * 0.5);
    if (k === 'garage') { g.fillStyle = '#1a1c1a'; g.fillRect(0, 0, W, H); } else { sky.addColorStop(0, '#9cc4e8'); sky.addColorStop(1, '#dbe9f4'); g.fillStyle = sky; g.fillRect(0, 0, W, H); }
    if (k === 'driveway') {
      for (let i = 0; i < 9; i++) { g.fillStyle = i % 2 ? '#2f5a34' : '#3b6b3f'; g.beginPath(); g.arc(30 + i * 75, gy * 0.62, 55 + (i % 3) * 14, 0, 7); g.fill(); }
      g.fillStyle = '#8a8f98'; g.fillRect(W * 0.7, gy * 0.35, W * 0.3, gy * 0.65); g.fillStyle = '#5b3a2e'; g.beginPath(); g.moveTo(W * 0.68, gy * 0.37); g.lineTo(W * 0.85, gy * 0.14); g.lineTo(W * 1.02, gy * 0.37); g.fill();
      g.fillStyle = '#4a7a3c'; g.fillRect(0, gy * 0.9, W, H); g.fillStyle = '#55565c'; g.beginPath(); g.moveTo(W * 0.1, H); g.lineTo(W * 0.38, gy * 0.92); g.lineTo(W * 0.74, gy * 0.92); g.lineTo(W * 0.96, H); g.fill();
      g.fillStyle = '#7a2e2e'; g.fillRect(W * 0.08, gy + 8, 120, 42); g.fillStyle = '#9cc'; g.fillRect(W * 0.1, gy + 2, 70, 14); g.fillStyle = '#111'; g.beginPath(); g.arc(W * 0.08 + 24, gy + 52, 11, 0, 7); g.arc(W * 0.08 + 98, gy + 52, 11, 0, 7); g.fill();
    } else if (k === 'porch') {
      g.fillStyle = '#c9b89a'; g.fillRect(0, 0, W, H); g.fillStyle = '#6b4a35'; g.fillRect(W * 0.38, H * 0.14, W * 0.24, H * 0.68); g.fillStyle = '#4e3626'; g.fillRect(W * 0.41, H * 0.2, W * 0.18, H * 0.34); g.fillStyle = '#e8c66a'; g.beginPath(); g.arc(W * 0.66, H * 0.26, 7, 0, 7); g.fill();
      g.fillStyle = '#8a7b66'; g.fillRect(0, gy, W, H); g.fillStyle = '#3d6b3a'; [0.12, 0.88].forEach((p) => { g.beginPath(); g.arc(W * p, H * 0.7, 36, 0, 7); g.fill(); }); g.fillStyle = '#5a4a3a'; g.fillRect(W * 0.42, gy + 10, W * 0.16, 22);
    } else if (k === 'yard') {
      g.fillStyle = '#4c8a43'; g.fillRect(0, H * 0.45, W, H); g.fillStyle = '#6b4f33'; for (let x = 0; x < W; x += 34) g.fillRect(x, H * 0.36, 22, H * 0.2); g.fillStyle = '#7a5c3a'; g.fillRect(0, H * 0.4, W, 8);
      g.fillStyle = '#2d5a30'; g.beginPath(); g.arc(W * 0.8, H * 0.28, 70, 0, 7); g.fill(); g.fillStyle = '#5b4026'; g.fillRect(W * 0.78, H * 0.3, 16, H * 0.3); g.fillStyle = '#9a9ea6'; g.fillRect(W * 0.1, H * 0.22, 110, 80);
    } else {
      g.fillStyle = '#272a27'; g.fillRect(0, gy * 0.5, W, H); g.fillStyle = '#343834'; for (let i = 0; i < 4; i++) g.fillRect(40 + i * 150, 40, 110, 12), g.fillRect(40 + i * 150, 120, 110, 12); g.fillStyle = '#3c413c'; g.fillRect(W * 0.3, gy - 30, 220, 60);
    }
    // shade, noise, vignette
    g.fillStyle = 'rgba(0,0,0,.05)'; g.fillRect(0, 0, W, H);
    for (let i = 0; i < 70; i++) { g.fillStyle = 'rgba(255,255,255,' + rnd(0.03, 0.12) + ')'; g.fillRect(Math.random() * W, Math.random() * H, 2, 2); }
    // actors
    c.actors.forEach((a) => drawActor(g, a, t, k === 'garage'));
    if (k === 'garage') { g.globalCompositeOperation = 'multiply'; g.fillStyle = '#7dff9a'; g.fillRect(0, 0, W, H); g.globalCompositeOperation = 'source-over'; }
    const v = g.createRadialGradient(W / 2, H / 2, H * 0.4, W / 2, H / 2, H * 0.95); v.addColorStop(0, 'rgba(0,0,0,0)'); v.addColorStop(1, 'rgba(0,0,0,.45)'); g.fillStyle = v; g.fillRect(0, 0, W, H);
  }
  function drawActor(g, a, t, ir) {
    const s = a.h, x = a.x, y = a.y; g.save();
    if (a.cls === 'person') {
      const sw = Math.sin(t / 150 + a.id) * s * 0.12;
      g.strokeStyle = a.leg; g.lineWidth = s * 0.07; g.lineCap = 'round'; g.beginPath(); g.moveTo(x, y - s * 0.4); g.lineTo(x + sw, y); g.moveTo(x, y - s * 0.4); g.lineTo(x - sw, y); g.stroke();
      g.fillStyle = a.col; g.fillRect(x - s * 0.14, y - s * 0.78, s * 0.28, s * 0.4); g.fillStyle = '#e0b48f'; g.beginPath(); g.arc(x, y - s * 0.88, s * 0.1, 0, 7); g.fill();
    } else if (a.cls === 'car') {
      g.fillStyle = a.col; g.fillRect(x - a.w / 2, y - s * 0.55, a.w, s * 0.4); g.beginPath(); g.moveTo(x - a.w * 0.28, y - s * 0.55); g.lineTo(x - a.w * 0.16, y - s * 0.9); g.lineTo(x + a.w * 0.2, y - s * 0.9); g.lineTo(x + a.w * 0.32, y - s * 0.55); g.fill();
      g.fillStyle = 'rgba(180,220,240,.8)'; g.fillRect(x - a.w * 0.14, y - s * 0.85, a.w * 0.32, s * 0.26); g.fillStyle = '#111'; [-0.3, 0.3].forEach((p) => { g.beginPath(); g.arc(x + a.w * p, y - s * 0.14, s * 0.17, 0, 7); g.fill(); });
      g.fillStyle = '#fff6c0'; g.fillRect(a.vx > 0 ? x + a.w / 2 - 6 : x - a.w / 2, y - s * 0.42, 6, 6);
    } else {
      const hop = Math.abs(Math.sin(t / 160 + a.id)) * 3; g.fillStyle = a.col; g.beginPath(); g.ellipse(x, y - s * 0.4 - hop, a.w * 0.4, s * 0.25, 0, 0, 7); g.fill(); g.beginPath(); g.arc(x + (a.vx > 0 ? 1 : -1) * a.w * 0.36, y - s * 0.55 - hop, s * 0.16, 0, 7); g.fill();
      g.strokeStyle = a.col; g.lineWidth = s * 0.07; g.beginPath(); [-0.25, 0.25].forEach((p) => { g.moveTo(x + a.w * p, y - s * 0.25); g.lineTo(x + a.w * p + Math.sin(t / 120) * 4, y); }); g.stroke();
    }
    g.restore();
  }
  function box(a) { return a.cls === 'car' ? { x: a.x - a.w / 2, y: a.y - a.h * 0.9, w: a.w, h: a.h * 0.9 } : a.cls === 'person' ? { x: a.x - a.h * 0.16, y: a.y - a.h * 0.98, w: a.h * 0.32, h: a.h * 0.98 } : { x: a.x - a.w * 0.5, y: a.y - a.h * 0.72, w: a.w, h: a.h * 0.72 }; }

  let aid = 0;
  function spawn(c) {
    const kinds = { driveway: ['person', 'car', 'car', 'person'], porch: ['person', 'person', 'animal'], yard: ['animal', 'animal', 'person'], garage: ['person'] }[c.kind];
    const cls = pick(kinds), dir = Math.random() < 0.5 ? 1 : -1, gy = groundY(c);
    const a = { id: ++aid, cls, vx: dir * (cls === 'car' ? rnd(110, 170) : cls === 'person' ? rnd(35, 70) : rnd(40, 80)), base: rnd(0.52, 0.96), last: 0 };
    a.y = gy + rnd(0, 26) * (cls === 'car' ? 0 : 1); a.h = cls === 'car' ? rnd(60, 80) : cls === 'person' ? rnd(110, 150) : rnd(40, 55); a.w = cls === 'car' ? rnd(130, 170) : cls === 'animal' ? rnd(56, 72) : 30;
    a.x = dir > 0 ? -a.w : W + a.w; a.col = cls === 'car' ? pick(['#2f6fbf', '#d0d4d8', '#222', '#b03030']) : cls === 'person' ? pick(['#c0392b', '#2c3e50', '#16a085', '#8e44ad']) : pick(['#8b5a2b', '#d7a56b', '#444']); a.leg = '#2a2f3a';
    if (cls === 'car') a.y = gy + 40; c.actors.push(a);
  }
  const iou = (a, b) => 0; void iou;
  function inZone(c, b) { const cx = (b.x + b.w / 2) / W, cy = (b.y + b.h / 2) / H; return c.zones.some((z) => cx >= z.x && cx <= z.x + z.w && cy >= z.y && cy <= z.y + z.h); }

  function addEvent(c, a, d) {
    const t = document.createElement('canvas'); t.width = 192; t.height = 108; const g = t.getContext('2d'); g.drawImage(c.cv, 0, 0, 192, 108);
    g.strokeStyle = COL[a.cls]; g.lineWidth = 2; g.strokeRect(d.box.x * 0.3, d.box.y * 0.3, d.box.w * 0.3, d.box.h * 0.3);
    const ev = { id: events.length + 1, t: new Date(), cam: c.id, cls: a.cls, conf: d.conf, img: t.toDataURL('image/jpeg', 0.7) }; events.unshift(ev); if (events.length > 80) events.pop();
    toast(`${a.cls[0].toUpperCase() + a.cls.slice(1)} detected · ${c.name} (${Math.round(d.conf * 100)}%)`);
    if (cfg.autoRecord) recordings.unshift({ id: 'r' + Date.now(), cam: c.id, start: new Date(), dur: Math.round(rnd(12, 28)), auto: true, cls: a.cls });
    if (recordings.length > 40) recordings.pop();
    if (['events', 'dashboard'].includes(route()[0]) || (route()[0] === 'cam' && route()[1] === c.id)) renderSide();
  }
  function detect(now) {
    if (cfg.mode === 'viewer') { cams.forEach((c) => (c.dets = [])); return; }
    const interval = cfg.mode === 'pi' ? 3000 : 250;
    cams.forEach((c) => {
      if (!c.online || !c.detect) { c.dets = []; return; }
      if (c.lastDet && now - c.lastDet < interval) return; c.lastDet = now;
      c.dets = [];
      c.actors.forEach((a) => {
        const b = box(a); if (b.x + b.w < 0 || b.x > W) return;
        const conf = Math.min(0.99, Math.max(0.3, a.base + rnd(-0.03, 0.03)));
        if (conf < c.conf || !c.classes[a.cls] || inZone(c, b)) return;
        const d = { id: a.id, cls: a.cls, conf, box: b }; c.dets.push(d);
        if (now - a.last > 12000) { a.last = now; addEvent(c, a, d); }
      });
    });
  }

  let lastTick = 0;
  function loop(now) {
    requestAnimationFrame(loop);
    if (now - lastTick < 66) return; const dt = Math.min(0.2, (now - lastTick) / 1000); lastTick = now;
    cams.forEach((c) => {
      if (!c.online) return;
      if (now > c.nextSpawn && c.actors.length < 3) { spawn(c); c.nextSpawn = now + rnd(4000, 9000); }
      c.actors.forEach((a) => { a.x += a.vx * dt; }); c.actors = c.actors.filter((a) => a.x > -220 && a.x < W + 220);
      drawScene(c, now); c.fps = 14.5 + Math.sin(now / 900 + c.seed) * 0.6;
    });
    detect(now);
    $$('canvas[data-cam]').forEach((cv) => paint(cv, cams.find((c) => c.id === cv.dataset.cam), now));
  }
  function paint(cv, c, now) {
    const g = cv.getContext('2d'); if (cv.width !== W) { cv.width = W; cv.height = H; }
    if (!c.online) { g.fillStyle = '#0a0a0d'; g.fillRect(0, 0, W, H); for (let i = 0; i < 400; i++) { g.fillStyle = 'rgba(255,255,255,' + rnd(0, 0.14) + ')'; g.fillRect(Math.random() * W, Math.random() * H, 3, 3); } g.fillStyle = '#9a9aa8'; g.font = '600 20px sans-serif'; g.textAlign = 'center'; g.fillText('NO SIGNAL', W / 2, H / 2); g.font = '13px sans-serif'; g.fillText(c.name + ' · reconnecting disabled in demo', W / 2, H / 2 + 24); g.textAlign = 'left'; return; }
    g.drawImage(c.cv, 0, 0);
    const big = !!cv.dataset.big;
    if (c.zones.length) c.zones.forEach((z) => { g.fillStyle = 'rgba(239,68,68,.16)'; g.strokeStyle = 'rgba(239,68,68,.8)'; g.setLineDash([6, 4]); g.fillRect(z.x * W, z.y * H, z.w * W, z.h * H); g.strokeRect(z.x * W, z.y * H, z.w * W, z.h * H); g.setLineDash([]); if (big) { g.fillStyle = '#fca5a5'; g.font = '11px sans-serif'; g.fillText('ignore', z.x * W + 4, z.y * H + 13); } });
    if (state.showBoxes) c.dets.forEach((d) => { g.strokeStyle = COL[d.cls]; g.lineWidth = 2; g.strokeRect(d.box.x, d.box.y, d.box.w, d.box.h); const label = d.cls + ' ' + Math.round(d.conf * 100) + '%'; g.font = '600 12px sans-serif'; const tw = g.measureText(label).width + 8; g.fillStyle = COL[d.cls]; g.fillRect(d.box.x, d.box.y - 17, tw, 17); g.fillStyle = '#041'; g.fillText(label, d.box.x + 4, d.box.y - 4); });
    g.fillStyle = 'rgba(0,0,0,.55)'; g.fillRect(8, H - 28, g.measureText(c.name).width + 26, 20); g.fillStyle = '#fff'; g.font = '600 13px sans-serif'; g.fillText(c.name, 16, H - 13);
    const ts = new Date().toLocaleString('en-US', { month: '2-digit', day: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }); g.font = '12px ui-monospace,monospace'; g.fillStyle = 'rgba(0,0,0,.55)'; g.fillRect(W - 170, 8, 162, 20); g.fillStyle = '#fff'; g.fillText(ts, W - 164, 22);
    if (c.rec) { g.fillStyle = (Math.floor(now / 500) % 2) ? '#ef4444' : 'transparent'; g.beginPath(); g.arc(24, 22, 6, 0, 7); g.fill(); g.fillStyle = '#fff'; g.font = '700 12px sans-serif'; g.fillText('REC', 36, 26); }
    if (cfg.mode === 'viewer' && big) { g.fillStyle = 'rgba(0,0,0,.5)'; g.fillRect(8, 36, 150, 20); g.fillStyle = '#fcd34d'; g.font = '12px sans-serif'; g.fillText('viewer mode: detection off', 14, 50); }
  }

  /* ---------- pages ---------- */
  const route = () => (location.hash.replace(/^#\/?/, '') || 'dashboard').split('/');
  const TABS = [['dashboard', 'Dashboard'], ['events', 'Detections'], ['recordings', 'Recordings'], ['health', 'Health & metrics'], ['settings', 'Settings']];
  const fmtT = (d) => d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  const camName = (id) => (cams.find((c) => c.id === id) || {}).name;
  const evRow = (e) => `<div class="ev"><img alt="" src="${e.img}"><div><b>${esc(e.cls)} · ${Math.round(e.conf * 100)}%</b><small>${esc(camName(e.cam))}</small></div><small>${fmtT(e.t)}</small></div>`;

  function shell(inner, page) {
    return `<div class="banner"><b>Interactive demo</b><span>Simulated cameras and AI detection running in your browser — no real video, nothing is uploaded.</span><span class="sp"></span><a href="https://github.com/248Tech/RTSPanda" target="_blank" rel="noopener">View source ↗</a><a href="../">← Overview</a></div>
    <header class="topbar"><div class="brand"><span>🐼</span>RTSPanda</div><nav class="tabs" aria-label="Primary">${TABS.map(([id, n]) => `<a class="${id === page ? 'on' : ''}" href="#/${id}">${n}</a>`).join('')}</nav><div class="top-r"><span class="badge info">${cfg.mode} mode</span><span>demo@rtspanda.test</span><button class="btn sm" id="logout">Sign out</button></div></header>
    <main class="content" id="page">${inner}</main>`;
  }
  const P = {};
  P.dashboard = () => {
    const online = cams.filter((c) => c.online).length;
    return `<div class="head"><h1>Cameras</h1><span class="sp"></span><span class="badge ok"><span class="dot"></span>${online}/${cams.length} online</span><span class="badge info" id="evcount">${events.length} detections</span></div>
    <div class="grid g2">${cams.map((c) => `<div class="cam" data-open="${c.id}" tabindex="0" role="link" aria-label="Open ${esc(c.name)}"><div class="v">${c.online ? '<span class="badge ok live pill-live"><span class="dot"></span>LIVE</span>' : '<span class="badge bad pill-live">OFFLINE</span>'}<canvas data-cam="${c.id}" width="${W}" height="${H}"></canvas></div><div class="m"><div><b>${esc(c.name)}</b><br><small class="mono" style="color:var(--muted)">${esc(c.rtsp)}</small></div>${c.online ? '<span class="badge ok">Online</span>' : `<button class="btn sm primary" data-rc="${c.id}">Reconnect</button>`}</div></div>`).join('')}</div>
    <div class="grid g2" style="margin-top:14px"><div class="card"><h3>Latest detections</h3><div class="list" id="side">${events.slice(0, 6).map(evRow).join('') || '<div class="empty">Waiting for activity…</div>'}</div></div><div class="card"><h3>Pipeline</h3><div class="row sb"><span>Mode</span><b>${cfg.mode}</b></div><div class="row sb"><span>Detector</span><b>${cfg.mode === 'viewer' ? 'none' : cfg.mode === 'pi' ? 'snapshot AI (' + cfg.provider + ')' : 'YOLO (' + cfg.aiMode + ')'}</b></div><div class="row sb"><span>Relay</span><b>mediamtx · LL-HLS</b></div><div class="row sb"><span>Auto-record on detection</span><b>${cfg.autoRecord ? 'on' : 'off'}</b></div></div></div>`;
  };
  P.dashboard.bind = () => {
    $$('[data-open]').forEach((el) => { const go = () => (location.hash = '#/cam/' + el.dataset.open); el.addEventListener('click', go); el.addEventListener('keydown', (e) => { if (e.key === 'Enter') go(); }); });
    $$('[data-rc]').forEach((b) => b.addEventListener('click', (e) => { e.stopPropagation(); b.textContent = 'Connecting…'; setTimeout(() => { const c = cams.find((x) => x.id === b.dataset.rc); c.online = true; toast(c.name + ' is back online'); render(); }, 1400); }));
  };
  function renderSide() { const s = $('#side'); if (!s) return; const r = route(); const list = r[0] === 'cam' ? events.filter((e) => e.cam === r[1]) : events; s.innerHTML = list.slice(0, r[0] === 'events' ? 30 : 8).map(evRow).join('') || '<div class="empty">Waiting for activity…</div>'; const c = $('#evcount'); if (c) c.textContent = events.length + ' detections'; }

  P.cam = (id) => {
    const c = cams.find((x) => x.id === id); if (!c) return '<div class="card">Camera not found. <a href="#/dashboard">Back</a></div>';
    return `<div class="head"><a href="#/dashboard">← Dashboard</a><h1>${esc(c.name)}</h1>${c.online ? '<span class="badge ok live"><span class="dot"></span>Online</span>' : '<span class="badge bad">Offline</span>'}<span class="sp"></span><button class="btn sm" id="snap">📷 Snapshot</button><button class="btn sm ${c.rec ? 'rec' : ''}" id="rec">${c.rec ? '■ Stop recording' : '● Record'}</button><button class="btn sm" id="fs">⛶ Fullscreen</button></div>
    <div class="layout"><div><div class="viewer ${state.drawMode ? 'draw' : ''}" id="viewer"><canvas data-cam="${c.id}" data-big="1" width="${W}" height="${H}"></canvas></div>
      <div class="ctrl"><button class="btn sm ${c.detect ? 'on' : ''}" id="tdet">Detection ${c.detect ? 'on' : 'off'}</button><button class="btn sm ${state.showBoxes ? 'on' : ''}" id="tbox">Boxes</button>${['person', 'car', 'animal'].map((k) => `<button class="btn sm ${c.classes[k] ? 'on' : ''}" data-cls="${k}" style="${c.classes[k] ? 'border-color:' + COL[k] : ''}">${k}</button>`).join('')}<label>Min confidence <input id="conf" type="range" min="30" max="95" value="${Math.round(c.conf * 100)}"><b id="confv">${Math.round(c.conf * 100)}%</b></label></div>
      <div class="card"><div class="row sb"><h3 style="margin:0">Ignore zones</h3><button class="btn sm ${state.drawMode ? 'primary' : ''}" id="draw">${state.drawMode ? 'Drag on video to draw…' : '+ Draw ignore zone'}</button></div><p style="color:var(--muted);font-size:.82rem;margin:.5rem 0">Detections whose centre falls inside a zone are suppressed — handy for trees, roads or reflections.</p>${c.zones.map((z, i) => `<div class="row sb" style="padding:.2rem 0"><span class="mono">zone ${i + 1}: ${Math.round(z.x * 100)}%,${Math.round(z.y * 100)}% · ${Math.round(z.w * 100)}×${Math.round(z.h * 100)}%</span><button class="btn sm" data-rz="${i}">Remove</button></div>`).join('') || '<small style="color:var(--muted)">No zones.</small>'}</div></div>
    <div><div class="card"><h3>Stream</h3><div class="row sb"><span>Status</span><b>${c.online ? 'online' : 'offline'}</b></div><div class="row sb"><span>Resolution</span><b>${c.res}</b></div><div class="row sb"><span>Codec</span><b>H.264 → HLS</b></div><div class="row sb"><span>Bitrate</span><b>${c.br} Mbps</b></div><div class="row sb"><span>Latency</span><b>~1.4 s (LL-HLS)</b></div></div>
      <div class="card" style="margin-top:14px"><h3>${esc(c.name)} detections</h3><div class="list" id="side">${events.filter((e) => e.cam === c.id).slice(0, 8).map(evRow).join('') || '<div class="empty">Waiting for activity…</div>'}</div></div></div></div>`;
  };
  P.cam.bind = (id) => {
    const c = cams.find((x) => x.id === id); if (!c) return;
    const rer = () => render();
    $('#tdet').addEventListener('click', () => { c.detect = !c.detect; rer(); }); $('#tbox').addEventListener('click', () => { state.showBoxes = !state.showBoxes; rer(); });
    $$('[data-cls]').forEach((b) => b.addEventListener('click', () => { c.classes[b.dataset.cls] = !c.classes[b.dataset.cls]; rer(); }));
    $('#conf').addEventListener('input', (e) => { c.conf = e.target.value / 100; $('#confv').textContent = e.target.value + '%'; });
    $$('[data-rz]').forEach((b) => b.addEventListener('click', () => { c.zones.splice(+b.dataset.rz, 1); rer(); }));
    $('#draw').addEventListener('click', () => { state.drawMode = !state.drawMode; rer(); });
    $('#fs').addEventListener('click', () => { const v = $('#viewer'); (v.requestFullscreen || (() => {})).call(v); });
    $('#snap').addEventListener('click', () => { if (!c.online) return toast('Camera offline'); c.cv.toBlob((b) => { const a = document.createElement('a'); a.href = URL.createObjectURL(b); a.download = c.name.toLowerCase().replace(/\s+/g, '-') + '-snapshot.png'; a.click(); setTimeout(() => URL.revokeObjectURL(a.href), 3000); toast('Snapshot saved'); }); });
    $('#rec').addEventListener('click', () => { if (!c.online) return toast('Camera offline'); if (c.rec) { const dur = Math.max(1, Math.round((Date.now() - c.rec) / 1000)); recordings.unshift({ id: 'r' + Date.now(), cam: c.id, start: new Date(c.rec), dur, auto: false }); c.rec = null; toast('Recording saved (' + dur + ' s)'); } else { c.rec = Date.now(); toast('Recording started'); } rer(); });
    const cv = $('canvas[data-cam]'), view = $('#viewer'); let st = null;
    const pos = (e) => { const r = cv.getBoundingClientRect(); return { x: Math.min(1, Math.max(0, (e.clientX - r.left) / r.width)), y: Math.min(1, Math.max(0, (e.clientY - r.top) / r.height)) }; };
    cv.addEventListener('mousedown', (e) => { if (!state.drawMode) return; st = pos(e); c.tmp = { x: st.x, y: st.y, w: 0, h: 0 }; c.zones.push(c.tmp); });
    window.addEventListener('mousemove', (e) => { if (!st || !c.tmp) return; const p = pos(e); c.tmp.x = Math.min(st.x, p.x); c.tmp.y = Math.min(st.y, p.y); c.tmp.w = Math.abs(p.x - st.x); c.tmp.h = Math.abs(p.y - st.y); });
    window.addEventListener('mouseup', () => { if (!st) return; if (c.tmp && (c.tmp.w < 0.02 || c.tmp.h < 0.02)) c.zones.pop(); st = null; c.tmp = null; if (state.drawMode) { state.drawMode = false; render(); } }, { once: true });
    void view;
  };

  P.events = () => `<div class="head"><h1>Detections</h1><span class="sp"></span><button class="btn sm" id="dtest">Preview Discord alert</button></div>
    <div class="grid g2"><div class="card"><h3>Event timeline</h3><div class="list" id="side" style="max-height:560px">${events.slice(0, 30).map(evRow).join('') || '<div class="empty">Waiting for activity… detections appear as simulated people, vehicles and animals pass the cameras.</div>'}</div></div>
    <div class="card"><h3>By class</h3>${['person', 'car', 'animal'].map((k) => { const n = events.filter((e) => e.cls === k).length; return `<div style="margin-bottom:10px"><div class="row sb"><span>${k}</span><b>${n}</b></div><div style="height:8px;background:#1d1d26;border-radius:6px"><div style="height:100%;width:${events.length ? (n / events.length) * 100 : 0}%;background:${COL[k]};border-radius:6px"></div></div></div>`; }).join('')}<p style="color:var(--muted);font-size:.82rem">Alerts can be posted to Discord with snapshot or clip attached, per-camera cooldowns and mention suppression.</p></div></div>`;
  P.events.bind = () => $('#dtest').addEventListener('click', () => discordPreview());
  function discordPreview() { const e = events[0]; modal(`<h3 style="margin:0 0 8px">Discord notification preview</h3><div class="discord"><div class="t">🐼 RTSPanda</div><div style="margin-top:4px"><b>${e ? esc(e.cls) + ' detected' : 'Test notification'}</b> on <b>${e ? esc(camName(e.cam)) : 'Driveway'}</b>${e ? ' (' + Math.round(e.conf * 100) + '% confidence)' : ''}<br><small style="color:#b5bac1">${new Date().toLocaleString()} · cooldown 60 s · mentions suppressed</small></div>${e ? `<img alt="Detection snapshot" src="${e.img}">` : ''}</div><div class="row" style="justify-content:flex-end;margin-top:12px"><button class="btn" data-close>Close</button></div>`); }

  P.recordings = () => `<div class="head"><h1>Recordings</h1><span class="sp"></span><span class="badge info">retention ${cfg.retention} days</span></div>
    <div class="tw"><table><thead><tr><th>Camera</th><th>Started</th><th>Length</th><th>Size</th><th>Trigger</th><th></th></tr></thead><tbody>${recordings.map((r) => `<tr><td>${esc(camName(r.cam))}</td><td>${r.start.toLocaleString()}</td><td>${r.dur}s</td><td>${(r.dur * 0.5).toFixed(1)} MB</td><td>${r.auto ? '<span class="badge info">' + esc(r.cls) + '</span>' : '<span class="badge">manual</span>'}</td><td><div class="row"><button class="btn sm" data-play="${r.id}">Play</button><button class="btn sm" data-del="${r.id}">Delete</button></div></td></tr>`).join('') || '<tr><td colspan="6" class="empty">No recordings yet. Press Record on a camera, or wait for a detection (auto-record is on).</td></tr>'}</tbody></table></div>`;
  P.recordings.bind = () => {
    $$('[data-del]').forEach((b) => b.addEventListener('click', () => { recordings.splice(recordings.findIndex((r) => r.id === b.dataset.del), 1); render(); }));
    $$('[data-play]').forEach((b) => b.addEventListener('click', () => { const r = recordings.find((x) => x.id === b.dataset.play); const c = cams.find((x) => x.id === r.cam); modal(`<h3 style="margin:0 0 8px">${esc(c.name)} · replay</h3><div class="viewer"><canvas data-cam="${c.id}" width="${W}" height="${H}"></canvas></div><p style="color:var(--muted);font-size:.8rem">Demo replays the simulated camera. Real recordings are MP4 files on disk with seek, download and retention clean-up.</p><div class="row" style="justify-content:flex-end"><button class="btn" data-close>Close</button></div>`); }));
  };

  const series = {}; const sd = (k, base, amp) => { const s = series[k] = series[k] || Array.from({ length: 48 }, () => base + rnd(-amp, amp)); s.push(Math.max(0, base + rnd(-amp, amp))); if (s.length > 48) s.shift(); return s; };
  P.health = () => {
    const on = cams.filter((c) => c.online).length;
    return `<div class="head"><h1>Health &amp; metrics</h1><span class="sp"></span><span class="badge ok">/api/v1/health ✔ ready</span></div>
    <div class="grid g4"><div class="card stat"><div class="l">Cameras online</div><div class="v">${on}/${cams.length}</div></div><div class="card stat"><div class="l">Detections</div><div class="v">${events.length}</div></div><div class="card stat"><div class="l">Detector latency</div><div class="v">${cfg.mode === 'standard' ? '38 ms' : cfg.mode === 'pi' ? '1.2 s' : '—'}</div></div><div class="card stat"><div class="l">Uptime</div><div class="v">6d 04h</div></div></div>
    <div class="grid g3" style="margin-top:14px">${[['cpu', 'CPU %', 34, 10], ['mem', 'Memory (GiB)', 2.6, 0.2], ['temp', 'SoC temp °C', 52, 3]].map(([k, l, b, a]) => `<div class="card"><h3>${l}</h3><canvas class="spark" data-spark="${k}" width="300" height="56"></canvas></div>`).join('')}</div>
    <div class="card" style="margin-top:14px"><h3>Prometheus exposition (sample)</h3><pre class="mono" id="prom" style="margin:0;font-size:.78rem;white-space:pre-wrap;color:#c9c3ff"></pre></div>`;
  };
  P.health.bind = () => {
    const draw = () => { if (!$('[data-spark]')) return; [['cpu', 34, 10, 100], ['mem', 2.6, 0.2, 4], ['temp', 52, 3, 80]].forEach(([k, b, a, m]) => { const s = sd(k, b, a), c = $('[data-spark="' + k + '"]'); if (!c) return; const g = c.getContext('2d'); g.clearRect(0, 0, 300, 56); g.beginPath(); s.forEach((v, i) => { const x = (i / 47) * 300, y = 52 - (v / m) * 48; i ? g.lineTo(x, y) : g.moveTo(x, y); }); g.strokeStyle = '#a99ff5'; g.lineWidth = 2; g.stroke(); g.lineTo(300, 56); g.lineTo(0, 56); g.closePath(); const gr = g.createLinearGradient(0, 0, 0, 56); gr.addColorStop(0, 'rgba(124,108,240,.35)'); gr.addColorStop(1, 'rgba(124,108,240,0)'); g.fillStyle = gr; g.fill(); });
      $('#prom').textContent = ['# TYPE rtspanda_cameras_online gauge', 'rtspanda_cameras_online ' + cams.filter((c) => c.online).length, '# TYPE rtspanda_detections_total counter', ...['person', 'car', 'animal'].map((k) => 'rtspanda_detections_total{class="' + k + '"} ' + events.filter((e) => e.cls === k).length), '# TYPE rtspanda_stream_fps gauge', ...cams.filter((c) => c.online).map((c) => 'rtspanda_stream_fps{camera="' + c.name + '"} ' + c.fps.toFixed(1)), 'process_resident_memory_bytes ' + Math.round(2.6 * 1073741824)].join('\n'); };
    draw(); const iv = setInterval(() => { if (!$('[data-spark]')) return clearInterval(iv); draw(); }, 1000); timers.push(iv);
  };

  P.settings = () => `<div class="head"><h1>Settings</h1></div>
    <div class="grid g2"><div class="card"><h3>Deployment mode</h3><div class="radio">${[['standard', 'Standard', 'x86 server · real-time YOLO detection (local or remote worker)'], ['pi', 'Raspberry Pi', 'ARM edge node · snapshot AI (Claude/OpenAI) or remote YOLO worker'], ['viewer', 'Viewer', 'Live view and optional recording only · no detection']].map(([v, t, d]) => `<label><input type="radio" name="mode" value="${v}" ${cfg.mode === v ? 'checked' : ''}><span><b>${t}</b><small>${d}</small></span></label>`).join('')}</div></div>
    <div class="card"><h3>Detection &amp; alerts</h3>
      <div class="row sb" style="margin-bottom:12px"><span>AI worker</span><select class="in" id="aim" style="width:auto"><option value="local" ${cfg.aiMode === 'local' ? 'selected' : ''}>Local YOLO</option><option value="remote" ${cfg.aiMode === 'remote' ? 'selected' : ''}>Remote worker</option></select></div>
      <div class="row sb" style="margin-bottom:12px"><span>Snapshot AI</span><button class="sw ${cfg.snapshotAI ? 'on' : ''}" id="sai" aria-label="Snapshot AI"></button></div>
      <div class="row sb" style="margin-bottom:12px"><span>Auto-record on detection</span><button class="sw ${cfg.autoRecord ? 'on' : ''}" id="aut" aria-label="Auto record"></button></div>
      <label class="f" style="display:block;font-size:.7rem;text-transform:uppercase;letter-spacing:.05em;color:var(--muted);margin-bottom:12px">Discord webhook<input class="in" id="wh" value="${esc(cfg.discord)}" style="margin-top:5px"></label>
      <div class="row"><button class="btn sm" id="whtest">Send test notification</button><label style="display:flex;gap:8px;align-items:center;color:var(--muted);font-size:.82rem">Retention <input class="in" id="ret" type="number" min="1" max="90" value="${cfg.retention}" style="width:70px"> days</label></div></div></div>`;
  P.settings.bind = () => {
    $$('input[name=mode]').forEach((r) => r.addEventListener('change', () => { cfg.mode = r.value; toast('Mode: ' + r.value + (r.value === 'viewer' ? ' — detection disabled' : r.value === 'pi' ? ' — snapshot AI every 3 s' : ' — real-time YOLO')); render(); }));
    $('#aim').addEventListener('change', (e) => { cfg.aiMode = e.target.value; toast('AI worker: ' + e.target.value); });
    $('#sai').addEventListener('click', (e) => { cfg.snapshotAI = !cfg.snapshotAI; e.target.classList.toggle('on'); }); $('#aut').addEventListener('click', (e) => { cfg.autoRecord = !cfg.autoRecord; e.target.classList.toggle('on'); });
    $('#wh').addEventListener('input', (e) => (cfg.discord = e.target.value)); $('#ret').addEventListener('change', (e) => (cfg.retention = +e.target.value || 14)); $('#whtest').addEventListener('click', () => discordPreview());
  };

  /* ---------- router / auth ---------- */
  const timers = [];
  function render() {
    timers.splice(0).forEach(clearInterval);
    if (!store.get('rp_demo_auth')) return renderLogin();
    const r = route(); const page = P[r[0]] ? r[0] : 'dashboard';
    app.innerHTML = shell(P[page](r[1]), page === 'cam' ? 'dashboard' : page);
    document.title = (page === 'cam' ? (camName(r[1]) || 'Camera') : TABS.find((t) => t[0] === page)[1]) + ' · RTSPanda Demo';
    $('#logout').addEventListener('click', () => { store.del('rp_demo_auth'); location.hash = ''; render(); });
    if (P[page].bind) P[page].bind(r[1]);
  }
  function renderLogin() {
    document.title = 'Sign in · RTSPanda Demo';
    app.innerHTML = `<div class="login"><form class="lcard" id="lf" novalidate><div class="logo">🐼</div><h1>RTSPanda</h1><p class="sub">Sign in to your camera platform</p><label class="f">Email<input class="in" id="em" type="email" autocomplete="username" value="${EMAIL}"></label><label class="f">Password<input class="in" id="pw" type="password" autocomplete="current-password" value="${PASS}"></label><div id="err" role="alert" style="color:#fca5a5;font-size:.8rem;min-height:1.1em;margin-bottom:6px"></div><button class="btn primary block" type="submit">Sign in</button><div class="divider">or</div><button class="btn block" type="button" id="quick">Enter as demo administrator</button><div class="hint"><b>Demo credentials are pre-filled.</b><br>Email <span class="mono">${EMAIL}</span><br>Password <span class="mono">${PASS}</span><br>Simulated cameras, in-browser only. No server, no real accounts.</div><p style="text-align:center;margin:14px 0 0"><a href="../">← Back to project overview</a></p></form></div>`;
    const go = () => { store.set('rp_demo_auth', '1'); location.hash = '#/dashboard'; render(); };
    $('#lf').addEventListener('submit', (e) => { e.preventDefault(); if ($('#em').value.trim().toLowerCase() === EMAIL && $('#pw').value === PASS) go(); else $('#err').textContent = 'Invalid credentials. Use the pre-filled demo account.'; });
    $('#quick').addEventListener('click', go);
  }
  window.addEventListener('hashchange', render);
  render(); requestAnimationFrame(loop);
})();
