import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/contexts/auth-context";
import { listSessions, revokeSession, type Session } from "@/lib/api";
import { Spinner } from "@/components/ui/spinner";

export function AccountSessionsPage() {
  const { sessionId: currentSessionId } = useAuth();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listSessions()
      .then((response) => {
        setSessions(response.items);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message || "Failed to load sessions");
        setLoading(false);
      });
  }, []);

  const handleRevoke = (sessionUuid: string) => {
    revokeSession(sessionUuid)
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

  const handleRevokeAll = () => {
    // TODO: Implement revoke all sessions API
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <h1 className="text-2xl font-semibold">Sessions</h1>
        <p className="text-muted-foreground">
          <Spinner />
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <h1 className="text-2xl font-semibold">Sessions</h1>
        <p className="text-destructive">{error}</p>
      </div>
    );
  }

  return (
    <div className="p-6 flex flex-col gap-4">
      <h1 className="text-2xl font-semibold">Sessions</h1>

      <div className="flex gap-2">
        <Button variant="destructive" onClick={handleRevokeAll}>
          Revoke all sessions
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
                <div className="flex items-center gap-2">
                  <code className="text-xs text-muted-foreground">
                    {session.uuid}
                  </code>
                  {session.uuid === currentSessionId && (
                    <Badge variant="secondary">Current</Badge>
                  )}
                </div>
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
