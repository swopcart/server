import { ListItem } from "@/components/list-item";
import { LucideChevronRight, LucideKeyRound } from "lucide-react";
import { useAuth } from "@/contexts/auth-context";
import { ChangePasswordDialog } from "@/components/change-password-dialog";

export function ChangePassword() {
  const { user } = useAuth();

  if (!user) return null;

  return (
    <ChangePasswordDialog userUuid={user.uuid} username={user.username}>
      <ListItem
        title="Change password"
        leading={<LucideKeyRound />}
        trailing={<LucideChevronRight />}
      />
    </ChangePasswordDialog>
  );
}
