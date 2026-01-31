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
import { disableTOTP } from "@/lib/api/users";
import { useAsyncFn } from "@/hooks/use-async";

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
  const [{ loading, error }, executeDisable, reset] = useAsyncFn(disableTOTP);

  const handleOpenChange = (open: boolean) => {
    setOpen(open);
    if (!open) {
      reset();
    }
  };

  const handleRemove = async () => {
    await executeDisable(userUuid);
    if (!error) {
      setOpen(false);
      onSuccess?.();
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

        {error && (
          <p className="text-destructive text-sm">
            {error instanceof Error ? error.message : "Failed to remove TOTP"}
          </p>
        )}

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>
            Cancel
          </DialogClose>
          <Button
            variant="destructive"
            onClick={handleRemove}
            disabled={loading}
          >
            {loading ? "Removing..." : "Remove TOTP"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
