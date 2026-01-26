import * as React from "react";
import { useNavigate } from "react-router";
import { ArrowLeftIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Separator } from "@/components/ui/separator";

interface HeaderProps {
  /** The page title to display */
  title: string;
  /** Optional actions to display on the right side of the header */
  actions?: React.ReactNode;
  /** Path to navigate to when back button is clicked. If provided, shows the back button */
  backHref?: string;
  /** Additional className for the header container */
  className?: string;
}

export function Header({ title, actions, backHref, className }: HeaderProps) {
  const navigate = useNavigate();

  return (
    <header
      data-slot="header"
      className={cn(
        "flex h-14 shrink-0 items-center gap-2 border-b px-4",
        className,
      )}
    >
      <div className="flex items-center gap-2">
        <SidebarTrigger className="md:hidden" />
        <Separator orientation="vertical" className="h-4 md:hidden" />
        {backHref && (
          <>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => navigate(backHref)}
              aria-label="Go back"
            >
              <ArrowLeftIcon />
            </Button>
            <Separator orientation="vertical" className="h-4" />
          </>
        )}
        <h1 className="text-base font-semibold">{title}</h1>
      </div>
      {actions && (
        <div className="ml-auto flex items-center gap-2">{actions}</div>
      )}
    </header>
  );
}
