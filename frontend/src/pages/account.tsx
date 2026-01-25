import { ListItem } from "@/components/list-item";
import { useAuth } from "@/contexts/auth-context";
import { LucideChevronRight, LucideLaptop, LucideLogOut } from "lucide-react";
import { useNavigate } from "react-router";
import { ChangePassword } from "./account/login/password";

export function AccountPage() {
  const navigate = useNavigate();
  const { user, logout } = useAuth();

  return (
    <div className="p-6 flex flex-col gap-4">
      <h1 className="text-2xl font-semibold">Account ({user?.username})</h1>

      <div className="flex flex-col gap-2">
        <ChangePassword />

        <ListItem
          title="Active sessions"
          leading={<LucideLaptop />}
          trailing={<LucideChevronRight />}
          onClick={() => navigate("/account/sessions")}
        />
        <ListItem
          title="Log out"
          leading={<LucideLogOut />}
          trailing={<LucideChevronRight />}
          onClick={() => logout()}
        />
      </div>
    </div>
  );
}
