# Checklist

- [x] Logger Init（INFO）在 DB Init 之前执行
- [x] DB/SettingService 就绪后，从库中读取 log_level 动态调级
- [x] 所有 Service 创建时 Logger 非 nil
- [x] startup() 中不存在日志初始化代码（Init/SetLevel/nil检查/Exit）
- [x] startup() 中不存在 `fmt.Printf`/`Println` 调用
- [x] 所有 `fmt.Printf`/`Println` 已替换为 `a.LogSvc.Logger.Errorw`/`Infow`
- [x] 移除了初始化后的三条回顾性 INFO 日志
- [x] NewApp 中 Init 失败且 Logger 为 nil 时 os.Exit(1)
- [x] migrateSensitiveKeys 中无 Logger nil 检查
- [x] 无编译错误
