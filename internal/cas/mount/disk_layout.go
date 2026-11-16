package mount

import (
	"path/filepath"

	"github.com/Skoowshot/qrix-runtime/internal/cas"
)

const shardingLevelSize = 2

// DiskLayout defines how the contents are aligned inside of disk.Mount.
type DiskLayout func(hash cas.Hash) string

// FlatDiskLayout aligns the contents flat, all files in the same directory.
func FlatDiskLayout() DiskLayout {
	return func(hash cas.Hash) string {
		return hash.HexString()
	}
}

// ShardedDiskLayout shards the contents by defined levels, files split into multiple directories.
func ShardedDiskLayout(shardingLevels int) DiskLayout {
	return func(hash cas.Hash) string {
		hexHash := hash.HexString()
		if shardingLevels <= 0 {
			return hexHash
		}

		prefixLen := shardingLevels * shardingLevelSize
		if len(hexHash) < prefixLen {
			return hexHash
		}

		parts := make([]string, 0, shardingLevels+1)
		for i := 0; i < prefixLen; i += shardingLevelSize {
			parts = append(parts, hexHash[i:i+shardingLevelSize])
		}
		parts = append(parts, hexHash)

		return filepath.Join(parts...)
	}
}
