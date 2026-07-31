import { defineConfig } from 'vite';

export default defineConfig({
    build: {
        // smol-toml 使用 BigInt 字面量，需要 es2021+ 支持
        target: 'es2021'
    },
    optimizeDeps: {
        esbuildOptions: {
            // dev server 预构建依赖也需要 es2021 以支持 BigInt
            target: 'es2021'
        }
    }
});