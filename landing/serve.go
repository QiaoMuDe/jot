// Jot 落地页单文件服务器（静态资源已通过 go:embed 嵌入二进制）
//
// 全部静态资源（HTML/CSS/JS/图片/视频/JSON）在编译时打包进二进制，
// 部署时只需拷贝一个文件，无需携带任何静态目录、无需安装任何运行时。
// 仅使用标准库，无需 go.mod。
//
// 用法：
//
//	go run serve.go                  # 默认端口 8123，自动打开浏览器
//	go run serve.go -port 9000       # 指定端口
//	go run serve.go -no-open         # 不自动打开浏览器
//	go run serve.go -host 0.0.0.0    # 局域网/公网可访问
//
// 部署：
//
//	go build -o jot-landing serve.go # 编译单文件二进制（无需 go.mod）
//	./jot-landing                    # 任意目录均可运行
//
// 注意：静态资源已内嵌，更新素材（如替换 videos/ 下的视频）后需要重新编译生效。
package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
)

const defaultPort = 8123

//go:embed index.html css js images videos media.json
var staticFiles embed.FS

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
	// 解析命令行参数
	host := flag.String("host", "127.0.0.1", "监听地址（默认 127.0.0.1）")
	port := flag.Int("port", defaultPort, "监听端口（默认 8123）")
	noOpen := flag.Bool("no-open", false, "不自动打开浏览器")
	flag.Parse()

	listener, actualPort := listen(*host, *port)
	defer func() { _ = listener.Close() }()

	url := fmt.Sprintf("http://%s:%d/index.html", *host, actualPort)
	fmt.Println("====================================================")
	fmt.Println("  Jot 落地页预览服务器已启动（静态资源已内嵌）")
	fmt.Printf("  访问地址 : %s\n", url)
	fmt.Println("  按 Ctrl+C 停止服务")
	fmt.Println("====================================================")

	// 自动打开浏览器预览
	if !*noOpen {
		if err := openBrowser(url); err != nil {
			fmt.Printf("[提示] 自动打开浏览器失败: %v\n", err)
		}
	}

	// 提供内嵌的静态文件服务
	log.Fatal(http.Serve(listener, http.FileServer(http.FS(staticFiles))))
}
