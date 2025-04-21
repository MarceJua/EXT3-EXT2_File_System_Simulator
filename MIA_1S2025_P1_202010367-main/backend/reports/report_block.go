package reports

import (
	"fmt"
	"os"
	"strings"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

func ReportBlock(sb *structures.SuperBlock, diskPath string) (string, error) {
	file, err := os.Open(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Leer bitmap de inodos
	_, err = file.Seek(int64(sb.S_bm_inode_start), 0)
	if err != nil {
		return "", fmt.Errorf("error buscando bitmap de inodos: %v", err)
	}
	bmInode := make([]byte, sb.S_inodes_count)
	_, err = file.Read(bmInode)
	if err != nil {
		return "", fmt.Errorf("error leyendo bitmap de inodos: %v", err)
	}

	var sbBuilder strings.Builder
	sbBuilder.WriteString("digraph G {\n")
	sbBuilder.WriteString("  node [shape=plaintext]\n")

	inodeSize := int(sb.S_inode_size)
	blockSize := int(sb.S_block_size)
	blockCounter := 0

	for i := int32(0); i < sb.S_inodes_count; i++ {
		if bmInode[i] != '1' { // Solo inodos ocupados
			continue
		}

		inode := &structures.Inode{}
		err := inode.Deserialize(diskPath, int64(sb.S_inode_start+(i*int32(inodeSize))))
		if err != nil {
			return "", fmt.Errorf("error deserializando inodo %d: %v", i, err)
		}

		var prevBlock int = -1
		for j := 0; j < 12; j++ {
			blockNum := inode.I_block[j]
			if blockNum == -1 {
				continue
			}
			blockOffset := int64(sb.S_block_start + (blockNum * int32(blockSize)))

			if inode.I_type[0] == '0' { // Carpeta
				folderBlock := &structures.FolderBlock{}
				err = folderBlock.Deserialize(diskPath, blockOffset)
				if err != nil {
					return "", fmt.Errorf("error deserializando bloque carpeta %d: %v", blockNum, err)
				}
				hasContent := false
				for _, content := range folderBlock.B_content {
					if content.B_inodo != -1 && strings.TrimRight(string(content.B_name[:]), "\x00") != "" {
						hasContent = true
						break
					}
				}
				if hasContent {
					sbBuilder.WriteString(fmt.Sprintf("  block%d [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n", blockCounter))
					sbBuilder.WriteString(fmt.Sprintf("    <TR><TD COLSPAN=\"2\">Bloque Carpeta %d</TD></TR>\n", blockNum))
					sbBuilder.WriteString("    <TR><TD>b_name</TD><TD>b_inodo</TD></TR>\n")
					for _, content := range folderBlock.B_content {
						name := strings.TrimRight(string(content.B_name[:]), "\x00")
						if name != "" && content.B_inodo != -1 {
							name = strings.ReplaceAll(name, "<", "<")
							name = strings.ReplaceAll(name, ">", ">")
							name = strings.ReplaceAll(name, "&", "&")
							sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>%s</TD><TD>%d</TD></TR>\n", name, content.B_inodo))
						}
					}
					sbBuilder.WriteString("  </TABLE>>];\n")
					if prevBlock != -1 {
						sbBuilder.WriteString(fmt.Sprintf("  block%d -> block%d;\n", prevBlock, blockCounter))
					}
					prevBlock = blockCounter
					blockCounter++
				}
			} else if inode.I_type[0] == '1' { // Archivo
				fileBlock := &structures.FileBlock{}
				err = fileBlock.Deserialize(diskPath, blockOffset)
				if err != nil {
					return "", fmt.Errorf("error deserializando bloque archivo %d: %v", blockNum, err)
				}
				content := strings.TrimRight(string(fileBlock.B_content[:]), "\x00")
				if content != "" {
					content = strings.ReplaceAll(content, "<", "<")
					content = strings.ReplaceAll(content, ">", ">")
					content = strings.ReplaceAll(content, "&", "&")
					content = strings.ReplaceAll(content, "\n", "<BR/>")
					sbBuilder.WriteString(fmt.Sprintf("  block%d [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n", blockCounter))
					sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>Bloque Archivo %d</TD></TR>\n", blockNum))
					sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>%s</TD></TR>\n", content))
					sbBuilder.WriteString("  </TABLE>>];\n")
					if prevBlock != -1 {
						sbBuilder.WriteString(fmt.Sprintf("  block%d -> block%d;\n", prevBlock, blockCounter))
					}
					prevBlock = blockCounter
					blockCounter++
				}
			}
		}
	}

	if blockCounter == 0 {
		sbBuilder.WriteString("  node0 [label=\"No hay bloques usados\"];\n")
	}

	sbBuilder.WriteString("}\n")
	return sbBuilder.String(), nil
}
