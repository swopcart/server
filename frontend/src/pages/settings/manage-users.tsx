import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import {
  listUsers,
  adminRemoveTotp,
  deleteUser,
  createUser,
  type UserListItem,
} from "@/lib/api";
import {
  LucideKeyRound,
  LucideShieldOff,
  LucideTrash2,
  LucideSettings,
  LucideUserPlus,
} from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { ChangePasswordDialog } from "@/components/change-password-dialog";

export function ManageUsersPage() {
  const navigate = useNavigate();
  const [users, setUsers] = useState<UserListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Remove TOTP dialog state
  const [totpDialogOpen, setTotpDialogOpen] = useState(false);
  const [totpDialogUser, setTotpDialogUser] = useState<UserListItem | null>(
    null,
  );
  const [totpLoading, setTotpLoading] = useState(false);

  // Delete user dialog state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteDialogUser, setDeleteDialogUser] = useState<UserListItem | null>(
    null,
  );
  const [deleteLoading, setDeleteLoading] = useState(false);

  // Create user dialog state
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [createUsername, setCreateUsername] = useState("");
  const [createPassword, setCreatePassword] = useState("");
  const [createAdmin, setCreateAdmin] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);

  useEffect(() => {
    listUsers()
      .then((response) => {
        setUsers(response.items);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message || "Failed to load users");
        setLoading(false);
      });
  }, []);

  const handleOpenTotpDialog = (user: UserListItem) => {
    setTotpDialogUser(user);
    setTotpDialogOpen(true);
  };

  const handleRemoveTotp = async () => {
    if (!totpDialogUser) return;

    setTotpLoading(true);

    try {
      await adminRemoveTotp(totpDialogUser.uuid);
      setUsers((prev) =>
        prev.map((u) =>
          u.uuid === totpDialogUser.uuid ? { ...u, totpEnabled: false } : u,
        ),
      );
      setTotpDialogOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to remove TOTP");
    } finally {
      setTotpLoading(false);
    }
  };

  const handleOpenDeleteDialog = (user: UserListItem) => {
    setDeleteDialogUser(user);
    setDeleteDialogOpen(true);
  };

  const handleDeleteUser = async () => {
    if (!deleteDialogUser) return;

    setDeleteLoading(true);

    try {
      await deleteUser(deleteDialogUser.uuid);
      setUsers((prev) => prev.filter((u) => u.uuid !== deleteDialogUser.uuid));
      setDeleteDialogOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete user");
    } finally {
      setDeleteLoading(false);
    }
  };

  const handleCreateUser = async () => {
    if (!createUsername || !createPassword) return;

    setCreateLoading(true);

    try {
      const newUser = await createUser({
        username: createUsername,
        password: createPassword,
        admin: createAdmin,
      });
      setUsers((prev) => [...prev, newUser]);
      setCreateDialogOpen(false);
      setCreateUsername("");
      setCreatePassword("");
      setCreateAdmin(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create user");
    } finally {
      setCreateLoading(false);
    }
  };

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
          <p className="text-destructive">{error}</p>
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
          <Button size="sm" onClick={() => setCreateDialogOpen(true)}>
            <LucideUserPlus className="size-4 md:mr-1" />
            <span className="hidden md:inline">Create User</span>
          </Button>
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
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleOpenTotpDialog(user)}
                  >
                    <LucideShieldOff className="size-4 mr-1" />
                    Remove TOTP
                  </Button>
                )}
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => handleOpenDeleteDialog(user)}
                >
                  <LucideTrash2 className="size-4 mr-1" />
                  Delete
                </Button>
              </div>
            </div>
          ))}

          {users.length === 0 && (
            <p className="text-muted-foreground">No users found.</p>
          )}
        </div>
      </Container>

      {/* Remove TOTP Dialog */}
      <Dialog open={totpDialogOpen} onOpenChange={setTotpDialogOpen}>
        <DialogContent>
          <DialogTitle>Remove Two-Factor Authentication</DialogTitle>
          <DialogDescription>
            Are you sure you want to remove two-factor authentication for{" "}
            {totpDialogUser?.username}? This will make their account less
            secure.
          </DialogDescription>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>
              Cancel
            </DialogClose>
            <Button
              variant="destructive"
              onClick={handleRemoveTotp}
              disabled={totpLoading}
            >
              {totpLoading ? "Removing..." : "Remove TOTP"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete User Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogTitle>Delete User</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete {deleteDialogUser?.username}? This
            action cannot be undone.
          </DialogDescription>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>
              Cancel
            </DialogClose>
            <Button
              variant="destructive"
              onClick={handleDeleteUser}
              disabled={deleteLoading}
            >
              {deleteLoading ? "Deleting..." : "Delete User"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create User Dialog */}
      <Dialog
        open={createDialogOpen}
        onOpenChange={(open) => {
          setCreateDialogOpen(open);
          if (!open) {
            setCreateUsername("");
            setCreatePassword("");
            setCreateAdmin(false);
          }
        }}
      >
        <DialogContent>
          <DialogTitle>Create User</DialogTitle>
          <DialogDescription>
            Create a new user account. The user will be able to log in with
            these credentials.
          </DialogDescription>

          <div className="flex flex-col gap-4 py-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={createUsername}
                onChange={(e) => setCreateUsername(e.target.value)}
                placeholder="Enter username"
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={createPassword}
                onChange={(e) => setCreatePassword(e.target.value)}
                placeholder="Enter password"
              />
            </div>
            <div className="flex items-center gap-2">
              <Switch
                checked={createAdmin}
                onCheckedChange={setCreateAdmin}
                id="admin"
              />
              <Label htmlFor="admin">Administrator</Label>
            </div>
          </div>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>
              Cancel
            </DialogClose>
            <Button
              onClick={handleCreateUser}
              disabled={createLoading || !createUsername || !createPassword}
            >
              {createLoading ? "Creating..." : "Create User"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
