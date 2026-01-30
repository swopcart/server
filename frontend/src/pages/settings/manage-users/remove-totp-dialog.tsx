import { useState, type ReactElement } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { adminRemoveTotp } from "@/lib/api";

interface RemoveTotpDialogProps {
  userUuid: string;
  username: string;
  children: ReactElement;
  onSuccess?: () => void;
}

export function RemoveTotpDialog({
  userUuid,
  username,
  children,
  onSuccess,
}: RemoveTotpDialogProps) {
  const [open, setOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleOpenChange = (open: boolean) => {
    setOpen(open);
    if (!open) {
      setError(null);
    }
  };

  const handleRemove = async () => {
    setIsLoading(true);
    setError(null);

    try {
      await adminRemoveTotp(userUuid);
      setOpen(false);
      onSuccess?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to remove TOTP");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={children} />
      <DialogContent>
        <DialogTitle>Remove Two-Factor Authentication</DialogTitle>
        <DialogDescription>
          Are you sure you want to remove two-factor authentication for{" "}
          {username}? This will make their account less secure.
        </DialogDescription>

        {error && <p className="text-destructive text-sm">{error}</p>}

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>
            Cancel
          </DialogClose>
          <Button
            variant="destructive"
            onClick={handleRemove}
            disabled={isLoading}
          >
            {isLoading ? "Removing..." : "Remove TOTP"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
