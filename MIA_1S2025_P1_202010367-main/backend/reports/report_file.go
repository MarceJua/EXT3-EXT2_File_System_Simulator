package reports

import (
	"fmt"
	"os"
	"strings"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

func ReportFile(sb *structures.SuperBlock, diskPath string, filePath string) (string, error) {
	file, err := os.Open(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Navegar hasta el inodo del archivo
	parts := strings.Split(strings.Trim(filePath, "/"), "/")
	currentInode := int32(0) // Raíz
	for i, part := range parts {
		inode := &structures.Inode{}
		err = inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
		if err != nil {
			return "", fmt.Errorf("error deserializando inodo %d: %v", currentInode, err)
		}
		if inode.I_type[0] != '0' && i < len(parts)-1 {
			return "", fmt.Errorf("ruta %s no es un directorio", strings.Join(parts[:i+1], "/"))
		}
		found := false
		for _, blockNum := range inode.I_block[:12] {
			if blockNum == -1 {
				break
			}
			folderBlock := &structures.FolderBlock{}
			err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return "", fmt.Errorf("error deserializando bloque %d: %v", blockNum, err)
			}
			for _, content := range folderBlock.B_content {
				name := strings.TrimRight(string(content.B_name[:]), "\x00")
				if name == part && content.B_inodo != -1 {
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
			return "", fmt.Errorf("archivo o directorio %s no encontrado", filePath)
		}
	}

	// Leer el inodo del archivo
	fileInode := &structures.Inode{}
	err = fileInode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
	if err != nil {
		return "", fmt.Errorf("error deserializando inodo del archivo %d: %v", currentInode, err)
	}
	if fileInode.I_type[0] != '1' {
		return "", fmt.Errorf("%s no es un archivo", filePath)
	}

	// Leer el contenido del archivo
	var content strings.Builder
	for _, blockNum := range fileInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		fileBlock := &structures.FileBlock{}
		err = fileBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return "", fmt.Errorf("error deserializando bloque de archivo %d: %v", blockNum, err)
		}
		blockContent := strings.TrimRight(string(fileBlock.B_content[:]), "\x00")
		content.WriteString(blockContent)
	}

	// Si el archivo está vacío, devolver ceros
	outputContent := content.String()
	if fileInode.I_size == 0 {
		outputContent = "0000000000000000000000000000000000000000000000000000000000000000" // 64 ceros
	}

	return outputContent, nil
}
