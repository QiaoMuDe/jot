/**
 * 日历视图模块 - 按创建日期浏览笔记
 */

// 当前显示的年月
let calendarYear = 0;
let calendarMonth = 0;

// DOM 引用
let calendarGrid, calendarTitle, calendarPrevBtn, calendarNextBtn;
let calendarDateLabel, calendarNotesList, calendarNotesEmpty, calendarBackBtn, calendarTodayBtn, calendarStats;

/**
 * 获取 closeEditorSafe 引用（在 main.js 中定义）
 * @returns {Function|undefined}
 */
function getCloseEditorSafe() {
    return window.closeEditorSafe;
}

/**
 * 初始化日历视图 - 获取 DOM 引用并绑定事件
 */
export function initCalendarView() {
    calendarGrid = document.getElementById('calendarGrid');
    calendarTitle = document.getElementById('calendarTitle');
    calendarPrevBtn = document.getElementById('calendarPrevBtn');
    calendarNextBtn = document.getElementById('calendarNextBtn');
    calendarDateLabel = document.getElementById('calendarDateLabel');
    calendarNotesList = document.getElementById('calendarNotesList');
    calendarNotesEmpty = document.getElementById('calendarNotesEmpty');
    calendarBackBtn = document.getElementById('calendarBackBtn');
    calendarTodayBtn = document.getElementById('calendarTodayBtn');
    calendarStats = document.getElementById('calendarStats');

    if (!calendarGrid) return;

    const now = new Date();
    calendarYear = now.getFullYear();
    calendarMonth = now.getMonth() + 1;

    calendarPrevBtn.addEventListener('click', () => {
        calendarMonth--;
        if (calendarMonth < 1) { calendarMonth = 12; calendarYear--; }
        renderCalendar(calendarYear, calendarMonth);
    });

    calendarNextBtn.addEventListener('click', () => {
        calendarMonth++;
        if (calendarMonth > 12) { calendarMonth = 1; calendarYear++; }
        renderCalendar(calendarYear, calendarMonth);
    });

    calendarBackBtn?.addEventListener('click', () => {
        window.switchView('grid');
    });

    calendarTodayBtn?.addEventListener('click', () => {
        const now = new Date();
        renderCalendar(now.getFullYear(), now.getMonth() + 1);
    });

    // 初始化滚动条自动显隐（滚动时显示，静止 1 秒后隐藏）
    let scrollTimer = null;
    calendarNotesList.addEventListener('scroll', (e) => {
        if (e.target !== calendarNotesList) return;
        calendarNotesList.classList.add('scrolling');
        clearTimeout(scrollTimer);
        scrollTimer = setTimeout(() => {
            calendarNotesList.classList.remove('scrolling');
        }, 1000);
    });

    // 初始渲染
    renderCalendar(calendarYear, calendarMonth);
}

/**
 * 渲染日历网格
 * @param {number} year - 年份
 * @param {number} month - 月份（1-12）
 */
async function renderCalendar(year, month) {
    calendarTitle.textContent = `${year}年 ${month}月`;

    // 获取当月笔记计数
    let counts = {};
    try {
        counts = await window.go.main.App.GetMonthNoteCounts(year, month) || {};
    } catch (e) {
        console.warn('GetMonthNoteCounts 失败:', e);
    }

    // 计算并渲染月份统计
    const stats = computeMonthStats(counts, year, month);
    renderMonthStats(stats);

    // 计算日期
    const firstDay = new Date(year, month - 1, 1);
    const lastDay = new Date(year, month, 0);
    const totalDays = lastDay.getDate();
    // 周一=0, 周二=1, ..., 周日=6
    let startWeekday = firstDay.getDay() - 1;
    if (startWeekday < 0) startWeekday = 6;

    // 上个月填充天数
    const prevMonthLastDay = new Date(year, month - 1, 0).getDate();

    // 今天
    const today = new Date();
    const todayStr = `${today.getFullYear()}-${today.getMonth()+1}-${today.getDate()}`;

    calendarGrid.innerHTML = '';

    // 星期头
    const weekdays = ['一', '二', '三', '四', '五', '六', '日'];
    weekdays.forEach(d => {
        const wd = document.createElement('div');
        wd.className = 'calendar-weekday';
        wd.textContent = d;
        calendarGrid.appendChild(wd);
    });

    let selectedDayEl = null;

    // 填充日期
    for (let i = 0; i < 42; i++) {
        let day, isOtherMonth = false;
        if (i < startWeekday) {
            // 上个月
            day = prevMonthLastDay - startWeekday + i + 1;
            isOtherMonth = true;
        } else if (i >= startWeekday + totalDays) {
            // 下个月
            day = i - startWeekday - totalDays + 1;
            isOtherMonth = true;
        } else {
            day = i - startWeekday + 1;
        }

        const el = document.createElement('div');
        el.className = 'calendar-day' + (isOtherMonth ? ' other-month' : '');
        el.textContent = day;
        el.dataset.year = year;
        el.dataset.month = month;

        if (!isOtherMonth) {
            el.dataset.day = day;
            const dateStr = `${year}-${month}-${day}`;

            // 标记今天
            if (dateStr === todayStr) {
                el.classList.add('today');
            }

            // 墨水圆点
            const count = counts[day] || 0;
            if (count > 0) {
                const dot = document.createElement('span');
                dot.className = 'ink-dot';
                if (count === 1) dot.classList.add('dot-1');
                else if (count <= 5) dot.classList.add('dot-2');
                else dot.classList.add('dot-3');
                el.appendChild(dot);
            }

            // 默认选中今天
            if (dateStr === todayStr) {
                selectedDayEl = el;
            }

            // 点击事件
            el.addEventListener('click', () => {
                selectDate(el, year, month, day);
            });
        }

        calendarGrid.appendChild(el);
    }

    // 默认选中今天并加载笔记
    if (selectedDayEl) {
        // 延迟一帧等 DOM 渲染完成
        requestAnimationFrame(() => {
            selectDate(selectedDayEl, year, month, today.getDate());
        });
    } else {
        calendarDateLabel.textContent = '选择日期查看笔记';
        calendarNotesList.innerHTML = '';
        calendarNotesEmpty.style.display = 'flex';
    }
}

