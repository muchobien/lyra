export function duration(start?: string | null, stop?: string | null) {
  if (!start) {
    return "Not started";
  }
  const seconds = Math.max(
    0,
    Math.floor(((stop ? Date.parse(stop) : Date.now()) - Date.parse(start)) / 1000),
  );
  return new Intl.DurationFormat(undefined, { style: "narrow" }).format({
    hours: Math.floor(seconds / 3600),
    minutes: Math.floor((seconds % 3600) / 60),
    seconds: seconds % 60,
  });
}
