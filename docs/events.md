# Waiting for events

Every write is recorded as an event. Agents wait for events instead of polling.

## `lyra event wait`

Blocks until a matching event happens and prints it. Actions are glob patterns (`push`,
`repo.*`).

- Without `--since`, it waits for events after it starts. With `--since ID`, it also returns
  events since that one, so an agent that saved the id of the last event it saw can't miss
  one that happened in between.
- It reconnects on its own if the server restarts.
- `--assignee me` returns when an issue or pull request is assigned to the caller (the agent
  name, or the login without one). This is a work queue; check
  `lyra issue list --assignee @me` first for work assigned earlier.

## `lyra pr wait` and `lyra issue wait`

Block until someone other than the caller acts on that pull request or issue, then print
the event, with the review or comment it carries.

- **Default actions:** pull requests wait for a review, comment, finished checks, merge or
  close; issues wait for anything but their opening. `--for` narrows it.
- **Who "the caller" is:** the token, or the agent and session when `LYRA_SESSION` is set,
  so another session of the same agent counts as someone else.
- **Where the wait starts:** without `--since`, after the later of the caller's last action
  on the item and the last event a wait returned to the caller. So a reply that arrived
  between pushing a fix and starting to wait returns at once, and waiting again doesn't
  return the same event twice. Lyra keeps that position per caller and item. With
  `--since ID`, the wait starts after that event instead.

## Over HTTP

The CLI uses `GET /api/events/stream`, a server-sent event stream. Each event's `id` is its
event id, and the stream accepts `Last-Event-ID` to resume. It starts with an `id` line
holding the current position, and sends a keepalive every 25 seconds. Tokens see only
events for the repositories they can read. See the
[API document](../api/openapi.yaml) for its filters.
