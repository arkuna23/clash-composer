import type { ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";

interface PageTransitionProps {
  children: ReactNode;
  className?: string;
}

/**
 * Re-mounts on route change (key=pathname) so the enter animation replays
 * when the user navigates between pages.
 */
export function PageTransition({ children, className }: PageTransitionProps) {
  const location = useLocation();
  return (
    <div
      key={location.pathname}
      className={cn(
        "animate-in fade-in slide-in-from-bottom-1 duration-300 ease-out motion-reduce:animate-none",
        className,
      )}
    >
      {children}
    </div>
  );
}
