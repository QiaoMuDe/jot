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
		return 13, 13, 13
	case "tokyo-night":
		return 26, 27, 38
	case "dracula":
		return 40, 42, 54
	case "one-dark-pro":
		return 40, 44, 52
	case "catppuccin-latte":
		return 239, 241, 245
	case "gruvbox-light":
		return 251, 241, 199
	case "nord":
		return 236, 239, 244
	case "light":
		return 250, 250, 250
	case "eye-protection":
		return 199, 237, 204
	case "quiet-light":
		return 245, 245, 245
	case "ysgrifennwr":
		return 245, 237, 218
	default: // "default" 主题
		return 247, 245, 240
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
		Width:            1080,
		Height:           768,
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
