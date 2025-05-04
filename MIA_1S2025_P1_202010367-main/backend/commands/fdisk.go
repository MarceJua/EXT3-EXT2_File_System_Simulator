package commands

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	utils "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/utils"
)

// FDISK estructura que representa el comando fdisk con sus parámetros
type FDISK struct {
	size   int    // Tamaño de la partición
	unit   string // Unidad de medida del tamaño (B, K o M)
	fit    string // Tipo de ajuste (BF, FF, WF)
	path   string // Ruta del archivo del disco
	typ    string // Tipo de partición (P, E, L)
	name   string // Nombre de la partición
	delete string // Modo de eliminación (fast, full)
	add    int    // Tamaño a añadir o reducir
}

// ParseFdisk parsea el comando fdisk y devuelve una instancia de FDISK
func ParseFdisk(tokens []string) (string, error) {
	cmd := &FDISK{}

	// Procesar cada token
	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "-size":
			size, err := strconv.Atoi(value)
			if err != nil || size <= 0 {
				return "", errors.New("el tamaño debe ser un número entero positivo")
			}
			cmd.size = size
		case "-unit":
			value = strings.ToUpper(value)
			if value != "B" && value != "K" && value != "M" {
				return "", errors.New("la unidad debe ser B, K o M")
			}
			cmd.unit = value
		case "-fit":
			value = strings.ToUpper(value)
			if value != "BF" && value != "FF" && value != "WF" {
				return "", errors.New("el ajuste debe ser BF, FF o WF")
			}
			cmd.fit = value
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-type":
			value = strings.ToUpper(value)
			if value != "P" && value != "E" && value != "L" {
				return "", errors.New("el tipo debe ser P, E o L")
			}
			cmd.typ = value
		case "-name":
			if value == "" {
				return "", errors.New("el nombre no puede estar vacío")
			}
			cmd.name = value
		case "-delete":
			value = strings.ToLower(value)
			if value != "fast" && value != "full" {
				return "", errors.New("el modo de eliminación debe ser fast o full")
			}
			cmd.delete = value
		case "-add":
			add, err := strconv.Atoi(value)
			if err != nil {
				return "", errors.New("el valor de add debe ser un entero")
			}
			cmd.add = add
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Validar parámetros requeridos
	if cmd.path == "" || cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -path, -name")
	}

	// Validar combinaciones de parámetros
	if cmd.delete != "" {
		if cmd.add != 0 || cmd.typ != "" || cmd.fit != "" {
			return "", errors.New("el parámetro -delete no puede combinarse con -add, -type o -fit")
		}
		if cmd.size > 0 {
			// Ignorar -size si está presente con -delete
			cmd.size = 0
		}
	} else if cmd.add != 0 {
		if cmd.delete != "" {
			return "", errors.New("el parámetro -add no puede combinarse con -delete")
		}
		if cmd.size > 0 {
			// Ignorar -size si está presente con -add
			cmd.size = 0
		}
	} else {
		if cmd.size == 0 {
			return "", errors.New("faltan parámetros requeridos: -size")
		}
	}

	// Establecer valores por defecto
	if cmd.unit == "" {
		cmd.unit = "K"
	}
	if cmd.fit == "" && cmd.delete == "" && cmd.add == 0 {
		cmd.fit = "WF"
	}
	if cmd.typ == "" && cmd.delete == "" && cmd.add == 0 {
		cmd.typ = "P"
	}

	// Ejecutar el comando
	err := commandFdisk(cmd)
	if err != nil {
		return "", fmt.Errorf("error al ejecutar FDISK: %v", err)
	}

	if cmd.delete != "" {
		return fmt.Sprintf("FDISK: Partición %s eliminada correctamente en %s", cmd.name, cmd.path), nil
	} else if cmd.add != 0 {
		return fmt.Sprintf("FDISK: Tamaño de partición %s ajustado correctamente en %s", cmd.name, cmd.path), nil
	}
	return fmt.Sprintf("FDISK: Partición %s creada correctamente en %s", cmd.name, cmd.path), nil
}

