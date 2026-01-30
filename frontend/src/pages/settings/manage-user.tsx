import { useState } from "react";
import { useParams } from "react-router";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { ListItem } from "@/components/list-item";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { getUserDetails, listUserSessions, disableTOTP } from "@/lib/api/users";
import type { UserDetails } from "@/lib/api/users";
import { revokeSession } from "@/lib/api/auth";
import {
  LucideShield,
  LucideLibrary,
  LucideShieldAlert,
  LucideKeyRound,
  LucideLaptop,
} from "lucide-react";
import { ChangePasswordDialog } from "@/components/change-password-dialog";
import { MockOverlay } from "@/components/mock-overlay";
import { useAsync } from "@/hooks/use-async";

export function ManageUserPage() {
  const { userUuid } = useParams<{ userUuid: string }>();
  const {
    data: user,
    loading,
    error,
  } = useAsync(() => getUserDetails(userUuid!), [userUuid]);

  if (loading) {
    return (
      <>
        <Header title="User Settings" backHref="/settings/users" />
        <div className="p-6 flex justify-center">
          <Spinner />
        </div>
      </>
    );
  }

  if (error || !user) {
    return (
      <>
        <Header title="User Settings" backHref="/settings/users" />
        <div className="p-6">
          <p className="text-destructive">
            {error?.message || "User not found"}
          </p>
        </div>
      </>
    );
  }

  return (
    <>
      <Header title={user.username} backHref="/settings/users" />
      <Container className="p-6 flex flex-col gap-4">
        <div className="flex items-center gap-2">
          <h1 className="text-2xl font-semibold">{user.username}</h1>
          {user.admin && <Badge variant="secondary">Admin</Badge>}
        </div>

        <Tabs defaultValue="admin">
          <TabsList variant="line">
            <TabsTrigger value="admin">
              <LucideShield className="size-4" />
              Admin
            </TabsTrigger>
            <TabsTrigger value="libraries">
              <LucideLibrary className="size-4" />
              Libraries
            </TabsTrigger>
            <TabsTrigger value="parental">
              <LucideShieldAlert className="size-4" />
              Parental
            </TabsTrigger>
            <TabsTrigger value="auth">
              <LucideKeyRound className="size-4" />
              Authentication
            </TabsTrigger>
            <TabsTrigger value="sessions">
              <LucideLaptop className="size-4" />
              Sessions
            </TabsTrigger>
          </TabsList>

          <TabsContent value="admin">
            <AdminTab user={user} />
          </TabsContent>

          <TabsContent value="libraries">
            <LibrariesTab />
          </TabsContent>

          <TabsContent value="parental">
            <ParentalTab />
          </TabsContent>

          <TabsContent value="auth">
            <AuthenticationTab user={user} />
          </TabsContent>

          <TabsContent value="sessions">
            <SessionsTab userUuid={userUuid!} />
          </TabsContent>
        </Tabs>
      </Container>
    </>
  );
}

function AdminTab({ user }: { user: UserDetails }) {
  const [isAdmin, setIsAdmin] = useState(user.admin);

  const handleToggleAdmin = () => {
    // Mock: In production this would call an API
    setIsAdmin(!isAdmin);
  };

  return (
    <div className="flex flex-col gap-4 pt-4">
      <MockOverlay>
        <ListItem
          title="Administrator"
          subtitle="Administrators have full access to all settings and can manage other users."
          htmlFor="admin-toggle"
          trailing={
            <Switch
              id="admin-toggle"
              checked={isAdmin}
              onCheckedChange={handleToggleAdmin}
            />
          }
        />
      </MockOverlay>
    </div>
  );
}

function LibrariesTab() {
  // Mock data for libraries
  const libraries = [
    { id: "1", name: "Games", enabled: true },
    { id: "2", name: "Movies", enabled: true },
    { id: "3", name: "Music", enabled: false },
    { id: "4", name: "Books", enabled: true },
  ];

  const [libraryAccess, setLibraryAccess] = useState(
    libraries.reduce(
      (acc, lib) => {
        acc[lib.id] = lib.enabled;
        return acc;
      },
      {} as Record<string, boolean>,
    ),
  );

  const handleToggleLibrary = (libraryId: string) => {
    // Mock: In production this would call an API
    setLibraryAccess((prev) => ({
      ...prev,
      [libraryId]: !prev[libraryId],
    }));
  };

  return (
    <MockOverlay>
      <div className="flex flex-col gap-4 pt-4">
        <p className="text-sm text-muted-foreground">
          Control which libraries this user can access.
        </p>
        <div className="flex flex-col gap-2">
          {libraries.map((library) => (
            <ListItem
              key={library.id}
              title={library.name}
              htmlFor={`library-${library.id}`}
              trailing={
                <Switch
                  id={`library-${library.id}`}
                  checked={libraryAccess[library.id]}
                  onCheckedChange={() => handleToggleLibrary(library.id)}
                />
              }
            />
          ))}
        </div>
      </div>
    </MockOverlay>
  );
}

