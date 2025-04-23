package commands

import (
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

type COPY struct {
	path    string
	destino string
}

func ParseCopy(tokens []string) (string, error) {
	cmd := &COPY{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := strings.Trim(parts[1], "\"")

		if key == "-path" {
			if value == "" {
				return "", errors.New("la ruta origen no puede estar vacía")
			}
			cmd.path = value
		} else if key == "-destino" {
			if value == "" {
				return "", errors.New("la ruta destino no puede estar vacía")
			}
			cmd.destino = value
		} else {
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.path == "" || cmd.destino == "" {
		return "", errors.New("faltan parámetros requeridos: -path, -destino")
	}

	err := commandCopy(cmd)
	if err != nil {
		return "", fmt.Errorf("error al copiar: %v", err)
	}

	return fmt.Sprintf("COPY: %s copiado a %s exitosamente", cmd.path, cmd.destino), nil
}

func commandCopy(copy *COPY) error {
	if stores.CurrentSession.ID == "" {
		return errors.New("no hay sesión activa, inicie sesión primero")
	}

	sb, mountedPartition, diskPath, err := stores.GetMountedPartitionSuperblock(stores.CurrentSession.ID)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %v", err)
	}

	file, err := os.OpenFile(diskPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Normalizar rutas
	srcParentDirs, srcName := utils.GetParentDirectories(copy.path)
	destParentDirs, destName := utils.GetParentDirectories(copy.destino)
	if srcName == "" || destName == "" {
		return errors.New("no se puede copiar o escribir en la raíz")
	}

	// Encontrar el inodo origen
	srcInodeNum, err := findInode(sb, diskPath, srcParentDirs, srcName)
	if err != nil {
		return fmt.Errorf("error al encontrar origen %s: %v", copy.path, err)
	}

	// Verificar permisos de lectura en el origen
	srcInode := &structures.Inode{}
	err = srcInode.Deserialize(diskPath, int64(sb.S_inode_start+srcInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo origen %d: %v", srcInodeNum, err)
	}
	if !checkReadPermission(srcInode, stores.CurrentSession) {
		return fmt.Errorf("no tiene permisos de lectura para %s", copy.path)
	}

	// Crear directorios padres del destino si no existen
	if len(destParentDirs) > 0 {
		err = createParentFolders(sb, diskPath, destParentDirs)
		if err != nil {
			return fmt.Errorf("error al crear directorios padres para %s: %v", copy.destino, err)
		}
		err = sb.Serialize(diskPath, int64(mountedPartition.Part_start))
		if err != nil {
			return fmt.Errorf("error al serializar superbloque tras crear carpetas: %v", err)
		}
	}

	// Verificar que el destino no exista
	if checkParentExists(sb, diskPath, append(destParentDirs, destName)) {
		return fmt.Errorf("ya existe %s en el directorio destino", destName)
	}

	// Encontrar el inodo del directorio padre del destino
	destParentInodeNum := int32(0) // Raíz
	if len(destParentDirs) > 0 {
		destParentInodeNum, err = findInode(sb, diskPath, destParentDirs[:len(destParentDirs)-1], destParentDirs[len(destParentDirs)-1])
		if err != nil {
			return fmt.Errorf("error al encontrar directorio padre %s: %v", strings.Join(destParentDirs, "/"), err)
		}
	}

	// Verificar permisos de escritura en el directorio padre del destino
	destParentInode := &structures.Inode{}
	err = destParentInode.Deserialize(diskPath, int64(sb.S_inode_start+destParentInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo padre destino %d: %v", destParentInodeNum, err)
	}
	if !checkWritePermission(destParentInode, stores.CurrentSession) {
		return fmt.Errorf("no tiene permisos de escritura en el directorio destino %s", strings.Join(destParentDirs, "/"))
	}

	// Copiar el archivo o carpeta
	_, err = copyInode(sb, diskPath, srcInodeNum, destParentInodeNum, destName)
	if err != nil {
		return fmt.Errorf("error al copiar %s: %v", copy.path, err)
	}

	// Serializar superbloque
	err = sb.Serialize(diskPath, int64(mountedPartition.Part_start))
	if err != nil {
		return fmt.Errorf("error al serializar superbloque: %v", err)
	}

	// Registrar en el Journal
	err = AddJournalEntry(sb, diskPath, "copy", copy.path, copy.destino)
	if err != nil {
		return fmt.Errorf("error al registrar en el Journal: %v", err)
	}

	return nil
}

// findInode encuentra el número de inodo para un archivo/carpeta dado su path
func findInode(sb *structures.SuperBlock, diskPath string, parentDirs []string, name string) (int32, error) {
	currentInodeNum := int32(0) // Raíz
	inode := &structures.Inode{}
	err := inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInodeNum*sb.S_inode_size))
	if err != nil {
		return -1, fmt.Errorf("error al leer inodo raíz: %v", err)
	}

	for _, dir := range parentDirs {
		if dir == "" {
			continue
		}
		if inode.I_type[0] != '0' {
			return -1, fmt.Errorf("el inodo %d no es una carpeta", currentInodeNum)
		}
		found := false
		for _, blockNum := range inode.I_block[:12] {
			if blockNum == -1 {
				break
			}
			folderBlock := &structures.FolderBlock{}
			err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return -1, fmt.Errorf("error al leer bloque %d: %v", blockNum, err)
			}
			for _, content := range folderBlock.B_content {
				contentName := strings.Trim(string(content.B_name[:]), "\x00")
				if contentName == dir {
					currentInodeNum = content.B_inodo
					err = inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInodeNum*sb.S_inode_size))
					if err != nil {
						return -1, fmt.Errorf("error al leer inodo %d: %v", currentInodeNum, err)
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
			return -1, fmt.Errorf("directorio %s no encontrado", dir)
		}
	}

	if inode.I_type[0] != '0' {
		return -1, fmt.Errorf("el inodo padre %d no es una carpeta", currentInodeNum)
	}
	for _, blockNum := range inode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return -1, fmt.Errorf("error al leer bloque %d: %v", blockNum, err)
		}
		for _, content := range folderBlock.B_content {
			contentName := strings.Trim(string(content.B_name[:]), "\x00")
			if contentName == name {
				return content.B_inodo, nil
			}
		}
	}
	return -1, fmt.Errorf("archivo/carpeta %s no encontrado", name)
}

// copyInode copia un inodo (archivo o carpeta) y sus bloques al destino
func copyInode(sb *structures.SuperBlock, diskPath string, srcInodeNum, destParentInodeNum int32, destName string) (int32, error) {
	srcInode := &structures.Inode{}
	err := srcInode.Deserialize(diskPath, int64(sb.S_inode_start+srcInodeNum*sb.S_inode_size))
	if err != nil {
		return -1, fmt.Errorf("error al leer inodo origen %d: %v", srcInodeNum, err)
	}

	// Crear nuevo inodo
	uid, err := strconv.Atoi(stores.CurrentSession.UID)
	if err != nil {
		return -1, fmt.Errorf("error convirtiendo UID: %v", err)
	}
	gid, err := strconv.Atoi(stores.CurrentSession.GID)
	if err != nil {
		return -1, fmt.Errorf("error convirtiendo GID: %v", err)
	}
	newInodeNum, err := sb.FindFreeInode(diskPath)
	if err != nil {
		return -1, fmt.Errorf("error al encontrar inodo libre: %v", err)
	}
	newInode := &structures.Inode{
		I_uid:   int32(uid),
		I_gid:   int32(gid),
		I_size:  srcInode.I_size,
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_type:  srcInode.I_type,
		I_perm:  srcInode.I_perm,
	}

	// Vincular al directorio padre
	destParentInode := &structures.Inode{}
	err = destParentInode.Deserialize(diskPath, int64(sb.S_inode_start+destParentInodeNum*sb.S_inode_size))
	if err != nil {
		return -1, fmt.Errorf("error al leer inodo padre destino %d: %v", destParentInodeNum, err)
	}
	var targetBlockIndex int32 = -1
	var blockToUpdate *structures.FolderBlock
	var contentIndex int
	for i, blockIndex := range destParentInode.I_block {
		if blockIndex != -1 {
			block := &structures.FolderBlock{}
			err = block.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
			if err != nil {
				return -1, err
			}
			for j := 0; j < len(block.B_content); j++ {
				if block.B_content[j].B_inodo == -1 || strings.Trim(string(block.B_content[j].B_name[:]), "\x00") == "" {
					targetBlockIndex = blockIndex
					blockToUpdate = block
					contentIndex = j
					break
				}
			}
			if targetBlockIndex != -1 {
				break
			}
		} else if i < 12 {
			newBlockNum, err := sb.FindFreeBlock(diskPath)
			if err != nil {
				return -1, fmt.Errorf("error al encontrar bloque libre: %v", err)
			}
			destParentInode.I_block[i] = newBlockNum
			blockToUpdate = &structures.FolderBlock{}
			targetBlockIndex = newBlockNum
			contentIndex = 0

			err = blockToUpdate.Serialize(diskPath, int64(sb.S_block_start+targetBlockIndex*sb.S_block_size))
			if err != nil {
				return -1, err
			}
			err = sb.UpdateBitmapBlock(diskPath, targetBlockIndex)
			if err != nil {
				return -1, err
			}
			sb.S_free_blocks_count--
			err = destParentInode.Serialize(diskPath, int64(sb.S_inode_start+destParentInodeNum*sb.S_inode_size))
			if err != nil {
				return -1, err
			}
			break
		}
	}
	if targetBlockIndex == -1 {
		return -1, errors.New("no hay espacio en el directorio destino para " + destName)
	}

	// Copiar contenido según el tipo
	if srcInode.I_type[0] == '1' { // Archivo
		for i, blockNum := range srcInode.I_block[:12] {
			if blockNum == -1 {
				break
			}
			newBlockNum, err := sb.FindFreeBlock(diskPath)
			if err != nil {
				return -1, fmt.Errorf("error al encontrar bloque libre: %v", err)
			}
			newInode.I_block[i] = newBlockNum

			srcBlock := &structures.FileBlock{}
			err = srcBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return -1, fmt.Errorf("error al leer bloque origen %d: %v", blockNum, err)
			}
			newBlock := &structures.FileBlock{B_content: srcBlock.B_content}
			err = newBlock.Serialize(diskPath, int64(sb.S_block_start+newBlockNum*sb.S_block_size))
			if err != nil {
				return -1, fmt.Errorf("error al escribir bloque %d: %v", newBlockNum, err)
			}
			err = sb.UpdateBitmapBlock(diskPath, newBlockNum)
			if err != nil {
				return -1, err
			}
			sb.S_free_blocks_count--
		}
	} else { // Carpeta
		// Crear bloque inicial
		newBlockNum, err := sb.FindFreeBlock(diskPath)
		if err != nil {
			return -1, fmt.Errorf("error al encontrar bloque libre: %v", err)
		}
		newInode.I_block[0] = newBlockNum
		newBlock := &structures.FolderBlock{
			B_content: [4]structures.FolderContent{
				{B_name: [12]byte{'.'}, B_inodo: newInodeNum},
				{B_name: [12]byte{'.', '.'}, B_inodo: destParentInodeNum},
				{B_name: [12]byte{'-'}, B_inodo: -1},
				{B_name: [12]byte{'-'}, B_inodo: -1},
			},
		}
		err = newBlock.Serialize(diskPath, int64(sb.S_block_start+newBlockNum*sb.S_block_size))
		if err != nil {
			return -1, fmt.Errorf("error al escribir bloque %d: %v", newBlockNum, err)
		}
		err = sb.UpdateBitmapBlock(diskPath, newBlockNum)
		if err != nil {
			return -1, err
		}
		sb.S_free_blocks_count--

		// Copiar contenido de la carpeta recursivamente
		currentFolderBlocks := []*structures.FolderBlock{newBlock}
		currentFolderBlockIndex := newBlockNum
		currentContentIndex := 2 // Después de . y ..

		for _, blockNum := range srcInode.I_block[:12] {
			if blockNum == -1 {
				continue
			}
			srcBlock := &structures.FolderBlock{}
			err = srcBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return -1, fmt.Errorf("error al leer bloque origen %d: %v", blockNum, err)
			}
			for _, content := range srcBlock.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				if name == "." || name == ".." || content.B_inodo == -1 {
					continue
				}

				// Asegurar espacio en el FolderBlock
				if currentContentIndex >= 4 {
					// Crear nuevo FolderBlock
					newBlockNum, err = sb.FindFreeBlock(diskPath)
					if err != nil {
						return -1, fmt.Errorf("error al encontrar bloque libre: %v", err)
					}
					for i, blockIndex := range newInode.I_block {
						if blockIndex == -1 && i < 12 {
							newInode.I_block[i] = newBlockNum
							break
						}
					}
					newBlock = &structures.FolderBlock{}
					err = newBlock.Serialize(diskPath, int64(sb.S_block_start+newBlockNum*sb.S_block_size))
					if err != nil {
						return -1, fmt.Errorf("error al escribir bloque %d: %v", newBlockNum, err)
					}
					err = sb.UpdateBitmapBlock(diskPath, newBlockNum)
					if err != nil {
						return -1, err
					}
					sb.S_free_blocks_count--
					currentFolderBlocks = append(currentFolderBlocks, newBlock)
					currentFolderBlockIndex = newBlockNum
					currentContentIndex = 0

					// Actualizar inodo
					err = newInode.Serialize(diskPath, int64(sb.S_inode_start+newInodeNum*sb.S_inode_size))
					if err != nil {
						return -1, fmt.Errorf("error al escribir inodo %d: %v", newInodeNum, err)
					}
				}

				// Copiar archivo/carpeta hijo
				newChildInodeNum, err := copyInode(sb, diskPath, content.B_inodo, newInodeNum, name)
				if err != nil {
					return -1, fmt.Errorf("error al copiar %s: %v", name, err)
				}

				// Añadir al FolderBlock actual
				currentFolderBlocks[len(currentFolderBlocks)-1].B_content[currentContentIndex].B_inodo = newChildInodeNum
				copy(currentFolderBlocks[len(currentFolderBlocks)-1].B_content[currentContentIndex].B_name[:], name)
				err = currentFolderBlocks[len(currentFolderBlocks)-1].Serialize(diskPath, int64(sb.S_block_start+currentFolderBlockIndex*sb.S_block_size))
				if err != nil {
					return -1, fmt.Errorf("error al actualizar bloque %d: %v", currentFolderBlockIndex, err)
				}
				currentContentIndex++
			}
		}
	}

	// Serializar nuevo inodo
	err = newInode.Serialize(diskPath, int64(sb.S_inode_start+newInodeNum*sb.S_inode_size))
	if err != nil {
		return -1, fmt.Errorf("error al escribir inodo %d: %v", newInodeNum, err)
	}
	err = sb.UpdateBitmapInode(diskPath, newInodeNum)
	if err != nil {
		return -1, err
	}
	sb.S_free_inodes_count--

	// Actualizar bloque padre
	blockToUpdate.B_content[contentIndex].B_inodo = newInodeNum
	copy(blockToUpdate.B_content[contentIndex].B_name[:], destName)
	err = blockToUpdate.Serialize(diskPath, int64(sb.S_block_start+targetBlockIndex*sb.S_block_size))
	if err != nil {
		return -1, fmt.Errorf("error al actualizar bloque padre %d: %v", targetBlockIndex, err)
	}

	return newInodeNum, nil
}

// checkReadPermission verifica si el usuario tiene permisos de lectura
func checkReadPermission(inode *structures.Inode, session stores.Session) bool {
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
	gid, err := strconv.Atoi(session.GID)
	if err != nil {
		return false
	}

	if int32(uid) == inode.I_uid {
		return userPerm&4 != 0 // Permiso de lectura para el propietario
	}
	if int32(gid) == inode.I_gid {
		return groupPerm&4 != 0 // Permiso de lectura para el grupo
	}
	return otherPerm&4 != 0 // Permiso de lectura para otros
}
