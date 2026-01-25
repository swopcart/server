import { BrowserRouter, Routes, Route } from "react-router";
import { AuthProvider, useAuth } from "./contexts/auth-context";
import { ThemeProvider } from "./components/theme-provider";
import { LoginPage } from "./pages/login";
import { HomePage } from "./pages/home";
import { LibraryPage } from "./pages/library";
import { MyGamesPage } from "./pages/my-games";
import { SettingsPage } from "./pages/settings";
import { AccountPage } from "./pages/account";
import { Spinner } from "./components/ui/spinner";
import {
  SidebarProvider,
  SidebarInset,
  SidebarTrigger,
} from "./components/ui/sidebar";
import { AppSidebar } from "./components/app-sidebar";
import { ManageUsersPage } from "./pages/settings/manage-users";
import { AccountSessionsPage } from "./pages/account/sessions";

function AuthenticatedLayout() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-14 items-center gap-2 border-b px-4 md:hidden">
          <SidebarTrigger />
          <span className="font-semibold">Swopcart</span>
        </header>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/library" element={<LibraryPage />} />
          <Route path="/my-games" element={<MyGamesPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/settings/users" element={<ManageUsersPage />} />
          <Route path="/account" element={<AccountPage />} />
          <Route path="/account/sessions" element={<AccountSessionsPage />} />
        </Routes>
      </SidebarInset>
    </SidebarProvider>
  );
}

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

  return <AuthenticatedLayout />;
}

export function App() {
  return (
    <BrowserRouter>
      <ThemeProvider defaultTheme="system">
        <AuthProvider>
          <AppContent />
        </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  );
}

export default App;
