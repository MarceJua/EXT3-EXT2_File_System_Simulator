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

type RENAME struct {
	path string
	name string
}

func ParseRename(tokens []string) (string, error) {
	cmd := &RENAME{}

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
		} else if key == "-name" {
			if value == "" {
				return "", errors.New("el nuevo nombre no puede estar vacío")
			}
			if strings.Contains(value, "/") {
				return "", errors.New("el nuevo nombre no puede contener '/'")
			}
			cmd.name = value
		} else {
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.path == "" || cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -path, -name")
	}

	err := commandRename(cmd)
	if err != nil {
		return "", fmt.Errorf("error al renombrar: %v", err)
	}

	return fmt.Sprintf("RENAME: %s renombrado a %s exitosamente", cmd.path, cmd.name), nil
}

func commandRename(rename *RENAME) error {
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
	parentDirs, targetName := utils.GetParentDirectories(rename.path)
	if targetName == "" {
		return errors.New("no se puede renombrar la raíz")
	}

	// Encontrar el inodo del archivo/carpeta
	currentInodeNum := int32(0) // Inodo raíz
	var targetInodeNum int32 = -1
	var parentInodeNum int32 = -1
	var targetBlockNum int32 = -1
	var targetContentIndex int = -1

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
			return fmt.Errorf("no se encontró %s en la ruta %s", dir, rename.path)
		}
	}

	// Verificar que el padre sea una carpeta
	if currentInode.I_type[0] != '0' {
		return fmt.Errorf("el inodo padre %d no es una carpeta", currentInodeNum)
	}
	parentInodeNum = currentInodeNum

	// Encontrar el inodo objetivo y verificar que el nuevo nombre no exista
	found := false
	nameExists := false
	for _, blockNum := range currentInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer bloque de carpeta %d: %v", blockNum, err)
		}

		for i, content := range folderBlock.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == targetName {
				targetInodeNum = content.B_inodo
				targetBlockNum = blockNum
				targetContentIndex = i
				found = true
			}
			if name == rename.name {
				nameExists = true
			}
		}
		if found && nameExists {
			break
		}
	}
	if !found {
		return fmt.Errorf("no se encontró %s en la ruta %s", targetName, rename.path)
	}
	if nameExists {
		return fmt.Errorf("ya existe un archivo o carpeta con el nombre %s en el directorio", rename.name)
	}

	// Leer el inodo objetivo
	targetInode := &structures.Inode{}
	err = targetInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+targetInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo objetivo %d: %v", targetInodeNum, err)
	}

	// Verificar permisos de escritura
	hasWritePermission := checkWritePermission(targetInode, stores.CurrentSession)
	if !hasWritePermission {
		return fmt.Errorf("no tiene permisos de escritura para %s", rename.path)
	}

	// Actualizar el nombre en el bloque de carpeta
	folderBlock := &structures.FolderBlock{}
	err = folderBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+targetBlockNum*partitionSuperblock.S_block_size))
	if err != nil {
		return fmt.Errorf("error al leer bloque de carpeta %d: %v", targetBlockNum, err)
	}
	folderBlock.B_content[targetContentIndex].B_name = structures.ToByte12(rename.name)
	err = folderBlock.Serialize(partitionPath, int64(partitionSuperblock.S_block_start+targetBlockNum*partitionSuperblock.S_block_size))
	if err != nil {
		return fmt.Errorf("error al actualizar bloque de carpeta %d: %v", targetBlockNum, err)
	}

	// Actualizar el inodo padre
	currentInode.I_mtime = float32(time.Now().Unix())
	err = currentInode.Serialize(partitionPath, int64(partitionSuperblock.S_inode_start+parentInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo padre %d: %v", parentInodeNum, err)
	}

	// Registrar en el Journal
	truncatedName := rename.name
	if len(truncatedName) > 64 {
		truncatedName = truncatedName[:64]
	}
	err = AddJournalEntry(partitionSuperblock, partitionPath, "rename", rename.path, truncatedName)
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
