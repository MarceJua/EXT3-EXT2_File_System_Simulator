package commands

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	"github.com/MarceJua/MIA_1S2025_P1_202010367/backend/utils"
)

type EDIT struct {
	path string
	cont string
}

func ParseEdit(tokens []string) (string, error) {
	cmd := &EDIT{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := strings.Trim(parts[1], "\"")

		if key == "-path" {
			if value == "" {
				return "", errors.New("la ruta no puede estar vacía")
			}
			cmd.path = value
		} else if key == "-cont" {
			cmd.cont = value
		} else {
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	err := commandEdit(cmd)
	if err != nil {
		return "", fmt.Errorf("error al editar: %v", err)
	}

	return fmt.Sprintf("EDIT: %s editado exitosamente", cmd.path), nil
}

func commandEdit(edit *EDIT) error {
	if stores.CurrentSession.ID == "" {
		return errors.New("no hay sesión activa, inicie sesión primero")
	}

	partitionSuperblock, _, partitionPath, err := stores.GetMountedPartitionSuperblock(stores.CurrentSession.ID)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %v", err)
	}

	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Normalizar la ruta
	parentDirs, destFile := utils.GetParentDirectories(edit.path)
	if destFile == "" {
		return errors.New("la ruta no puede ser la raíz")
	}

	// Encontrar el inodo del archivo
	currentInodeNum := int32(0) // Inodo raíz
	var targetInodeNum int32 = -1
	//targetName := destFile

	currentInode := &structures.Inode{}
	err = currentInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start))
	if err != nil {
		return fmt.Errorf("error al leer inodo raíz: %v", err)
	}

	// Navegar los directorios padres
	for _, dir := range parentDirs {
		if dir == "" {
			continue
		}
		if currentInode.I_type[0] != '0' {
			return fmt.Errorf("el inodo %d no es una carpeta", currentInodeNum)
		}

		found := false
		for _, blockNum := range currentInode.I_block[:12] {
			if blockNum == -1 {
				break
			}
			folderBlock := &structures.FolderBlock{}
			err = folderBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
			if err != nil {
				return fmt.Errorf("error al leer bloque de carpeta %d: %v", blockNum, err)
			}

			for _, content := range folderBlock.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				if name == dir {
					currentInodeNum = content.B_inodo
					err = currentInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+content.B_inodo*partitionSuperblock.S_inode_size))
					if err != nil {
						return fmt.Errorf("error al leer inodo %d: %v", content.B_inodo, err)
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("no se encontró %s en la ruta %s", dir, edit.path)
		}
	}

	// Encontrar el inodo del archivo
	if currentInode.I_type[0] != '0' {
		return fmt.Errorf("el inodo padre %d no es una carpeta", currentInodeNum)
	}
	found := false
	for _, blockNum := range currentInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer bloque de carpeta %d: %v", blockNum, err)
		}

		for _, content := range folderBlock.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == destFile {
				targetInodeNum = content.B_inodo
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("no se encontró %s en la ruta %s", destFile, edit.path)
	}

	// Leer el inodo objetivo
	targetInode := &structures.Inode{}
	err = targetInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+targetInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo objetivo %d: %v", targetInodeNum, err)
	}

	// Verificar que sea un archivo
	if targetInode.I_type[0] != '1' {
		return fmt.Errorf("%s no es un archivo", edit.path)
	}

	// Verificar permisos de escritura
	hasWritePermission := checkWritePermission(targetInode, stores.CurrentSession)
	if !hasWritePermission {
		return fmt.Errorf("no tiene permisos de escritura para %s", edit.path)
	}

	// Dividir el contenido en bloques de 64 bytes
	chunks := utils.SplitStringIntoChunks(edit.cont)
	newBlocksNeeded := len(chunks)
	currentBlocks := 0
	for _, blockNum := range targetInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		currentBlocks++
	}

	// Verificar si hay suficientes bloques libres
	if newBlocksNeeded > currentBlocks {
		availableBlocks := int(partitionSuperblock.S_free_blocks_count)
		if newBlocksNeeded-currentBlocks > availableBlocks {
			return fmt.Errorf("no hay suficientes bloques libres para almacenar el contenido")
		}
	}

	// Liberar bloques actuales si son más de los necesarios
	for i := newBlocksNeeded; i < currentBlocks; i++ {
		blockNum := targetInode.I_block[i]
		if blockNum != -1 {
			err = partitionSuperblock.UpdateBitmapBlock(partitionPath, blockNum)
			if err != nil {
				return fmt.Errorf("error al liberar bloque %d: %v", blockNum, err)
			}
			partitionSuperblock.S_free_blocks_count++
			targetInode.I_block[i] = -1
		}
	}

	// Asignar nuevos bloques si son necesarios
	for i := currentBlocks; i < newBlocksNeeded; i++ {
		newBlockNum, err := partitionSuperblock.FindFreeBlock(partitionPath)
		if err != nil {
			return fmt.Errorf("error al encontrar bloque libre: %v", err)
		}
		err = partitionSuperblock.UpdateBitmapBlock(partitionPath, newBlockNum)
		if err != nil {
			return fmt.Errorf("error al marcar bloque %d: %v", newBlockNum, err)
		}
		partitionSuperblock.S_free_blocks_count--
		targetInode.I_block[i] = newBlockNum
	}

	// Escribir el nuevo contenido
	for i, chunk := range chunks {
		fileBlock := &structures.FileBlock{}
		copy(fileBlock.B_content[:], chunk)
		err = fileBlock.Serialize(partitionPath, int64(partitionSuperblock.S_block_start+targetInode.I_block[i]*partitionSuperblock.S_block_size))
		if err != nil {
			return fmt.Errorf("error al escribir bloque %d: %v", targetInode.I_block[i], err)
		}
	}

	// Actualizar el inodo
	targetInode.I_size = int32(len(edit.cont))
	targetInode.I_mtime = float32(time.Now().Unix())
	err = targetInode.Serialize(partitionPath, int64(partitionSuperblock.S_inode_start+targetInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo %d: %v", targetInodeNum, err)
	}

	// Registrar en el Journal
	truncatedCont := edit.cont
	if len(truncatedCont) > 64 {
		truncatedCont = truncatedCont[:64]
	}
	err = AddJournalEntry(partitionSuperblock, partitionPath, "edit", edit.path, truncatedCont)
	if err != nil {
		return fmt.Errorf("error al registrar en el Journal: %v", err)
	}

	// Actualizar el superbloque
	err = partitionSuperblock.Serialize(partitionPath, int64(partitionSuperblock.S_bm_inode_start)-int64(binary.Size(structures.SuperBlock{})))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	return nil
}
