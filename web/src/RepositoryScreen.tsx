import { useParams } from "@tanstack/react-router";
import { ItemList } from "./ItemList";
import { ItemDetail } from "./ItemDetail";
import { CodeScreens } from "./CodeScreens";
import { ActionsList, RunDetail } from "./Actions";
import { Releases, Containers } from "./Releases";
import { Activity } from "./Activity";
import { Settings } from "./Settings";
import styles from "./Workspace.module.css";

export function RepositoryScreen() {
  const { section = "", _splat = "", owner = "", repo = "" } = useParams({ strict: false });
  const key = `${owner}/${repo}/${section}/${_splat}`;
  if (section === "pulls" || section === "issues") {
    if (_splat) {
      return <ItemDetail key={key} kind={section} number={_splat} />;
    }
    return <ItemList key={key} kind={section} />;
  }
  if (["commits", "commit", "tags"].includes(section)) {
    return <CodeScreens key={key} section={section} target={_splat} />;
  }
  if (section === "runs") {
    return _splat ? <RunDetail key={key} number={_splat} /> : <ActionsList key={key} />;
  }
  if (section === "releases") {
    return <Releases key={key} tag={_splat.startsWith("tag/") ? _splat.slice(4) : ""} />;
  }
  if (section === "images") {
    return <Containers key={key} />;
  }
  if (section === "activity") {
    return <Activity key={key} />;
  }
  if (section === "settings") {
    return <Settings key={key} />;
  }
  return (
    <div className={styles.panel}>
      <p className={styles.empty}>Page not found.</p>
    </div>
  );
}
