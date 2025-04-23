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

type CHMOD struct {
	path string
	ugo  string
	r    bool
}

func ParseChmod(tokens []string) (string, error) {
	cmd := &CHMOD{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 && parts[0] != "-r" {
			return "", fmt.Errorf("formato de parámetro inválido: %s", token)
		}
		key := strings.ToLower(parts[0])

		if key == "-path" {
			if len(parts) != 2 {
				return "", errors.New("formato inválido para -path")
			}
			value := strings.Trim(parts[1], "\"")
			if value == "" {
				return "", errors.New("la ruta no puede estar vacía")
			}
			cmd.path = value
		} else if key == "-ugo" {
			if len(parts) != 2 {
				return "", errors.New("formato inválido para -ugo")
			}
			value := parts[1]
			if !isValidUgo(value) {
				return "", fmt.Errorf("permisos UGO inválidos: %s (debe ser 000-777)", value)
			}
			cmd.ugo = value
		} else if key == "-r" {
			if len(parts) != 1 {
				return "", errors.New("formato inválido para -r")
			}
			cmd.r = true
		} else {
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.path == "" || cmd.ugo == "" {
		return "", errors.New("faltan parámetros requeridos: -path, -ugo")
	}

	err := commandChmod(cmd)
	if err != nil {
		return "", fmt.Errorf("error al cambiar permisos: %v", err)
	}

	return fmt.Sprintf("CHMOD: Permisos de %s cambiados a %s exitosamente", cmd.path, cmd.ugo), nil
}

// isValidUgo verifica si los permisos UGO son válidos (000-777)
func isValidUgo(ugo string) bool {
	if len(ugo) != 3 {
		return false
	}
	perm, err := strconv.Atoi(ugo)
	if err != nil {
		return false
	}
	return perm >= 0 && perm <= 777
}

func commandChmod(chmod *CHMOD) error {
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
	parentDirs, targetName := utils.GetParentDirectories(chmod.path)
	if targetName == "" {
		return errors.New("no se puede cambiar permisos de la raíz")
	}

	// Encontrar el inodo del archivo/carpeta
	currentInodeNum := int32(0) // Inodo raíz
	var targetInodeNum int32 = -1

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
			return fmt.Errorf("no se encontró %s en la ruta %s", dir, chmod.path)
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
			if name == targetName {
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
		return fmt.Errorf("no se encontró %s en la ruta %s", targetName, chmod.path)
	}

	// Leer el inodo objetivo
	targetInode := &structures.Inode{}
	err = targetInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+targetInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo objetivo %d: %v", targetInodeNum, err)
	}

	// Verificar permisos (solo propietario o root)
	hasPermission := checkChangePermission(targetInode, stores.CurrentSession)
	if !hasPermission {
		return fmt.Errorf("solo el propietario o root pueden cambiar permisos de %s", chmod.path)
	}

	// Cambiar permisos (recursivamente si -r)
	err = changePermissions(partitionSuperblock, partitionPath, targetInodeNum, chmod.ugo, chmod.r)
	if err != nil {
		return fmt.Errorf("error al cambiar permisos: %v", err)
	}

	// Registrar en el Journal
	err = AddJournalEntry(partitionSuperblock, partitionPath, "chmod", chmod.path, chmod.ugo)
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

// checkChangePermission verifica si el usuario puede cambiar permisos (propietario o root)
func checkChangePermission(inode *structures.Inode, session stores.Session) bool {
	if session.Username == "root" {
		return true
	}
	uid, err := strconv.Atoi(session.UID)
	if err != nil {
		return false
	}
	return int32(uid) == inode.I_uid
}

// changePermissions aplica los permisos al inodo y, si es recursivo, a sus hijos
func changePermissions(sb *structures.SuperBlock, path string, inodeNum int32, ugo string, recursive bool) error {
	inode := &structures.Inode{}
	err := inode.Deserialize(path, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo %d: %v", inodeNum, err)
	}

	// Actualizar permisos
	inode.I_perm = [3]byte{ugo[0], ugo[1], ugo[2]}
	inode.I_mtime = float32(time.Now().Unix())
	err = inode.Serialize(path, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo %d: %v", inodeNum, err)
	}

	// Si es carpeta y -r, aplicar recursivamente
	if inode.I_type[0] == '0' && recursive {
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
				err = changePermissions(sb, path, content.B_inodo, ugo, recursive)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
