/* ========== Navigation Scroll Effect ========== */
const navbar = document.getElementById('navbar');
let lastScroll = 0;

window.addEventListener('scroll', function() {
  const currentScroll = window.pageYOffset || document.documentElement.scrollTop;
  if (currentScroll > 40) {
    navbar.classList.add('scrolled');
  } else {
    navbar.classList.remove('scrolled');
  }
  lastScroll = currentScroll;
}, { passive: true });

/* ========== Intersection Observer (Reveal Animations) ========== */
const revealElements = document.querySelectorAll('.reveal');

const revealObserver = new IntersectionObserver(function(entries) {
  entries.forEach(function(entry) {
    if (entry.isIntersecting) {
      entry.target.classList.add('visible');
      revealObserver.unobserve(entry.target);
    }
  });
}, {
  threshold: 0.08,
  rootMargin: '0px 0px -40px 0px'
});

revealElements.forEach(function(el) {
  revealObserver.observe(el);
});

/* ========== 滚动进度条 ========== */
const progressBar = document.getElementById('scroll-progress');
function updateProgress() {
  const doc = document.documentElement;
  const max = doc.scrollHeight - window.innerHeight;
  const p = max > 0 ? (window.pageYOffset || doc.scrollTop) / max : 0;
  progressBar.style.transform = 'scaleX(' + p + ')';
}
window.addEventListener('scroll', updateProgress, { passive: true });
window.addEventListener('resize', updateProgress);
updateProgress();

/* ========== 导航当前章节高亮 ========== */
const navSectionLinks = [];
document.querySelectorAll('.nav-links a[href^="#"]:not(.nav-cta)').forEach(function(link) {
  const target = document.querySelector(link.getAttribute('href'));
  if (target) navSectionLinks.push({ link: link, section: target });
});
const navHighlightObserver = new IntersectionObserver(function(entries) {
  entries.forEach(function(entry) {
    navSectionLinks.forEach(function(item) {
      if (item.section === entry.target) {
        item.link.classList.toggle('active', entry.isIntersecting);
      }
    });
  });
}, { rootMargin: '-40% 0px -50% 0px' });
navSectionLinks.forEach(function(item) { navHighlightObserver.observe(item.section); });

/* ========== Hero 光晕滚动视差（尊重 reduced-motion） ========== */
const heroGlowWrap = document.querySelector('.hero-glow-wrap');
if (heroGlowWrap && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
  let parallaxTicking = false;
  function updateHeroParallax() {
    parallaxTicking = false;
    const y = Math.min(window.pageYOffset || document.documentElement.scrollTop, window.innerHeight);
    heroGlowWrap.style.transform = 'translateY(' + (y * 0.35) + 'px)';
  }
  window.addEventListener('scroll', function() {
    if (!parallaxTicking) {
      parallaxTicking = true;
      requestAnimationFrame(updateHeroParallax);
    }
  }, { passive: true });
}

/* ========== Smooth Anchor Scroll (fallback for older browsers) ========== */
document.querySelectorAll('a[href^="#"]').forEach(function(anchor) {
  anchor.addEventListener('click', function(e) {
    const href = this.getAttribute('href');
    if (!href || href === '#') return;
    const target = document.querySelector(href);
    if (target) {
      e.preventDefault();
      const offset = 80;
      const targetPos = target.getBoundingClientRect().top + window.pageYOffset - offset;
      window.scrollTo({ top: targetPos, behavior: 'smooth' });
    }
  });
});

/* ========== 联系作者按钮（占位，待补充真实链接） ========== */
document.querySelectorAll('.contact-btn').forEach(function(btn) {
  btn.addEventListener('click', function(e) {
    e.preventDefault();
  });
});

