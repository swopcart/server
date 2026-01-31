import { useState } from "react";
import { useNavigate } from "react-router";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { listUsers } from "@/lib/api/users";
import type { UserListItem } from "@/lib/api/users";
import {
  LucideKeyRound,
  LucideShieldOff,
  LucideTrash2,
  LucideSettings,
  LucideUserPlus,
} from "lucide-react";
import { ChangePasswordDialog } from "@/components/change-password-dialog";
import { RemoveTotpDialog } from "./manage-users/remove-totp-dialog";
import { DeleteUserDialog } from "./manage-users/delete-user-dialog";
import { CreateUserDialog } from "./manage-users/create-user-dialog";
import { useAsync } from "@/hooks/use-async";

export function ManageUsersPage() {
  const navigate = useNavigate();
  const { data: usersData, loading, error } = useAsync(listUsers);
  const [users, setUsers] = useState<UserListItem[]>(usersData?.items ?? []);

  // Sync users state when data changes
  if (usersData && users.length === 0 && usersData.items.length > 0) {
    setUsers(usersData.items);
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  if (loading) {
    return (
      <>
        <Header title="Manage Users" backHref="/settings" />
        <div className="p-6 flex justify-center">
          <Spinner />
        </div>
      </>
    );
  }

  if (error) {
    return (
      <>
        <Header title="Manage Users" backHref="/settings" />
        <div className="p-6">
          <p className="text-destructive">
            {error.message || "Failed to load users"}
          </p>
        </div>
      </>
    );
  }

  return (
    <>
      <Header
        title="Manage Users"
        backHref="/settings"
        actions={
          <CreateUserDialog
            onSuccess={(newUser) => setUsers((prev) => [...prev, newUser])}
          >
            <Button size="sm">
              <LucideUserPlus className="size-4 md:mr-1" />
              <span className="hidden md:inline">Create User</span>
            </Button>
          </CreateUserDialog>
        }
      />
      <Container className="p-6 flex flex-col gap-4">
        <div className="flex flex-col gap-3">
          {users.map((user) => (
            <div
              key={user.uuid}
              className="p-4 border rounded-lg flex flex-col gap-3"
            >
              <div className="flex justify-between items-start">
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{user.username}</span>
                    {user.admin && <Badge variant="secondary">Admin</Badge>}
                    {user.totpEnabled && <Badge variant="outline">2FA</Badge>}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Created: {formatDate(user.createdAt)}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => navigate(`/settings/users/${user.uuid}`)}
                  title="User settings"
                >
                  <LucideSettings className="size-4" />
                </Button>
              </div>
              <div className="flex gap-2 flex-wrap">
                <ChangePasswordDialog
                  userUuid={user.uuid}
                  username={user.username}
                  adminMode
                >
                  <Button variant="outline" size="sm">
                    <LucideKeyRound className="size-4 mr-1" />
                    Change Password
                  </Button>
                </ChangePasswordDialog>
                {user.totpEnabled && (
                  <RemoveTotpDialog
                    userUuid={user.uuid}
                    username={user.username}
                    onSuccess={() =>
                      setUsers((prev) =>
                        prev.map((u) =>
                          u.uuid === user.uuid
                            ? { ...u, totpEnabled: false }
                            : u,
                        ),
                      )
                    }
                  >
                    <Button variant="outline" size="sm">
                      <LucideShieldOff className="size-4 mr-1" />
                      Remove TOTP
                    </Button>
                  </RemoveTotpDialog>
                )}
                <DeleteUserDialog
                  userUuid={user.uuid}
                  username={user.username}
                  onSuccess={() =>
                    setUsers((prev) => prev.filter((u) => u.uuid !== user.uuid))
                  }
                >
                  <Button variant="destructive" size="sm">
                    <LucideTrash2 className="size-4 mr-1" />
                    Delete
                  </Button>
                </DeleteUserDialog>
              </div>
            </div>
          ))}

          {users.length === 0 && (
            <p className="text-muted-foreground">No users found.</p>
          )}
        </div>
      </Container>
    </>
  );
}
