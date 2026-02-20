import { useState, useMemo } from "react";
import { useNavigate } from "react-router";
import { Container } from "@/components/container";
import { Header } from "@/components/header";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import type { Game, Library } from "@/lib/api/types";
import { useAsync } from "@/hooks/use-async";
import {
  LucideChevronLeft,
  LucideChevronRight,
  LucideSearch,
  LucideX,
} from "lucide-react";

interface Filter {
  type: "search" | "region" | "platform" | "library";
  value: string;
  label: string;
}

export function LibraryPage() {
  const navigate = useNavigate();
  const { data: libraries, loading: loadingLibraries } = useAsync(
    listLibraries,
    [],
  );

  const [filters, setFilters] = useState<Filter[]>([]);
  const [showSearchModal, setShowSearchModal] = useState(false);
  const [searchInput, setSearchInput] = useState("");
  const [currentPage, setCurrentPage] = useState(0);

  const pageSize = 50;

  // Create library lookup map
  const libraryMap = useMemo(() => {
    const map = new Map<string, Library>();
    libraries?.forEach((lib) => {
      map.set(lib.id, lib);
    });
    return map;
  }, [libraries]);

  // Get unique platforms from libraries
  const platforms = useMemo(() => {
    const platformSet = new Set<string>();
    libraries?.forEach((lib) => {
      if (lib.platformName) {
        platformSet.add(lib.platformName);
      }
    });
    return Array.from(platformSet).sort();
  }, [libraries]);

  // Get all games from all libraries, then filter
  const { data: allGamesData, loading: loadingGames } = useAsync(async () => {
    if (!libraries || libraries.length === 0) {
      return { items: [], total: 0, offset: 0 };
    }

    try {
      // For now, fetch from first library and combine
      // In a production app, you'd want a "list all games" API endpoint
      const allGames: Game[] = [];

      for (const lib of libraries) {
        const data = await listGamesByLibrary(lib.id, 0, 1000); // Fetch all
        if (data && "items" in data) {
          allGames.push(...data.items);
        }
      }

      return {
        items: allGames,
        total: allGames.length,
        offset: 0,
      };
    } catch (error) {
      console.error("Failed to load games:", error);
      return { items: [], total: 0, offset: 0 };
    }
  }, [libraries]) as {
    data: { items: Game[]; total: number; offset: number };
    loading: boolean;
    error: Error | null;
  };

  // Filter games based on active filters
  const filteredGames = useMemo(() => {
    if (!allGamesData?.items) return [];

    return allGamesData.items.filter((game) => {
      for (const filter of filters) {
        switch (filter.type) {
          case "search": {
            const query = filter.value.toLowerCase();
            const matchesTitle = game.title.toLowerCase().includes(query);
            const matchesVersion = game.versions?.some((v) =>
              v.versionName?.toLowerCase().includes(query),
            );
            if (!matchesTitle && !matchesVersion) return false;
            break;
          }
          case "region": {
            // Check if any version has this region
            const hasRegion = game.versions?.some((v) => {
              const regions = v.metadataJson?.regions;
              return Array.isArray(regions) && regions.includes(filter.value);
            });
            if (!hasRegion) return false;
            break;
          }
          case "platform": {
            // Check if game's library matches the selected platform
            const lib = libraryMap.get(game.libraryId);
            if (lib?.platformName !== filter.value) return false;
            break;
          }
          case "library": {
            // Check if game is in the selected library
            if (game.libraryId !== filter.value) return false;
            break;
          }
        }
      }
      return true;
    });
  }, [allGamesData, filters, libraryMap]);

  // Paginate filtered games
  const paginatedGames = useMemo(() => {
    const start = currentPage * pageSize;
    const end = start + pageSize;
    return filteredGames.slice(start, end);
  }, [filteredGames, currentPage]);

  const totalPages = Math.ceil((filteredGames.length || 0) / pageSize);

  const addSearchFilter = () => {
    if (searchInput.trim()) {
      setFilters([
        ...filters,
        {
          type: "search",
          value: searchInput,
          label: `Search: "${searchInput}"`,
        },
      ]);
      setSearchInput("");
      setShowSearchModal(false);
      setCurrentPage(0);
    }
  };

  const addFilter = (
    type: Filter["type"],
    value: string | null,
    label: string,
  ) => {
    if (value && !filters.some((f) => f.type === type && f.value === value)) {
      setFilters([...filters, { type, value, label }]);
      setCurrentPage(0);
    }
  };

  const removeFilter = (index: number) => {
    setFilters(filters.filter((_, i) => i !== index));
    setCurrentPage(0);
  };

  const regions = useMemo(() => {
    const regionSet = new Set<string>();
    allGamesData?.items?.forEach((game) => {
      game.versions?.forEach((v) => {
        const regions = v.metadataJson?.regions;
        if (Array.isArray(regions)) {
          regions.forEach((r: string) => regionSet.add(r));
        }
      });
    });
    return Array.from(regionSet).sort();
  }, [allGamesData?.items]);

  return (
    <>
      <Header
        title="All Games"
        actions={
          <Button
            size="icon"
            variant="outline"
            onClick={() => setShowSearchModal(true)}
          >
            <LucideSearch className="w-4 h-4" />
          </Button>
        }
      />

      <Container>
        <div className="space-y-4 pt-4">
          {/* Search Modal */}
          <Dialog open={showSearchModal} onOpenChange={setShowSearchModal}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Search Games</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <Input
                  placeholder="Search by title, version name, keywords..."
                  value={searchInput}
                  onChange={(e) => setSearchInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      addSearchFilter();
                    }
                  }}
                  autoFocus
                />
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    onClick={() => setShowSearchModal(false)}
                  >
                    Cancel
                  </Button>
                  <Button
                    onClick={addSearchFilter}
                    disabled={!searchInput.trim()}
                  >
                    Add Search Filter
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>

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
              {/* Filter Pills */}
              <div className="space-y-3">
                {filters.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {filters.map((filter, index) => (
                      <Badge
                        key={index}
                        variant="outline"
                        className="gap-2 pl-3 pr-1"
                      >
                        {filter.label}
                        <button
                          onClick={() => removeFilter(index)}
                          className="hover:bg-gray-200 dark:hover:bg-gray-700 rounded p-1"
                        >
                          <LucideX className="w-3 h-3" />
                        </button>
                      </Badge>
                    ))}
                  </div>
                )}

                {/* Filter Dropdowns */}
                <div className="flex flex-wrap gap-2">
                  {regions.length > 0 && (
                    <Select
                      onValueChange={(value) =>
                        addFilter(
                          "region",
                          (value as string) || "",
                          `Region: ${value}`,
                        )
                      }
                    >
                      <SelectTrigger className="w-[140px]">
                        <SelectValue placeholder="Region" />
                      </SelectTrigger>
                      <SelectContent>
                        {regions.map((region) => (
                          <SelectItem key={region} value={region}>
                            {region}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}

                  {platforms.length > 0 && (
                    <Select
                      onValueChange={(value) =>
                        addFilter(
                          "platform",
                          (value as string) || "",
                          `Platform: ${value}`,
                        )
                      }
                    >
                      <SelectTrigger className="w-[140px]">
                        <SelectValue placeholder="Platform" />
                      </SelectTrigger>
                      <SelectContent>
                        {platforms.map((platform) => (
                          <SelectItem key={platform} value={platform}>
                            {platform}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}

                  {libraries.length > 1 && (
                    <Select
                      onValueChange={(value) =>
                        addFilter(
                          "library",
                          (value as string) || "",
                          `Library: ${libraries.find((l) => l.id === value)?.name || value}`,
                        )
                      }
                    >
                      <SelectTrigger className="w-[140px]">
                        <SelectValue placeholder="Library" />
                      </SelectTrigger>
                      <SelectContent>
                        {libraries.map((lib) => (
                          <SelectItem key={lib.id} value={lib.id}>
                            {lib.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
              </div>

              {/* Games Grid */}
              {loadingGames ? (
                <div className="flex justify-center items-center py-8">
                  <Spinner />
                </div>
              ) : !paginatedGames || paginatedGames.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-gray-500">
                    {filters.length > 0
                      ? "No games match your filters"
                      : "No games found"}
                  </p>
                </div>
              ) : (
                <>
                  <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 items-start">
                    {paginatedGames.map((game) => (
                      <button
                        key={game.id}
                        onClick={() => navigate(`/game/${game.id}`)}
                        className="group cursor-pointer"
                      >
                        {/* Cover Art Placeholder */}
                        <div className="relative aspect-[2/3] bg-gradient-to-br from-gray-300 to-gray-400 dark:from-gray-700 dark:to-gray-800 rounded-lg overflow-hidden mb-2 group-hover:shadow-lg transition-shadow">
                          <div className="w-full h-full flex items-center justify-center">
                            <span className="text-gray-600 dark:text-gray-400 text-xs text-center px-2">
                              No Cover
                            </span>
                          </div>
                        </div>

                        {/* Game Info */}
                        <div className="space-y-1">
                          <h3 className="font-semibold text-sm line-clamp-2 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                            {game.title}
                          </h3>
                          {game.releasedDate && (
                            <p className="text-xs text-gray-600 dark:text-gray-400">
                              {new Date(game.releasedDate).getFullYear()}
                            </p>
                          )}
                          {game.versions && game.versions.length > 0 && (
                            <p className="text-xs text-gray-500 dark:text-gray-500">
                              {game.versions.length} version
                              {game.versions.length !== 1 ? "s" : ""}
                            </p>
                          )}
                        </div>
                      </button>
                    ))}
                  </div>

                  {/* Pagination */}
                  {totalPages > 1 && (
                    <div className="flex justify-between items-center mt-8 pt-4 border-t">
                      <div className="text-sm text-gray-600 dark:text-gray-400">
                        Showing {currentPage * pageSize + 1} to{" "}
                        {Math.min(
                          (currentPage + 1) * pageSize,
                          filteredGames.length,
                        )}{" "}
                        of {filteredGames.length} games
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
                  )}
                </>
              )}
            </>
          )}
        </div>
      </Container>
    </>
  );
}
