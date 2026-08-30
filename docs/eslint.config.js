import js from "@eslint/js";
import eslintPluginAstro from "eslint-plugin-astro";

export default [
  {
    ignores: [".astro/**", "dist/**"],
  },
  js.configs.recommended,
  ...eslintPluginAstro.configs.recommended,
  {
    files: ["astro.config.mjs"],
    languageOptions: {
      globals: {
        URL: "readonly",
      },
    },
  },
];
