package main

import (
	"embed"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	home, _ := os.UserHomeDir()
	imageDir := filepath.Join(home, ".jot", "images")

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "jot",
		Width:            1024,
		Height:           768,
		Frameless:        true,
		CSSDragProperty:  "--wails-draggable",
		CSSDragValue:     "drag",
		BackgroundColour: &options.RGBA{R: r, G: g, B: b, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
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
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
