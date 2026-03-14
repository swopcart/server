import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { FieldGroup, FieldLabel, FieldSet } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Spinner } from "@/components/ui/spinner";
import {
  updateLibrary,
  type Library,
  type UpdateLibraryRequest,
} from "@/lib/api/libraries";
import { useAsyncFn } from "@/hooks/use-async";
import { useFieldErrors } from "@/hooks/use-field-errors";

interface EditLibraryDialogProps {
  library: Library;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function EditLibraryDialog({
  library,
  onOpenChange,
  onSuccess,
}: EditLibraryDialogProps) {
  const [name, setName] = useState(library.name);
  const [description, setDescription] = useState(library.description);
  const [paths, setPaths] = useState(library.paths.join("\n"));
  const [validationError, setValidationError] = useState<string | null>(null);

  const [{ loading, error }, executeUpdate, reset] = useAsyncFn(updateLibrary);
  const fieldErrors = useFieldErrors(error);

  useEffect(() => {
    setName(library.name);
    setDescription(library.description);
    setPaths(library.paths.join("\n"));
  }, [library]);

  const handleOpenChange = (open: boolean) => {
    onOpenChange(open);
    if (!open) {
      reset();
      setValidationError(null);
    }
  };

  const handleSubmit = async () => {
    // Validate
    if (!name.trim()) {
      setValidationError("Library name is required");
      return;
    }
    if (!paths.trim()) {
      setValidationError("At least one path is required");
      return;
    }

    setValidationError(null);

    const pathList = paths
      .split("\n")
      .map((p: string) => p.trim())
      .filter((p: string) => p.length > 0);

    const request: UpdateLibraryRequest = {
      name: name.trim(),
      description: description.trim(),
      paths: pathList,
    };

    await executeUpdate(library.id, request);

    if (!error) {
      handleOpenChange(false);
      onSuccess();
    }
  };

  return (
    <Dialog open={true} onOpenChange={handleOpenChange}>
      <DialogContent className="w-full max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Library</DialogTitle>
          <DialogDescription>Update library settings</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {validationError && (
            <div className="p-3 bg-red-100 text-red-800 rounded-lg text-sm">
              {validationError}
            </div>
          )}

          {error && !fieldErrors.name && !fieldErrors.paths && (
            <div className="p-3 bg-red-100 text-red-800 rounded-lg text-sm">
              {error.message}
            </div>
          )}

          <FieldGroup>
            <FieldSet>
              <FieldLabel htmlFor="name">Library Name</FieldLabel>
              <Input
                id="name"
                placeholder="e.g., My NES Games"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={loading}
              />
              {fieldErrors.name && (
                <p className="text-sm text-red-600 dark:text-red-400 mt-1">
                  {fieldErrors.name}
                </p>
              )}
            </FieldSet>
          </FieldGroup>

          <FieldGroup>
            <FieldSet>
              <FieldLabel htmlFor="description">Description</FieldLabel>
              <Textarea
                id="description"
                placeholder="Optional description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={loading}
                rows={3}
              />
            </FieldSet>
          </FieldGroup>

          <FieldGroup>
            <FieldSet>
              <FieldLabel htmlFor="paths">Paths (one per line)</FieldLabel>
              <Textarea
                id="paths"
                placeholder="/mnt/games/nes&#10;/home/user/games/nes"
                value={paths}
                onChange={(e) => setPaths(e.target.value)}
                disabled={loading}
                rows={4}
              />
              {fieldErrors.paths && (
                <p className="text-sm text-red-600 dark:text-red-400 mt-1">
                  {fieldErrors.paths}
                </p>
              )}
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Enter one directory path per line.
              </p>
            </FieldSet>
          </FieldGroup>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={loading}
            className="flex gap-2 items-center"
          >
            {loading && <Spinner className="w-4 h-4" />}
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
