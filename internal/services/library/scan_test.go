package library

import "testing"

// TestScan_NoMetadata tests scanning when metadata.toml is completely missing.
// The scanner should generate basic metadata from the filename.
//
// Database state:
//   - Empty (no games in database)
//
// Library directory structure:
//
//	library/
//	  platform-name/
//	    game-rom-file.zip
func TestScan_NoMetadata(t *testing.T) {
	t.Fatal("Not implemented")
}

// TestScan_PartialMetadata tests scanning when metadata.toml exists but is incomplete.
// The scanner should use existing metadata fields and discover ROM files in the directory.
//
// Database state:
//   - Empty (no games in database)
//
// Library directory structure:
//
//	library/
//	  platform-name/
//	    metadata.toml  (contains title, developer, publisher, but no versions section)
//	    game-rom-v1.zip
//	    game-rom-v2.zip
func TestScan_PartialMetadata(t *testing.T) {
	t.Fatal("Not implemented")
}

// TestScan_FullGenericMetadata tests scanning when complete metadata exists without UUID.
// The scanner should create a new game instance with all metadata intact.
//
// Database state:
//   - Empty (no games in database)
//
// Library directory structure:
//
//	library/
//	  platform-name/
//	    metadata.toml  (complete: title, developer, publisher, versions, but no uuid field)
//	    game-rom-v1.zip
//	    game-rom-v2.zip
func TestScan_FullGenericMetadata(t *testing.T) {
	t.Fatal("Not implemented")
}

// TestScan_InstanceMetadata tests scanning when metadata.toml has an instance UUID.
// The scanner should match the existing game in the database and update it.
//
// Database state:
//   - Game exists with UUID "123e4567-e89b-12d3-a456-426614174000"
//   - Linked to path "library/platform-name"
//
// Library directory structure:
//
//	library/
//	  platform-name/
//	    metadata.toml  (includes uuid = "123e4567-e89b-12d3-a456-426614174000")
//	    game-rom-v1.zip
//	    game-rom-v2.zip
func TestScan_InstanceMetadata(t *testing.T) {
	t.Fatal("Not implemented")
}

// TestScan_UUIDCollision tests that scanner rejects simultaneous UUID conflicts.
// If a UUID exists in the DB for an active game and another game tries to use it,
// scanning should fail/warn. Multiple games cannot have the same UUID at the same time.
//
// Database state:
//   - Game A exists with UUID "123e4567-e89b-12d3-a456-426614174000"
//   - Linked to path "library/game-a"
//   - Status: active
//
// Library directory structure:
//
//	library/
//	  game-a/
//	    metadata.toml  (includes uuid = "123e4567-e89b-12d3-a456-426614174000")
//	    game-a.zip
//	  game-b/
//	    metadata.toml  (includes uuid = "123e4567-e89b-12d3-a456-426614174000")  // CONFLICT!
//	    game-b.zip
func TestScan_UUIDCollision(t *testing.T) {
	t.Fatal("Not implemented")
}

// TestScan_UUIDReuse tests that scanner allows UUID reuse after original game is deleted.
// If a game is deleted from the database, its UUID can be reassigned to a new game.
// This is valid because only one game has the UUID at any given time.
//
// Database state:
//   - Game was previously deleted (UUID "123e4567-e89b-12d3-a456-426614174000" no longer in DB)
//   - Or game exists with status: deleted/removed
//
// Library directory structure:
//
//	library/
//	  new-game/
//	    metadata.toml  (includes uuid = "123e4567-e89b-12d3-a456-426614174000")
//	    game-rom.zip
func TestScan_UUIDReuse(t *testing.T) {
	t.Fatal("Not implemented")
}

// TestScan_RemovedMetadata tests that scanner regenerates metadata when metadata.toml is deleted.
// If only the metadata file is removed but game files remain, the scanner should re-trigger
// metadata search and fall back to generic data generation (treating it like a new game).
// This may result in a new UUID being assigned.
//
// Database state:
//   - Game exists with UUID "123e4567-e89b-12d3-a456-426614174000"
//   - Linked to path "library/platform-name"
//
// Library directory structure:
//
//	library/
//	  platform-name/
//	    game-rom.zip         (still exists)
//	    (metadata.toml deleted)
func TestScan_RemovedMetadata(t *testing.T) {
	t.Fatal("Not implemented")
}

// TestScan_RemovedGame tests that scanner deletes game from database when entire directory is removed.
// If the game directory is completely deleted, the game should be removed from the database.
//
// Database state:
//   - Game exists with UUID "123e4567-e89b-12d3-a456-426614174000"
//   - Linked to path "library/platform-name"
//
// Library directory structure:
//
//	library/
//	  (platform-name/ directory completely removed)
func TestScan_RemovedGame(t *testing.T) {
	t.Fatal("Not implemented")
}

// TestScan_MetadataWriteback tests that scanner writes complete instance metadata back to disk.
// After scanning, metadata.toml should be updated with the assigned UUID.
//
// Database state:
//   - Empty (no games in database)
//
// Library directory structure (before scan):
//
//	library/
//	  platform-name/
//	    metadata.toml  (no uuid field)
//	    game-rom.zip
//
// Expected directory structure (after scan):
//
//	library/
//	  platform-name/
//	    metadata.toml  (now includes generated uuid field)
//	    game-rom.zip
func TestScan_MetadataWriteback(t *testing.T) {
	t.Fatal("Not implemented")
}
