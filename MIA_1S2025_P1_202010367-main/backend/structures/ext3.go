package structures

import (
	"encoding/binary"
	"math"
	"os"
	"time"
)

// CalculateStructures calcula el número de inodos, bloques y entradas del Journal.
func CalculateStructures(partitionSize int32) (inodes, blocks, journalEntries int32) {
	superblockSize := int32(binary.Size(SuperBlock{})) // 76 bytes
	journalSize := int32(binary.Size(Journal{}))       // 114 bytes
	inodeSize := int32(binary.Size(Inode{}))           // 88 bytes
	blockSize := int32(binary.Size(FolderBlock{}))     // 64 bytes
	journalEntries = 50                                // Constante según el enunciado
	n := float64(partitionSize-superblockSize-journalEntries*journalSize) / float64(4+inodeSize+3*blockSize)
	inodes = int32(math.Floor(n))
	blocks = 3 * inodes
	if inodes < 2 {
		inodes = 2
		blocks = 6
	}
	return inodes, blocks, journalEntries
}

// FormatEXT3 formatea una partición con el sistema de archivos EXT3.
func FormatEXT3(path string, start, size int32) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	inodes, blocks, journalEntries := CalculateStructures(size)
	sb := SuperBlock{
		S_filesystem_type:   3,
		S_inodes_count:      inodes,
		S_blocks_count:      blocks,
		S_free_inodes_count: inodes - 2, // Raíz y users.txt
		S_free_blocks_count: blocks - 2, // Raíz y users.txt
		S_mtime:             float32(time.Now().Unix()),
		S_umtime:            float32(time.Now().Unix()),
		S_mnt_count:         1,
		S_magic:             0xEF53,
		S_inode_size:        int32(binary.Size(Inode{})),
		S_block_size:        64,
		S_first_ino:         2,
		S_first_blo:         2,
		S_journal_count:     journalEntries,
	}
	sb.S_journal_start = start + int32(binary.Size(sb))
	sb.S_bm_inode_start = sb.S_journal_start + journalEntries*int32(binary.Size(Journal{}))
	sb.S_bm_block_start = sb.S_bm_inode_start + inodes
	sb.S_inode_start = sb.S_bm_block_start + blocks
	sb.S_block_start = sb.S_inode_start + inodes*sb.S_inode_size

	if err := sb.Serialize(path, int64(start)); err != nil {
		return err
	}

	for i := int32(0); i < journalEntries; i++ {
		journal := Journal{Count: i}
		if err := journal.Serialize(path, int64(sb.S_journal_start)+int64(i)*int64(binary.Size(Journal{}))); err != nil {
			return err
		}
	}

	if err := sb.CreateBitMaps(file); err != nil {
		return err
	}
	if err := sb.CreateUsersFile(path); err != nil {
		return err
	}

	if err := sb.Serialize(path, int64(start)); err != nil {
		return err
	}

	return nil
}
