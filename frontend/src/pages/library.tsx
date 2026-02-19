import { useState } from "react";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { listLibraries } from "@/lib/api/libraries";
import { listGamesByLibrary } from "@/lib/api/games";
import type { Game } from "@/lib/api/libraries";
import { useAsync } from "@/hooks/use-async";
import { LucideChevronLeft, LucideChevronRight } from "lucide-react";
import { GameDetailsModal } from "./library/game-details-modal";

export function LibraryPage() {
  const { data: libraries, loading: loadingLibraries } = useAsync(
    listLibraries,
    [],
  );

  const [selectedLibraryId, setSelectedLibraryId] = useState<string | null>(
    null,
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [currentPage, setCurrentPage] = useState(0);
  const [selectedGame, setSelectedGame] = useState<Game | null>(null);

  const pageSize = 50;

  const selectedLibrary = libraries?.find(
    (lib) => lib.id === selectedLibraryId,
  );

  const [{ data: gamesData, loading: loadingGames }] = useAsync(
    selectedLibraryId
      ? () =>
          listGamesByLibrary(
            selectedLibraryId,
            currentPage * pageSize,
            pageSize,
            searchQuery || undefined,
          )
      : null,
    [selectedLibraryId, currentPage, searchQuery],
  );

  const handleSearch = (query: string) => {
    setSearchQuery(query);
    setCurrentPage(0);
  };

  const totalPages = gamesData ? Math.ceil(gamesData.total / pageSize) : 0;

  return (
    <Container>
      <Header title="Library" subtitle="Browse and discover games" />

      <div className="space-y-4">
        {/* Library Selection */}
        {loadingLibraries ? (
          <div className="flex items-center gap-2">
            <Spinner className="w-4 h-4" /> Loading libraries...
          </div>
        ) : !libraries || libraries.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-500 mb-4">
              No libraries available. Ask an administrator to create one.
            </p>
          </div>
        ) : (
          <>
            <div className="flex gap-2">
              <Select
                value={selectedLibraryId || ""}
                onValueChange={setSelectedLibraryId}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select a library" />
                </SelectTrigger>
                <SelectContent>
                  {libraries.map((library) => (
                    <SelectItem key={library.id} value={library.id}>
                      {library.name} ({library.gameCount || 0} games)
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {selectedLibraryId && selectedLibrary && (
              <div className="bg-gray-50 dark:bg-gray-900 p-4 rounded-lg">
                <h2 className="font-semibold mb-2">{selectedLibrary.name}</h2>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {selectedLibrary.description}
                </p>
              </div>
            )}

            {selectedLibraryId && (
              <>
                {/* Search Bar */}
                <div className="flex gap-2">
                  <Input
                    placeholder="Search games by title..."
                    value={searchQuery}
                    onChange={(e) => handleSearch(e.target.value)}
                    disabled={loadingGames}
                  />
                </div>

                {/* Games Grid */}
                {loadingGames ? (
                  <div className="flex justify-center items-center py-8">
                    <Spinner />
                  </div>
                ) : !gamesData || gamesData.items.length === 0 ? (
                  <div className="text-center py-8">
                    <p className="text-gray-500">
                      {searchQuery
                        ? `No games found matching "${searchQuery}"`
                        : "No games in this library yet"}
                    </p>
                  </div>
                ) : (
                  <>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      {gamesData.items.map((game) => (
                        <button
                          key={game.id}
                          onClick={() => setSelectedGame(game)}
                          className="border rounded-lg p-4 hover:bg-gray-100 dark:hover:bg-gray-800 transition text-left"
                        >
                          <h3 className="font-semibold truncate">
                            {game.title}
                          </h3>
                          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                            {game.versions?.length || 0} version
                            {(game.versions?.length || 0) !== 1 ? "s" : ""}
                          </p>
                          {game.developer && (
                            <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                              by {game.developer}
                            </p>
                          )}
                        </button>
                      ))}
                    </div>

                    {/* Pagination */}
                    <div className="flex justify-between items-center mt-6">
                      <div className="text-sm text-gray-600 dark:text-gray-400">
                        Showing {gamesData.offset + 1} to{" "}
                        {Math.min(gamesData.offset + pageSize, gamesData.total)}{" "}
                        of {gamesData.total} games
                      </div>
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            setCurrentPage((p) => Math.max(0, p - 1))
                          }
                          disabled={currentPage === 0 || loadingGames}
                          className="flex gap-1"
                        >
                          <LucideChevronLeft className="w-4 h-4" /> Previous
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setCurrentPage((p) => p + 1)}
                          disabled={
                            currentPage >= totalPages - 1 || loadingGames
                          }
                          className="flex gap-1"
                        >
                          Next <LucideChevronRight className="w-4 h-4" />
                        </Button>
                      </div>
                    </div>
                  </>
                )}
              </>
            )}
          </>
        )}
      </div>

      {selectedGame && (
        <GameDetailsModal
          game={selectedGame}
          onOpenChange={(open) => {
            if (!open) setSelectedGame(null);
          }}
        />
      )}
    </Container>
  );
}
