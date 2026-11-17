type DateProps = { value?: string | null };

export function DateTime({ value }: DateProps) {
  if (!value) {
    return <span>—</span>;
  }
  return (
    <time dateTime={value} title={new Date(value).toLocaleString()}>
      {new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(value))}
    </time>
  );
}
