package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
)

type LOSS struct {
	id string
}

func ParseLoss(tokens []string) (string, error) {
	cmd := &LOSS{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "-id":
			if value == "" {
				return "", errors.New("el id no puede estar vacío")
			}
			cmd.id = value
		default:
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.id == "" {
		return "", errors.New("faltan parámetros requeridos: -id")
	}

	err := commandLoss(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("LOSS: Simulación de pérdida ejecutada en la partición %s", cmd.id), nil
}

func commandLoss(loss *LOSS) error {
	// Obtener la partición montada
	superblock, _, diskPath, err := stores.GetMountedPartitionSuperblock(loss.id)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %v", err)
	}

	// Verificar que sea EXT3
	if superblock.S_filesystem_type != 3 {
		return fmt.Errorf("la partición %s no soporta Journaling (no es EXT3)", loss.id)
	}

	file, err := os.OpenFile(diskPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Sobrescribir Bitmap de Inodos con ceros
	_, err = file.Seek(int64(superblock.S_bm_inode_start), 0)
	if err != nil {
		return fmt.Errorf("error al posicionarse en bitmap de inodos: %v", err)
	}
	inodeBitmapSize := superblock.S_inodes_count
	zeroBuffer := make([]byte, inodeBitmapSize)
	_, err = file.Write(zeroBuffer)
	if err != nil {
		return fmt.Errorf("error al sobrescribir bitmap de inodos: %v", err)
	}

	// Sobrescribir Bitmap de Bloques con ceros
	_, err = file.Seek(int64(superblock.S_bm_block_start), 0)
	if err != nil {
		return fmt.Errorf("error al posicionarse en bitmap de bloques: %v", err)
	}
	blockBitmapSize := superblock.S_blocks_count
	zeroBuffer = make([]byte, blockBitmapSize)
	_, err = file.Write(zeroBuffer)
	if err != nil {
		return fmt.Errorf("error al sobrescribir bitmap de bloques: %v", err)
	}

	// Sobrescribir Tabla de Inodos con ceros
	_, err = file.Seek(int64(superblock.S_inode_start), 0)
	if err != nil {
		return fmt.Errorf("error al posicionarse en tabla de inodos: %v", err)
	}
	inodeTableSize := superblock.S_inodes_count * superblock.S_inode_size
	zeroBuffer = make([]byte, inodeTableSize)
	_, err = file.Write(zeroBuffer)
	if err != nil {
		return fmt.Errorf("error al sobrescribir tabla de inodos: %v", err)
	}

	// Sobrescribir Tabla de Bloques con ceros
	_, err = file.Seek(int64(superblock.S_block_start), 0)
	if err != nil {
		return fmt.Errorf("error al posicionarse en tabla de bloques: %v", err)
	}
	blockTableSize := superblock.S_blocks_count * superblock.S_block_size
	zeroBuffer = make([]byte, blockTableSize)
	_, err = file.Write(zeroBuffer)
	if err != nil {
		return fmt.Errorf("error al sobrescribir tabla de bloques: %v", err)
	}

	return nil
}
