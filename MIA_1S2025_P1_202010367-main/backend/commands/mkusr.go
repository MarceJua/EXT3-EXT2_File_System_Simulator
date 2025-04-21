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

type MKUSR struct {
	user string
	pass string
	grp  string
}

func ParseMkusr(tokens []string) (string, error) {
	cmd := &MKUSR{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := strings.Trim(parts[1], "\"")

		switch key {
		case "-user":
			if value == "" || len(value) > 10 {
				return "", errors.New("el usuario debe tener entre 1 y 10 caracteres")
			}
			cmd.user = value
		case "-pass":
			if value == "" || len(value) > 10 {
				return "", errors.New("la contraseña debe tener entre 1 y 10 caracteres")
			}
			cmd.pass = value
		case "-grp":
			if value == "" || len(value) > 10 {
				return "", errors.New("el grupo debe tener entre 1 y 10 caracteres")
			}
			cmd.grp = value
		default:
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.user == "" || cmd.pass == "" || cmd.grp == "" {
		return "", errors.New("faltan parámetros requeridos: -user, -pass, -grp")
	}

	err := commandMkusr(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("MKUSR: Usuario %s creado exitosamente", cmd.user), nil
}

func commandMkusr(mkusr *MKUSR) error {
	if stores.CurrentSession.ID == "" {
		return errors.New("no hay sesión activa, inicie sesión primero")
	}
	if stores.CurrentSession.Username != "root" {
		return errors.New("solo el usuario root puede crear usuarios")
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

	// Encontrar el inodo de users.txt desde la raíz
	rootInode := &structures.Inode{}
	err = rootInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start))
	if err != nil {
		return fmt.Errorf("error al leer inodo raíz: %v", err)
	}
	var usersInodeNum int32 = -1
	for _, blockNum := range rootInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
		if err != nil {
			return err
		}
		for _, content := range folderBlock.B_content {
			if strings.Trim(string(content.B_name[:]), "\x00") == "users.txt" {
				usersInodeNum = content.B_inodo
				break
			}
		}
		if usersInodeNum != -1 {
			break
		}
	}
	if usersInodeNum == -1 {
		return errors.New("users.txt no encontrado")
	}

	usersInode := &structures.Inode{}
	err = usersInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+usersInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer el inodo de users.txt: %v", err)
	}
	if usersInode.I_type[0] != '1' {
		return errors.New("users.txt no es un archivo válido")
	}

	// Leer contenido actual
	var content strings.Builder
	for i, blockNum := range usersInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		fileBlock := &structures.FileBlock{}
		err = fileBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer el bloque %d de users.txt: %v", blockNum, err)
		}
		content.Write(bytes.Trim(fileBlock.B_content[:], "\x00"))
		fmt.Printf("DEBUG: Bloque %d leído: %s\n", i, strings.Trim(string(fileBlock.B_content[:]), "\x00"))
	}
	usersContent := strings.TrimSpace(content.String())
	fmt.Printf("DEBUG: Contenido actual de users.txt:\n%s\n", usersContent)

	// Validar usuario y grupo
	lines := strings.Split(usersContent, "\n")
	maxUID := 0
	grpExists := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		if parts[1] == "U" && parts[3] == mkusr.user && parts[0] != "0" {
			return errors.New("el usuario ya existe")
		}
		if parts[1] == "G" && parts[2] == mkusr.grp && parts[0] != "0" {
			grpExists = true
		}
		if uid, err := strconv.Atoi(parts[0]); err == nil && uid > maxUID {
			maxUID = uid
		}
	}
	if !grpExists {
		return errors.New("el grupo especificado no existe o está eliminado")
	}

	// Agregar nuevo usuario
	newUID := maxUID + 1
	newLine := fmt.Sprintf("%d,U,%s,%s,%s", newUID, mkusr.grp, mkusr.user, mkusr.pass)
	updatedContent := usersContent + "\n" + newLine
	fmt.Printf("DEBUG: Nuevo contenido de users.txt:\n%s\n", updatedContent)

	// Dividir contenido en bloques de 64 bytes
	blockSize := int(partitionSuperblock.S_block_size)
	contentBytes := []byte(updatedContent)
	numBlocksNeeded := (len(contentBytes) + blockSize - 1) / blockSize
	if numBlocksNeeded > 12 {
		return errors.New("el archivo users.txt excede el límite de bloques directos (12)")
	}

	// Actualizar bloques
	for i := 0; i < 12; i++ {
		if i < numBlocksNeeded {
			start := i * blockSize
			end := start + blockSize
			if end > len(contentBytes) {
				end = len(contentBytes)
			}
			blockContent := contentBytes[start:end]
			fmt.Printf("DEBUG: Escribiendo bloque %d: %s\n", i, string(blockContent))

			var blockNum int32
			if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
				blockNum = usersInode.I_block[i] // Reutilizar bloque existente
			} else {
				// Buscar un bloque libre en el bitmap
				blockNum, err = findFreeBlock(partitionSuperblock, partitionPath)
				if err != nil {
					return err
				}
				usersInode.I_block[i] = blockNum
				err = partitionSuperblock.UpdateBitmapBlock(partitionPath, blockNum)
				if err != nil {
					return fmt.Errorf("error al actualizar bitmap de bloques: %v", err)
				}
				partitionSuperblock.S_free_blocks_count--
			}

			fileBlock := &structures.FileBlock{B_content: [64]byte{}}
			copy(fileBlock.B_content[:], blockContent)
			err = fileBlock.Serialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*int32(partitionSuperblock.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al escribir bloque %d: %v", blockNum, err)
			}
		} else if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
			// Liberar bloques sobrantes
			blockNum := usersInode.I_block[i]
			err = partitionSuperblock.UpdateBitmapBlock(partitionPath, blockNum) // Marcar como libre (esto debería ser '0', revisar bitmaps.go)
			if err != nil {
				return fmt.Errorf("error al liberar bloque %d: %v", blockNum, err)
			}
			partitionSuperblock.S_free_blocks_count++
			usersInode.I_block[i] = -1
		}
	}

	usersInode.I_size = int32(len(contentBytes))
	err = usersInode.Serialize(partitionPath, int64(partitionSuperblock.S_inode_start+usersInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo: %v", err)
	}

	err = partitionSuperblock.Serialize(partitionPath, int64(partitionSuperblock.S_bm_inode_start)-int64(binary.Size(partitionSuperblock)))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	return nil
}

func findFreeBlock(sb *structures.SuperBlock, path string) (int32, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	_, err = file.Seek(int64(sb.S_bm_block_start), 0)
	if err != nil {
		return -1, err
	}

	bm := make([]byte, sb.S_blocks_count)
	_, err = file.Read(bm)
	if err != nil {
		return -1, err
	}

	for i := int32(0); i < sb.S_blocks_count; i++ {
		if bm[i] == '0' {
			return i, nil
		}
	}
	return -1, errors.New("no hay bloques libres disponibles")
}
