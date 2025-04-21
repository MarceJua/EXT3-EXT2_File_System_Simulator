package reports

import (
	"fmt"
	"os"
	"strings"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

func ReportTree(sb *structures.SuperBlock, diskPath string) (string, error) {
	file, err := os.Open(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	var sbBuilder strings.Builder
	sbBuilder.WriteString("digraph Tree {\n")
	sbBuilder.WriteString("  node [shape=box]\n")

	inodeSize := int(sb.S_inode_size)
	blockSize := int(sb.S_block_size)
	processedInodes := make(map[int32]bool)

	var buildTree func(inodoNum int32, parentPath string) error
	buildTree = func(inodoNum int32, parentPath string) error {
		if processedInodes[inodoNum] {
			return nil
		}
		processedInodes[inodoNum] = true

		inode := &structures.Inode{}
		err := inode.Deserialize(diskPath, int64(sb.S_inode_start+(inodoNum*int32(inodeSize))))
		if err != nil {
			return fmt.Errorf("error deserializando inodo %d: %v", inodoNum, err)
		}

		currentPath := "\"/\""
		if parentPath != "" {
			currentPath = fmt.Sprintf("\"%s\"", parentPath)
		}

		if inode.I_type[0] == '0' { // Carpeta
			for _, blockNum := range inode.I_block[:12] { // Procesar todos los bloques directos
				if blockNum == -1 {
					break
				}
				folderBlock := &structures.FolderBlock{}
				err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+(blockNum*int32(blockSize))))
				if err != nil {
					return fmt.Errorf("error deserializando bloque carpeta %d: %v", blockNum, err)
				}

				for _, content := range folderBlock.B_content {
					name := strings.TrimRight(string(content.B_name[:]), "\x00")
					childInodo := content.B_inodo
					if name != "" && childInodo != -1 && name != "." && name != ".." {
						childPath := fmt.Sprintf("%s/%s", strings.Trim(parentPath, "\""), name)
						if parentPath == "" {
							childPath = "/" + name
						}
						childName := fmt.Sprintf("\"%s\"", childPath)
						sbBuilder.WriteString(fmt.Sprintf("  %s -> %s\n", currentPath, childName))
						err = buildTree(childInodo, childPath)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if inode.I_type[0] == '1' { // Archivo
			for i, blockNum := range inode.I_block[:12] {
				if blockNum == -1 {
					break
				}
				fileBlock := &structures.FileBlock{}
				err = fileBlock.Deserialize(diskPath, int64(sb.S_block_start+(blockNum*int32(blockSize))))
				if err != nil {
					return fmt.Errorf("error deserializando bloque archivo %d: %v", blockNum, err)
				}
				content := strings.TrimRight(string(fileBlock.B_content[:]), "\x00")
				if content != "" && i == 0 { // Solo conectar el primer bloque como nodo archivo
					sbBuilder.WriteString(fmt.Sprintf("  %s -> %s\n", currentPath, currentPath)) // Conectar al padre
				}
			}
		}
		return nil
	}

	err = buildTree(0, "")
	if err != nil {
		return "", err
	}

	sbBuilder.WriteString("}\n")
	return sbBuilder.String(), nil
}
