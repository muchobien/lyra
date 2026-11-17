import "./styles.css";
import { useLogic, type FileIconProps } from "./logic";

export function FileIcon({ name, folder = false }: FileIconProps) {
  const { icons } = useLogic({ name, folder });
  return (
    <span className="lyra-file-icon-icon" aria-hidden="true">
      {icons?.dark && (
        <img
          className={icons.light === icons.dark ? undefined : "lyra-file-icon-dark"}
          src={icons.dark}
          alt=""
        />
      )}
      {icons?.light && icons.light !== icons.dark && (
        <img className="lyra-file-icon-light" src={icons.light} alt="" />
      )}
    </span>
  );
}
