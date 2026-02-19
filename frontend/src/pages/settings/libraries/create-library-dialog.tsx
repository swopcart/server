import { useState } from "react";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { createLibrary, type CreateLibraryRequest } from "@/lib/api/libraries";
import { listPlatforms } from "@/lib/api/platforms";
import { useAsync, useAsyncFn } from "@/hooks/use-async";
import { useFieldErrors } from "@/hooks/use-field-errors";

interface CreateLibraryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function CreateLibraryDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateLibraryDialogProps) {
  const { data: platforms, loading: loadingPlatforms } = useAsync(
    listPlatforms,
    [],
  );

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [platformId, setPlatformId] = useState("");
  const [paths, setPaths] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);

  const [{ loading, error }, executeCreate, reset] = useAsyncFn(createLibrary);
  const fieldErrors = useFieldErrors(error);

  const handleOpenChange = (newOpen: boolean) => {
    onOpenChange(newOpen);
    if (!newOpen) {
      // Reset form
      setName("");
      setDescription("");
      setPlatformId("");
      setPaths("");
      setValidationError(null);
      reset();
    }
  };

  const handleSubmit = async () => {
    // Validate
    if (!name.trim()) {
      setValidationError("Library name is required");
      return;
    }
    if (!platformId) {
      setValidationError("Platform is required");
      return;
    }
    if (!paths.trim()) {
      setValidationError("At least one path is required");
      return;
    }

    setValidationError(null);

    const pathList = paths
      .split("\n")
      .map((p) => p.trim())
      .filter((p) => p.length > 0);

    const request: CreateLibraryRequest = {
      name: name.trim(),
      description: description.trim(),
      platformId: parseInt(platformId),
      paths: pathList,
    };

    await executeCreate(request);

    if (!error) {
      handleOpenChange(false);
      onSuccess();
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="w-full max-w-md">
        <DialogHeader>
          <DialogTitle>Create Library</DialogTitle>
          <DialogDescription>
            Create a new game library by specifying a platform and paths to scan
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {validationError && (
            <div className="p-3 bg-red-100 text-red-800 rounded-lg text-sm">
              {validationError}
            </div>
          )}

          {error &&
            !fieldErrors.name &&
            !fieldErrors.platformId &&
            !fieldErrors.paths && (
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
              <FieldLabel htmlFor="platform">Platform</FieldLabel>
              {loadingPlatforms ? (
                <div className="flex items-center gap-2 p-2">
                  <Spinner className="w-4 h-4" /> Loading platforms...
                </div>
              ) : (
                <Select value={platformId} onValueChange={setPlatformId}>
                  <SelectTrigger id="platform" disabled={loading}>
                    <SelectValue placeholder="Select a platform" />
                  </SelectTrigger>
                  <SelectContent>
                    {platforms?.map((platform) => (
                      <SelectItem
                        key={platform.id}
                        value={platform.id.toString()}
                      >
                        {platform.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              {fieldErrors.platformId && (
                <p className="text-sm text-red-600 dark:text-red-400 mt-1">
                  {fieldErrors.platformId}
                </p>
              )}
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
                Enter one directory path per line. These paths will be scanned
                for games.
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
            Create Library
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
