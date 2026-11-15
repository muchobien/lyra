import { useEffect, useState } from "react";
import styles from "./FileIcon.module.css";

type Icons = { dark: string | undefined; light: string | undefined };
type FileIconProps = { name: string; folder?: boolean };

export function FileIcon({ name, folder = false }: FileIconProps) {
  const [icons, setIcons] = useState<Icons>();
  useEffect(() => {
    let active = true;
    void import("./fileIcons").then(({ fileIcon }) => {
      if (active) {
        setIcons(fileIcon(name, folder));
      }
    });
    return () => {
      active = false;
    };
  }, [name, folder]);
  return (
    <span className={styles.icon} aria-hidden="true">
      {icons?.dark && (
        <img
          className={icons.light === icons.dark ? undefined : styles.dark}
          src={icons.dark}
          alt=""
        />
      )}
      {icons?.light && icons.light !== icons.dark && (
        <img className={styles.light} src={icons.light} alt="" />
      )}
    </span>
  );
}
