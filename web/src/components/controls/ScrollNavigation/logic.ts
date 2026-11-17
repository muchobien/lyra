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
    const element = navigation.current;
    const selected = element?.querySelector<HTMLElement>('[aria-current="page"]');
    if (!element || !selected || element.scrollWidth <= element.clientWidth) {
      return;
    }
    const offset = selected.getBoundingClientRect().left - element.getBoundingClientRect().left;
    element.scrollLeft += offset - (element.clientWidth - selected.clientWidth) / 2;
  }, [active]);

  return { navigation };
}
