import { ListItem } from "@/components/list-item";
import { useAuth } from "@/contexts/auth-context";
import { LucideChevronRight, LucideLaptop, LucideLogOut } from "lucide-react";
import { useNavigate } from "react-router";
import { ChangePassword } from "./account/login/password";
import { Header } from "@/components/header";
import { TotpSettings } from "./account/login/totp";

export function AccountPage() {
  const navigate = useNavigate();
  const { logout } = useAuth();

  return (
    <>
      <Header title="Account" backHref="/" />
      <div className="p-6 flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          <ChangePassword />
          <TotpSettings />

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
    </>
  );
}
