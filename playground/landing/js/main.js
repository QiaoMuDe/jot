/* ============================================================
   Jot 落地页 · main.js（ES5，深空 HUD 版）
   ============================================================ */
(function () {
  'use strict';

  var reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  /* ---------- 1. 预加载进度条 ---------- */
  var preloader = document.getElementById('preloader');
  var plFill = document.getElementById('pl-fill');
  var plNum = document.getElementById('pl-num');
  var progress = 0;

  (function preloadRun() {
    progress += Math.random() * 22 + 10;
    if (progress >= 100) progress = 100;
    plFill.style.width = progress + '%';
    plNum.textContent = Math.round(progress) + '%';
    if (progress < 100) {
      setTimeout(preloadRun, 120 + Math.random() * 180);
    } else {
      setTimeout(function () {
        preloader.classList.add('done');
        setTimeout(function () { preloader.style.display = 'none'; }, 800);
      }, 260);
    }
  })();

  /* ---------- 2. Hero 粒子网络（AURORA 移植） ---------- */
  var canvas = document.getElementById('stars');
  var ctx = (canvas && canvas.getContext) ? canvas.getContext('2d') : null;

  function initStars() {
    if (!ctx) return;
    var W, H, parts = [], mouse = { x: -1e4, y: -1e4 };
    var LINK = 132, REPEL = 96;
    var running = !reduced;

    function resize() {
      W = canvas.clientWidth;
      H = canvas.clientHeight;
      canvas.width = W;
      canvas.height = H;
      var n = Math.min(150, Math.floor(W * H / 11000));
      parts = [];
      for (var i = 0; i < n; i++) {
        parts.push({
          x: Math.random() * W,
          y: Math.random() * H,
          r: Math.random() * 1.4 + 0.4,
          vx: (Math.random() - .5) * .22,
          vy: (Math.random() - .5) * .22,
          tw: Math.random() * Math.PI * 2
        });
      }
      if (!running) drawStatic();
    }
    function drawStatic() {
      ctx.clearRect(0, 0, W, H);
      ctx.fillStyle = 'rgba(219,234,254,.7)';
      for (var s = 0; s < parts.length; s++) {
        ctx.beginPath(); ctx.arc(parts[s].x, parts[s].y, parts[s].r, 0, 6.283); ctx.fill();
      }
    }
    function frame() {
      ctx.clearRect(0, 0, W, H);
      var i, j, p, q, dx, dy, d;
      for (i = 0; i < parts.length; i++) {
        p = parts[i];
        p.x += p.vx; p.y += p.vy; p.tw += .02;
        if (p.x < -20) p.x = W + 20; if (p.x > W + 20) p.x = -20;
        if (p.y < -20) p.y = H + 20; if (p.y > H + 20) p.y = -20;
        // 鼠标轻微排斥
        dx = p.x - mouse.x; dy = p.y - mouse.y;
        d = Math.sqrt(dx * dx + dy * dy);
        if (d < REPEL && d > .01) { var f = (REPEL - d) / REPEL * 1.1; p.x += dx / d * f; p.y += dy / d * f; }
      }
      // 粒子连线
      ctx.lineWidth = 1;
      for (i = 0; i < parts.length; i++) {
        p = parts[i];
        for (j = i + 1; j < parts.length; j++) {
          q = parts[j];
          dx = p.x - q.x; dy = p.y - q.y; d = dx * dx + dy * dy;
          if (d < LINK * LINK) {
            var a = (1 - Math.sqrt(d) / LINK) * .34;
            ctx.strokeStyle = 'rgba(62,231,255,' + a.toFixed(3) + ')';
            ctx.beginPath(); ctx.moveTo(p.x, p.y); ctx.lineTo(q.x, q.y); ctx.stroke();
          }
        }
        // 与鼠标连线（琥珀色）
        dx = p.x - mouse.x; dy = p.y - mouse.y; d = dx * dx + dy * dy;
        if (d < LINK * LINK) {
          var ma = (1 - Math.sqrt(d) / LINK) * .6;
          ctx.strokeStyle = 'rgba(255,196,107,' + ma.toFixed(3) + ')';
          ctx.beginPath(); ctx.moveTo(p.x, p.y); ctx.lineTo(mouse.x, mouse.y); ctx.stroke();
        }
      }
      // 星点
      for (i = 0; i < parts.length; i++) {
        p = parts[i];
        var alpha = .35 + Math.sin(p.tw) * .3;
        ctx.fillStyle = 'rgba(219,234,254,' + alpha.toFixed(3) + ')';
        ctx.beginPath(); ctx.arc(p.x, p.y, p.r, 0, 6.283); ctx.fill();
      }
      // 鼠标光环
      ctx.beginPath(); ctx.arc(mouse.x, mouse.y, 10, 0, 6.283);
      ctx.strokeStyle = 'rgba(62,231,255,.55)'; ctx.lineWidth = 1.2; ctx.stroke();
      ctx.beginPath(); ctx.arc(mouse.x, mouse.y, 3, 0, 6.283);
      ctx.fillStyle = 'rgba(62,231,255,.85)'; ctx.fill();
      requestAnimationFrame(frame);
    }
    window.addEventListener('resize', resize);
    window.addEventListener('mousemove', function (e) {
      var r = canvas.getBoundingClientRect();
      mouse.x = e.clientX - r.left; mouse.y = e.clientY - r.top;
    });
    window.addEventListener('mouseout', function () { mouse.x = -1e4; mouse.y = -1e4; });
    resize();
    if (running) requestAnimationFrame(frame);
  }
  initStars();

  /* ---------- 3. 打字机 ---------- */
  var typedEl = document.getElementById('typed');
  var lines = [
    '本地存储 · 数据完全由你掌控',
    'AI 智能体 · 深度思考 · 16 个内置工具',
    '向量语义召回 · 让笔记真正能被理解',
    '异构文件一键清洗 · 沉淀为知识库',
    'Card 卡片式设计 · 界面清爽 · 交互流畅'
  ];
  function typewriter() {
    if (!typedEl || reduced) {
      if (typedEl) typedEl.textContent = lines[0];
      return;
    }
    var li = 0, ci = 0, deleting = false;
    function tick() {
      var cur = lines[li];
      if (!deleting) {
        ci++;
        typedEl.textContent = cur.slice(0, ci);
        if (ci === cur.length) { deleting = true; setTimeout(tick, 1800); return; }
        setTimeout(tick, 70 + Math.random() * 60);
      } else {
        ci--;
        typedEl.textContent = cur.slice(0, ci);
        if (ci === 0) {
          deleting = false;
          li = (li + 1) % lines.length;
          setTimeout(tick, 350);
          return;
        }
        setTimeout(tick, 32);
      }
    }
    setTimeout(tick, 900);
  }
  typewriter();

  /* ---------- 4. 导航状态 ---------- */
  var nav = document.getElementById('nav');
  var progressBar = document.getElementById('scroll-progress');
  var navLinks = nav.querySelectorAll('.links a');
  var sections = [];

  function buildSections() {
    sections = [];
    for (var i = 0; i < navLinks.length; i++) {
      var id = navLinks[i].getAttribute('href');
      if (id && id.charAt(0) === '#') {
        var el = document.querySelector(id);
        if (el) sections.push({ el: el, link: navLinks[i] });
      }
    }
  }
  buildSections();

  function onScroll() {
    var top = window.pageYOffset || document.documentElement.scrollTop;
    nav.classList.toggle('scrolled', top > 40);

    if (progressBar) {
      var doc = document.documentElement;
      var max = doc.scrollHeight - window.innerHeight;
      var sc = max > 0 ? top / max : 0;
      progressBar.style.transform = 'scaleX(' + Math.min(1, Math.max(0, sc)).toFixed(4) + ')';
    }

    var cur = 0;
    for (var i = 0; i < sections.length; i++) {
      if (top >= sections[i].el.offsetTop - 160) cur = i;
    }
    for (var j = 0; j < sections.length; j++) {
      sections[j].link.classList.toggle('active', j === cur);
    }
  }
  window.addEventListener('scroll', onScroll, { passive: true });
  onScroll();

  /* ---------- 5. 移动菜单 ---------- */
  var burger = document.getElementById('burger');
  function closeMenu() {
    document.body.classList.remove('nav-open');
    if (burger) burger.setAttribute('aria-expanded', 'false');
  }
  if (burger) {
    burger.addEventListener('click', function () {
      var open = document.body.classList.toggle('nav-open');
      burger.setAttribute('aria-expanded', open ? 'true' : 'false');
    });
  }
  var mmLinks = document.querySelectorAll('#mobile-menu a');
  for (var mm = 0; mm < mmLinks.length; mm++) {
    mmLinks[mm].addEventListener('click', closeMenu);
  }

  /* 导航/按钮平滑滚动（带头部偏移） */
  document.addEventListener('click', function (e) {
    var a = e.target.closest ? e.target.closest('a[href^="#"]') : null;
    if (!a) return;
    var id = a.getAttribute('href');
    if (id.length < 2) return;
    var target = document.querySelector(id);
    if (!target) return;
    e.preventDefault();
    var offset = target.getBoundingClientRect().top + window.pageYOffset - (nav ? nav.offsetHeight + 14 : 70);
    window.scrollTo({ top: offset, behavior: reduced ? 'auto' : 'smooth' });
  });

  /* ---------- 6. 入场观察（.up / .reveal） ---------- */
  function observeReveals() {
    var items = document.querySelectorAll('.up, .reveal');
    if (reduced) {
      for (var i = 0; i < items.length; i++) items[i].classList.add('in');
      return;
    }
    if (!('IntersectionObserver' in window)) {
      for (var j = 0; j < items.length; j++) items[j].classList.add('in');
      return;
    }
    var io = new IntersectionObserver(function (entries) {
      for (var k = 0; k < entries.length; k++) {
        if (entries[k].isIntersecting) {
          entries[k].target.classList.add('in');
          io.unobserve(entries[k].target);
        }
      }
    }, { threshold: 0.12 });
    for (var n = 0; n < items.length; n++) io.observe(items[n]);
  }
  observeReveals();

  /* 优先触发首屏 .up */
  function flashAboveFold() {
    var heroUp = document.querySelectorAll('#hero .up');
    if (!reduced && window.pageYOffset < 60) {
      setTimeout(function () {
        for (var i = 0; i < heroUp.length; i++) heroUp[i].classList.add('in');
      }, 700);
    }
  }
  flashAboveFold();

  /* ---------- 7. 3D 倾斜卡片 ---------- */
  function initTilt() {
    if (reduced || !window.matchMedia('(hover:hover)').matches) return;
    var cards = document.querySelectorAll('[data-tilt]');
    for (var i = 0; i < cards.length; i++) (function (card) {
      var max = 7;
      var rect = null;
      function onMove(e) {
        if (!rect) rect = card.getBoundingClientRect();
        var px = (e.clientX - rect.left) / rect.width;
        var py = (e.clientY - rect.top) / rect.height;
        var rx = (0.5 - py) * max;
        var ry = (px - 0.5) * max;
        card.style.setProperty('--mx', (px * 100).toFixed(1) + '%');
        card.style.setProperty('--my', (py * 100).toFixed(1) + '%');
        card.style.transform = 'perspective(900px) rotateX(' + rx.toFixed(2) + 'deg) rotateY(' + ry.toFixed(2) + 'deg)';
      }
      function onLeave() {
        rect = null;
        card.style.transform = '';
      }
      card.addEventListener('mousemove', onMove);
      card.addEventListener('mouseleave', onLeave);
    })(cards[i]);
  }
  initTilt();

  /* ---------- 8. 媒体渲染（截图 / 视频） ---------- */
  var basePath = (window.BASE_PATH || '').replace(/\/$/, '');

  var screenshotsGrid = document.getElementById('screenshots-grid');
  var videosGrid = document.getElementById('videos-grid');

  var lightbox = document.getElementById('lightbox');
  var lightboxImg = document.getElementById('lightbox-img');
  var lightboxCaption = document.getElementById('lightbox-caption');
  var videoModal = document.getElementById('video-modal');
  var videoModalVideo = document.getElementById('video-modal-video');
  var videoModalTitle = document.getElementById('video-modal-title');

  function closeLightbox() { if (lightbox) lightbox.classList.remove('active'); document.body.style.overflow = ''; }
  function closeVideoModal() { if (videoModalVideo) videoModalVideo.pause(); if (videoModal) videoModal.classList.remove('active'); document.body.style.overflow = ''; }
  if (lightbox) lightbox.addEventListener('click', function (e) { if (e.target === lightbox) closeLightbox(); });
  if (videoModal) videoModal.addEventListener('click', function (e) { if (e.target === videoModal) closeVideoModal(); });
  if (document.querySelector('.lightbox-close')) document.querySelector('.lightbox-close').addEventListener('click', closeLightbox);
  if (document.querySelector('.video-modal-close')) document.querySelector('.video-modal-close').addEventListener('click', closeVideoModal);

  fetch(basePath + '/media.json').then(function (res) { return res.json(); }).then(function (data) {
    if (screenshotsGrid && data.screenshots && data.screenshots.length) {
      data.screenshots.forEach(function (item) {
        var card = document.createElement('div');
        card.className = 'screenshot-card';
        card.innerHTML =
          '<img src="' + basePath + '/' + item.src + '" alt="' + (item.caption || '') + '" loading="lazy">' +
          '<div class="screenshot-caption"><strong>' + (item.alt || '') + '</strong><span>' + (item.caption || '') + '</span></div>';
        card.addEventListener('click', function () {
          lightboxImg.src = basePath + '/' + item.src;
          lightboxCaption.textContent = item.title || '';
          lightbox.classList.add('active');
          document.body.style.overflow = 'hidden';
        });
        screenshotsGrid.appendChild(card);
      });
    }
    if (videosGrid && data.videos && data.videos.length) {
      data.videos.forEach(function (item) {
        var card = document.createElement('div');
        card.className = 'video-card';
        card.innerHTML =
          '<div class="video-card-poster">' +
          '<img src="' + basePath + '/' + item.poster + '" alt="' + (item.caption || '') + '" loading="lazy">' +
          '<div class="video-card-play"><svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg></div>' +
          '</div>' +
          '<div class="video-card-caption"><strong>' + (item.title || '') + '</strong><span>' + (item.caption || '') + '</span></div>';
        card.addEventListener('click', function () {
          videoModalVideo.src = basePath + '/' + item.src;
          videoModalTitle.textContent = item.title || '';
          videoModal.classList.add('active');
          document.body.style.overflow = 'hidden';
          videoModalVideo.play();
        });
        videosGrid.appendChild(card);
      });
    }
    if (screenshotsGrid) { screenshotsGrid.classList.add('rendered'); }
    if (videosGrid) { videosGrid.classList.add('rendered'); }
  }).catch(function (err) {
    if (window.console) console.error('加载 media.json 失败:', err);
  });

  /* ---------- 9. ESC 关闭弹窗 ---------- */
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' || e.keyCode === 27) {
      closeLightbox();
      closeVideoModal();
      closeMenu();
    }
  });
})();