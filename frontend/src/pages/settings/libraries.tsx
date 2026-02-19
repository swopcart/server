import { useState } from "react";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import {
  listLibraries,
  deleteLibrary,
  triggerLibraryScan,
} from "@/lib/api/libraries";
import type { Library } from "@/lib/api/libraries";
import { useAsync, useAsyncFn } from "@/hooks/use-async";
import { LucidePlay, LucideTrash2 } from "lucide-react";
import { CreateLibraryDialog } from "./libraries/create-library-dialog";
import { EditLibraryDialog } from "./libraries/edit-library-dialog";

export function LibrariesPage() {
  const [refreshCounter, setRefreshCounter] = useState(0);
  const {
    data: libraries,
    loading,
    error,
  } = useAsync(listLibraries, [refreshCounter]);
  const [{ error: deleteError }, executeDelete, resetDelete] =
    useAsyncFn(deleteLibrary);
  const [{ loading: scanning, error: scanError }, executeScan, resetScan] =
    useAsyncFn(triggerLibraryScan);

  const [deletingLib, setDeletingLib] = useState<string | null>(null);
  const [scanningLib, setScanningLib] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [editingLib, setEditingLib] = useState<Library | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  const formatScanStatus = (status: string): JSX.Element => {
    const variants: Record<string, "default" | "secondary" | "destructive"> = {
      idle: "default",
      scanning: "secondary",
      error: "destructive",
    };
    return <Badge variant={variants[status] || "default"}>{status}</Badge>;
  };

  const formatDate = (dateString: string | null | undefined) => {
    if (!dateString) return "Never";
    return new Date(dateString).toLocaleString();
  };

  const handleDelete = async (library: Library) => {
    if (!confirm(`Are you sure you want to delete "${library.name}"?`)) {
      return;
    }

    setDeletingLib(library.id);
    setSuccessMessage(null);
    resetDelete();

    await executeDelete(library.id);

    if (!deleteError) {
      setSuccessMessage(`Library "${library.name}" deleted`);
      setTimeout(() => setSuccessMessage(null), 5000);
      setRefreshCounter((prev) => prev + 1);
    }

    setDeletingLib(null);
  };

  const handleScan = async (library: Library) => {
    setScanningLib(library.id);
    setSuccessMessage(null);
    resetScan();

    await executeScan(library.id);

    if (!scanError) {
      setSuccessMessage(`Scan started for "${library.name}"`);
      setTimeout(() => setSuccessMessage(null), 5000);
      // Refresh after a short delay to show the scanning status
      setTimeout(() => setRefreshCounter((prev) => prev + 1), 500);
    }

    setScanningLib(null);
  };

  if (loading) {
    return (
      <Container>
        <Header title="Libraries" subtitle="Manage your game libraries" />
        <div className="flex justify-center items-center py-8">
          <Spinner />
        </div>
      </Container>
    );
  }

  return (
    <Container>
      <div className="flex justify-between items-center mb-6">
        <Header title="Libraries" subtitle="Manage your game libraries" />
        <Button onClick={() => setShowCreateDialog(true)}>
          Create Library
        </Button>
      </div>

      {successMessage && (
        <div className="mb-4 p-4 bg-green-100 text-green-800 rounded-lg">
          {successMessage}
        </div>
      )}

      {error && (
        <div className="mb-4 p-4 bg-red-100 text-red-800 rounded-lg">
          Error loading libraries: {error.message}
        </div>
      )}

      {deleteError && (
        <div className="mb-4 p-4 bg-red-100 text-red-800 rounded-lg">
          Error deleting library: {deleteError.message}
        </div>
      )}

      {scanError && (
        <div className="mb-4 p-4 bg-red-100 text-red-800 rounded-lg">
          Error starting scan: {scanError.message}
        </div>
      )}

      {!libraries || libraries.length === 0 ? (
        <div className="text-center py-8">
          <p className="text-gray-500 mb-4">No libraries yet</p>
          <Button onClick={() => setShowCreateDialog(true)}>
            Create your first library
          </Button>
        </div>
      ) : (
        <div className="space-y-4">
          {libraries.map((library) => (
            <div
              key={library.id}
              className="border rounded-lg p-4 hover:bg-gray-50 dark:hover:bg-gray-900 transition"
            >
              <div className="flex justify-between items-start mb-3">
                <div className="flex-1">
                  <h3 className="font-semibold text-lg">{library.name}</h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                    {library.description}
                  </p>
                  <div className="mt-2 flex gap-2 items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-300">
                      Platform: <strong>{library.platformName}</strong>
                    </span>
                    <span className="text-sm text-gray-600 dark:text-gray-300">
                      •
                    </span>
                    <span className="text-sm text-gray-600 dark:text-gray-300">
                      Games: <strong>{library.gameCount || 0}</strong>
                    </span>
                  </div>
                </div>
                <div className="flex gap-2">
                  {formatScanStatus(library.scanStatus)}
                </div>
              </div>

              <div className="text-xs text-gray-500 dark:text-gray-400 mb-3">
                <p>Paths: {library.paths.join(", ")}</p>
                <p>Last scanned: {formatDate(library.lastScannedAt)}</p>
                {library.lastScanError && (
                  <p className="text-red-600 dark:text-red-400">
                    Last error: {library.lastScanError}
                  </p>
                )}
              </div>

              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setEditingLib(library)}
                  disabled={scanning === true}
                >
                  Edit
                </Button>
                <Button
                  size="sm"
                  onClick={() => handleScan(library)}
                  disabled={
                    scanningLib === library.id ||
                    library.scanStatus === "scanning"
                  }
                  className="flex gap-2 items-center"
                >
                  {scanningLib === library.id ? (
                    <>
                      <Spinner className="w-4 h-4" /> Scanning...
                    </>
                  ) : (
                    <>
                      <LucidePlay className="w-4 h-4" /> Scan
                    </>
                  )}
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => handleDelete(library)}
                  disabled={deletingLib === library.id}
                  className="flex gap-2 items-center"
                >
                  {deletingLib === library.id ? (
                    <>
                      <Spinner className="w-4 h-4" /> Deleting...
                    </>
                  ) : (
                    <>
                      <LucideTrash2 className="w-4 h-4" /> Delete
                    </>
                  )}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <CreateLibraryDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSuccess={() => {
          setShowCreateDialog(false);
          setRefreshCounter((prev) => prev + 1);
        }}
      />

      {editingLib && (
        <EditLibraryDialog
          library={editingLib}
          onOpenChange={(open) => {
            if (!open) setEditingLib(null);
          }}
          onSuccess={() => {
            setEditingLib(null);
            setRefreshCounter((prev) => prev + 1);
          }}
        />
      )}
    </Container>
  );
}
