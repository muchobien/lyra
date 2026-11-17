import { useLogic, type ResponsiveDetailsProps } from "./logic";

export function ResponsiveDetails({ query, className, children }: ResponsiveDetailsProps) {
  const { initiallyOpen } = useLogic({ query });
  return (
    <details className={className} open={initiallyOpen}>
      {children}
    </details>
  );
}
