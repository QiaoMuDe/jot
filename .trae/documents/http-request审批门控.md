# http\_request 接入审批门控计划

## Summary

给 `http_request` 工具接入与 run\_command/manage\_\* 同款的 Approver 审批门控：**所有方法都走审批**，GET 作为普通操作（critical=false），POST/PUT/DELETE/PATCH 为高危（critical=true）。confirm\_every 模式下全部确认；review 模式下仅非 GET 强制确认、GET 自动放行；auto 模式下全部放行、非 GET 额外审计留痕。

## 当前状态分析

* [http\_request.go](internal/agent/tools/http_request.go) 的 `invoke`（L133-250）支持 GET/POST/PUT/DELETE/PATCH（L139-143 白名单校验），**全程无任何审批调用**——模型可未经确认向任意公网 API 发起写请求（POST/PUT/DELETE/PATCH 产生外部副作用）。

* 审批基础设施已就绪：`Context.Approver`（[context.go](internal/agent/tools/context.go#L104-L109) 接口 + L137-142 注释）由父包 agentSession 注入；各工具统一实现 `requestApproval(ctx, summary, critical)` 帮助方法（ctx nil 放行、Approver nil fail-fast、否则调 `ctx.Approver.RequestApproval`）——manage\_note/manage\_notebook/manage\_tag/manage\_todo 均同款（如 [manage\_note.go](internal/agent/tools/manage_note.go#L403-L411)）。

* 摘要截断工具 `TruncateRunes` 已在 tools 包内（[http\_request.go](internal/agent/tools/http_request.go#L82) 的 ActionText 已使用）。

* 现有测试文件 [http\_request\_test.go](internal/agent/tools/http_request_test.go) 直接调用 `invoke`，构造的 `httpRequestTool` 无 ctx（ctx nil → 审批放行），故插入审批不影响既有测试。

## Proposed Changes

### 1. [http\_request.go](internal/agent/tools/http_request.go) — 接入审批门控

* **新增** **`requestApproval`** **帮助方法**（与 manage\_\* 同款，放 `NewHTTP` 附近）：

  * `h.ctx == nil` → 返回 nil（裸工具/既有测试放行）

  * `h.ctx.Approver == nil` → 返回 `errors.New("http_request 需要审批确认，但当前未配置审批机制")`（fail-fast）

  * 否则 `h.ctx.Approver.RequestApproval(ctx, "http_request", summary, critical)`

* **在** **`invoke`** **内 URL 校验之后、构造请求之前**（当前 L160 与 L162 之间）插入审批调用：

  * 此时 method 已白名单校验（L139-143）、body 长度已校验（L146-150）、URL 已校验（L154-160），符合"先校验后审批"约定。

  * `critical := method != http.MethodGet`（GET 普通、非 GET 高危）

  * summary 构造（函数级注释说明）：

    * 通用：`fmt.Sprintf("发送 %s 请求到 %s", method, TruncateRunes(target, 60))`

    * 非 GET 且 body 非空时追加：`，请求体：%s`（`TruncateRunes(args.Body, 100)` 截断）

  * 审批返回错误直接 `return "", err`（拒绝/取消均回填模型）。

* **更新** **`Info`** **的 Desc**：补充"所有请求按当前审批模式门控（GET 为常规、写方法 POST/PUT/DELETE/PATCH 为高危需确认）"。

* **更新文件头注释**：说明审批接入。

### 2. [context.go](internal/agent/tools/context.go) — 注释同步

* L97-103（Approver 接口注释）与 L137-142（Context.Approver 字段注释）把"http\_request 写方法（POST/PUT/DELETE/PATCH）"补入覆盖范围说明。

### 3. 测试 — [http\_request\_test.go](internal/agent/tools/http_request_test.go) 追加审批用例

复用同包 `mockApprover`（fs\_tools\_test.go 定义，含 `gotSum/gotCritical/gotCall/allow/reject`）。新增用例：

* **GET 批准放行**：`approver.allow=true`，`invoke` 走通，断言 `gotCritical==false`、`gotSum` 含 "GET" 与 URL、请求已发送（响应正常）。

* **GET 拒绝**：`approver.reject=true`，断言返回拒绝错误、请求**未发送**。

* **POST critical=true**：断言 `gotCritical==true`。

* **POST 批准放行**：请求发出（httptest 服务器收到 POST）。

* **Approver 缺失 fail-fast**：`ctx 非 nil 但 Approver==nil` → 返回"未配置审批机制"错误。

* **裸工具放行**：`ctx==nil`（现有测试形态）→ 不报错正常发送。

* 测试工具构造：`&httpRequestTool{setting:…, ctx: &Context{Approver: approver}, skipURLGuard: true}` + `buildClient(false)`（沿用现有测试访问 httptest 的注入缝）。

### 4. 文档

* [TOOLS.md](internal/agent/TOOLS.md)：如已有 http\_request 条目，补充审批说明；如无则不加（保持最小改动，改动以代码注释为准）。

## Assumptions & Decisions

* **GET 也走审批门控（用户明确要求）**：critical=false → confirm\_every 确认、review/auto 自动放行（即"完全与自动审批模式无需审批 GET"）。

* **非 GET 一律 critical=true**：外部写请求目标不可控、不可撤销，review 模式强制确认、auto 留痕。

* **审批位置**：`invoke` 内参数校验后（先校验后审批，沿用审查修复后的约定）；既有测试因 ctx nil 放行而不受影响。

* **Approver 缺失语义**：与 manage\_\* 一致 fail-fast（生产装配遗漏时显式暴露，而非静默放行）。

* **摘要截断**：URL 60 rune、body 100 rune，防长内容撑爆审批弹窗。

## Verification

1. `gofmt -l internal/agent/tools/` 无输出
2. `go build ./...`
3. `go vet ./internal/agent/tools/`
4. `go test -count=1 ./internal/agent/tools/`（新旧用例全绿，含 http\_request 既有用例不受影响）
5. 复核：GET 用例 gotCritical=false；POST 用例 gotCritical=true；拒绝用例请求未发出

