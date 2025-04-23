package commands

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	"github.com/MarceJua/MIA_1S2025_P1_202010367/backend/utils"
)

type REMOVE struct {
	path string
}

func ParseRemove(tokens []string) (string, error) {
	cmd := &REMOVE{}

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
		} else {
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	err := commandRemove(cmd)
	if err != nil {
		return "", fmt.Errorf("error al eliminar: %v", err)
	}

	return fmt.Sprintf("REMOVE: %s eliminado exitosamente", cmd.path), nil
}

func commandRemove(remove *REMOVE) error {
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

	// Normalizar la ruta usando utils.GetParentDirectories
	parentDirs, destDir := utils.GetParentDirectories(remove.path)
	if destDir == "" {
		return errors.New("la ruta no puede ser la raíz")
	}

	// Encontrar el inodo del archivo/carpeta
	currentInodeNum := int32(0) // Inodo raíz
	var targetInodeNum int32 = -1
	var parentInodeNum int32 = -1
	targetName := destDir

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
			return fmt.Errorf("no se encontró %s en la ruta %s", dir, remove.path)
		}
	}

	// Encontrar el inodo objetivo
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
			if name == destDir {
				targetInodeNum = content.B_inodo
				parentInodeNum = currentInodeNum
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("no se encontró %s en la ruta %s", destDir, remove.path)
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
		return fmt.Errorf("no tiene permisos de escritura para %s", remove.path)
	}

	// Si es una carpeta, verificar permisos recursivamente
	if targetInode.I_type[0] == '0' {
		canDelete, err := canDeleteFolder(partitionSuperblock, partitionPath, targetInodeNum, stores.CurrentSession)
		if err != nil {
			return fmt.Errorf("error al verificar permisos de la carpeta: %v", err)
		}
		if !canDelete {
			return fmt.Errorf("no se puede eliminar %s debido a falta de permisos en su contenido", remove.path)
		}
	}

	// Eliminar recursivamente el contenido
	err = deleteInode(partitionSuperblock, partitionPath, targetInodeNum)
	if err != nil {
		return fmt.Errorf("error al eliminar inodo %d: %v", targetInodeNum, err)
	}

	// Actualizar el inodo padre (eliminar la entrada)
	parentInode := &structures.Inode{}
	err = parentInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+parentInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo padre %d: %v", parentInodeNum, err)
	}

	for _, blockNum := range parentInode.I_block[:12] {
		if blockNum == -1 {
			continue
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer bloque de carpeta %d: %v", blockNum, err)
		}

		for i, content := range folderBlock.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == targetName {
				folderBlock.B_content[i] = structures.FolderContent{B_name: structures.ToByte12("-"), B_inodo: -1}
				err = folderBlock.Serialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
				if err != nil {
					return fmt.Errorf("error al actualizar bloque de carpeta %d: %v", blockNum, err)
				}
				break
			}
		}
	}

	// Actualizar el inodo padre
	parentInode.I_mtime = float32(time.Now().Unix())
	err = parentInode.Serialize(partitionPath, int64(partitionSuperblock.S_inode_start+parentInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo padre: %v", err)
	}

	// Registrar en el Journal
	err = AddJournalEntry(partitionSuperblock, partitionPath, "remove", remove.path, targetName)
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

// checkWritePermission verifica si el usuario tiene permisos de escritura
func checkWritePermission(inode *structures.Inode, session stores.Session) bool {
	permStr := string(inode.I_perm[:])
	permInt, err := strconv.Atoi(permStr)
	if err != nil {
		return false
	}

	userPerm := (permInt / 100) % 10
	groupPerm := (permInt / 10) % 10
	otherPerm := permInt % 10

	if session.Username == "root" {
		return true
	}
	uid, err := strconv.Atoi(session.UID)
	if err != nil {
		return false
	}
	if int32(uid) == inode.I_uid {
		return userPerm&2 != 0 // Permiso de escritura para el propietario
	}
	gid, err := strconv.Atoi(session.GID)
	if err != nil {
		return false
	}
	if int32(gid) == inode.I_gid {
		return groupPerm&2 != 0 // Permiso de escritura para el grupo
	}
	return otherPerm&2 != 0 // Permiso de escritura para otros
}

// canDeleteFolder verifica si se pueden eliminar todos los elementos de una carpeta
func canDeleteFolder(sb *structures.SuperBlock, path string, inodeNum int32, session stores.Session) (bool, error) {
	inode := &structures.Inode{}
	err := inode.Deserialize(path, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return false, fmt.Errorf("error al leer inodo %d: %v", inodeNum, err)
	}

	if inode.I_type[0] != '0' {
		return checkWritePermission(inode, session), nil
	}

	for _, blockNum := range inode.I_block[:12] {
		if blockNum == -1 {
			continue
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(path, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return false, fmt.Errorf("error al leer bloque de carpeta %d: %v", blockNum, err)
		}

		for _, content := range folderBlock.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == "." || name == ".." || content.B_inodo == -1 {
				continue
			}
			canDelete, err := canDeleteFolder(sb, path, content.B_inodo, session)
			if err != nil {
				return false, err
			}
			if !canDelete {
				return false, nil
			}
		}
	}
	return true, nil
}

// deleteInode elimina un inodo y sus bloques recursivamente
func deleteInode(sb *structures.SuperBlock, path string, inodeNum int32) error {
	inode := &structures.Inode{}
	err := inode.Deserialize(path, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo %d: %v", inodeNum, err)
	}

	// Si es una carpeta, eliminar sus hijos
	if inode.I_type[0] == '0' {
		for _, blockNum := range inode.I_block[:12] {
			if blockNum == -1 {
				continue
			}
			folderBlock := &structures.FolderBlock{}
			err = folderBlock.Deserialize(path, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al leer bloque de carpeta %d: %v", blockNum, err)
			}

			for _, content := range folderBlock.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				if name == "." || name == ".." || content.B_inodo == -1 {
					continue
				}
				err = deleteInode(sb, path, content.B_inodo)
				if err != nil {
					return err
				}
			}

			// Limpiar el bloque
			err = sb.UpdateBitmapBlock(path, blockNum)
			if err != nil {
				return fmt.Errorf("error al liberar bloque %d: %v", blockNum, err)
			}
			sb.S_free_blocks_count++
		}
	} else {
		// Si es un archivo, liberar sus bloques
		for _, blockNum := range inode.I_block[:12] {
			if blockNum == -1 {
				continue
			}
			err = sb.UpdateBitmapBlock(path, blockNum)
			if err != nil {
				return fmt.Errorf("error al liberar bloque %d: %v", blockNum, err)
			}
			sb.S_free_blocks_count++
		}
	}

	// Liberar el inodo
	err = sb.UpdateBitmapInode(path, inodeNum)
	if err != nil {
		return fmt.Errorf("error al liberar inodo %d: %v", inodeNum, err)
	}
	sb.S_free_inodes_count++

	// Limpiar el inodo
	inode = &structures.Inode{}
	err = inode.Serialize(path, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al limpiar inodo %d: %v", inodeNum, err)
	}

	return nil
}
