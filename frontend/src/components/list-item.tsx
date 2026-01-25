import { type ReactNode } from "react";

export interface ListItemProps {
  title: ReactNode;
  subtitle?: ReactNode;
  leading?: ReactNode;
  trailing?: ReactNode;
  onClick?: () => void;
}

export function ListItem({
  title,
  subtitle,
  leading,
  trailing,
  onClick,
}: ListItemProps) {
  return (
    <div
      className="flex gap-3 p-3 focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:aria-invalid:border-destructive/50 rounded-md border bg-clip-padding focus-visible:ring-[3px] aria-invalid:ring-[3px] [&_svg:not([class*='size-'])]:size-4 items-center justify-center whitespace-nowrap transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none shrink-0 [&_svg]:shrink-0 outline-none group/button select-none border-border bg-background hover:bg-muted hover:text-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50 aria-expanded:bg-muted aria-expanded:text-foreground shadow-xs text-left"
      onClick={onClick}
    >
      {leading}
      <div className="flex flex-col grow gap-1">
        <div className="font-medium">{title}</div>
        {subtitle ? <div className="text-sm">{subtitle}</div> : <></>}
      </div>
      {trailing}
    </div>
  );
}
