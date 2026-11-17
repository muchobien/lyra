import { useLogic, type ScrollNavigationProps } from "./logic";

export function ScrollNavigation({ label, active, className, children }: ScrollNavigationProps) {
  const { navigation } = useLogic({ active });
  return (
    <nav ref={navigation} className={className} aria-label={label}>
      {children}
    </nav>
  );
}
