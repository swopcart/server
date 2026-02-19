import { useParams, useNavigate } from "react-router";
import { Container } from "@/components/container";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { MockOverlay } from "@/components/mock-overlay";
import { getGame, downloadGameVersion } from "@/lib/api/games";
import { useAsync } from "@/hooks/use-async";
import {
  LucideChevronLeft,
  LucideDownload,
  LucidePlay,
  LucideUsers,
  LucideCalendar,
} from "lucide-react";

export function GameDetailsPage() {
  const { gameId } = useParams<{ gameId: string }>();
  const navigate = useNavigate();

  const {
    data: game,
    loading,
    error,
  } = useAsync(
    () => (gameId ? getGame(gameId) : Promise.reject(new Error("No game ID"))),
    [gameId],
  );

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <Spinner />
      </div>
    );
  }

  if (error || !game) {
    return (
      <Container>
        <div className="py-8">
          <Button
            variant="outline"
            onClick={() => navigate(-1)}
            className="flex gap-2 items-center mb-4"
          >
            <LucideChevronLeft className="w-4 h-4" /> Back
          </Button>
          <div className="text-center py-12">
            <p className="text-red-600 mb-4">Error loading game</p>
            <p className="text-gray-500">
              {error?.message || "Game not found"}
            </p>
          </div>
        </div>
      </Container>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-black">
      {/* Hero Section */}
      <div className="relative h-96 bg-gradient-to-b from-gray-200 to-gray-100 dark:from-gray-800 dark:to-gray-900 overflow-hidden">
        <MockOverlay className="w-full h-full">
          <div className="w-full h-full" />
        </MockOverlay>

        {/* Metadata Overlay */}
        <div className="absolute inset-0 flex flex-col justify-end p-6 bg-gradient-to-t from-black/80 via-transparent to-transparent">
          <div className="flex items-center gap-4 mb-4">
            <div>
              <h1 className="text-4xl font-bold text-white mb-2">
                {game.title}
              </h1>
              <div className="flex items-center gap-6 text-gray-200">
                {game.releasedDate && (
                  <div className="flex items-center gap-2">
                    <LucideCalendar className="w-4 h-4" />
                    <span>{new Date(game.releasedDate).getFullYear()}</span>
                  </div>
                )}
                {game.developer && (
                  <div className="flex items-center gap-2">
                    <LucideUsers className="w-4 h-4" />
                    <span>{game.developer}</span>
                  </div>
                )}
              </div>
              {game.publisher && (
                <p className="text-sm text-gray-300 mt-2">
                  Published by {game.publisher}
                </p>
              )}
            </div>
          </div>
        </div>

        {/* Back Button */}
        <div className="absolute top-6 left-6">
          <Button
            variant="outline"
            size="icon"
            onClick={() => navigate(-1)}
            className="bg-black/50 border-white/20 hover:bg-black/70"
          >
            <LucideChevronLeft className="w-4 h-4 text-white" />
          </Button>
        </div>
      </div>

      {/* Content Section */}
      <Container className="py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Left: Description */}
          <div className="lg:col-span-2">
            <div className="space-y-6">
              {game.description && (
                <div>
                  <h2 className="text-2xl font-semibold mb-4">About</h2>
                  <p className="text-gray-700 dark:text-gray-300 leading-relaxed">
                    {game.description}
                  </p>
                </div>
              )}

              {/* Versions Section */}
              {game.versions && game.versions.length > 0 && (
                <div>
                  <h2 className="text-2xl font-semibold mb-4">
                    Available Versions
                  </h2>
                  <div className="space-y-3">
                    {game.versions.map((version) => (
                      <div
                        key={version.id}
                        className="border rounded-lg p-4 hover:bg-gray-50 dark:hover:bg-gray-900/50 transition"
                      >
                        <div className="flex justify-between items-start">
                          <div className="flex-1">
                            <h3 className="font-semibold">
                              {version.versionName}
                            </h3>
                            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                              {(version.fileSize / 1024 / 1024).toFixed(2)} MB
                            </p>
                            {version.md5 && (
                              <p className="text-xs text-gray-500 dark:text-gray-500 mt-2 font-mono break-all">
                                MD5: {version.md5}
                              </p>
                            )}
                          </div>
                          <VersionDownloadButton
                            gameId={game.id}
                            version={version}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Right: Sidebar Cards */}
          <div className="space-y-4">
            {/* Play Button Card */}
            <MockOverlay
              className="border rounded-lg p-6 bg-white dark:bg-gray-900"
              variant="yellow"
            >
              <div className="space-y-4">
                <h3 className="font-semibold">Quick Play</h3>
                <Button className="w-full flex gap-2 items-center justify-center">
                  <LucidePlay className="w-4 h-4" /> Play Game
                </Button>
                <p className="text-xs text-gray-500">
                  Emulation and launching coming soon
                </p>
              </div>
            </MockOverlay>

            {/* Play Time Card */}
            <MockOverlay
              className="border rounded-lg p-6 bg-white dark:bg-gray-900"
              variant="yellow"
            >
              <div className="space-y-3">
                <h3 className="font-semibold">Your Stats</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-600 dark:text-gray-400">
                      Play Time:
                    </span>
                    <span className="font-medium">-- hours</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600 dark:text-gray-400">
                      Last Played:
                    </span>
                    <span className="font-medium">Never</span>
                  </div>
                </div>
                <p className="text-xs text-gray-500">
                  Playtime tracking coming soon
                </p>
              </div>
            </MockOverlay>

            {/* Save Files Card */}
            <MockOverlay
              className="border rounded-lg p-6 bg-white dark:bg-gray-900"
              variant="yellow"
            >
              <div className="space-y-3">
                <h3 className="font-semibold">Save Files</h3>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  No save files found
                </p>
                <p className="text-xs text-gray-500">
                  Save file sync coming soon
                </p>
              </div>
            </MockOverlay>

            {/* Global Stats Card */}
            <MockOverlay
              className="border rounded-lg p-6 bg-white dark:bg-gray-900"
              variant="yellow"
            >
              <div className="space-y-3">
                <h3 className="font-semibold">Community Stats</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-600 dark:text-gray-400">
                      Played by:
                    </span>
                    <span className="font-medium">-- users</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600 dark:text-gray-400">
                      Avg. Time:
                    </span>
                    <span className="font-medium">-- hours</span>
                  </div>
                </div>
                <p className="text-xs text-gray-500">
                  Community stats coming soon
                </p>
              </div>
            </MockOverlay>
          </div>
        </div>
      </Container>
    </div>
  );
}

interface VersionDownloadButtonProps {
  gameId: string;
  version: Record<string, unknown>;
}

function VersionDownloadButton({
  gameId,
  version,
}: VersionDownloadButtonProps) {
  const handleDownload = async () => {
    try {
      const { blob, filename } = await downloadGameVersion(gameId, version.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Download failed:", error);
      alert("Failed to download file. Please try again.");
    }
  };

  return (
    <Button
      size="sm"
      onClick={handleDownload}
      className="flex gap-1 items-center"
    >
      <LucideDownload className="w-4 h-4" /> Download
    </Button>
  );
}
