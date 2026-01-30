import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { useState } from "react";
import { useAuth } from "@/contexts/auth-context";
import { listSessions, revokeSession } from "@/lib/api/auth";
import type { Session } from "@/lib/api/types";

export function BackendTest() {
  const { user, logout } = useAuth();
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<string | undefined>();
  const [sessions, setSessions] = useState<Session[] | null>(null);
  const [sessionsLoading, setSessionsLoading] = useState(false);

  const onPingClicked = () => {
    setPending(true);
    fetch("/api/v0/ping")
      .then((response) => {
        return response.json();
      })
      .then((body) => {
        setMessage(
          `Server responded with ${JSON.stringify(body, undefined, 2)}`,
        );
      })
      .catch((error) => {
        setMessage(`Failed to ping server: ${error}`);
      })
      .finally(() => setPending(false));
  };

  const onListSessionsClicked = async () => {
    setSessionsLoading(true);
    try {
      const response = await listSessions();
      setSessions(response.items);
    } catch (error) {
      setMessage(`Failed to list sessions: ${error}`);
    } finally {
      setSessionsLoading(false);
    }
  };

  const onRevokeSession = async (sessionUuid: string) => {
    try {
      await revokeSession(sessionUuid);
      setSessions(
        (prev) => prev?.filter((s) => s.uuid !== sessionUuid) ?? null,
      );
    } catch (error) {
      setMessage(`Failed to revoke session: ${error}`);
    }
  };

  const onLogoutClicked = async () => {
    setPending(true);
    try {
      await logout();
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="flex flex-col justify-center items-center min-h-screen gap-4 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Welcome, {user?.username}</CardTitle>
        </CardHeader>
        <CardContent>
          {pending ? (
            <Spinner />
          ) : (
            (message ?? "No message received from server.")
          )}
        </CardContent>
        <CardFooter className="flex-col gap-2">
          <Button className="w-full" disabled={pending} onClick={onPingClicked}>
            Ping server
          </Button>
          <Button
            className="w-full"
            variant="outline"
            disabled={sessionsLoading}
            onClick={onListSessionsClicked}
          >
            {sessionsLoading ? <Spinner /> : "List Sessions"}
          </Button>
          <Button
            className="w-full"
            variant="destructive"
            disabled={pending}
            onClick={onLogoutClicked}
          >
            Log out
          </Button>
        </CardFooter>
      </Card>

      {sessions && (
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle>Active Sessions ({sessions.length})</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {sessions.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No active sessions
              </p>
            ) : (
              sessions.map((session) => (
                <div
                  key={session.uuid}
                  className="flex flex-col gap-1 p-3 border rounded-md"
                >
                  <div className="flex justify-between items-start">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">
                        {session.userAgent || "Unknown device"}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {session.ipAddress}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        Created: {new Date(session.createdAt).toLocaleString()}
                      </p>
                    </div>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => onRevokeSession(session.uuid)}
                      disabled={!session.active}
                    >
                      Revoke
                    </Button>
                  </div>
                </div>
              ))
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
