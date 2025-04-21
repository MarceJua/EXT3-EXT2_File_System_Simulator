package stores

import (
	"errors"
	"fmt"
	"os"
	"strings"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

// Carnet de estudiante
const Carnet string = "67" // 202010367

// Session representa una sesión activa
type Session struct {
	ID       string // ID de la partición
	Username string // Nombre del usuario
	UID      string // ID del usuario
	GID      string // ID del grupo
}

// CurrentSession almacena la sesión actual
var CurrentSession Session

// Declaración de variables globales
var MountedPartitions = make(map[string]string)

func GetMountedPartitionRep(id string) (*structures.MBR, *structures.SuperBlock, string, error) {
	path, exists := MountedPartitions[id]
	if !exists {
		return nil, nil, "", errors.New("partición no montada")
	}

	var mbr structures.MBR
	if err := mbr.Deserialize(path); err != nil {
		return nil, nil, "", fmt.Errorf("error deserializando MBR: %v", err)
	}

	// Buscar partición primaria
	for _, p := range mbr.Mbr_partitions {
		if strings.Trim(string(p.Part_id[:]), "\x00") == id {
			var sb structures.SuperBlock
			err := sb.Deserialize(path, int64(p.Part_start))
			if err != nil {
				return &mbr, nil, path, nil // Devolver sin superbloque si falla (para mbr/disk)
			}
			return &mbr, &sb, path, nil
		}
	}

	// Buscar en particiones lógicas
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, nil, "", fmt.Errorf("error abriendo disco: %v", err)
	}
	defer file.Close()

	var extPartition *structures.Partition
	for _, p := range mbr.Mbr_partitions {
		if p.Part_type[0] == 'E' && p.Part_status[0] != 'N' {
			extPartition = &p
			break
		}
	}
	if extPartition == nil {
		return &mbr, nil, path, nil // Sin extendida, devolvemos solo MBR
	}

	var currentEBR structures.EBR
	currentOffset := int64(extPartition.Part_start)
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, nil, "", fmt.Errorf("error obteniendo tamaño del archivo: %v", err)
	}
	fileSize := fileInfo.Size()

	for currentOffset < fileSize {
		if err := currentEBR.Deserialize(file, currentOffset); err != nil {
			return nil, nil, "", fmt.Errorf("error leyendo EBR en offset %d: %v", currentOffset, err)
		}
		if strings.Trim(string(currentEBR.Part_id[:]), "\x00") == id {
			var sb structures.SuperBlock
			err := sb.Deserialize(path, int64(currentEBR.Part_start))
			if err != nil {
				return &mbr, nil, path, nil // Sin superbloque si falla
			}
			return &mbr, &sb, path, nil
		}
		if currentEBR.Part_next == -1 {
			break
		}
		currentOffset = int64(currentEBR.Part_next)
	}

	return &mbr, nil, path, nil // Si no se encuentra, devolvemos solo MBR
}

// GetMountedPartitionSuperblock obtiene el SuperBlock de la partición montada con el id especificado
func GetMountedPartitionSuperblock(id string) (*structures.SuperBlock, *structures.Partition, string, error) {
	path := MountedPartitions[id]
	if path == "" {
		return nil, nil, "", errors.New("la partición no está montada")
	}
	var mbr structures.MBR
	err := mbr.Deserialize(path)
	if err != nil {
		return nil, nil, "", err
	}
	partition, err := mbr.GetPartitionByID(id)
	if partition == nil {
		return nil, nil, "", err
	}
	var sb structures.SuperBlock
	err = sb.Deserialize(path, int64(partition.Part_start))
	if err != nil {
		return nil, nil, "", err
	}
	return &sb, partition, path, nil
}
