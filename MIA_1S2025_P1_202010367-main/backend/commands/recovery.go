package commands

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

type RECOVERY struct {
	id string
}

func ParseRecovery(tokens []string) (string, error) {
	cmd := &RECOVERY{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "-id":
			if value == "" {
				return "", errors.New("el id no puede estar vacío")
			}
			cmd.id = value
		default:
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.id == "" {
		return "", errors.New("faltan parámetros requeridos: -id")
	}

	err := commandRecovery(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("RECOVERY: Sistema restaurado exitosamente en la partición %s", cmd.id), nil
}

func commandRecovery(recovery *RECOVERY) error {
	// Obtener la partición montada
	superblock, _, diskPath, err := stores.GetMountedPartitionSuperblock(recovery.id)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %v", err)
	}

	// Verificar que sea EXT3
	if superblock.S_filesystem_type != 3 {
		return fmt.Errorf("la partición %s no soporta Journaling (no es EXT3)", recovery.id)
	}

	// Formatear la partición (como si ejecutáramos mkfs -fs=3fs)
	err = structures.FormatEXT3(diskPath, int32(superblock.S_inode_start-int32(binary.Size(structures.SuperBlock{}))), superblock.S_blocks_count*superblock.S_block_size)
	if err != nil {
		return fmt.Errorf("error al reformatear la partición: %v", err)
	}

	// Actualizar el SuperBlock
	err = superblock.Serialize(diskPath, int64(superblock.S_inode_start-int32(binary.Size(structures.SuperBlock{}))))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	// Leer las entradas del Journal
	var entries []structures.Journal
	for i := int32(0); i < superblock.S_journal_count; i++ {
		journalEntry := &structures.Journal{}
		offset := int64(superblock.S_journal_start) + int64(i*int32(binary.Size(journalEntry)))
		err := journalEntry.Deserialize(diskPath, offset)
		if err != nil {
			return fmt.Errorf("error al deserializar entrada %d: %v", i, err)
		}
		operation := strings.Trim(string(journalEntry.Content.Operation[:]), "\x00")
		if journalEntry.Count == 0 || operation == "" {
			break
		}
		entries = append(entries, *journalEntry)
	}

	// Reaplicar las operaciones del Journal
	for _, entry := range entries {
		operation := strings.Trim(string(entry.Content.Operation[:]), "\x00")
		path := strings.Trim(string(entry.Content.Path[:]), "\x00")
		content := strings.Trim(string(entry.Content.Content[:]), "\x00")

		switch operation {
		case "mkdir":
			err = reapplyMkdir(superblock, diskPath, path)
			if err != nil {
				return fmt.Errorf("error al reaplicar mkdir para %s: %v", path, err)
			}
		case "mkfile":
			err = reapplyMkfile(superblock, diskPath, path, content)
			if err != nil {
				return fmt.Errorf("error al reaplicar mkfile para %s: %v", path, err)
			}
		case "chgrp":
			user := path
			groups := strings.Split(content, " -> ")
			if len(groups) != 2 {
				return fmt.Errorf("formato de contenido inválido en chgrp: %s", content)
			}
			newGroup := groups[1]
			err = reapplyChgrp(superblock, diskPath, user, newGroup)
			if err != nil {
				return fmt.Errorf("error al reaplicar chgrp para usuario %s: %v", user, err)
			}
		case "mkusr":
			user := path
			passParts := strings.Split(content, "pass: ")
			if len(passParts) != 2 {
				return fmt.Errorf("formato de contenido inválido en mkusr: %s", content)
			}
			pass := passParts[1]
			err = reapplyMkusr(superblock, diskPath, user, pass, "users") // Asumiendo grupo por defecto "users"
			if err != nil {
				return fmt.Errorf("error al reaplicar mkusr para usuario %s: %v", user, err)
			}
		case "mkgrp":
			group := path
			err = reapplyMkgrp(superblock, diskPath, group)
			if err != nil {
				return fmt.Errorf("error al reaplicar mkgrp para grupo %s: %v", group, err)
			}
		case "mkfs":
			// No necesitamos hacer nada, ya formateamos al inicio
			continue
		default:
			fmt.Printf("Operación no soportada en RECOVERY: %s\n", operation)
		}
	}

	return nil
}

func reapplyMkdir(sb *structures.SuperBlock, diskPath, path string) error {
	// Implementación simplificada de mkdir
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	currentInode := int32(0)

	for i, dir := range pathParts {
		if dir == "" {
			continue
		}

		inode := &structures.Inode{}
		err := inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
		if err != nil {
			return fmt.Errorf("error al leer inodo %d: %v", currentInode, err)
		}

		if inode.I_type[0] != '0' {
			return fmt.Errorf("el path %s no es un directorio", strings.Join(pathParts[:i+1], "/"))
		}

		found := false
		for _, blockIndex := range inode.I_block {
			if blockIndex == -1 {
				break
			}
			folderBlock := &structures.FolderBlock{}
			err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al leer bloque %d: %v", blockIndex, err)
			}
			for _, content := range folderBlock.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				if name == dir {
					currentInode = content.B_inodo
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			if i < len(pathParts)-1 {
				return fmt.Errorf("el directorio intermedio %s no existe", strings.Join(pathParts[:i+1], "/"))
			}

			// Crear nuevo inodo para el directorio
			newInodeNum := sb.S_first_ino
			sb.S_first_ino++
			sb.S_free_inodes_count--

			newInode := &structures.Inode{}
			newInode.I_uid = 1
			newInode.I_gid = 1
			newInode.I_size = 0
			now := float32(time.Now().Unix())
			newInode.I_atime = now
			newInode.I_ctime = now
			newInode.I_mtime = now
			newInode.I_type = [1]byte{'0'}
			newInode.I_perm = [3]byte{'6', '6', '4'}

			// Asignar un bloque para el nuevo directorio
			newBlockNum := sb.S_first_blo
			sb.S_first_blo++
			sb.S_free_blocks_count--
			newInode.I_block[0] = newBlockNum

			// Crear el bloque de carpeta
			newBlock := &structures.FolderBlock{}
			copy(newBlock.B_content[0].B_name[:], ".")
			newBlock.B_content[0].B_inodo = newInodeNum
			copy(newBlock.B_content[1].B_name[:], "..")
			newBlock.B_content[1].B_inodo = currentInode

			// Actualizar el bloque del directorio padre
			parentInode := &structures.Inode{}
			err = parentInode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
			if err != nil {
				return fmt.Errorf("error al leer inodo padre %d: %v", currentInode, err)
			}
			var parentBlockIndex int32 = -1
			for i, blockIndex := range parentInode.I_block {
				if blockIndex == -1 {
					parentBlockIndex = sb.S_first_blo
					sb.S_first_blo++
					sb.S_free_blocks_count--
					parentInode.I_block[i] = parentBlockIndex
					break
				}
			}
			if parentBlockIndex == -1 {
				return fmt.Errorf("no hay bloques disponibles en el directorio padre")
			}

			parentBlock := &structures.FolderBlock{}
			if parentInode.I_block[0] != -1 {
				err = parentBlock.Deserialize(diskPath, int64(sb.S_block_start+parentInode.I_block[0]*sb.S_block_size))
				if err != nil {
					return fmt.Errorf("error al leer bloque padre: %v", err)
				}
			}
			for i := range parentBlock.B_content {
				if strings.Trim(string(parentBlock.B_content[i].B_name[:]), "\x00") == "" {
					copy(parentBlock.B_content[i].B_name[:], dir)
					parentBlock.B_content[i].B_inodo = newInodeNum
					break
				}
			}

			// Escribir las estructuras
			err = newInode.Serialize(diskPath, int64(sb.S_inode_start+newInodeNum*sb.S_inode_size))
			if err != nil {
				return fmt.Errorf("error al escribir nuevo inodo: %v", err)
			}
			err = newBlock.Serialize(diskPath, int64(sb.S_block_start+newBlockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al escribir nuevo bloque: %v", err)
			}
			err = parentBlock.Serialize(diskPath, int64(sb.S_block_start+parentInode.I_block[0]*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al escribir bloque padre: %v", err)
			}
			err = parentInode.Serialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
			if err != nil {
				return fmt.Errorf("error al escribir inodo padre: %v", err)
			}
			err = sb.Serialize(diskPath, int64(sb.S_inode_start-int32(binary.Size(structures.SuperBlock{}))))
			if err != nil {
				return fmt.Errorf("error al actualizar superbloque: %v", err)
			}
		}
	}

	return nil
}

func reapplyMkfile(sb *structures.SuperBlock, diskPath, path, content string) error {
	// Implementación simplificada de mkfile
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	filename := pathParts[len(pathParts)-1]
	parentPath := strings.Join(pathParts[:len(pathParts)-1], "/")
	currentInode := int32(0)

	// Navegar al directorio padre
	for i, dir := range pathParts[:len(pathParts)-1] {
		if dir == "" {
			continue
		}

		inode := &structures.Inode{}
		err := inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
		if err != nil {
			return fmt.Errorf("error al leer inodo %d: %v", currentInode, err)
		}

		if inode.I_type[0] != '0' {
			return fmt.Errorf("el path %s no es un directorio", strings.Join(pathParts[:i+1], "/"))
		}

		found := false
		for _, blockIndex := range inode.I_block {
			if blockIndex == -1 {
				break
			}
			folderBlock := &structures.FolderBlock{}
			err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al leer bloque %d: %v", blockIndex, err)
			}
			for _, content := range folderBlock.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				if name == dir {
					currentInode = content.B_inodo
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			return fmt.Errorf("el directorio padre %s no existe", parentPath)
		}
	}

	// Crear nuevo inodo para el archivo
	newInodeNum := sb.S_first_ino
	sb.S_first_ino++
	sb.S_free_inodes_count--

	newInode := &structures.Inode{}
	newInode.I_uid = 1
	newInode.I_gid = 1
	newInode.I_size = int32(len(content))
	now := float32(time.Now().Unix())
	newInode.I_atime = now
	newInode.I_ctime = now
	newInode.I_mtime = now
	newInode.I_type = [1]byte{'1'}
	newInode.I_perm = [3]byte{'6', '6', '4'}

	// Asignar bloques para el contenido
	contentBytes := []byte(content)
	blockSize := int(sb.S_block_size)
	numBlocksNeeded := (len(contentBytes) + blockSize - 1) / blockSize
	for i := 0; i < numBlocksNeeded; i++ {
		newBlockNum := sb.S_first_blo
		sb.S_first_blo++
		sb.S_free_blocks_count--
		newInode.I_block[i] = newBlockNum

		fileBlock := &structures.FileBlock{}
		start := i * blockSize
		end := start + blockSize
		if end > len(contentBytes) {
			end = len(contentBytes)
		}
		copy(fileBlock.B_content[:], contentBytes[start:end])

		err := fileBlock.Serialize(diskPath, int64(sb.S_block_start+newBlockNum*sb.S_block_size))
		if err != nil {
			return fmt.Errorf("error al escribir bloque %d: %v", newBlockNum, err)
		}
	}

	// Actualizar el directorio padre
	parentInode := &structures.Inode{}
	err := parentInode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo padre %d: %v", currentInode, err)
	}
	var parentBlockIndex int32 = parentInode.I_block[0]
	if parentBlockIndex == -1 {
		parentBlockIndex = sb.S_first_blo
		sb.S_first_blo++
		sb.S_free_blocks_count--
		parentInode.I_block[0] = parentBlockIndex
	}

	parentBlock := &structures.FolderBlock{}
	err = parentBlock.Deserialize(diskPath, int64(sb.S_block_start+parentBlockIndex*sb.S_block_size))
	if err != nil {
		return fmt.Errorf("error al leer bloque padre: %v", err)
	}
	for i := range parentBlock.B_content {
		if strings.Trim(string(parentBlock.B_content[i].B_name[:]), "\x00") == "" {
			copy(parentBlock.B_content[i].B_name[:], filename)
			parentBlock.B_content[i].B_inodo = newInodeNum
			break
		}
	}

	// Escribir las estructuras
	err = newInode.Serialize(diskPath, int64(sb.S_inode_start+newInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al escribir nuevo inodo: %v", err)
	}
	err = parentBlock.Serialize(diskPath, int64(sb.S_block_start+parentBlockIndex*sb.S_block_size))
	if err != nil {
		return fmt.Errorf("error al escribir bloque padre: %v", err)
	}
	err = parentInode.Serialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al escribir inodo padre: %v", err)
	}
	err = sb.Serialize(diskPath, int64(sb.S_inode_start-int32(binary.Size(structures.SuperBlock{}))))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	return nil
}

func reapplyChgrp(sb *structures.SuperBlock, diskPath, user, newGroup string) error {
	usersInode := &structures.Inode{}
	err := usersInode.Deserialize(diskPath, int64(sb.S_inode_start+sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo de users.txt: %v", err)
	}

	var content strings.Builder
	for _, blockNum := range usersInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		fileBlock := &structures.FileBlock{}
		err = fileBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer bloque de users.txt: %v", err)
		}
		content.Write(bytes.Trim(fileBlock.B_content[:], "\x00"))
	}
	usersContent := strings.TrimSpace(content.String())

	lines := strings.Split(usersContent, "\n")
	userFound := false
	grpExists := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		if parts[1] == "U" && parts[3] == user && parts[0] != "0" {
			userFound = true
		}
		if parts[1] == "G" && parts[2] == newGroup && parts[0] != "0" {
			grpExists = true
		}
	}

	if !userFound {
		return fmt.Errorf("usuario %s no existe", user)
	}
	if !grpExists {
		return fmt.Errorf("grupo %s no existe", newGroup)
	}

	for i, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 4 && parts[1] == "U" && parts[3] == user && parts[0] != "0" {
			lines[i] = fmt.Sprintf("%s,U,%s,%s,%s", parts[0], newGroup, parts[3], parts[4])
			break
		}
	}

	updatedContent := strings.Join(lines, "\n")
	contentBytes := []byte(updatedContent)
	blockSize := int(sb.S_block_size)
	numBlocksNeeded := (len(contentBytes) + blockSize - 1) / blockSize

	for i := 0; i < 12; i++ {
		if i < numBlocksNeeded {
			start := i * blockSize
			end := start + blockSize
			if end > len(contentBytes) {
				end = len(contentBytes)
			}
			blockContent := contentBytes[start:end]

			var blockNum int32
			if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
				blockNum = usersInode.I_block[i]
			} else {
				blockNum = sb.S_first_blo
				sb.S_first_blo++
				sb.S_free_blocks_count--
				usersInode.I_block[i] = blockNum
			}

			fileBlock := &structures.FileBlock{}
			for j := range fileBlock.B_content {
				fileBlock.B_content[j] = 0
			}
			copy(fileBlock.B_content[:], blockContent)
			err = fileBlock.Serialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al escribir bloque %d: %v", blockNum, err)
			}
		} else if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
			blockNum := usersInode.I_block[i]
			fileBlock := &structures.FileBlock{}
			for j := range fileBlock.B_content {
				fileBlock.B_content[j] = 0
			}
			err = fileBlock.Serialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al limpiar bloque %d: %v", blockNum, err)
			}
			sb.S_free_blocks_count++
			usersInode.I_block[i] = -1
		}
	}

	usersInode.I_size = int32(len(contentBytes))
	usersInode.I_mtime = float32(time.Now().Unix())
	err = usersInode.Serialize(diskPath, int64(sb.S_inode_start+sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo: %v", err)
	}

	err = sb.Serialize(diskPath, int64(sb.S_inode_start-int32(binary.Size(structures.SuperBlock{}))))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	return nil
}

