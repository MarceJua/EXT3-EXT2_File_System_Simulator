package commands

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"         // Paquete que contiene las estructuras de datos necesarias para el manejo de discos y particiones
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures" // Paquete que contiene las estructuras de datos necesarias para el manejo de discos y particiones
)

// MKFS estructura que representa el comando mkfs con sus parámetros
type MKFS struct {
	id  string // ID del disco
	typ string // Tipo de formato (full)
	fs  string // Tipo de sistema de archivos (2fs o 3fs)
}

/*
   mkfs -id=vd1 -type=full
   mkfs -id=vd2
*/

func ParseMkfs(tokens []string) (string, error) {
	cmd := &MKFS{typ: "full", fs: "2fs"}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "-id":
			if value == "" {
				return "", errors.New("el id no puede estar vacío")
			}
			cmd.id = value
		case "-type":
			if value != "full" {
				return "", errors.New("el tipo debe ser full")
			}
			cmd.typ = value
		case "-fs":
			value = strings.ToLower(value)
			if value != "2fs" && value != "3fs" {
				return "", errors.New("el fs debe ser 2fs o 3fs")
			}
			cmd.fs = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.id == "" {
		return "", errors.New("faltan parámetros requeridos: -id")
	}

	err := commandMkfs(cmd)
	if err != nil {
		return "", fmt.Errorf("error al formatear la partición: %v", err)
	}

	return fmt.Sprintf("MKFS: Partición %s formateada con éxito con sistema %s", cmd.id, cmd.fs), nil
}

func commandMkfs(mkfs *MKFS) error {
	partitionPath, exists := stores.MountedPartitions[mkfs.id]
	if !exists {
		return errors.New("partición no montada")
	}

	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	var mbr structures.MBR
	if err := mbr.Deserialize(partitionPath); err != nil {
		return fmt.Errorf("error al deserializar MBR: %v", err)
	}

	var partition *structures.Partition
	var startOffset int64
	var partitionSize int32
	for _, p := range mbr.Mbr_partitions {
		partID := strings.Trim(string(p.Part_id[:]), "\x00")
		if partID == mkfs.id {
			partition = &p
			startOffset = int64(p.Part_start)
			partitionSize = p.Part_size
			break
		}
	}

	if partition == nil {
		var extPartition *structures.Partition
		for _, p := range mbr.Mbr_partitions {
			if p.Part_type[0] == 'E' && p.Part_status[0] != 'N' {
				extPartition = &p
				break
			}
		}
		if extPartition == nil {
			return fmt.Errorf("partición %s no encontrada en el disco", mkfs.id)
		}

		var currentEBR structures.EBR
		currentOffset := int64(extPartition.Part_start)
		for {
			if err := currentEBR.Deserialize(file, currentOffset); err != nil {
				return fmt.Errorf("error al leer EBR: %v", err)
			}
			partID := strings.Trim(string(currentEBR.Part_id[:]), "\x00")
			if partID == mkfs.id {
				startOffset = int64(currentEBR.Part_start)
				partitionSize = currentEBR.Part_size
				break
			}
			if currentEBR.Part_next == -1 {
				return fmt.Errorf("partición lógica %s no encontrada", mkfs.id)
			}
			currentOffset = int64(currentEBR.Part_next)
		}
	}

	var sbCheck structures.SuperBlock
	if err := sbCheck.Deserialize(partitionPath, startOffset); err == nil && sbCheck.S_magic == 0xEF53 {
		return errors.New("la partición ya está formateada")
	}

	n := calculateN(partitionSize)
	fmt.Printf("DEBUG: partitionSize=%d, n=%d\n", partitionSize, n)
	superBlock := createSuperBlock(startOffset, n, mkfs.fs)

	// Crear bitmaps y users.txt
	if err := superBlock.CreateBitMaps(file); err != nil {
		return err
	}
	if err := superBlock.CreateUsersFile(partitionPath); err != nil {
		return err
	}
	if err := superBlock.Serialize(partitionPath, startOffset); err != nil {
		return err
	}

	return nil
}
func calculateN(size int32) int32 {
	numerator := int(size) - binary.Size(structures.SuperBlock{})
	denominator := 4 + binary.Size(structures.Inode{}) + 3*binary.Size(structures.FileBlock{})
	return int32(math.Floor(float64(numerator) / float64(denominator)))
}

func createSuperBlock(startOffset int64, n int32, fs string) *structures.SuperBlock {
	bm_inode_start := int32(startOffset) + int32(binary.Size(structures.SuperBlock{}))
	bm_block_start := bm_inode_start + n
	inode_start := bm_block_start + (3 * n)
	block_start := inode_start + (int32(binary.Size(structures.Inode{})) * n)

	fsType := int32(2)
	if fs == "3fs" {
		fsType = 3
	}

	totalInodes := n
	totalBlocks := 3 * n
	freeInodes := n - 2 // Raíz y users.txt
	freeBlocks := 3*n - 2

	if n < 2 {
		totalInodes = 2
		totalBlocks = 6
		freeInodes = 0
		freeBlocks = 4
	}

	return &structures.SuperBlock{
		S_filesystem_type:   fsType,
		S_inodes_count:      totalInodes,
		S_blocks_count:      totalBlocks,
		S_free_inodes_count: freeInodes,
		S_free_blocks_count: freeBlocks,
		S_mtime:             float32(time.Now().Unix()),
		S_umtime:            float32(time.Now().Unix()),
		S_mnt_count:         1,
		S_magic:             0xEF53,
		S_inode_size:        int32(binary.Size(structures.Inode{})),
		S_block_size:        int32(binary.Size(structures.FileBlock{})),
		S_first_ino:         2, // Próximo inodo libre
		S_first_blo:         2, // Próximo bloque libre
		S_bm_inode_start:    bm_inode_start,
		S_bm_block_start:    bm_block_start,
		S_inode_start:       inode_start,
		S_block_start:       block_start,
	}
}
