import { AuthProvider, useAuth } from "./contexts/auth-context";
import { BackendTest } from "./pages/backend-test";
import { LoginPage } from "./pages/login";
import { Spinner } from "./components/ui/spinner";

function AppContent() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <Spinner />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginPage />;
  }

  return <BackendTest />;
}

export function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}

export default App;
