/**
 * 预览渲染 Web Worker
 * 在后台线程执行 marked.parse() 离线程解析，不阻塞主线程 UI
 */
import { marked } from 'marked';

// 与主线程一致的 marked 选项
marked.setOptions({
    breaks: true,
    gfm: true,
});

// 监听主线程消息
self.onmessage = function (e) {
    const content = e.data;
    try {
        const html = marked.parse(content);
        self.postMessage({ html });
    } catch (err) {
        self.postMessage({ error: err.message });
    }
};
