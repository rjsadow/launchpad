# React + TypeScript + Vite

This template provides a minimal setup to get React working in Vite with HMR
and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react][plugin-react] uses [Babel][babel] (or [oxc][oxc] when
  used in [rolldown-vite][rolldown]) for Fast Refresh
- [@vitejs/plugin-react-swc][plugin-react-swc] uses [SWC][swc] for Fast Refresh

## React Compiler

The React Compiler is not enabled on this template because of its impact on
dev & build performances. To add it, see [this documentation][compiler-docs].

## Expanding the ESLint configuration

If you are developing a production application, we recommend updating the
configuration to enable type-aware lint rules:

```js
export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...

      // Remove tseslint.configs.recommended and replace with this
      tseslint.configs.recommendedTypeChecked,
      // Alternatively, use this for stricter rules
      tseslint.configs.strictTypeChecked,
      // Optionally, add this for stylistic rules
      tseslint.configs.stylisticTypeChecked,

      // Other configs...
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```

You can also install [eslint-plugin-react-x][react-x] and
[eslint-plugin-react-dom][react-dom] for React-specific lint rules:

```js
// eslint.config.js
import reactX from 'eslint-plugin-react-x'
import reactDom from 'eslint-plugin-react-dom'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...
      // Enable lint rules for React
      reactX.configs['recommended-typescript'],
      // Enable lint rules for React DOM
      reactDom.configs.recommended,
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```

<!-- Reference links -->
[plugin-react]: https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react
[plugin-react-swc]: https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc
[babel]: https://babeljs.io/
[oxc]: https://oxc.rs
[rolldown]: https://vite.dev/guide/rolldown
[swc]: https://swc.rs/
[compiler-docs]: https://react.dev/learn/react-compiler/installation
[react-x]: https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-x
[react-dom]: https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-dom
