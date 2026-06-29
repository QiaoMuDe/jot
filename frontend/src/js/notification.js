/* ===== 统一通知系统 ===== */

/**
 * 右上角浮动通知管理器
 * 单例模式，全局可引用
 */
export class NotificationManager {
    constructor() {
        this.container = document.getElementById('notificationContainer');
        if (!this.container) {
            this.container = document.createElement('div');
            this.container.id = 'notificationContainer';
            this.container.className = 'notification-container';
            document.body.appendChild(this.container);
        }
    }

    /**
     * 显示通知
     * @param {string} message - 通知内容
     * @param {string} type - 类型：'success' | 'error' | 'warning' | 'info'
     * @param {number} duration - 自动消失毫秒数（默认 3000）
     */
    show(message, type = 'info', duration = 3000) {
        const el = document.createElement('div');
        el.className = `notification ${type}`;

        const SVGS = window.SVGS;
        const iconSvg = { success: SVGS.checkmark, error: SVGS.windowClose, warning: SVGS.warning, info: SVGS.info };
        el.innerHTML = `
            <span class="notification-icon">${iconSvg[type] || SVGS.info}</span>
            <span class="notification-msg">${this._esc(message)}</span>
            <button class="notification-close" aria-label="关闭">${SVGS.windowClose}</button>
        `;

        this.container.appendChild(el);

        // 关闭按钮
        el.querySelector('.notification-close').addEventListener('click', () => this._dismiss(el));

        // 自动消失
        const timer = setTimeout(() => this._dismiss(el), duration);
        el._timer = timer;

        return el;
    }

    /**
     * 显示可撤销通知
     * @param {string} message - 通知内容
     * @param {Function} onUndo - 点击撤销的回调函数
     * @param {number} duration - 自动消失毫秒数（默认 5000）
     */
    showUndo(message, onUndo, duration = 5000) {
        const el = document.createElement('div');
        el.className = 'notification undo';
        const SVGS = window.SVGS;
        el.innerHTML = `
            <span class="notification-icon" style="color:var(--accent,#6366f1)">${SVGS.undo}</span>
            <span class="notification-msg">${this._esc(message)}</span>
            <button class="notification-undo-btn">撤销</button>
        `;

        this.container.appendChild(el);

        // 撤销按钮
        el.querySelector('.notification-undo-btn').addEventListener('click', () => {
            if (typeof onUndo === 'function') onUndo();
            this._dismiss(el);
        });

        // 自动消失
        const timer = setTimeout(() => this._dismiss(el), duration);
        el._timer = timer;

        return el;
    }

    /**
     * 手动销毁通知
     */
    _dismiss(el) {
        if (el._dismissing) return;
        el._dismissing = true;
        clearTimeout(el._timer);
        el.classList.add('exit');
        el.addEventListener('animationend', () => {
            if (el.parentNode) el.parentNode.removeChild(el);
        }, { once: true });
    }

    /**
     * HTML 转义
     */
    _esc(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

window.NotificationManager = NotificationManager;
window.showNotification = (msg, type = 'info', duration) => {
    // 复用已存在的全局通知管理器实例，避免重复创建
    if (!window.__nm) window.__nm = new NotificationManager();
    window.__nm.show(msg, type, duration);
};

/* ===== 模拟数据（后端未绑定时使用） ===== */

// Mock 数据的可变副本，确保修改可持久化
let mockNotes = null;

export function getMockNotes() {
    return [
        {
            id: 1,
            title: '欢迎使用 jot',
            content: '这是一条示例笔记。你可以在这里记录你的想法、灵感、待办事项等。\n\n点击左侧 "+" 按钮创建新笔记，或点击笔记进行编辑。',
            pinned: true,
            created_at: new Date(Date.now() - 3600000).toISOString(),
            updated_at: new Date(Date.now() - 1800000).toISOString(),
            tags: [
                { id: 1, name: '入门', color: '#6366f1' },
            ],
        },
        {
            id: 2,
            title: '设计思路',
            content: 'jot 是一款简洁的笔记应用，采用双栏布局设计。\n左侧导航栏包含搜索、笔记列表和设置。\n右侧主内容区展示笔记卡片网格。',
            pinned: false,
            created_at: new Date(Date.now() - 86400000).toISOString(),
            updated_at: new Date(Date.now() - 43200000).toISOString(),
            tags: [
                { id: 2, name: '设计', color: '#8b5cf6' },
            ],
        },
        {
            id: 3,
            title: '待办事项',
            content: '- 实现后端 CRUD\n- 实现前端布局\n- 联调测试',
            pinned: false,
            created_at: new Date(Date.now() - 172800000).toISOString(),
            updated_at: new Date(Date.now() - 86400000).toISOString(),
            tags: [
                { id: 3, name: '待办', color: '#f59e0b' },
            ],
        },
    ];
}

export function getMockTags() {
    return [
        { id: 1, name: '入门', color: '#6366f1' },
        { id: 2, name: '设计', color: '#8b5cf6' },
        { id: 3, name: '待办', color: '#f59e0b' },
    ];
}

window.getMockNotes = getMockNotes;
window.getMockTags = getMockTags;