// commandFdisk implementa la lógica para crear, añadir o eliminar particiones
func commandFdisk(fdisk *FDISK) error {
	// Crear directorios padre si no existen
	if err := utils.CreateParentDirs(fdisk.path); err != nil {
		return err
	}

	// Verificar si la partición está montada
	for id, path := range stores.MountedPartitions {
		if path == fdisk.path {
			var mbr structures.MBR
			if err := mbr.Deserialize(fdisk.path); err != nil {
				return fmt.Errorf("error al deserializar MBR: %v", err)
			}
			partition, _ := mbr.GetPartitionByNameFromID(id)
			if partition != nil && strings.Trim(string(partition.Part_name[:]), "\x00") == fdisk.name {
				return errors.New("no se puede modificar una partición montada")
			}
		}
	}

	// Abrir disco
	file, err := os.OpenFile(fdisk.path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Leer MBR
	var mbr structures.MBR
	if err := mbr.Deserialize(fdisk.path); err != nil {
		return fmt.Errorf("error al deserializar MBR: %v", err)
	}

	// Manejar -delete
	if fdisk.delete != "" {
		return deletePartition(&mbr, fdisk, file)
	}

	// Manejar -add
	if fdisk.add != 0 {
		return addPartitionSize(&mbr, fdisk, file)
	}

	// Crear nueva partición
	// Convertir el tamaño a bytes
	sizeBytes, err := utils.ConvertToBytes(fdisk.size, fdisk.unit)
	if err != nil {
		return fmt.Errorf("error al convertir tamaño: %v", err)
	}

	// Validar nombre duplicado en primarias/extendidas
	if _, idx := mbr.GetPartitionByName(fdisk.name); idx != -1 {
		return fmt.Errorf("el nombre '%s' ya existe en particiones primarias/extendidas", fdisk.name)
	}

	switch fdisk.typ {
	case "P":
		return createPrimaryPartition(fdisk, sizeBytes)
	case "E":
		return createExtendedPartition(fdisk, sizeBytes)
	case "L":
		return createLogicalPartition(fdisk, sizeBytes)
	default:
		return errors.New("tipo de partición inválido")
	}
}

// deletePartition elimina una partición primaria o lógica
func deletePartition(mbr *structures.MBR, fdisk *FDISK, file *os.File) error {
	// Buscar partición primaria
	partition, _ := mbr.GetPartitionByName(fdisk.name)
	if partition != nil {
		if fdisk.delete == "full" {
			// Sobrescribir con ceros
			if _, err := file.Seek(int64(partition.Part_start), 0); err != nil {
				return err
			}
			zeros := make([]byte, partition.Part_size)
			if _, err := file.Write(zeros); err != nil {
				return err
			}
		}
		// Marcar como no usada
		partition.Part_status = [1]byte{'N'}
		partition.Part_size = 0
		partition.Part_start = -1
		partition.Part_name = [16]byte{}
		partition.Part_id = [4]byte{}
		partition.Part_type = [1]byte{}
		partition.Part_fit = [1]byte{}
		partition.Part_correlative = 0
		return mbr.Serialize(fdisk.path)
	}

	// Buscar partición lógica
	var extPartition *structures.Partition
	for _, p := range mbr.Mbr_partitions {
		if p.Part_type[0] == 'E' && p.Part_status[0] != 'N' {
			extPartition = &p
			break
		}
	}
	if extPartition == nil {
		return fmt.Errorf("partición %s no encontrada", fdisk.name)
	}

	var currentEBR structures.EBR
	currentOffset := int64(extPartition.Part_start)
	var prevOffset int64 = -1
	for {
		if err := currentEBR.Deserialize(file, currentOffset); err != nil {
			return fmt.Errorf("error al leer EBR: %v", err)
		}
		if strings.Trim(string(currentEBR.Part_name[:]), "\x00") == fdisk.name {
			if fdisk.delete == "full" {
				// Sobrescribir con ceros
				if _, err := file.Seek(int64(currentEBR.Part_start), 0); err != nil {
					return err
				}
				zeros := make([]byte, currentEBR.Part_size)
				if _, err := file.Write(zeros); err != nil {
					return err
				}
			}
			// Actualizar el enlace del EBR anterior
			if prevOffset != -1 {
				var prevEBR structures.EBR
				if err := prevEBR.Deserialize(file, prevOffset); err != nil {
					return fmt.Errorf("error al leer EBR anterior: %v", err)
				}
				prevEBR.Part_next = currentEBR.Part_next
				if err := prevEBR.Serialize(file, prevOffset); err != nil {
					return fmt.Errorf("error al serializar EBR anterior: %v", err)
				}
			} else {
				// Si es el primer EBR, inicializar un nuevo EBR vacío o copiar el siguiente
				if currentEBR.Part_next != -1 {
					var nextEBR structures.EBR
					if err := nextEBR.Deserialize(file, int64(currentEBR.Part_next)); err != nil {
						return fmt.Errorf("error al leer EBR siguiente: %v", err)
					}
					if err := nextEBR.Serialize(file, currentOffset); err != nil {
						return fmt.Errorf("error al serializar nuevo EBR inicial: %v", err)
					}
				} else {
					// No hay más EBRs, inicializar uno vacío
					emptyEBR := structures.EBR{
						Part_status: [1]byte{'N'},
						Part_start:  -1,
						Part_size:   0,
						Part_next:   -1,
					}
					if err := emptyEBR.Serialize(file, currentOffset); err != nil {
						return fmt.Errorf("error al serializar EBR vacío: %v", err)
					}
				}
			}
			return nil
		}
		if currentEBR.Part_next == -1 {
			break
		}
		prevOffset = currentOffset
		currentOffset = int64(currentEBR.Part_next)
	}
	return fmt.Errorf("partición lógica %s no encontrada", fdisk.name)
}

// addPartitionSize ajusta el tamaño de una partición primaria o lógica
func addPartitionSize(mbr *structures.MBR, fdisk *FDISK, file *os.File) error {
	// Convertir add a bytes
	addBytes, err := utils.ConvertToBytes(fdisk.add, fdisk.unit)
	if err != nil {
		return err
	}
	if addBytes == 0 {
		return errors.New("el valor de add no puede ser cero")
	}

	// Buscar partición primaria
	partition, _ := mbr.GetPartitionByName(fdisk.name)
	if partition != nil {
		newSize := int(partition.Part_size) + addBytes
		if newSize <= 0 {
			return errors.New("el nuevo tamaño de la partición no puede ser menor o igual a cero")
		}
		// Verificar espacio disponible
		fileInfo, err := file.Stat()
		if err != nil {
			return err
		}
		diskSize := fileInfo.Size()
		endPosition := int64(partition.Part_start) + int64(newSize)
		if endPosition > diskSize {
			return errors.New("no hay suficiente espacio en el disco")
		}
		// Verificar colisión con otras particiones
		for _, p := range mbr.Mbr_partitions {
			if p.Part_status[0] == 'N' || p.Part_start == -1 {
				continue
			}
			if p.Part_start > partition.Part_start && p.Part_start < int32(endPosition) {
				return errors.New("el nuevo tamaño colisiona con otra partición")
			}
		}
		partition.Part_size = int32(newSize)
		return mbr.Serialize(fdisk.path)
	}

	// Buscar partición lógica
	var extPartition *structures.Partition
	for _, p := range mbr.Mbr_partitions {
		if p.Part_type[0] == 'E' && p.Part_status[0] != 'N' {
			extPartition = &p
			break
		}
	}
	if extPartition == nil {
		return fmt.Errorf("partición %s no encontrada", fdisk.name)
	}

	var currentEBR structures.EBR
	currentOffset := int64(extPartition.Part_start)
	for {
		if err := currentEBR.Deserialize(file, currentOffset); err != nil {
			return fmt.Errorf("error al leer EBR: %v", err)
		}
		if strings.Trim(string(currentEBR.Part_name[:]), "\x00") == fdisk.name {
			newSize := int(currentEBR.Part_size) + addBytes
			if newSize <= 0 {
				return errors.New("el nuevo tamaño de la partición no puede ser menor o igual a cero")
			}
			// Verificar espacio en partición extendida
			endPosition := int64(currentEBR.Part_start) + int64(newSize)
			if endPosition > int64(extPartition.Part_start+extPartition.Part_size) {
				return errors.New("no hay suficiente espacio en la partición extendida")
			}
			// Verificar colisión con EBR siguiente
			if currentEBR.Part_next != -1 {
				var nextEBR structures.EBR
				if err := nextEBR.Deserialize(file, int64(currentEBR.Part_next)); err != nil {
					return err
				}
				if int64(nextEBR.Part_start) < endPosition {
					return errors.New("el nuevo tamaño colisiona con la siguiente partición lógica")
				}
			}
			currentEBR.Part_size = int32(newSize)
			return currentEBR.Serialize(file, currentOffset)
		}
		if currentEBR.Part_next == -1 {
			break
		}
		currentOffset = int64(currentEBR.Part_next)
	}
	return fmt.Errorf("partición lógica %s no encontrada", fdisk.name)
}

// createPrimaryPartition crea una partición primaria
func createPrimaryPartition(fdisk *FDISK, sizeBytes int) error {
	var mbr structures.MBR
	if err := mbr.Deserialize(fdisk.path); err != nil {
		return fmt.Errorf("error deserializando el MBR: %v", err)
	}

	// Contar particiones primarias/extendidas activas
	count := 0
	for _, p := range mbr.Mbr_partitions {
		if p.Part_status[0] != 'N' {
			count++
		}
	}
	if count >= 4 {
		return errors.New("máximo de 4 particiones primarias/extendidas alcanzado")
	}

	partition, start, idx := mbr.GetFirstAvailablePartition()
	if partition == nil {
		return errors.New("no hay particiones disponibles en el MBR")
	}

	// Verificar espacio disponible desde el inicio hasta el final del disco
	availableSpace := int(mbr.Mbr_size) - start
	if sizeBytes > availableSpace {
		return errors.New("no hay espacio suficiente en el disco")
	}

	partition.CreatePartition(start, sizeBytes, fdisk.typ, fdisk.fit, fdisk.name)
	mbr.Mbr_partitions[idx] = *partition
	if err := mbr.Serialize(fdisk.path); err != nil {
		return fmt.Errorf("error serializando el MBR: %v", err)
	}

	return nil
}

// createExtendedPartition crea una partición extendida
func createExtendedPartition(fdisk *FDISK, sizeBytes int) error {
	var mbr structures.MBR
	if err := mbr.Deserialize(fdisk.path); err != nil {
		return fmt.Errorf("error deserializando el MBR: %v", err)
	}

	// Validar que no exista otra extendida
	for _, p := range mbr.Mbr_partitions {
		if p.Part_type[0] == 'E' && p.Part_status[0] != 'N' {
			return errors.New("ya existe una partición extendida en el disco")
		}
	}

	// Contar particiones primarias/extendidas activas
	count := 0
	for _, p := range mbr.Mbr_partitions {
		if p.Part_status[0] != 'N' {
			count++
		}
	}
	if count >= 4 {
		return errors.New("máximo de 4 particiones primarias/extendidas alcanzado")
	}

	partition, start, idx := mbr.GetFirstAvailablePartition()
	if partition == nil {
		return errors.New("no hay particiones disponibles en el MBR")
	}

	availableSpace := int(mbr.Mbr_size) - start
	if sizeBytes > availableSpace {
		return errors.New("no hay espacio suficiente en el disco")
	}

	partition.CreatePartition(start, sizeBytes, "E", fdisk.fit, fdisk.name)
	mbr.Mbr_partitions[idx] = *partition
	if err := mbr.Serialize(fdisk.path); err != nil {
		return fmt.Errorf("error serializando el MBR: %v", err)
	}

	return nil
}

// createLogicalPartition crea una partición lógica dentro de una extendida
func createLogicalPartition(fdisk *FDISK, sizeBytes int) error {
	var mbr structures.MBR
	if err := mbr.Deserialize(fdisk.path); err != nil {
		return fmt.Errorf("error al deserializar MBR: %v", err)
	}

	// Buscar partición extendida
	var extPartition *structures.Partition
	for _, p := range mbr.Mbr_partitions {
		if p.Part_type[0] == 'E' && p.Part_status[0] != 'N' {
			extPartition = &p
			break
		}
	}
	if extPartition == nil {
		return errors.New("no hay partición extendida para crear lógicas")
	}

	file, err := os.OpenFile(fdisk.path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	startExt := int64(extPartition.Part_start)
	availableSpace := int(extPartition.Part_size)

	var currentEBR structures.EBR
	err = currentEBR.Deserialize(file, startExt)
	if err != nil || currentEBR.Part_status[0] == 0 || currentEBR.Part_status[0] == 'N' {
		// Primer EBR
		ebrSize := int(binary.Size(structures.EBR{}))
		if sizeBytes+ebrSize > availableSpace {
			return errors.New("no hay espacio suficiente en la partición extendida")
		}
		currentEBR = structures.EBR{
			Part_status: [1]byte{'0'},
			Part_fit:    [1]byte{fdisk.fit[0]},
			Part_start:  extPartition.Part_start,
			Part_size:   int32(sizeBytes),
			Part_next:   -1,
		}
		copy(currentEBR.Part_name[:], fdisk.name)
		if err := currentEBR.Serialize(file, startExt); err != nil {
			return fmt.Errorf("error al crear primer EBR: %v", err)
		}
		return nil
	}

	// Recorrer EBRs existentes
	currentOffset := startExt
	for {
		if string(currentEBR.Part_name[:len(fdisk.name)]) == fdisk.name {
			return fmt.Errorf("el nombre '%s' ya existe en particiones lógicas", fdisk.name)
		}
		if currentEBR.Part_next == -1 {
			break
		}
		currentOffset = int64(currentEBR.Part_next)
		if err := currentEBR.Deserialize(file, currentOffset); err != nil {
			return fmt.Errorf("error al leer EBR: %v", err)
		}
	}

	// Crear nuevo EBR
	ebrSize := int(binary.Size(structures.EBR{}))
	nextStart := currentOffset + int64(currentEBR.Part_size) // Ajustar para empezar después de la partición anterior
	availableSpace = int(extPartition.Part_size) - int(nextStart-startExt) - ebrSize

	if sizeBytes+ebrSize > availableSpace {
		return errors.New("no hay espacio suficiente en la partición extendida")
	}

	newEBR := structures.EBR{
		Part_status: [1]byte{'0'},
		Part_fit:    [1]byte{fdisk.fit[0]},
		Part_start:  int32(nextStart),
		Part_size:   int32(sizeBytes),
		Part_next:   -1,
	}
	copy(newEBR.Part_name[:], fdisk.name)

	currentEBR.Part_next = int32(nextStart)
	if err := currentEBR.Serialize(file, currentOffset); err != nil {
		return fmt.Errorf("error al actualizar EBR anterior: %v", err)
	}
	if err := newEBR.Serialize(file, int64(newEBR.Part_start)); err != nil {
		return fmt.Errorf("error al crear nuevo EBR: %v", err)
	}

	return nil
}
