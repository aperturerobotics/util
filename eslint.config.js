import eslint from '@eslint/js'
import tseslint from '@typescript-eslint/eslint-plugin'
import prettier from 'eslint-config-prettier'

const runtimeGlobals = {
  console: 'readonly',
  process: 'readonly',
  setTimeout: 'readonly',
}

export default [
  {
    ignores: [
      'node_modules/**',
      'dist/**',
      'coverage/**',
      'bundle/**',
      'runtime/**',
      '.tools/**',
      'vendor/**',
      '**/wasm_exec.js',
      '**/*.pb.ts',
    ],
  },
  eslint.configs.recommended,
  ...tseslint.configs['flat/recommended'],
  {
    languageOptions: {
      globals: runtimeGlobals,
    },
    rules: {
      '@typescript-eslint/explicit-module-boundary-types': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
    },
  },
  prettier,
]
