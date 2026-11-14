import { defineConfig } from "orval";

export default defineConfig({
  lyra: {
    input: {
      target: "../api/openapi.yaml",
      filters: { operationIds: ["streamEvents", "createBackup"], mode: "exclude" },
    },
    output: {
      target: "src/api/generated/lyra.ts",
      schemas: { path: "src/api/generated/models.ts", mode: "single" },
      mode: "split",
      client: "react-query",
      httpClient: "fetch",
      baseUrl: "",
      clean: true,
      urlEncodeParameters: true,
      override: {
        header: false,
        fetch: { includeHttpResponseReturnType: false },
        mutator: { path: "src/api/transport.ts", name: "orvalFetch" },
        query: { signal: true },
        operations: {
          renderMarkdown: { query: { useQuery: true, useMutation: false } },
          previewReleaseNotes: { query: { useQuery: false, useMutation: true } },
          downloadAsset: { requestOptions: { responseType: "blob" } },
          downloadAssetByTag: { requestOptions: { responseType: "blob" } },
          downloadLatestAsset: { requestOptions: { responseType: "blob" } },
          getRepositoryRawFile: { requestOptions: { responseType: "blob" } },
        },
      },
    },
  },
});
