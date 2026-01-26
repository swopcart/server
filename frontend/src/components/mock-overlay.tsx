import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface MockOverlayProps {
  children: ReactNode;
  variant?: "yellow" | "red";
  className?: string;
}

export function MockOverlay({
  children,
  variant = "yellow",
  className,
}: MockOverlayProps) {
  const stripeColor =
    variant === "red" ? "rgba(239, 68, 68, 0.15)" : "rgba(234, 179, 8, 0.15)";

  return (
    <div className={cn("relative", className)}>
      {children}
      <div
        className="absolute inset-0 pointer-events-none rounded-[inherit]"
        style={{
          backgroundImage: `repeating-linear-gradient(
            -45deg,
            ${stripeColor},
            ${stripeColor} 10px,
            transparent 10px,
            transparent 20px
          )`,
        }}
      />
    </div>
  );
}
