# -*- coding: utf-8 -*-
"""
Jot 落地页本地预览服务器

在 landing 目录启动一个静态 Web 服务器，方便本地预览页面效果。
用法（在 landing 目录下执行）：
    python serve.py                  # 默认端口 8123，自动打开浏览器
    python serve.py --port 9000      # 指定端口
    python serve.py --no-open        # 不自动打开浏览器
    python serve.py --host 0.0.0.0   # 局域网可访问
"""

import argparse
import os
import sys
import webbrowser
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer

# 脚本位于 landing 目录下，站点根目录即脚本所在目录
SITE_ROOT = os.path.dirname(os.path.abspath(__file__))
DEFAULT_PORT = 8123


class NoCacheHandler(SimpleHTTPRequestHandler):
    """静态文件处理器：禁用浏览器缓存，保证改动即时生效。"""

    def end_headers(self):
        self.send_header("Cache-Control", "no-store")
        super().end_headers()


def build_handler(site_root):
    """构建静态文件处理器，将请求目录固定为站点根目录。

    Args:
        site_root: 站点根目录的绝对路径

    Returns:
        SimpleHTTPRequestHandler 子类
    """
    handler = partial(NoCacheHandler, directory=site_root)
    return handler


def parse_args():
    """解析命令行参数。

    Returns:
        包含 host、port、open_browser 的命名空间
    """
    parser = argparse.ArgumentParser(description="Jot 落地页本地预览服务器")
    parser.add_argument("--host", default="127.0.0.1", help="监听地址（默认 127.0.0.1）")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT, help=f"监听端口（默认 {DEFAULT_PORT}）")
    parser.add_argument("--no-open", action="store_true", help="不自动打开浏览器")
    return parser.parse_args()


def main():
    """程序入口：启动静态 Web 服务器并可选打开浏览器。"""
    args = parse_args()

    # Windows 终端兼容 UTF-8 中文输出
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
        sys.stderr.reconfigure(encoding="utf-8")

    server = ThreadingHTTPServer((args.host, args.port), build_handler(SITE_ROOT))

    # 自动挑选可用端口（指定端口被占用时顺延）
    actual_port = server.server_address[1]
    if actual_port != args.port:
        print(f"[提示] 端口 {args.port} 已被占用，已改用端口 {actual_port}")

    url = f"http://{args.host}:{actual_port}/index.html"
    print("=" * 52)
    print("  Jot 落地页预览服务器已启动")
    print(f"  站点目录 : {SITE_ROOT}")
    print(f"  访问地址 : {url}")
    print("  按 Ctrl+C 停止服务")
    print("=" * 52)

    # 自动打开浏览器预览
    if not args.no_open:
        webbrowser.open(url)

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[提示] 已停止服务")
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
