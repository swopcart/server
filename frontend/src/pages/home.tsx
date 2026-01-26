import { useAuth } from "@/contexts/auth-context";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { MockOverlay } from "@/components/mock-overlay";

export function HomePage() {
  const { user } = useAuth();

  return (
    <>
      <Header title={`Welcome, ${user?.username}`} />
      <Container className="p-6">
        <MockOverlay>
          <p className="text-muted-foreground">
            Your gaming dashboard and recent activity.
          </p>
        </MockOverlay>
      </Container>
    </>
  );
}
