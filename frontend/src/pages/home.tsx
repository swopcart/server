import { useAuth } from "@/contexts/auth-context";

export function HomePage() {
  const { user } = useAuth();

  return (
    <div className="p-6">
      <h1 className="text-2xl font-semibold mb-4">Welcome, {user?.username}</h1>
      <p className="text-muted-foreground">
        Your gaming dashboard and recent activity.
      </p>
    </div>
  );
}
