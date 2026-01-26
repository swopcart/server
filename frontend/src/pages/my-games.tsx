import { Header } from "@/components/header";

export function MyGamesPage() {
  return (
    <>
      <Header title="My Games" />
      <div className="p-6">
        <p className="text-muted-foreground">
          View and manage your game collection.
        </p>
      </div>
    </>
  );
}
