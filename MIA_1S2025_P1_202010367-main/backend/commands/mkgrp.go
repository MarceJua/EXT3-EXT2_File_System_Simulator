package commands

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

type MKGRP struct {
	name string
}

func ParseMkgrp(tokens []string) (string, error) {
	cmd := &MKGRP{}

	// Procesar cada token
	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := parts[1]

		if key == "-name" {
			if value == "" || len(value) > 10 {
				return "", errors.New("el nombre del grupo debe tener entre 1 y 10 caracteres")
			}
			cmd.name = value
		} else {
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Validar parámetro requerido
	if cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -name")
	}

	// Ejecutar el comando
	err := commandMkgrp(cmd)
	if err != nil {
		return "", fmt.Errorf("error al crear el grupo: %v", err)
	}

	return fmt.Sprintf("MKGRP: Grupo %s creado exitosamente", cmd.name), nil
}

// commandMkgrp implementa la lógica para crear el grupo
func commandMkgrp(mkgrp *MKGRP) error {
	// Verificar sesión activa y permisos
	if stores.CurrentSession.ID == "" {
		return errors.New("no hay sesión activa, inicie sesión primero")
	}
	if stores.CurrentSession.Username != "root" {
		return errors.New("solo el usuario root puede crear grupos")
	}

	// Obtener la partición montada
	partitionSuperblock, _, partitionPath, err := stores.GetMountedPartitionSuperblock(stores.CurrentSession.ID)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %v", err)
	}

	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Leer el inodo de users.txt (inodo 1)
	usersInode := &structures.Inode{}
	err = usersInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer el inodo de users.txt: %v", err)
	}

	if usersInode.I_type[0] != '1' {
		return errors.New("users.txt no es un archivo válido")
	}

	// Leer el contenido actual de users.txt
	var content strings.Builder
	for _, blockNum := range usersInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		fileBlock := &structures.FileBlock{}
		err = fileBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer el bloque de users.txt: %v", err)
		}
		content.Write(bytes.Trim(fileBlock.B_content[:], "\x00"))
	}
	usersContent := strings.TrimSpace(content.String())

	// Procesar contenido para encontrar GID máximo y verificar duplicados
	lines := strings.Split(usersContent, "\n")
	maxGID := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		if parts[1] == "G" && parts[2] == mkgrp.name && parts[0] != "0" {
			return errors.New("el grupo ya existe")
		}
		if parts[1] == "G" {
			gid, err := strconv.Atoi(parts[0])
			if err == nil && gid > maxGID {
				maxGID = gid
			}
		}
	}

	// Crear nueva línea
	newGID := maxGID + 1
	newLine := fmt.Sprintf("%d,G,%s", newGID, mkgrp.name)
	updatedContent := usersContent + "\n" + newLine

	// Escribir el contenido actualizado
	blockSize := int(partitionSuperblock.S_block_size) // 64 bytes
	contentBytes := []byte(updatedContent)
	numBlocksNeeded := (len(contentBytes) + blockSize - 1) / blockSize

	if numBlocksNeeded > 12 {
		return errors.New("el archivo users.txt excede el límite de bloques directos (12)")
	}

	// Reutilizar o asignar bloques
	for i := 0; i < numBlocksNeeded; i++ {
		start := i * blockSize
		end := start + blockSize
		if end > len(contentBytes) {
			end = len(contentBytes)
		}
		blockContent := contentBytes[start:end]

		var blockNum int32
		if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
			blockNum = usersInode.I_block[i] // Reutilizar bloque existente
		} else {
			// Asignar nuevo bloque
			if partitionSuperblock.S_free_blocks_count <= 0 {
				return errors.New("no hay bloques libres disponibles")
			}
			blockNum = partitionSuperblock.S_first_blo
			partitionSuperblock.S_first_blo++
			partitionSuperblock.S_free_blocks_count--
			partitionSuperblock.S_blocks_count++
			usersInode.I_block[i] = blockNum

			// Actualizar bitmap de bloques
			err = setBitmapBit(partitionPath, int64(partitionSuperblock.S_bm_block_start), int(blockNum), 1)
			if err != nil {
				return fmt.Errorf("error al actualizar bitmap de bloques: %v", err)
			}
		}

		fileBlock := &structures.FileBlock{}
		copy(fileBlock.B_content[:], blockContent)
		err = fileBlock.Serialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*int32(partitionSuperblock.S_block_size)))
		if err != nil {
			return fmt.Errorf("error al escribir bloque %d: %v", blockNum, err)
		}
	}

	// Actualizar inodo
	usersInode.I_size = int32(len(contentBytes))
	err = usersInode.Serialize(partitionPath, int64(partitionSuperblock.S_inode_start+partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo: %v", err)
	}

	// Actualizar superbloque
	err = partitionSuperblock.Serialize(partitionPath, int64(partitionSuperblock.S_inode_start-int32(binary.Size(structures.SuperBlock{}))))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	return nil
}

// setBitmapBit actualiza un bit en el bitmap (0 = libre, 1 = ocupado)
func setBitmapBit(path string, offset int64, bitIndex int, value byte) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	byteIndex := bitIndex / 8
	bitOffset := bitIndex % 8

	var currentByte [1]byte
	_, err = file.ReadAt(currentByte[:], offset+int64(byteIndex))
	if err != nil {
		return err
	}

	if value == 1 {
		currentByte[0] |= 1 << bitOffset
	} else {
		currentByte[0] &= ^(1 << bitOffset)
	}

	_, err = file.WriteAt(currentByte[:], offset+int64(byteIndex))
	return err
}
