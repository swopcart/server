import { useEffect, useState } from "react";
import { useParams } from "react-router";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { ListItem } from "@/components/list-item";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import {
  getUserDetails,
  listUserSessions,
  adminRevokeSession,
  type UserDetails,
  type Session,
} from "@/lib/api";
import {
  LucideShield,
  LucideLibrary,
  LucideShieldAlert,
  LucideKeyRound,
  LucideLaptop,
} from "lucide-react";
import { ChangePasswordDialog } from "@/components/change-password-dialog";

export function ManageUserPage() {
  const { userUuid } = useParams<{ userUuid: string }>();
  const [user, setUser] = useState<UserDetails | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!userUuid) return;

    getUserDetails(userUuid)
      .then((data) => {
        setUser(data);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message || "Failed to load user");
        setLoading(false);
      });
  }, [userUuid]);

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
          <p className="text-destructive">{error || "User not found"}</p>
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
                  setSettings({ ...settings, requirePinForPurchases: checked })
                }
              />
            }
          />
        </>
      )}
    </div>
  );
}

function AuthenticationTab({ user }: { user: UserDetails }) {
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
            <Button variant="destructive" size="sm">
              Remove TOTP
            </Button>
          ) : undefined
        }
      />

      <ListItem
        title="Passkeys"
        subtitle="No passkeys registered."
        trailing={
          <Button variant="outline" size="sm" disabled>
            Manage
          </Button>
        }
      />
    </div>
  );
}

function SessionsTab({ userUuid }: { userUuid: string }) {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listUserSessions(userUuid)
      .then((response) => {
        setSessions(response.items);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message || "Failed to load sessions");
        setLoading(false);
      });
  }, [userUuid]);

  const handleRevoke = (sessionUuid: string) => {
    adminRevokeSession(userUuid, sessionUuid)
      .then(() => {
        setSessions((prev) =>
          prev.map((s) =>
            s.uuid === sessionUuid
              ? { ...s, active: false, revokedAt: new Date().toISOString() }
              : s,
          ),
        );
      })
      .catch((err) => {
        setError(err.message || "Failed to revoke session");
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
        <p className="text-destructive">{error}</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 pt-4">
      <div className="flex gap-2">
        <Button variant="destructive" size="sm">
          Revoke All Sessions
        </Button>
      </div>

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
