import globals from 'globals';
import prettier from 'eslint-config-prettier';

export default [
    {
        ignores: [
            'dist/**',
            'node_modules/**',
            // 第三方语法高亮语言数据文件（大量引用运行时全局，不参与检查）
            'src/js/hljs-themes-data.js',
        ],
    },
    {
        // 主线程脚本（Vite 入口与页面逻辑）
        files: ['src/**/*.js'],
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            globals: {
                ...globals.browser,
                // 项目惯例：由各模块挂载到 window 的隐式全局（以裸标识符调用）
                nm: 'readonly',
                SVGS: 'readonly',
                getMockNotes: 'readonly',
                exportNote: 'readonly',
                initEditorActionsMenu: 'readonly',
                switchEditorMode: 'readonly',
                // 历史遗留：main.js 直接引用但无声明/无 window 挂载（疑似死代码，见检查报告）
                mockNotes: 'readonly',
            },
        },
        rules: {
            // ---- 核心风险检查（错误级）----
            'no-unreachable': 'error', // 不可达代码
            'no-dupe-keys': 'error', // 对象重复键
            'use-isnan': 'error', // 用 isNaN() 判断 NaN
            // ---- 提醒级（历史代码噪音多，降为 warn 自查）----
            'no-undef': 'warn', // 未定义变量（含项目大量 window 隐式全局）
            'no-unused-vars': ['warn', { args: 'none', caughtErrors: 'none', varsIgnorePattern: '^_' }],
            'no-constant-condition': ['warn', { checkLoops: false }],
            // ---- 项目风格（关闭）----
            'no-empty': 'off', // catch 空块用于静默忽略
            'no-redeclare': 'off', // var 重复声明是项目既有风格
        },
    },
    {
        // Web Worker 脚本（使用 Worker 全局，无 DOM）
        files: ['src/js/preview-worker.js'],
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            globals: {
                ...globals.worker,
            },
        },
        rules: {
            'no-undef': 'warn',
            'no-unused-vars': ['warn', { args: 'none', caughtErrors: 'none', varsIgnorePattern: '^_' }],
        },
    },
    // 关闭与 Prettier 冲突的格式类规则（必须放最后）
    prettier,
];
