import { useState } from "react";
import { Container } from "@/components/container";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/contexts/auth-context";
import { listSessions, revokeSession } from "@/lib/api/auth";
import { Spinner } from "@/components/ui/spinner";
import { MockOverlay } from "@/components/mock-overlay";
import { useAsync } from "@/hooks/use-async";

export function AccountSessionsPage() {
  const { sessionId: currentSessionId } = useAuth();
  const { data: sessionsData, loading, error } = useAsync(listSessions);
  const sessions = sessionsData?.items ?? [];
  const [revokeError, setRevokeError] = useState<string | null>(null);

  const handleRevoke = (sessionUuid: string) => {
    revokeSession(sessionUuid)
      .then(() => {
        // Note: This is optimistic local state update
        // A full solution would use query invalidation or refetch
        window.location.reload();
      })
      .catch((err) => {
        setRevokeError(err.message || "Failed to revoke session");
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
      <Container className="p-6 flex flex-col gap-4">
        <h1 className="text-2xl font-semibold">Sessions</h1>
        <p className="text-muted-foreground">
          <Spinner />
        </p>
      </Container>
    );
  }

  if (error) {
    return (
      <Container className="p-6 flex flex-col gap-4">
        <h1 className="text-2xl font-semibold">Sessions</h1>
        <p className="text-destructive">
          {error.message || "Failed to load sessions"}
        </p>
      </Container>
    );
  }

  return (
    <Container className="p-6 flex flex-col gap-4">
      <h1 className="text-2xl font-semibold">Sessions</h1>

      {revokeError && <p className="text-destructive text-sm">{revokeError}</p>}

      <MockOverlay className="w-fit">
        <Button variant="destructive" onClick={handleRevokeAll}>
          Revoke all sessions
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
    </Container>
  );
}
