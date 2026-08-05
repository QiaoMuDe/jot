// Jot 落地页本地预览服务器（Go 标准库实现）
//
// 在 landing 目录启动一个静态 Web 服务器，方便本地预览或服务器部署页面效果。
// 仅使用标准库，无需 go.mod，用法与 Python 版 serve.py 保持一致：
//
// 用法（在 landing 目录下执行）：
//     go run serve.go                  # 默认端口 8123，自动打开浏览器
//     go run serve.go -port 9000       # 指定端口
//     go run serve.go -no-open         # 不自动打开浏览器
//     go run serve.go -host 0.0.0.0    # 局域网/公网可访问
//
// 服务器部署：
//     go build -o serve serve.go       # 编译单文件二进制（无需 go.mod）
//     ./serve                          # 二进制与静态文件放在同一目录下运行
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const defaultPort = 8123

// openBrowser 调用系统默认浏览器打开指定 URL（跨平台支持 Windows/macOS/Linux）。
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// resolveSiteRoot 定位站点根目录：优先取二进制所在目录（go build 部署场景），
// go run 场景下二进制位于临时缓存目录，回退到当前工作目录。
func resolveSiteRoot() string {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("无法定位程序路径: %v", err)
	}
	root := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("无法定位站点根目录: %v", err)
		}
		root = cwd
	}
	return root
}

// listen 监听指定地址端口；端口被占用时自动顺延为系统分配的可用端口。
func listen(host string, port int) (net.Listener, int) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		fmt.Printf("[提示] 端口 %d 已被占用，已改用系统自动分配的端口\n", port)
		listener, err = net.Listen("tcp", fmt.Sprintf("%s:0", host))
		if err != nil {
			log.Fatalf("监听失败: %v", err)
		}
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	return listener, actualPort
}

func main() {
	// 解析命令行参数，与 Python 版用法保持一致
	host := flag.String("host", "127.0.0.1", "监听地址（默认 127.0.0.1）")
	port := flag.Int("port", defaultPort, "监听端口（默认 8123）")
	noOpen := flag.Bool("no-open", false, "不自动打开浏览器")
	flag.Parse()

	root := resolveSiteRoot()

	listener, actualPort := listen(*host, *port)
	defer listener.Close()

	url := fmt.Sprintf("http://%s:%d/index.html", *host, actualPort)
	fmt.Println("====================================================")
	fmt.Println("  Jot 落地页预览服务器已启动")
	fmt.Printf("  站点目录 : %s\n", root)
	fmt.Printf("  访问地址 : %s\n", url)
	fmt.Println("  按 Ctrl+C 停止服务")
	fmt.Println("====================================================")

	// 自动打开浏览器预览
	if !*noOpen {
		if err := openBrowser(url); err != nil {
			fmt.Printf("[提示] 自动打开浏览器失败: %v\n", err)
		}
	}

	// 静态文件服务
	log.Fatal(http.Serve(listener, http.FileServer(http.Dir(root))))
}
