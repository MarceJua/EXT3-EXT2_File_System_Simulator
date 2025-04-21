package commands

import (
	"errors"
	"fmt"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	utils "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/utils"
)

// CAT estructura que representa el comando cat con sus parámetros
type CAT struct {
	files []string // Lista de rutas de archivos a leer
}

/*
   cat -file1=/test.txt
   cat -file1=/file.txt -file2=/folder/subfolder/newfile.txt
*/

func ParseCat(tokens []string) (string, error) {
	cmd := &CAT{files: []string{}}

	for _, token := range tokens {
		if strings.HasPrefix(token, "-file") {
			parts := strings.SplitN(token, "=", 2)
			if len(parts) != 2 || parts[1] == "" {
				return "", fmt.Errorf("formato inválido para %s, debe ser -fileX=PATH", parts[0])
			}
			filePath := strings.Trim(parts[1], "\"")
			cmd.files = append(cmd.files, filePath)
		} else {
			return "", fmt.Errorf("parámetro inválido: %s", token)
		}
	}

	if len(cmd.files) == 0 {
		return "", errors.New("faltan parámetros requeridos: al menos -file1 es obligatorio")
	}

	output, err := commandCat(cmd)
	if err != nil {
		return "", err
	}

	return output, nil
}

func commandCat(cat *CAT) (string, error) {
	if stores.CurrentSession.ID == "" {
		return "", errors.New("debe iniciar sesión primero")
	}

	partitionSuperblock, _, partitionPath, err := stores.GetMountedPartitionSuperblock(stores.CurrentSession.ID)
	if err != nil {
		return "", fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	var output strings.Builder
	for i, filePath := range cat.files {
		content, err := readFile(partitionSuperblock, partitionPath, filePath)
		if err != nil {
			return "", fmt.Errorf("error al leer %s: %w", filePath, err)
		}
		output.WriteString(fmt.Sprintf("Contenido de %s:\n%s\n", filePath, content))
		if i < len(cat.files)-1 {
			output.WriteString("\n")
		}
	}

	return output.String(), nil
}

func readFile(sb *structures.SuperBlock, diskPath string, filePath string) (string, error) {
	parentDirs, fileName := utils.GetParentDirectories(filePath)
	currentInode := int32(0) // Empezamos en la raíz

	// Navegar a través de los directorios padres
	for _, dir := range parentDirs {
		inode := &structures.Inode{}
		err := inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
		if err != nil {
			return "", err
		}
		if inode.I_type[0] != '0' {
			return "", fmt.Errorf("ruta %s inválida: %s no es un directorio", filePath, dir)
		}

		found := false
		for _, blockIndex := range inode.I_block {
			if blockIndex == -1 {
				break
			}
			block := &structures.FolderBlock{}
			err = block.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
			if err != nil {
				return "", err
			}
			for _, content := range block.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				if name == dir && content.B_inodo != -1 {
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
			return "", fmt.Errorf("directorio %s no encontrado en la ruta %s", dir, filePath)
		}
	}

	// Buscar el archivo en el directorio final
	inode := &structures.Inode{}
	err := inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
	if err != nil {
		return "", err
	}
	if inode.I_type[0] != '0' {
		return "", fmt.Errorf("ruta %s inválida: el padre de %s no es un directorio", filePath, fileName)
	}

	var fileInodeIndex int32 = -1
	for _, blockIndex := range inode.I_block {
		if blockIndex == -1 {
			break
		}
		block := &structures.FolderBlock{}
		err = block.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
		if err != nil {
			return "", err
		}
		for _, content := range block.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == fileName && content.B_inodo != -1 {
				fileInodeIndex = content.B_inodo
				break
			}
		}
		if fileInodeIndex != -1 {
			break
		}
	}
	if fileInodeIndex == -1 {
		return "", fmt.Errorf("archivo %s no encontrado", filePath)
	}

	// Leer el inodo del archivo
	fileInode := &structures.Inode{}
	err = fileInode.Deserialize(diskPath, int64(sb.S_inode_start+fileInodeIndex*sb.S_inode_size))
	if err != nil {
		return "", err
	}
	if fileInode.I_type[0] != '1' {
		return "", fmt.Errorf("%s no es un archivo", filePath)
	}

	// Leer los bloques de datos
	var content strings.Builder
	for _, blockIndex := range fileInode.I_block {
		if blockIndex == -1 || blockIndex == 0 { // 0 podría ser un valor inicial no usado
			break
		}
		fileBlock := &structures.FileBlock{}
		err = fileBlock.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
		if err != nil {
			return "", err
		}
		content.WriteString(strings.Trim(string(fileBlock.B_content[:]), "\x00"))
	}

	return content.String(), nil
}
