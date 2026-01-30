import { useState, type ReactElement } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { createUser } from "@/lib/api/users";
import type { UserListItem } from "@/lib/api/users";
import { useAsyncFn } from "@/hooks/use-async";
import { useFieldErrors } from "@/hooks/use-field-errors";

interface CreateUserDialogProps {
  children: ReactElement;
  onSuccess?: (user: UserListItem) => void;
}

export function CreateUserDialog({
  children,
  onSuccess,
}: CreateUserDialogProps) {
  const [open, setOpen] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [admin, setAdmin] = useState(false);
  const [{ data, loading, error }, executeCreate, reset] =
    useAsyncFn(createUser);
  const fieldErrors = useFieldErrors(error);

  const resetState = () => {
    setUsername("");
    setPassword("");
    setAdmin(false);
    reset();
  };

  const handleOpenChange = (open: boolean) => {
    setOpen(open);
    if (!open) {
      resetState();
    }
  };

  const handleCreate = async () => {
    if (!username || !password) return;

    await executeCreate({ username, password, admin });
    if (data) {
      setOpen(false);
      onSuccess?.(data);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={children} />
      <DialogContent>
        <DialogTitle>Create User</DialogTitle>
        <DialogDescription>
          Create a new user account. The user will be able to log in with these
          credentials.
        </DialogDescription>

        <div className="flex flex-col gap-4 py-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="username">Username</Label>
            <Input
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Enter username"
            />
            {fieldErrors.username && (
              <p className="text-destructive text-sm">{fieldErrors.username}</p>
            )}
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter password"
            />
            {fieldErrors.password && (
              <p className="text-destructive text-sm">{fieldErrors.password}</p>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Switch checked={admin} onCheckedChange={setAdmin} id="admin" />
            <Label htmlFor="admin">Administrator</Label>
          </div>
        </div>

        {error && !fieldErrors.username && !fieldErrors.password && (
          <p className="text-destructive text-sm">
            {error instanceof Error ? error.message : "Failed to create user"}
          </p>
        )}

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>
            Cancel
          </DialogClose>
          <Button
            onClick={handleCreate}
            disabled={loading || !username || !password}
          >
            {loading ? "Creating..." : "Create User"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
