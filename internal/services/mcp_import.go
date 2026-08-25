package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"jot/internal/models"

	"gitee.com/MM-Q/fastlog"
)

// rawMCPServer 导入 JSON 的中间结构(不直接对应 models.MCPServer,便于容错与字段推导)
type rawMCPServer struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
}

// ParseMCPServersImport 仅做 JSON 解析与字段校验，不入库
// 整体结果通过返回的 ParseMCPServersResult.OK 判断；失败时 Error 必填
func ParseMCPServersImport(logSvc *LogService, input string) models.ParseMCPServersResult {
	raws, err := parseMCPImportInput(input)
	if err != nil {
		logSvc.Logger.Errorw("ParseMCPServersImport JSON 解析失败", fastlog.Error(err))
		return models.ParseMCPServersResult{OK: false, Error: "JSON 解析失败: " + err.Error()}
	}
	if len(raws) == 0 {
		return models.ParseMCPServersResult{OK: false, Error: "未找到任何服务器配置"}
	}

	items := make([]models.ImportMCPServerItem, 0, len(raws))
	allValid := true
	for i, raw := range raws {
		res := models.ImportMCPServerItem{Index: i + 1, Name: strings.TrimSpace(raw.Name)}
		if _, errMsg := buildMCPServerFromRaw(raw); errMsg != "" {
			res.Error = errMsg
			allValid = false
		} else {
			res.OK = true // B3: 校验通过的条目显式置 true
		}
		items = append(items, res)
	}
	return models.ParseMCPServersResult{OK: allValid, Items: items}
}

// ImportMCPServers 解析用户粘贴的 JSON 并批量入库 MCP 服务器
// 所有解析/校验/入库错误均自动写入 logs/app.log(由 logSvc 输出)
// input 支持三种格式: 裸数组 / {servers:[..]} / 单个对象
// 返回每条处理结果: JSON 整体解析失败时返回单条 {ok:false, error:...}
func ImportMCPServers(logSvc *LogService, mcpSvc *MCPServerService, input string) []models.ImportMCPServerItem {
	logSvc.Logger.Debugw("ImportMCPServers", fastlog.Int("inputLen", len(input)))

	raws, err := parseMCPImportInput(input)
	if err != nil {
		logSvc.Logger.Errorw("ImportMCPServers JSON 解析失败", fastlog.Error(err))
		return []models.ImportMCPServerItem{{Index: 0, OK: false, Error: "JSON 解析失败: " + err.Error()}}
	}
	if len(raws) == 0 {
		return []models.ImportMCPServerItem{{Index: 0, OK: false, Error: "未找到任何服务器配置"}}
	}

	results := make([]models.ImportMCPServerItem, 0, len(raws))
	for i, raw := range raws {
		res := models.ImportMCPServerItem{Index: i + 1, Name: strings.TrimSpace(raw.Name)}

		server, errMsg := buildMCPServerFromRaw(raw)
		if errMsg != "" {
			res.Error = errMsg
			logSvc.Logger.Errorw("ImportMCPServers 字段校验失败",
				fastlog.Int("index", i+1),
				fastlog.String("name", res.Name),
				fastlog.String("reason", errMsg))
		} else {
			if err := mcpSvc.Save(server); err != nil {
				res.Error = err.Error()
				logSvc.Logger.Errorw("ImportMCPServers 入库失败",
					fastlog.Int("index", i+1),
					fastlog.String("name", server.Name),
					fastlog.Error(err))
			} else {
				res.OK = true
				logSvc.Logger.Infow("ImportMCPServers 入库成功",
					fastlog.Int("index", i+1),
					fastlog.String("name", server.Name))
			}
		}
		results = append(results, res)
	}
	return results
}

// parseMCPImportInput 三格式容错: 裸数组 / {servers:[..]} / 单个对象
// 顶层直接是空数组 [] 时返回"未找到任何服务器配置（输入为空数组）"友好提示
func parseMCPImportInput(s string) ([]rawMCPServer, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("输入为空")
	}

	// B4: 顶层直接是空数组时返回友好提示,避免落到末尾的"无法识别"
	var probe []any
	if err := json.Unmarshal([]byte(s), &probe); err == nil && len(probe) == 0 {
		return nil, errors.New("未找到任何服务器配置（输入为空数组）")
	}

	raws, err := tryParseInput([]byte(s))
	if err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return nil, errors.New("未找到任何服务器配置")
	}
	return raws, nil
}

