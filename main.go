package main

import (
	"embed"
	"net/http"
	"strings"

	"jot/internal/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// themeBG 返回主题名称对应的窗口背景色 RGBA 值（与 variables.css 的 --bg 一致）
func themeBG(theme string) (uint8, uint8, uint8) {
	switch theme {
	case "dark":
		return 15, 15, 18
	case "tokyo-night":
		return 26, 27, 38
	case "dracula":
		return 26, 27, 37
	case "catppuccin-latte":
		return 237, 231, 229
	case "gruvbox-light":
		return 240, 232, 201
	case "nord":
		return 228, 233, 240
	case "light":
		return 244, 244, 247
	case "eye-protection":
		return 228, 236, 217
	case "quiet-light":
		return 240, 234, 236
	case "ysgrifennwr":
		return 240, 229, 208
	case "mono":
		return 245, 245, 245
	default: // "default" 主题
		return 242, 237, 227
	}
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// 在窗口创建前读取已保存的主题，设置 WebView2 初始背景色
	r, g, b := themeBG(app.settingService.Get("theme"))

	// 计算图片存储目录路径
	imageDir, err := config.SubDir(config.DirImages)
	if err != nil {
		println("获取图片存储目录失败:", err.Error())
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:            "jot",
		Width:            1280,
		Height:           800,
		Frameless:        true,
		CSSDragProperty:  "--wails-draggable",
		CSSDragValue:     "drag",
		BackgroundColour: &options.RGBA{R: r, G: g, B: b, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 入口页禁用缓存，避免 WebView2 缓存旧资源引用导致前端修改不生效
					if r.URL.Path == "/" || r.URL.Path == "/index.html" {
						w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
					}
					next.ServeHTTP(w, r)
				})
			},
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/images/") {
					http.StripPrefix("/images/", http.FileServer(http.Dir(imageDir))).ServeHTTP(w, r)
					return
				}
				http.NotFound(w, r)
			}),
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
