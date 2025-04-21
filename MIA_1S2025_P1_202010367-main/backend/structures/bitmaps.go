package structures

import (
	"fmt"
	"os"
)

// CreateBitMaps crea los Bitmaps de inodos y bloques en el archivo especificado
func (sb *SuperBlock) CreateBitMaps(file *os.File) error {
	// Bitmap de inodos
	_, err := file.Seek(int64(sb.S_bm_inode_start), 0)
	if err != nil {
		return err
	}

	totalInodes := sb.S_inodes_count
	buffer := make([]byte, totalInodes)
	for i := range buffer {
		buffer[i] = '0'
	}
	// Marcar algunos inodos como ocupados (ejemplo: root y users.txt)
	if totalInodes >= 2 {
		buffer[0] = '1' // Inodo 0 ocupado (root)
		buffer[1] = '1' // Inodo 1 ocupado (users.txt)
	}

	_, err = file.Write(buffer)
	if err != nil {
		return err
	}

	// Bitmap de bloques
	_, err = file.Seek(int64(sb.S_bm_block_start), 0)
	if err != nil {
		return err
	}

	totalBlocks := sb.S_blocks_count
	buffer = make([]byte, totalBlocks)
	for i := range buffer {
		buffer[i] = '0'
	}
	if totalBlocks >= 2 {
		buffer[0] = '1'
		buffer[1] = '1'
	}

	_, err = file.Write(buffer)
	if err != nil {
		return err
	}

	return nil
}

// UpdateBitmapInode actualiza un inodo específico en el bitmap
func (sb *SuperBlock) UpdateBitmapInode(path string, inodeIndex int32) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if inodeIndex >= sb.S_inodes_count {
		return fmt.Errorf("índice de inodo fuera de rango: %d", inodeIndex)
	}

	_, err = file.Seek(int64(sb.S_bm_inode_start)+int64(inodeIndex), 0)
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{'1'})
	if err != nil {
		return err
	}

	return nil
}

// UpdateBitmapBlock (similar ajuste)
func (sb *SuperBlock) UpdateBitmapBlock(path string, blockIndex int32) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if blockIndex >= sb.S_blocks_count {
		return fmt.Errorf("índice de bloque fuera de rango: %d", blockIndex)
	}

	_, err = file.Seek(int64(sb.S_bm_block_start)+int64(blockIndex), 0)
	if err != nil {
		return err
	}

	_, err = file.Write([]byte{'1'})
	if err != nil {
		return err
	}

	return nil
}