// tryParseInput 统一三格式解析(B10 抽取的公共函数,消除三处 fallback 重复)
// 顺序: 1) 裸数组  2) {servers:[..]}  3) 单个对象(至少含 name/command/url 之一)
func tryParseInput(input []byte) ([]rawMCPServer, error) {
	var arr []rawMCPServer
	if err := json.Unmarshal(input, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}
	var wrapped struct {
		Servers []rawMCPServer `json:"servers"`
	}
	if err := json.Unmarshal(input, &wrapped); err == nil && len(wrapped.Servers) > 0 {
		return wrapped.Servers, nil
	}
	var single rawMCPServer
	if err := json.Unmarshal(input, &single); err == nil &&
		(single.Name != "" || single.Command != "" || single.URL != "") {
		return []rawMCPServer{single}, nil
	}
	return nil, errors.New("无法识别为 [..] / {servers:[..]} / 单个对象")
}

// buildMCPServerFromRaw 字段校验 + 推导 transport + 构造 MCPServer
// 返回 (nil, errMsg) 表示校验失败;返回 (server, "") 表示成功
// 校验规则与服务层 MCPServerService.Save 完全一致(B2),使阶段 1 就能拦截所有非法配置
func buildMCPServerFromRaw(raw rawMCPServer) (*models.MCPServer, string) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return nil, "名称不能为空"
	}
	// B2: 名称不能含空白(与服务层 Save 一致,避免阶段 2 才拦截)
	if strings.ContainsAny(name, " \t\r\n") {
		return nil, "名称不能包含空格等空白字符"
	}

	// 推导 transport(显式标注优先,否则按 command/url 推断)
	transport := strings.ToLower(strings.TrimSpace(raw.Transport))
	if transport == "" {
		switch {
		case raw.Command != "":
			transport = "stdio"
		case raw.URL != "":
			transport = "sse" // 默认 sse,与项目既有约定一致
		default:
			return nil, "缺少 command 或 url"
		}
	}
	if transport != "stdio" && transport != "sse" && transport != "http" {
		return nil, fmt.Sprintf("transport 非法: %q", raw.Transport)
	}

	// 构造 MCPServer;字段冲突清零(由 service.Save 最终落地)
	server := &models.MCPServer{
		Name:    name,
		Enabled: false, // 导入默认禁用,需用户手动启用
	}
	switch transport {
	case "stdio":
		if raw.Command == "" {
			return nil, "stdio 模式必须提供 command"
		}
		// B2: env KEY 不能含空白或等号(与服务层一致)
		for k := range raw.Env {
			if strings.ContainsAny(k, " \t\r\n=") {
				return nil, "环境变量 KEY「" + k + "」不能包含空白或等号"
			}
		}
		server.Transport = "stdio"
		server.Command = raw.Command
		if len(raw.Args) > 0 {
			server.Args = append([]string(nil), raw.Args...)
		}
		if len(raw.Env) > 0 {
			server.Env = make(map[string]string, len(raw.Env))
			for k, v := range raw.Env {
				server.Env[k] = v
			}
		}
	case "sse", "http":
		if raw.URL == "" {
			return nil, transport + " 模式必须提供 url"
		}
		// B2: headers KEY 不能含空白或等号(与服务层一致)
		for k := range raw.Headers {
			if strings.ContainsAny(k, " \t\r\n=") {
				return nil, "请求头 KEY「" + k + "」不能包含空白或等号"
			}
		}
		server.Transport = transport
		server.URL = raw.URL
		if len(raw.Headers) > 0 {
			server.Headers = make(map[string]string, len(raw.Headers))
			for k, v := range raw.Headers {
				server.Headers[k] = v
			}
		}
	}
	return server, ""
}
