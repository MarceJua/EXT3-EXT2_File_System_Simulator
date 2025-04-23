package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	"github.com/MarceJua/MIA_1S2025_P1_202010367/backend/utils"
)

type MOVE struct {
	path    string
	destino string
}

func ParseMove(tokens []string) (string, error) {
	cmd := &MOVE{}

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

	err := commandMove(cmd)
	if err != nil {
		return "", fmt.Errorf("error al mover: %v", err)
	}

	return fmt.Sprintf("MOVE: %s movido a %s exitosamente", cmd.path, cmd.destino), nil
}

func commandMove(move *MOVE) error {
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
	srcParentDirs, srcName := utils.GetParentDirectories(move.path)
	destParentDirs, destName := utils.GetParentDirectories(move.destino)
	if srcName == "" || destName == "" {
		return errors.New("no se puede mover o escribir en la raíz")
	}

	// Encontrar el inodo origen y su padre
	srcParentInodeNum := int32(0) // Raíz
	if len(srcParentDirs) > 0 {
		srcParentInodeNum, err = findInode(sb, diskPath, srcParentDirs[:len(srcParentDirs)-1], srcParentDirs[len(srcParentDirs)-1])
		if err != nil {
			return fmt.Errorf("error al encontrar directorio padre origen %s: %v", strings.Join(srcParentDirs, "/"), err)
		}
	}
	srcInodeNum, err := findInode(sb, diskPath, srcParentDirs, srcName)
	if err != nil {
		return fmt.Errorf("error al encontrar origen %s: %v", move.path, err)
	}

	// Verificar permisos en el origen
	srcInode := &structures.Inode{}
	err = srcInode.Deserialize(diskPath, int64(sb.S_inode_start+srcInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo origen %d: %v", srcInodeNum, err)
	}
	if !checkReadPermission(srcInode, stores.CurrentSession) {
		return fmt.Errorf("no tiene permisos de lectura para %s", move.path)
	}

	// Verificar permisos de escritura en el directorio padre origen
	srcParentInode := &structures.Inode{}
	err = srcParentInode.Deserialize(diskPath, int64(sb.S_inode_start+srcParentInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo padre origen %d: %v", srcParentInodeNum, err)
	}
	if !checkWritePermission(srcParentInode, stores.CurrentSession) {
		return fmt.Errorf("no tiene permisos de escritura en el directorio padre origen %s", strings.Join(srcParentDirs, "/"))
	}

	// Crear directorios padres del destino si no existen
	if len(destParentDirs) > 0 {
		err = createParentFolders(sb, diskPath, destParentDirs)
		if err != nil {
			return fmt.Errorf("error al crear directorios padres para %s: %v", move.destino, err)
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
			return fmt.Errorf("error al encontrar directorio padre destino %s: %v", strings.Join(destParentDirs, "/"), err)
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

	// Verificar que no se cree un ciclo si es una carpeta
	if srcInode.I_type[0] == '0' {
		isDescendant, err := isDescendant(sb, diskPath, srcInodeNum, destParentInodeNum)
		if err != nil {
			return fmt.Errorf("error al verificar ciclos: %v", err)
		}
		if isDescendant {
			return fmt.Errorf("no se puede mover %s a un subdirectorio de sí mismo", move.path)
		}
	}

	// Mover el inodo
	err = moveInode(sb, diskPath, srcInodeNum, srcParentInodeNum, srcName, destParentInodeNum, destName)
	if err != nil {
		return fmt.Errorf("error al mover %s: %v", move.path, err)
	}

	// Serializar superbloque
	err = sb.Serialize(diskPath, int64(mountedPartition.Part_start))
	if err != nil {
		return fmt.Errorf("error al serializar superbloque: %v", err)
	}

	// Registrar en el Journal
	err = AddJournalEntry(sb, diskPath, "move", move.path, move.destino)
	if err != nil {
		return fmt.Errorf("error al registrar en el Journal: %v", err)
	}

	return nil
}

// isDescendant verifica si destParentInodeNum es un descendiente de srcInodeNum
func isDescendant(sb *structures.SuperBlock, diskPath string, srcInodeNum, destParentInodeNum int32) (bool, error) {
	if srcInodeNum == destParentInodeNum {
		return true, nil
	}

	destInode := &structures.Inode{}
	err := destInode.Deserialize(diskPath, int64(sb.S_inode_start+destParentInodeNum*sb.S_inode_size))
	if err != nil {
		return false, fmt.Errorf("error al leer inodo destino %d: %v", destParentInodeNum, err)
	}
	if destInode.I_type[0] != '0' {
		return false, nil // No es una carpeta, no puede ser descendiente
	}

	for _, blockNum := range destInode.I_block[:12] {
		if blockNum == -1 {
			continue
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return false, fmt.Errorf("error al leer bloque %d: %v", blockNum, err)
		}
		for _, content := range folderBlock.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == "." || name == ".." || content.B_inodo == -1 {
				continue
			}
			isDesc, err := isDescendant(sb, diskPath, srcInodeNum, content.B_inodo)
			if err != nil {
				return false, err
			}
			if isDesc {
				return true, nil
			}
		}
	}
	return false, nil
}

// moveInode mueve un inodo del padre origen al padre destino
func moveInode(sb *structures.SuperBlock, diskPath string, srcInodeNum, srcParentInodeNum int32, srcName string, destParentInodeNum int32, destName string) error {
	// Actualizar inodo origen
	srcInode := &structures.Inode{}
	err := srcInode.Deserialize(diskPath, int64(sb.S_inode_start+srcInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo origen %d: %v", srcInodeNum, err)
	}
	srcInode.I_mtime = float32(time.Now().Unix())
	err = srcInode.Serialize(diskPath, int64(sb.S_inode_start+srcInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo origen %d: %v", srcInodeNum, err)
	}

	// Eliminar entrada del directorio padre origen
	srcParentInode := &structures.Inode{}
	err = srcParentInode.Deserialize(diskPath, int64(sb.S_inode_start+srcParentInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo padre origen %d: %v", srcParentInodeNum, err)
	}
	for _, blockNum := range srcParentInode.I_block[:12] {
		if blockNum == -1 {
			continue
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer bloque padre origen %d: %v", blockNum, err)
		}
		for i, content := range folderBlock.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == srcName && content.B_inodo == srcInodeNum {
				folderBlock.B_content[i] = structures.FolderContent{B_name: structures.ToByte12("-"), B_inodo: -1}
				err = folderBlock.Serialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
				if err != nil {
					return fmt.Errorf("error al actualizar bloque padre origen %d: %v", blockNum, err)
				}
				break
			}
		}
	}
	srcParentInode.I_mtime = float32(time.Now().Unix())
	err = srcParentInode.Serialize(diskPath, int64(sb.S_inode_start+srcParentInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo padre origen %d: %v", srcParentInodeNum, err)
	}

	// Añadir entrada al directorio padre destino
	destParentInode := &structures.Inode{}
	err = destParentInode.Deserialize(diskPath, int64(sb.S_inode_start+destParentInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo padre destino %d: %v", destParentInodeNum, err)
	}
	var targetBlockIndex int32 = -1
	var blockToUpdate *structures.FolderBlock
	var contentIndex int
	for i, blockIndex := range destParentInode.I_block {
		if blockIndex != -1 {
			block := &structures.FolderBlock{}
			err = block.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
			if err != nil {
				return err
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
				return fmt.Errorf("error al encontrar bloque libre: %v", err)
			}
			destParentInode.I_block[i] = newBlockNum
			blockToUpdate = &structures.FolderBlock{}
			targetBlockIndex = newBlockNum
			contentIndex = 0

			err = blockToUpdate.Serialize(diskPath, int64(sb.S_block_start+targetBlockIndex*sb.S_block_size))
			if err != nil {
				return err
			}
			err = sb.UpdateBitmapBlock(diskPath, targetBlockIndex)
			if err != nil {
				return err
			}
			sb.S_free_blocks_count--
			err = destParentInode.Serialize(diskPath, int64(sb.S_inode_start+destParentInodeNum*sb.S_inode_size))
			if err != nil {
				return err
			}
			break
		}
	}
	if targetBlockIndex == -1 {
		return errors.New("no hay espacio en el directorio destino para " + destName)
	}

	// Actualizar bloque padre destino
	blockToUpdate.B_content[contentIndex].B_inodo = srcInodeNum
	copy(blockToUpdate.B_content[contentIndex].B_name[:], destName)
	err = blockToUpdate.Serialize(diskPath, int64(sb.S_block_start+targetBlockIndex*sb.S_block_size))
	if err != nil {
		return fmt.Errorf("error al actualizar bloque padre destino %d: %v", targetBlockIndex, err)
	}
	destParentInode.I_mtime = float32(time.Now().Unix())
	err = destParentInode.Serialize(diskPath, int64(sb.S_inode_start+destParentInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo padre destino %d: %v", destParentInodeNum, err)
	}

	return nil
}