/**
 * 选择日期 - 高亮对应日期格并加载该日笔记
 * @param {HTMLElement} el - 日期 DOM 元素
 * @param {number} year - 年份
 * @param {number} month - 月份
 * @param {number} day - 日期
 */
function selectDate(el, year, month, day) {
    // 清除其他选中
    document.querySelectorAll('.calendar-day.selected').forEach(d => d.classList.remove('selected'));
    el.classList.add('selected');

    const weekdayNames = ['日', '一', '二', '三', '四', '五', '六'];
    const weekday = weekdayNames[new Date(year, month - 1, day).getDay()];
    calendarDateLabel.textContent = `${month}月${day}日 星期${weekday}`;

    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
    loadNotesForDate(dateStr);

    // 更新回到今天按钮显隐
    updateTodayBtnVisibility(year, month, day);
}

/**
 * 加载指定日期的笔记列表
 * @param {string} dateStr - 日期字符串，格式 YYYY-MM-DD
 */
async function loadNotesForDate(dateStr) {
    calendarNotesList.innerHTML = '';
    calendarNotesEmpty.style.display = 'none';

    try {
        const notes = await window.go.main.App.GetNotesByDate(dateStr);
        if (!notes || notes.length === 0) {
            calendarNotesEmpty.style.display = 'flex';
            return;
        }
        renderNotes(notes);
    } catch (e) {
        console.warn('GetNotesByDate 失败:', e);
        calendarNotesEmpty.style.display = 'flex';
        calendarNotesEmpty.querySelector('p').textContent = '加载失败';
    }
}

/**
 * 渲染笔记列表
 * @param {Array} notes - 笔记数组
 */
function renderNotes(notes) {
    const fragment = document.createDocumentFragment();
    notes.forEach((note, index) => {
        const item = document.createElement('div');
        item.className = 'calendar-note-item';
        item.style.animationDelay = `${index * 0.05}s`;

        const title = document.createElement('div');
        title.className = 'calendar-note-title';
        title.textContent = note.title || '无标题';

        const meta = document.createElement('div');
        meta.className = 'calendar-note-meta';

        const notebookSpan = document.createElement('span');
        notebookSpan.className = 'calendar-note-notebook';
        notebookSpan.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg> ${note.notebook?.name || '默认笔记本'}`;

        const timeSpan = document.createElement('span');
        timeSpan.className = 'calendar-note-time';
        timeSpan.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg> ${formatTime(note.created_at)}`;

        meta.appendChild(notebookSpan);
        meta.appendChild(timeSpan);
        item.appendChild(title);
        item.appendChild(meta);

        item.addEventListener('click', () => openNoteFromCalendar(note));
        fragment.appendChild(item);
    });
    calendarNotesList.appendChild(fragment);
}

/**
 * 从日历打开笔记 - 在日历视图内直接打开编辑器查看模式
 * @param {Object} note - 笔记对象
 */
