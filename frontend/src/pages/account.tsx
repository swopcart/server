import { useAuth } from "@/contexts/auth-context";

export function AccountPage() {
  const { user } = useAuth();

  return (
    <div className="p-6">
      <h1 className="text-2xl font-semibold mb-4">Account</h1>
      <p className="text-muted-foreground">
        Manage your account settings, {user?.username}.
      </p>
    </div>
  );
}
