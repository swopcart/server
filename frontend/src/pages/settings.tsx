import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { ListItem } from "@/components/list-item";
import { useAuth } from "@/contexts/auth-context";
import {
  LucideChevronRight,
  LucideCircleCheckBig,
  LucideInfo,
  LucideLibraryBig,
  LucideUsers,
} from "lucide-react";
import { useNavigate } from "react-router";

export function SettingsPage() {
  const navigate = useNavigate();
  const auth = useAuth();

  const adminSection = auth.user?.admin ? (
    <div className="flex flex-col gap-2">
      <h1 className="text-1xl font-semibold">Administration</h1>

      <ListItem
        title="Manage users"
        leading={<LucideUsers />}
        trailing={<LucideChevronRight />}
        onClick={() => navigate("/settings/users")}
      />
      <ListItem
        title="Manage library"
        leading={<LucideLibraryBig />}
        trailing={<LucideChevronRight />}
        onClick={() => navigate("/settings/library")}
      />
      <ListItem
        title="Background jobs"
        leading={<LucideCircleCheckBig />}
        trailing={<LucideChevronRight />}
        onClick={() => navigate("/settings/jobs")}
      />
    </div>
  ) : (
    <></>
  );

  return (
    <>
      <Header title="Settings" backHref="/" />
      <Container className="p-6 flex flex-col gap-4">
        <h1 className="text-2xl font-semibold">Settings</h1>

        <div className="flex flex-col gap-2">
          <ListItem
            title="Instance information"
            leading={<LucideInfo />}
            trailing={<LucideChevronRight />}
            onClick={() => navigate("/settings/info")}
          />
        </div>

        {adminSection}
      </Container>
    </>
  );
}