async function openNoteFromCalendar(note) {
    // 仅在编辑器实际打开时关闭，避免 closeEditor 的 200ms 清理定时器竞态关闭新开的编辑器
    const viewEditor = document.getElementById('viewEditor');
    if (viewEditor && viewEditor.classList.contains('active')) {
        const closeEditorSafe = getCloseEditorSafe();
        if (closeEditorSafe) await closeEditorSafe();
        // 等待 closeEditor 的 200ms 清理定时器完成，防止其撤销 openEditor 的设置
        await new Promise(r => setTimeout(r, 200));
    }

    // 直接在日历视图内打开编辑器查看模式，编辑器是独立浮层，不依赖视图切换
    window.openEditor(note.id, true);
}

/**
 * 刷新日历视图数据（切换到今天）
 */
window.refreshCalendarView = function () {
    const now = new Date();
    calendarYear = now.getFullYear();
    calendarMonth = now.getMonth() + 1;
    renderCalendar(calendarYear, calendarMonth);
};

/**
 * 格式化笔记创建时间为 HH:mm
 * @param {string} dateStr - ISO 日期字符串
 * @returns {string}
 */
function formatTime(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    const h = String(d.getHours()).padStart(2, '0');
    const m = String(d.getMinutes()).padStart(2, '0');
    return `${h}:${m}`;
}

/**
 * 根据所选日期是否今天，控制"回到今天"按钮的显隐
 * @param {number} year - 年份
 * @param {number} month - 月份（1-12）
 * @param {number} day - 日期
 */
function updateTodayBtnVisibility(year, month, day) {
    const now = new Date();
    const isToday = (year === now.getFullYear() && month === now.getMonth() + 1 && day === now.getDate());
    calendarTodayBtn.style.display = isToday ? 'none' : '';
}

/**
 * 从月份笔记计数中计算统计数据
 * @param {Object} counts - { day: count } 映射
 * @param {number} year - 年份
 * @param {number} month - 月份（1-12）
 * @returns {{ totalNotes: number, activeDays: number, streak: number, passedDays: number }}
 */
function computeMonthStats(counts, year, month) {
    const days = Object.keys(counts).map(Number).filter(d => d > 0);
    const totalNotes = days.reduce((sum, d) => sum + (counts[d] || 0), 0);
    const activeDays = days.length;

    // 计算当月已过天数
    const now = new Date();
    const isCurrentMonth = (year === now.getFullYear() && month === now.getMonth() + 1);
    const lastDay = new Date(year, month, 0).getDate();
    const passedDays = isCurrentMonth ? now.getDate() : lastDay;

    // 计算最长连续记笔天数
    let streak = 0;
    if (days.length > 0) {
        days.sort((a, b) => a - b);
        let currentStreak = 1;
        for (let i = 1; i < days.length; i++) {
            if (days[i] === days[i - 1] + 1) {
                currentStreak++;
            } else {
                streak = Math.max(streak, currentStreak);
                currentStreak = 1;
            }
        }
        streak = Math.max(streak, currentStreak);
    }

    return { totalNotes, activeDays, streak, passedDays };
}

/**
 * 渲染月份统计摘要
 * @param {{ totalNotes: number, activeDays: number, streak: number, passedDays: number }} stats - 统计数据
 */
function renderMonthStats(stats) {
    const barPercent = stats.passedDays > 0 ? Math.round((stats.activeDays / stats.passedDays) * 100) : 0;

    calendarStats.innerHTML = `
        <div class="calendar-stats-title">本月统计</div>
        <div class="calendar-stats-grid">
            <div class="calendar-stat-item">
                <span class="calendar-stat-value">${stats.totalNotes}</span>
                <span class="calendar-stat-label">总笔记</span>
            </div>
            <div class="calendar-stat-item">
                <span class="calendar-stat-value">${stats.activeDays}</span>
                <span class="calendar-stat-label">记天数</span>
            </div>
            <div class="calendar-stat-item">
                <span class="calendar-stat-value">${stats.streak}</span>
                <span class="calendar-stat-label">最长连续</span>
            </div>
        </div>
        <div class="calendar-stats-bar">
            <div class="calendar-stats-bar-label">
                <span>活跃度</span>
                <span class="stats-bar-percent">${barPercent}%</span>
            </div>
            <div class="calendar-stats-bar-track">
                <div class="calendar-stats-bar-fill" style="width:0%"></div>
            </div>
        </div>
    `;

    // 延迟一帧触发填充动画，使每次渲染都有从0展开的效果
    const fill = calendarStats.querySelector('.calendar-stats-bar-fill');
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            fill.style.width = barPercent + '%';
        });
    });
}
