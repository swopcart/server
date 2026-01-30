import { useState, type ReactElement } from "react";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel, FieldSet } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { changePassword } from "@/lib/api/users";
import { useAsyncFn } from "@/hooks/use-async";
import { useFieldErrors } from "@/hooks/use-field-errors";

interface ChangePasswordDialogProps {
  userUuid: string;
  username: string;
  /** Whether this is an admin changing another user's password (skips current password requirement) */
  adminMode?: boolean;
  children: ReactElement;
  onSuccess?: () => void;
}

export function ChangePasswordDialog({
  userUuid,
  username,
  adminMode = false,
  children,
  onSuccess,
}: ChangePasswordDialogProps) {
  const [open, setOpen] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);
  const [{ loading, error }, executeChange, reset] = useAsyncFn(
    (userUuid: string, newPassword: string, currentPassword?: string) =>
      changePassword(userUuid, newPassword, currentPassword),
  );
  const fieldErrors = useFieldErrors(error);

  const resetState = () => {
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setValidationError(null);
    reset();
  };

  const handleOpenChange = (open: boolean) => {
    setOpen(open);
    if (!open) {
      resetState();
    }
  };

  const handleSubmit = async () => {
    // Validation
    if (!adminMode && !currentPassword) {
      setValidationError("Current password is required");
      return;
    }
    if (!newPassword) {
      setValidationError("New password is required");
      return;
    }
    if (newPassword !== confirmPassword) {
      setValidationError("Passwords do not match");
      return;
    }

    setValidationError(null);

    if (adminMode) {
      await executeChange(userUuid, newPassword);
    } else {
      await executeChange(userUuid, newPassword, currentPassword);
    }

    if (!error) {
      setOpen(false);
      onSuccess?.();
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={children} />
      <DialogContent>
        <DialogTitle>Change password</DialogTitle>
        <DialogDescription>
          {adminMode
            ? `Set a new password for ${username}.`
            : "Enter your current password and choose a new password."}
        </DialogDescription>

        <FieldSet>
          {!adminMode && (
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="current-password">
                  Current password
                </FieldLabel>
                <Input
                  id="current-password"
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                />
              </Field>
            </FieldGroup>
          )}
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="new-password">New password</FieldLabel>
              <Input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
              {fieldErrors.newPassword && (
                <p className="text-destructive text-sm">
                  {fieldErrors.newPassword}
                </p>
              )}
            </Field>
            <Field>
              <FieldLabel htmlFor="confirm-password">
                Confirm password
              </FieldLabel>
              <Input
                id="confirm-password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
              />
            </Field>
          </FieldGroup>
          {validationError && (
            <p className="text-destructive text-sm">{validationError}</p>
          )}
          {error &&
            !fieldErrors.newPassword &&
            !fieldErrors.currentPassword && (
              <p className="text-destructive text-sm">
                {error instanceof Error
                  ? error.message
                  : "Failed to change password"}
              </p>
            )}
        </FieldSet>

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>
            Cancel
          </DialogClose>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? "Changing..." : "Change password"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
