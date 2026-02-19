import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { FieldGroup, FieldLabel, FieldSet } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  getGame,
  updateGameMetadata,
  type Game,
  type UpdateGameMetadataRequest,
} from "@/lib/api/games";
import type { GameVersion } from "@/lib/api/types";
import { useAsync, useAsyncFn } from "@/hooks/use-async";
import { useFieldErrors } from "@/hooks/use-field-errors";
import { useAuth } from "@/contexts/auth-context";
import { LucideDownload } from "lucide-react";

interface GameDetailsModalProps {
  game: Game;
  onOpenChange: (open: boolean) => void;
}

export function GameDetailsModal({
  game: initialGame,
  onOpenChange,
}: GameDetailsModalProps) {
  const { user } = useAuth();
  const isAdmin = user?.admin || false;

  const { data: game, loading: loadingGame } = useAsync(
    () => getGame(initialGame.id),
    [initialGame.id],
  );
  const [editMode, setEditMode] = useState(false);
  const [title, setTitle] = useState(initialGame.title);
  const [developer, setDeveloper] = useState(initialGame.developer || "");
  const [publisher, setPublisher] = useState(initialGame.publisher || "");
  const [description, setDescription] = useState(initialGame.description || "");

  const [
    { loading: updating, error: updateError },
    executeUpdate,
    resetUpdate,
  ] = useAsyncFn(updateGameMetadata);
  const fieldErrors = useFieldErrors(updateError);

  const handleSaveMetadata = async () => {
    resetUpdate();

    const request: UpdateGameMetadataRequest = {
      title: title.trim(),
      developer: developer.trim() || undefined,
      publisher: publisher.trim() || undefined,
      description: description.trim() || undefined,
    };

    await executeUpdate(game!.id, request);

    if (!updateError) {
      setEditMode(false);
    }
  };

  const handleDownload = (versionId: string, fileName?: string) => {
    // Note: Full download implementation would use the downloadGameVersion API
    // For now, we'll provide a link that the user can click
    const link = `/api/v0/games/${game!.id}/versions/${versionId}/download`;
    const a = document.createElement("a");
    a.href = link;
    a.download = fileName || `${game!.title}.rom`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  const displayGame = game || initialGame;

  return (
    <Dialog open={true} onOpenChange={onOpenChange}>
      <DialogContent className="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{displayGame.title}</DialogTitle>
          <DialogDescription>Game details and versions</DialogDescription>
        </DialogHeader>

        {loadingGame && (
          <div className="flex justify-center items-center py-8">
            <Spinner />
          </div>
        )}

        {!loadingGame && displayGame && (
          <div className="space-y-6">
            {/* Metadata Section */}
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <h3 className="font-semibold text-lg">Details</h3>
                {isAdmin && !editMode && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setTitle(displayGame.title);
                      setDeveloper(displayGame.developer || "");
                      setPublisher(displayGame.publisher || "");
                      setDescription(displayGame.description || "");
                      setEditMode(true);
                    }}
                  >
                    Edit
                  </Button>
                )}
              </div>

              {editMode ? (
                <div className="space-y-3 border rounded-lg p-4 bg-gray-50 dark:bg-gray-900">
                  {updateError && (
                    <div className="p-3 bg-red-100 text-red-800 rounded-lg text-sm">
                      {updateError.message}
                    </div>
                  )}

                  <FieldGroup>
                    <FieldSet>
                      <FieldLabel htmlFor="edit-title">Title</FieldLabel>
                      <Input
                        id="edit-title"
                        value={title}
                        onChange={(e) => setTitle(e.target.value)}
                        disabled={updating}
                      />
                      {fieldErrors.title && (
                        <p className="text-sm text-red-600 dark:text-red-400 mt-1">
                          {fieldErrors.title}
                        </p>
                      )}
                    </FieldSet>
                  </FieldGroup>

                  <FieldGroup>
                    <FieldSet>
                      <FieldLabel htmlFor="edit-developer">
                        Developer
                      </FieldLabel>
                      <Input
                        id="edit-developer"
                        value={developer}
                        onChange={(e) => setDeveloper(e.target.value)}
                        disabled={updating}
                      />
                    </FieldSet>
                  </FieldGroup>

                  <FieldGroup>
                    <FieldSet>
                      <FieldLabel htmlFor="edit-publisher">
                        Publisher
                      </FieldLabel>
                      <Input
                        id="edit-publisher"
                        value={publisher}
                        onChange={(e) => setPublisher(e.target.value)}
                        disabled={updating}
                      />
                    </FieldSet>
                  </FieldGroup>

                  <FieldGroup>
                    <FieldSet>
                      <FieldLabel htmlFor="edit-description">
                        Description
                      </FieldLabel>
                      <Textarea
                        id="edit-description"
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        disabled={updating}
                        rows={4}
                      />
                    </FieldSet>
                  </FieldGroup>

                  <div className="flex gap-2 pt-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setEditMode(false)}
                      disabled={updating}
                    >
                      Cancel
                    </Button>
                    <Button
                      size="sm"
                      onClick={handleSaveMetadata}
                      disabled={updating}
                      className="flex gap-2 items-center"
                    >
                      {updating && <Spinner className="w-4 h-4" />}
                      Save
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="space-y-2">
                  {displayGame.developer && (
                    <div>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        Developer
                      </p>
                      <p className="font-medium">{displayGame.developer}</p>
                    </div>
                  )}
                  {displayGame.publisher && (
                    <div>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        Publisher
                      </p>
                      <p className="font-medium">{displayGame.publisher}</p>
                    </div>
                  )}
                  {displayGame.description && (
                    <div>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        Description
                      </p>
                      <p className="text-sm">{displayGame.description}</p>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Versions Section */}
            {displayGame.versions && displayGame.versions.length > 0 && (
              <div className="space-y-3">
                <h3 className="font-semibold text-lg">Versions</h3>
                <div className="space-y-2 border rounded-lg p-4 bg-gray-50 dark:bg-gray-900">
                  {displayGame.versions.map((version: GameVersion) => (
                    <div
                      key={version.id}
                      className="flex justify-between items-start border-b pb-2 last:border-b-0 last:pb-0"
                    >
                      <div className="flex-1">
                        <p className="font-medium">{version.versionName}</p>
                        <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                          {version.filePath}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                          {(version.fileSize / 1024 / 1024).toFixed(2)} MB
                        </p>
                        {version.md5 && (
                          <p className="text-xs text-gray-500 dark:text-gray-500 mt-1 font-mono break-all">
                            MD5: {version.md5}
                          </p>
                        )}
                      </div>
                      <Button
                        size="sm"
                        onClick={() =>
                          handleDownload(
                            version.id,
                            `${displayGame.title}-${version.versionName}`,
                          )
                        }
                        className="flex gap-1 items-center mt-2"
                      >
                        <LucideDownload className="w-4 h-4" /> Download
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Metadata JSON (if available) */}
            {displayGame.versions?.[0]?.metadataJson && (
              <div className="space-y-3">
                <h3 className="font-semibold text-lg">Additional Metadata</h3>
                <div className="border rounded-lg p-4 bg-gray-50 dark:bg-gray-900 text-sm space-y-2">
                  {displayGame.versions[0].metadataJson?.regions && (
                    <div>
                      <p className="text-gray-600 dark:text-gray-400">
                        Regions
                      </p>
                      <div className="flex gap-2 flex-wrap mt-1">
                        {displayGame.versions[0].metadataJson.regions.map(
                          (region: string) => (
                            <Badge key={region}>{region}</Badge>
                          ),
                        )}
                      </div>
                    </div>
                  )}
                  {displayGame.versions[0].metadataJson?.tags && (
                    <div>
                      <p className="text-gray-600 dark:text-gray-400">Tags</p>
                      <div className="flex gap-2 flex-wrap mt-1">
                        {displayGame.versions[0].metadataJson.tags.map(
                          (tag: string) => (
                            <Badge key={tag} variant="secondary">
                              {tag}
                            </Badge>
                          ),
                        )}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
