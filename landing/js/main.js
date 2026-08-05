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

/* ========== Counter Animation ========== */
const counters = document.querySelectorAll('.stat-counter');

const counterObserver = new IntersectionObserver(function(entries) {
  entries.forEach(function(entry) {
    if (entry.isIntersecting) {
      const counter = entry.target;
      const target = parseInt(counter.getAttribute('data-target'), 10);
      const duration = 1800;
      const start = performance.now();

      function updateCounter(currentTime) {
        const elapsed = currentTime - start;
        const progress = Math.min(elapsed / duration, 1);
        // easeOutExpo
        const eased = progress === 1 ? 1 : 1 - Math.pow(2, -10 * progress);
        const current = Math.floor(eased * target);
        counter.textContent = current;

        if (progress < 1) {
          requestAnimationFrame(updateCounter);
        } else {
          counter.textContent = target;
        }
      }

      requestAnimationFrame(updateCounter);
      counterObserver.unobserve(counter);
    }
  });
}, { threshold: 0.5 });

counters.forEach(function(counter) {
  counterObserver.observe(counter);
});

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

    // 有封面图则显示封面，否则用渐变底色 + 播放按钮兜底
    if (item.poster) {
      var img = document.createElement('img');
      img.src = item.poster;
      img.alt = item.title;
      img.loading = 'lazy';
      poster.appendChild(img);
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
      document.getElementById('video-modal-video').src = item.src;
      document.getElementById('video-modal-title').textContent = item.title;
      document.getElementById('video-modal').classList.add('active');
      document.getElementById('video-modal-video').play();
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
