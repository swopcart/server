import { Header } from "@/components/header";

export function LibraryPage() {
  return (
    <>
      <Header title="Library" />
      <div className="p-6">
        <p className="text-muted-foreground">
          Browse and discover games in the library.
        </p>
      </div>
    </>
  );
}