/* ========== 截图动态渲染 ========== */
fetch('media.json')
  .then(function(res) {
    if (!res.ok) throw new Error('HTTP ' + res.status);
    return res.json();
  })
  .then(function(data) {
    var grid = document.getElementById('screenshots-grid');
    data.screenshots.forEach(function(item) {
      var card = document.createElement('div');
      card.className = 'screenshot-card';

      var img = document.createElement('img');
      img.src = item.src;
      img.alt = item.alt;
      img.loading = 'lazy';

      var caption = document.createElement('div');
      caption.className = 'screenshot-caption';
      caption.innerHTML = '<strong>' + item.alt + '</strong><span>' + item.caption + '</span>';

      card.appendChild(img);
      card.appendChild(caption);

      // 点击放大
      card.addEventListener('click', function() {
        document.getElementById('lightbox-img').src = item.src;
        document.getElementById('lightbox-caption').textContent = item.alt;
        document.getElementById('lightbox').classList.add('active');
      });

      grid.appendChild(card);
    });

    renderVideos(data.videos);
  })
  .catch(function(err) {
    console.error('截图加载失败:', err);
    var msg = '截图加载失败，请刷新重试';
    // 直接双击打开页面（file:// 协议）时，浏览器禁止 fetch 读取本地 JSON
    if (window.location.protocol === 'file:') {
      msg = '无法在文件模式下加载数据：请在 landing 目录运行 python serve.py，再通过 http 地址访问';
    }
    document.getElementById('screenshots-grid').innerHTML =
      '<p style="text-align:center;color:var(--text-light);padding:40px 0;">' + msg + '</p>';
  });

/* ========== Lightbox 控制 ========== */
var lightbox = document.getElementById('lightbox');
lightbox.addEventListener('click', function(e) {
  if (e.target === this || e.target.classList.contains('lightbox-close')) {
    this.classList.remove('active');
  }
});
document.addEventListener('keydown', function(e) {
  if (e.key === 'Escape') {
    lightbox.classList.remove('active');
  }
});

/* ========== 视频动态渲染 ========== */
/**
 * 从已加载的视频元素截取当前帧，生成一个 img 元素。
 * 限制画布最大宽度 1280，避免超大视频生成过大的封面图。
 * @param {HTMLVideoElement} video - 已定位到目标时间点的视频元素
 * @returns {HTMLImageElement|null} 截帧成功返回 img 元素，失败返回 null
 */
function captureVideoFrame(video) {
  try {
    var scale = Math.min(1, 1280 / video.videoWidth);
    var canvas = document.createElement('canvas');
    canvas.width = video.videoWidth * scale;
    canvas.height = video.videoHeight * scale;
    canvas.getContext('2d').drawImage(video, 0, 0, canvas.width, canvas.height);
    var img = document.createElement('img');
    img.src = canvas.toDataURL('image/jpeg', 0.85);
    img.alt = '';
    return img;
  } catch (e) {
    return null;
  }
}

/**
 * 页面加载时主动截取视频帧作为卡片封面图（针对未配置 poster 的视频）。
 * 使用隐藏的 video 元素加载视频并 seek 到 1 秒处截帧，成功后替换渐变底色，
 * 失败（如视频文件不存在）时静默回退到渐变底色。
 * @param {string} src - 视频地址
 * @param {HTMLElement} posterEl - 视频卡片的封面容器
 */
function fetchVideoPoster(src, posterEl) {
  if (!src || !posterEl) return;
  var video = document.createElement('video');
  video.muted = true;        // 静音 + 内联，避免触发浏览器自动播放限制
  video.playsInline = true;
  video.preload = 'metadata'; // 只预加载元数据，seek 时按需拉取数据，节省流量
  video.src = src;
  video.style.display = 'none';
  document.body.appendChild(video);

  // 清理临时视频元素并解绑事件
  function cleanup() {
    video.removeEventListener('loadedmetadata', onLoaded);
    video.removeEventListener('seeked', onSeeked);
    video.removeEventListener('error', onError);
    video.removeAttribute('src');
    video.load();
    video.remove();
  }
  // 元数据就绪后跳转到 1 秒处，确保截到有效画面
  function onLoaded() {
    try { video.currentTime = Math.min(1, video.duration || 1); } catch (e) {}
  }
  // 定位完成后截帧并填充卡片封面
  function onSeeked() {
    if (posterEl.classList.contains('has-poster')) { cleanup(); return; }
    var img = captureVideoFrame(video);
    if (img) {
      posterEl.insertBefore(img, posterEl.firstChild);
      posterEl.classList.add('has-poster');
    }
    cleanup();
  }
  // 加载失败时静默回退（保持渐变底色兜底）
  function onError() { cleanup(); }

  video.addEventListener('loadedmetadata', onLoaded);
  video.addEventListener('seeked', onSeeked);
  video.addEventListener('error', onError);
}

