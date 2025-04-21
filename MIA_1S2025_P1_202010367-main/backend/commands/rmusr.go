package commands

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

type RMUSR struct {
	user string
}

func ParseRmusr(tokens []string) (string, error) {
	cmd := &RMUSR{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := strings.Trim(parts[1], "\"")

		if key == "-user" {
			if value == "" || len(value) > 10 {
				return "", errors.New("el usuario debe tener entre 1 y 10 caracteres")
			}
			cmd.user = value
		} else {
			return "", fmt.Errorf("parámetro inválido: %s", key)
		}
	}

	if cmd.user == "" {
		return "", errors.New("faltan parámetros requeridos: -user")
	}

	err := commandRmusr(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("RMUSR: Usuario %s eliminado exitosamente", cmd.user), nil
}

func commandRmusr(rmusr *RMUSR) error {
	if stores.CurrentSession.ID == "" {
		return errors.New("no hay sesión activa, inicie sesión primero")
	}
	if stores.CurrentSession.Username != "root" {
		return errors.New("solo el usuario root puede eliminar usuarios")
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

	usersInode := &structures.Inode{}
	err = usersInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer el inodo de users.txt: %v", err)
	}

	if usersInode.I_type[0] != '1' {
		return errors.New("users.txt no es un archivo válido")
	}

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
	fmt.Printf("DEBUG: Contenido actual de users.txt en RMUSR:\n%s\n", usersContent)

	lines := strings.Split(usersContent, "\n")
	found := false
	for i, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		if parts[1] == "U" && parts[3] == rmusr.user && parts[0] != "0" {
			lines[i] = fmt.Sprintf("0,U,%s,%s,%s", parts[2], parts[3], parts[4])
			found = true
			break
		}
	}

	if !found {
		return errors.New("el usuario no existe o ya está eliminado")
	}

	updatedContent := strings.Join(lines, "\n")
	fmt.Printf("DEBUG: Nuevo contenido de users.txt en RMUSR:\n%s\n", updatedContent)

	blockSize := int(partitionSuperblock.S_block_size)
	contentBytes := []byte(updatedContent)
	numBlocksNeeded := (len(contentBytes) + blockSize - 1) / blockSize

	if numBlocksNeeded > 12 {
		return errors.New("el archivo users.txt excede el límite de bloques directos (12)")
	}

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
				if partitionSuperblock.S_free_blocks_count <= 0 {
					return errors.New("no hay bloques libres disponibles")
				}
				blockNum = partitionSuperblock.S_first_blo
				partitionSuperblock.S_first_blo++
				partitionSuperblock.S_free_blocks_count--
				partitionSuperblock.S_blocks_count++
				usersInode.I_block[i] = blockNum

				err = setBitmapBit(partitionPath, int64(partitionSuperblock.S_bm_block_start), int(blockNum), 1)
				if err != nil {
					return fmt.Errorf("error al actualizar bitmap de bloques: %v", err)
				}
			}

			fileBlock := &structures.FileBlock{}
			// Limpiar el bloque completo antes de escribir
			for j := range fileBlock.B_content {
				fileBlock.B_content[j] = 0
			}
			copy(fileBlock.B_content[:], blockContent)
			err = fileBlock.Serialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*int32(partitionSuperblock.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al escribir bloque %d: %v", blockNum, err)
			}
		} else if i < len(usersInode.I_block) && usersInode.I_block[i] != -1 {
			blockNum := usersInode.I_block[i]
			fileBlock := &structures.FileBlock{}
			// Limpiar el bloque completo
			for j := range fileBlock.B_content {
				fileBlock.B_content[j] = 0
			}
			err = fileBlock.Serialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*int32(partitionSuperblock.S_block_size)))
			if err != nil {
				return fmt.Errorf("error al limpiar bloque %d: %v", blockNum, err)
			}
			err = setBitmapBit(partitionPath, int64(partitionSuperblock.S_bm_block_start), int(blockNum), 0)
			if err != nil {
				return fmt.Errorf("error al actualizar bitmap de bloques: %v", err)
			}
			partitionSuperblock.S_free_blocks_count++
			usersInode.I_block[i] = -1
		}
	}

	usersInode.I_size = int32(len(contentBytes))
	err = usersInode.Serialize(partitionPath, int64(partitionSuperblock.S_inode_start+partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al actualizar inodo: %v", err)
	}

	err = partitionSuperblock.Serialize(partitionPath, int64(partitionSuperblock.S_inode_start-int32(binary.Size(structures.SuperBlock{}))))
	if err != nil {
		return fmt.Errorf("error al actualizar superbloque: %v", err)
	}

	return nil
}
