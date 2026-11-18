import { useEffect, useRef, type ReactNode } from "react";

export type ScrollNavigationProps = {
  label: string;
  active: string;
  className: string;
  children: ReactNode;
};

export function useLogic({ active }: Pick<ScrollNavigationProps, "active">) {
  const navigation = useRef<HTMLElement>(null);
  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      navigation.current?.querySelector('[aria-current="page"]')?.scrollIntoView({
        inline: "center",
        block: "nearest",
        container: "nearest",
      } as ScrollIntoViewOptions);
    });
    return () => cancelAnimationFrame(frame);
  }, [active]);

  return { navigation };
}
