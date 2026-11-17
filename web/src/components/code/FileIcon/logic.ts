import { useEffect, useState } from "react";

type Icons = { dark: string | undefined; light: string | undefined };

export type FileIconProps = { name: string; folder?: boolean };

export function useLogic({ name, folder = false }: Pick<FileIconProps, "name" | "folder">) {
  const [icons, setIcons] = useState<Icons>();
  useEffect(() => {
    let active = true;
    void import("./icons").then(({ fileIcon }) => {
      if (active) {
        setIcons(fileIcon(name, folder));
      }
    });
    return () => {
      active = false;
    };
  }, [name, folder]);

  return { icons };
}
