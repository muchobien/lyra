import { type Actor } from "@/api";
import "@/styles/workspace.css";

type ActorProps = { actor: Actor };

export function ActorName({ actor }: ActorProps) {
  return (
    <span title={actor.agent ? `${actor.agent} for ${actor.user}` : actor.user}>
      {actor.agent ? (
        <>
          <strong>{actor.agent}</strong>
          <span className="lyra-workspace-muted"> for </span>
          {actor.user}
        </>
      ) : (
        actor.user
      )}
    </span>
  );
}
