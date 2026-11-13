import { defineConfig } from "vite-plus";
import react from "@vitejs/plugin-react";

export default defineConfig(({ command }) => {
  const backend = {
    target: process.env.LYRA_DEV_URL ?? "http://127.0.0.1:8080",
    headers: process.env.LYRA_DEV_LOGIN
      ? { "Tailscale-User-Login": process.env.LYRA_DEV_LOGIN }
      : undefined,
  };
  return {
    plugins: [react()],
    base: command === "build" ? "/static/" : "/",
    build: {
      target: "chrome157",
      cssTarget: "chrome157",
      outDir: "../internal/webui/dist",
      emptyOutDir: true,
    },
    server: {
      proxy: {
        "/api": backend,
        "/v2": backend,
        "^/[^/]+/[^/]+[.]git/": backend,
        "^/[^/]+/[^/]+/(pulls|issues|runs|commits|commit|tags|releases|images|tree|blob|info|git-upload-pack|git-receive-pack)(/|$)":
          backend,
      },
    },
    fmt: {},
    lint: {
      plugins: ["typescript", "react", "jsx-a11y"],
      jsPlugins: [{ name: "vite-plus", specifier: "vite-plus/oxlint-plugin" }],
      rules: { "vite-plus/prefer-vite-plus-imports": "error" },
      options: { typeAware: true, typeCheck: true },
    },
  };
});
