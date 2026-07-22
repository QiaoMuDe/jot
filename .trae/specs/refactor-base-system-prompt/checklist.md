# Checklist

- [x] 基础 prompt 已定义为包级常量，包含三层结构（身份层/规范层/边界层）
- [x] `CallAIStream` 和 `CallAIStreamRegenerate` 中的硬编码字符串已替换为常量引用
- [x] 无技能激活时正确注入三层结构 prompt
- [x] 有技能激活时行为不变（跳过基础 prompt）
- [x] 编译通过，无错误