function ParentalTab() {
  const [settings, setSettings] = useState({
    enabled: false,
    maxRating: "PG-13",
    restrictAdultContent: true,
    requirePinForPurchases: false,
  });

  return (
    <MockOverlay>
      <div className="flex flex-col gap-4 pt-4">
        <ListItem
          title="Enable Parental Controls"
          subtitle="Restrict content based on age ratings and categories."
          htmlFor="parental-enabled"
          trailing={
            <Switch
              id="parental-enabled"
              checked={settings.enabled}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, enabled: checked })
              }
            />
          }
        />

        {settings.enabled && (
          <>
            <ListItem
              title="Maximum Content Rating"
              subtitle={`Current: ${settings.maxRating}`}
              trailing={
                <Button variant="outline" size="sm">
                  Change
                </Button>
              }
            />

            <ListItem
              title="Restrict Adult Content"
              subtitle="Hide content marked as adult or mature."
              htmlFor="restrict-adult"
              trailing={
                <Switch
                  id="restrict-adult"
                  checked={settings.restrictAdultContent}
                  onCheckedChange={(checked) =>
                    setSettings({ ...settings, restrictAdultContent: checked })
                  }
                />
              }
            />

            <ListItem
              title="Require PIN for Purchases"
              subtitle="Require a PIN to make any purchases."
              htmlFor="require-pin"
              trailing={
                <Switch
                  id="require-pin"
                  checked={settings.requirePinForPurchases}
                  onCheckedChange={(checked) =>
                    setSettings({
                      ...settings,
                      requirePinForPurchases: checked,
                    })
                  }
                />
              }
            />
          </>
        )}
      </div>
    </MockOverlay>
  );
}

function AuthenticationTab({ user }: { user: UserDetails }) {
  const [totpDialogOpen, setTotpDialogOpen] = useState(false);
  const [totpLoading, setTotpLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleRemoveTotp = async () => {
    setTotpLoading(true);
    setError(null);

    try {
      await disableTOTP(user.uuid);
      setTotpDialogOpen(false);
      // Note: Without cache invalidation, user needs to refresh to see change
      window.location.reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to remove TOTP");
    } finally {
      setTotpLoading(false);
    }
  };

  return (
    <div className="flex flex-col gap-4 pt-4">
      <ListItem
        title="Password"
        subtitle="Change the user's password."
        trailing={
          <ChangePasswordDialog
            userUuid={user.uuid}
            username={user.username}
            adminMode
          >
            <Button variant="outline" size="sm">
              Change Password
            </Button>
          </ChangePasswordDialog>
        }
      />

      <ListItem
        title="Two-Factor Authentication"
        subtitle={
          user.totpEnabled
            ? "TOTP is currently enabled."
            : "TOTP is not enabled."
        }
        trailing={
          user.totpEnabled ? (
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setTotpDialogOpen(true)}
            >
              Remove TOTP
            </Button>
          ) : undefined
        }
      />

      <MockOverlay>
        <ListItem
          title="Passkeys"
          subtitle="No passkeys registered."
          trailing={
            <Button variant="outline" size="sm" disabled>
              Manage
            </Button>
          }
        />
      </MockOverlay>

      <Dialog open={totpDialogOpen} onOpenChange={setTotpDialogOpen}>
        <DialogContent>
          <DialogTitle>Remove Two-Factor Authentication</DialogTitle>
          <DialogDescription>
            Are you sure you want to remove two-factor authentication for{" "}
            {user.username}? This will make their account less secure.
          </DialogDescription>

          {error && <p className="text-sm text-destructive">{error}</p>}

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
    </div>
  );
}

function SessionsTab({ userUuid }: { userUuid: string }) {
  const {
    data: sessionsData,
    loading,
    error,
  } = useAsync(() => listUserSessions(userUuid), [userUuid]);
  const sessions = sessionsData?.items ?? [];

  const handleRevoke = (sessionUuid: string) => {
    revokeSession(sessionUuid)
      .then(() => {
        // Reload to show updated sessions
        window.location.reload();
      })
      .catch(() => {
        // Error handling could be improved with state management
        alert("Failed to revoke session");
      });
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <div className="pt-4 flex justify-center">
        <Spinner />
      </div>
    );
  }

  if (error) {
    return (
      <div className="pt-4">
        <p className="text-destructive">
          {error.message || "Failed to load sessions"}
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 pt-4">
      <MockOverlay className="w-fit">
        <Button variant="destructive" size="sm">
          Revoke All Sessions
        </Button>
      </MockOverlay>

      <div className="flex flex-col gap-3">
        {sessions.map((session) => (
          <div
            key={session.uuid}
            className={`p-4 border rounded-lg flex flex-col gap-2 ${
              !session.active ? "opacity-50" : ""
            }`}
          >
            <div className="flex justify-between items-start">
              <div className="flex flex-col gap-1">
                <code className="text-xs text-muted-foreground">
                  {session.uuid}
                </code>
                <p className="text-sm">{session.userAgent}</p>
                <p className="text-xs text-muted-foreground">
                  Created: {formatDate(session.createdAt)}
                </p>
                {session.revokedAt && (
                  <p className="text-xs text-muted-foreground">
                    Revoked: {formatDate(session.revokedAt)}
                  </p>
                )}
              </div>
              {session.active && (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => handleRevoke(session.uuid)}
                >
                  Revoke
                </Button>
              )}
            </div>
          </div>
        ))}

        {sessions.length === 0 && (
          <p className="text-muted-foreground">No sessions found.</p>
        )}
      </div>
    </div>
  );
}
