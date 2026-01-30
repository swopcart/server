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
import { deleteUser } from "@/lib/api/users";
import { useAsyncFn } from "@/hooks/use-async";

interface DeleteUserDialogProps {
  userUuid: string;
  username: string;
  children: ReactElement;
  onSuccess?: () => void;
}

export function DeleteUserDialog({
  userUuid,
  username,
  children,
  onSuccess,
}: DeleteUserDialogProps) {
  const [open, setOpen] = useState(false);
  const [{ loading, error }, executeDelete, reset] = useAsyncFn(deleteUser);

  const handleOpenChange = (open: boolean) => {
    setOpen(open);
    if (!open) {
      reset();
    }
  };

  const handleDelete = async () => {
    await executeDelete(userUuid);
    if (!error) {
      setOpen(false);
      onSuccess?.();
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={children} />
      <DialogContent>
        <DialogTitle>Delete User</DialogTitle>
        <DialogDescription>
          Are you sure you want to delete {username}? This action cannot be
          undone.
        </DialogDescription>

        {error && (
          <p className="text-destructive text-sm">
            {error instanceof Error ? error.message : "Failed to delete user"}
          </p>
        )}

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>
            Cancel
          </DialogClose>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={loading}
          >
            {loading ? "Deleting..." : "Delete User"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
