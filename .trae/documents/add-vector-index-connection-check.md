# 量化弹窗打开时增加 API 可用性校验

## 摘要

在「数据管理 → 笔记量化」弹窗打开时，新增量化服务（embedding API）连通性校验：通过轻量 GET 请求（openai 走 `/models`、ollama 走 `/api/tags`）测试服务是否可用。**校验完全异步执行，不阻塞弹窗打开**——弹窗秒开，测试在后台进行，失败时用 toast 通知提示用户检查服务是否启动；成功时静默，不打扰。

## 现状分析

- 弹窗打开入口：[data-management.js openVectorIndexModal](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js#L577-L639) 已调用 `ValidateVectorIndexConfig` 做**配置完整性**校验（provider/baseURL/model、openai 的 key），但只校验"配置是否填了"，**不校验服务是否真的在运行**。
- 后端已具备现成的连通性测试能力，无需新造轮子：
  - [ai_service.go TestConnection](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L267-L276)：openai 调 `GET /models`（5s 超时）、ollama 调 `GET /api/tags`（5s 超时）、other 简单对话（10s 超时），非 2xx 返回具体状态码错误。
  - [app.go testAIConnection](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L1677-L1695)：已封装的公共分发实现，设置页「测试连接」按钮即通过它工作。
  - [app.go GetEmbedConfig](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L1530-L1537)：读取量化配置（provider/baseURL/apiKey，apiKey 已解码）。
- 现有结构化结果类型 `CardRecallCheckResult{OK, Message}`（app.go L1540-1543）可复用，符合项目"多返回值用 struct"约定。
- Wails v2 绑定整个 App struct（[main.go](file:///d:/资源池/下水道/Dev/本地项目/jot/main.go#L92-L94)），新增导出方法自动绑定；`frontend/wailsjs/` 为构建时自动生成。

## 超时/过渡处理（用户关注点）

测试耗时最坏 5 秒（网络超时）。处理策略：

1. **弹窗立即打开**：连通性测试放在弹窗打开逻辑之后 fire-and-forget（不 `await`），弹窗秒开、用户可立即正常操作，不存在"弹窗卡住等待"的情况。
2. **超时兜底**：后端 `httpGetJSON` 自带 5s 超时，超时即返回错误 → toast 提示"量化服务连接失败"。用户看到提示时弹窗早已打开并可正常使用。
3. **失败不阻断**：连通性失败**不阻止**用户继续操作弹窗（用户仍可尝试开始量化，后端 embedding 调用失败时已有单篇错误提示兜底），仅作提醒。

## 改动方案

### 1. 后端：新增绑定方法 `TestVectorIndexConnection`（[app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go)，置于 `testAIConnection` 附近）

**目的**：前端不需要也无法拿到解码后的 apiKey，由后端读取量化配置并完成连通性测试，返回结构化结果。

```go
// TestVectorIndexConnection 测试量化服务连通性（量化弹窗打开时异步调用，不阻塞弹窗）
// 通过轻量 GET 请求检测服务可用性：openai 走 /models，ollama 走 /api/tags（均 5s 超时）
func (a *App) TestVectorIndexConnection() CardRecallCheckResult {
	provider, baseURL, apiKey, _, _ := a.GetEmbedConfig()
	if provider == "" || baseURL == "" {
		return CardRecallCheckResult{OK: false, Message: "请先在设置中配置量化连接与量化模型"}
	}
	ok, err := a.testAIConnection(provider, baseURL, apiKey, "TestVectorIndexConnection")
	if err != nil {
		return CardRecallCheckResult{OK: false, Message: "量化服务连接失败，请检查服务是否已启动（" + err.Error() + "）"}
	}
	if !ok {
		return CardRecallCheckResult{OK: false, Message: "量化服务连接失败，请检查服务是否已启动"}
	}
	return CardRecallCheckResult{OK: true, Message: ""}
}
```

要点：
- 复用 `testAIConnection` 公共实现，不新增连接逻辑。
- 错误消息带上具体原因（如连接拒绝/超时/HTTP 状态码），遵循 [improve-test-connection-notifications.md](file:///d:/资源池/下水道/Dev/本地项目/jot/.trae/documents/improve-test-connection-notifications.md) 的"失败提示包含具体原因"约定。
- 配置缺失分支返回引导消息（正常流程下前端已由 `ValidateVectorIndexConfig` 拦截，此分支兜底）。

### 2. 前端：弹窗打开后异步触发校验（[data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js)）

**改动 A**：`openVectorIndexModal` 中，`ValidateVectorIndexConfig` 校验通过之后（约 L591 后）、复位弹窗状态之前，插入一行 fire-and-forget 调用：

```js
// 配置校验通过后，异步测试量化服务连通性（不阻塞弹窗打开；失败时 toast 提示，成功静默）
checkVectorIndexConnection();
```

**改动 B**：在 AI 量化索引模块区域（`openVectorIndexModal` 函数附近）新增模块级函数：

```js
/**
 * 异步测试量化服务连通性（弹窗打开后后台执行，不阻塞弹窗）
 * 失败时 toast 提示，成功静默；接口异常时忽略，由开始量化时的后端校验兜底
 */
async function checkVectorIndexConnection() {
    const app = window.go?.main?.App;
    if (!app?.TestVectorIndexConnection) return;
    try {
        const res = await app.TestVectorIndexConnection();
        if (res && !res.ok && res.message) {
            window.nm?.show?.(res.message, 'error');
        }
    } catch (_) { /* 忽略异常，后端 startVectorIndex 兜底 */ }
}
```

要点：
- **不 await**：弹窗立即打开，测试后台执行，无等待感。
- **成功静默、失败提示**：符合用户选择，避免每次打开弹窗被成功 toast 打扰；失败提示使用 `error` 级别，引导用户检查云服务是否运行 / 本地 ollama 是否启动。
- 接口异常忽略（如 wailsjs 未重新生成时），与现有 `ValidateVectorIndexConfig` 的异常放行策略一致。

### 3. 重新生成 wailsjs 绑定

新增后端方法后，运行 `wails generate module`（或下次 `wails build`）重新生成 `frontend/wailsjs/go/main/App.js` 与 `App.d.ts`，使 `window.go.main.App.TestVectorIndexConnection` 可用且类型声明同步。该目录为构建时自动生成，不手动编辑。

## 假设与决策

- 量化 provider 仅有 openai/ollama 两类，`testAIConnection` 的 other 分支（需 model 的简单对话）不会走到；如需绝对稳妥可后续补 model 透传，本次不涉及。
- 连通性失败不阻断弹窗使用，仅提醒——与用户"校验然后通知提示一下"的诉求一致，且避免因测试失败误伤正常流程。
- 不新增弹窗内 UI（用户已确认用 toast 通知形式）。
- 不做请求节流：成功静默不会刷屏；失败提示是真实反馈（服务确实不可用），每次打开弹窗提示合理。

## 验证

1. `go build ./...` 编译通过。
2. 运行 `wails generate module` 重新生成绑定，确认 `frontend/wailsjs/go/main/App.js` 包含 `TestVectorIndexConnection`。
3. 手动场景测试（`wails dev` 或构建后运行）：
   - a. 配置正常 + 服务运行中（云 API 或 ollama 已启动）→ 打开量化弹窗**立即弹出、无成功提示**。
   - b. 配置正常 + 服务未启动（关闭 ollama）→ 打开弹窗立即弹出，约 5s 后出现 error toast「量化服务连接失败，请检查服务是否已启动（...）」。
   - c. 未配置量化连接 → 弹窗不打开，提示引导配置（现有逻辑，回归验证不变）。
   - d. 连通性失败时仍可正常操作弹窗、查看笔记列表。