func reapplyMkusr(sb *structures.SuperBlock, diskPath, user, pass, group string) error {
	usersInode := &structures.Inode{}
	err := usersInode.Deserialize(diskPath, int64(sb.S_inode_start+sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo de users.txt: %v", err)
	}

	var content strings.Builder
	for _, blockNum := range usersInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		fileBlock := &structures.FileBlock{}
		err = fileBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer bloque de users.txt: %v", err)
		}
		content.Write(bytes.Trim(fileBlock.B_content[:], "\x00"))
	}
	usersContent := strings.TrimSpace(content.String())

	lines := strings.Split(usersContent, "\n")
	var maxID int
	grpExists := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		var id int
		fmt.Sscanf(parts[0], "%d", &id)
		if id > maxID {
			maxID = id
		}
		if parts[1] == "G" && parts[2] == group && parts[0] != "0" {
			grpExists = true
		}
	}

	if !grpExists {
		return fmt.Errorf("grupo %s no existe", group)
	}

	newID := maxID + 1
	newLine := fmt.Sprintf("%d,U,%s,%s,%s\n", newID, group, user, pass)
	updatedContent := usersContent + "\n" + newLine
	contentBytes := []byte(updatedContent)
	blockSize := int(sb.S_block_size)
	numBlocksNeeded := (len(contentBytes) + blockSize - 1) / blockSize

	for i := 0; i < 12; i++ {
		if i < numBlocksNeeded {
			start := i * blockSize
			end := start + blockSize
			if end > len(contentBytes) {
				end = len(contentBytes)
			}
			blockContent := contentBytes[start:end]

			var blockNum int32
			if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
				blockNum = usersInode.I_block[i]
			} else {
				blockNum = sb.S_first_blo
				sb.S_first_blo++
				sb.S_free_blocks_count--
				usersInode.I_block[i] = blockNum
			}

			fileBlock := &structures.FileBlock{}
			for j := range fileBlock.B_content {
				fileBlock.B_content[j] = 0
			}
			copy(fileBlock.B_content[:], blockContent)
			err = fileBlock.Serialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al escribir bloque %d: %v", blockNum, err)
			}
		} else if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
			blockNum := usersInode.I_block[i]
			fileBlock := &structures.FileBlock{}
			for j := range fileBlock.B_content {
				fileBlock.B_content[j] = 0
			}
			err = fileBlock.Serialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al limpiar bloque %d: %v", blockNum, err)
			}
			sb.S_free_blocks_count++
			usersInode.I_block[i] = -1
		}
	}

	usersInode.I_size = int32(len(contentBytes))
	usersInode.I_mtime = float32(time.Now().Unix())
	err = usersInode.Serialize(diskPath, int64(sb.S_inode_start+sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo: %v", err)
	}

	err = sb.Serialize(diskPath, int64(sb.S_inode_start-int32(binary.Size(structures.SuperBlock{}))))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	return nil
}