/**
 * 视频播放时自动截取一帧作为卡片封面图（仅当 media.json 未配置 poster 时使用）。
 * 作为 fetchVideoPoster 的补充：页面加载时截帧失败的情况下，打开视频后补一次机会。
 * 每个视频元素只绑定一次事件，截帧成功后记录 has-poster 状态防止重复截帧。
 * @param {HTMLVideoElement} video - 弹窗中的视频元素
 * @param {HTMLElement} posterEl - 视频卡片的封面容器
 */
function autoCapturePoster(video, posterEl) {
  if (!video || !posterEl) return;
  // 记录本次打开的目标封面容器，目标随每次打开动态更新
  video._posterTarget = posterEl;
  // 每个视频元素只绑定一次事件，避免重复监听
  if (video.dataset.captureBound) return;
  video.dataset.captureBound = '1';

  video.addEventListener('loadeddata', function() {
    // 视频数据就绪后跳转到 1 秒处，确保截到有效画面
    try { video.currentTime = Math.min(1, video.duration || 1); } catch (e) {}
  });
  video.addEventListener('seeked', function() {
    var target = video._posterTarget;
    if (!target || target.classList.contains('has-poster')) return;
    var img = captureVideoFrame(video);
    if (img) {
      target.insertBefore(img, target.firstChild);
      target.classList.add('has-poster');
    }
  });
}

/**
 * 根据 media.json 中的 videos 数组渲染视频卡片
 * @param {Array} videos - 视频数据数组，每项含 id/src/poster/title/caption
 */
function renderVideos(videos) {
  var grid = document.getElementById('videos-grid');
  if (!videos || !videos.length) {
    grid.innerHTML =
      '<p style="text-align:center;color:var(--text-light);padding:40px 0;">视频准备中，敬请期待</p>';
    return;
  }

  videos.forEach(function(item) {
    var card = document.createElement('div');
    card.className = 'video-card';

    var poster = document.createElement('div');
    poster.className = 'video-card-poster';

    // 有封面图则直接显示封面；没有则主动截取视频帧作为预览图
    if (item.poster) {
      var img = document.createElement('img');
      img.src = item.poster;
      img.alt = item.title;
      img.loading = 'lazy';
      poster.appendChild(img);
    } else if (item.src) {
      fetchVideoPoster(item.src, poster);
    }

    var play = document.createElement('div');
    play.className = 'video-card-play';
    play.innerHTML =
      '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>';
    poster.appendChild(play);

    var caption = document.createElement('div');
    caption.className = 'video-card-caption';
    caption.innerHTML = '<strong>' + item.title + '</strong><span>' + item.caption + '</span>';

    card.appendChild(poster);
    card.appendChild(caption);

    // 点击卡片播放
    function openVideo() {
      var video = document.getElementById('video-modal-video');
      video.src = item.src;
      document.getElementById('video-modal-title').textContent = item.title;
      document.getElementById('video-modal').classList.add('active');
      video.play();
      // 未配置封面图时，打开视频后自动截帧生成预览图
      if (!item.poster) {
        autoCapturePoster(video, poster);
      }
    }
    card.addEventListener('click', openVideo);

    // 键盘无障碍支持
    card.addEventListener('keydown', function(e) {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        openVideo();
      }
    });

    grid.appendChild(card);
  });
}

/* ========== 视频弹窗控制 ========== */
var videoModal = document.getElementById('video-modal');
var videoModalVideo = document.getElementById('video-modal-video');

/**
 * 关闭视频弹窗，并释放视频资源避免后台继续下载
 */
function closeVideoModal() {
  videoModal.classList.remove('active');
  videoModalVideo.pause();
  videoModalVideo.removeAttribute('src');
  videoModalVideo.load();
}

videoModal.addEventListener('click', function(e) {
  if (e.target === this || e.target.classList.contains('video-modal-close')) {
    closeVideoModal();
  }
});

document.addEventListener('keydown', function(e) {
  if (e.key === 'Escape' && videoModal.classList.contains('active')) {
    closeVideoModal();
  }
});
