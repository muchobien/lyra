import type { Manifest } from "material-icon-theme";
import manifestURL from "material-icon-theme/dist/material-icons.json?url";

type Associations = Record<string, string> | undefined;

const iconURLs = import.meta.glob<string>("/node_modules/material-icon-theme/icons/*.svg", {
  eager: true,
  query: "?url",
  import: "default",
});
const manifest: Manifest = await (await fetch(manifestURL)).json();

function iconURL(id: string) {
  const path = manifest.iconDefinitions?.[id]?.iconPath.replace("./../icons/", "");
  return iconURLs[`/node_modules/material-icon-theme/icons/${path}`];
}

function find(names: Associations[], extensions: Associations[], name: string) {
  for (const map of names) {
    if (map?.[name]) {
      return map[name];
    }
  }
  const parts = name.split(".");
  for (let index = 1; index < parts.length; index++) {
    const extension = parts.slice(index).join(".");
    for (const map of extensions) {
      if (map?.[extension]) {
        return map[extension];
      }
    }
  }
  return undefined;
}

export function fileIcon(name: string, folder: boolean) {
  const lower = name.toLowerCase();
  const light = manifest.light;
  if (folder) {
    const dark = manifest.folderNames?.[lower] ?? manifest.folder ?? "folder";
    return { dark: iconURL(dark), light: iconURL(light?.folderNames?.[lower] ?? dark) };
  }
  const dark =
    find([manifest.fileNames], [manifest.fileExtensions], lower) ?? manifest.file ?? "file";
  const lightID =
    find(
      [light?.fileNames, manifest.fileNames],
      [light?.fileExtensions, manifest.fileExtensions],
      lower,
    ) ?? dark;
  return { dark: iconURL(dark), light: iconURL(lightID) };
}