func reapplyMkgrp(sb *structures.SuperBlock, diskPath, group string) error {
	usersInode := &structures.Inode{}
	err := usersInode.Deserialize(diskPath, int64(sb.S_inode_start+sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo de users.txt: %v", err)
	}

	var content strings.Builder
	for _, blockNum := range usersInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		fileBlock := &structures.FileBlock{}
		err = fileBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer bloque de users.txt: %v", err)
		}
		content.Write(bytes.Trim(fileBlock.B_content[:], "\x00"))
	}
	usersContent := strings.TrimSpace(content.String())

	lines := strings.Split(usersContent, "\n")
	var maxID int
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		var id int
		fmt.Sscanf(parts[0], "%d", &id)
		if id > maxID {
			maxID = id
		}
	}

	newID := maxID + 1
	newLine := fmt.Sprintf("%d,G,%s\n", newID, group)
	updatedContent := usersContent + "\n" + newLine
	contentBytes := []byte(updatedContent)
	blockSize := int(sb.S_block_size)
	numBlocksNeeded := (len(contentBytes) + blockSize - 1) / blockSize

	for i := 0; i < 12; i++ {
		if i < numBlocksNeeded {
			start := i * blockSize
			end := start + blockSize
			if end > len(contentBytes) {
				end = len(contentBytes)
			}
			blockContent := contentBytes[start:end]

			var blockNum int32
			if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
				blockNum = usersInode.I_block[i]
			} else {
				blockNum = sb.S_first_blo
				sb.S_first_blo++
				sb.S_free_blocks_count--
				usersInode.I_block[i] = blockNum
			}

			fileBlock := &structures.FileBlock{}
			for j := range fileBlock.B_content {
				fileBlock.B_content[j] = 0
			}
			copy(fileBlock.B_content[:], blockContent)
			err = fileBlock.Serialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al escribir bloque %d: %v", blockNum, err)
			}
		} else if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
			blockNum := usersInode.I_block[i]
			fileBlock := &structures.FileBlock{}
			for j := range fileBlock.B_content {
				fileBlock.B_content[j] = 0
			}
			err = fileBlock.Serialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al limpiar bloque %d: %v", blockNum, err)
			}
			sb.S_free_blocks_count++
			usersInode.I_block[i] = -1
		}
	}

	usersInode.I_size = int32(len(contentBytes))
	usersInode.I_mtime = float32(time.Now().Unix())
	err = usersInode.Serialize(diskPath, int64(sb.S_inode_start+sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo: %v", err)
	}

	err = sb.Serialize(diskPath, int64(sb.S_inode_start-int32(binary.Size(structures.SuperBlock{}))))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	return nil
}
