package reports

import (
	"fmt"
	"os"
	"strings"
	"time"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

func ReportLS(sb *structures.SuperBlock, diskPath string, dirPath string) (string, error) {
	file, err := os.Open(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Navegar hasta el inodo del directorio
	parts := strings.Split(strings.Trim(dirPath, "/"), "/")
	currentInode := int32(0) // Raíz
	for _, dir := range parts {
		inode := &structures.Inode{}
		err = inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
		if err != nil {
			return "", fmt.Errorf("error deserializando inodo %d: %v", currentInode, err)
		}
		if inode.I_type[0] != '0' {
			return "", fmt.Errorf("ruta %s no es un directorio", dir)
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
			return "", fmt.Errorf("directorio %s no encontrado", dir)
		}
	}

	// Leer el inodo del directorio
	dirInode := &structures.Inode{}
	err = dirInode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
	if err != nil {
		return "", fmt.Errorf("error deserializando inodo de directorio %d: %v", currentInode, err)
	}
	if dirInode.I_type[0] != '0' {
		return "", fmt.Errorf("%s no es un directorio", dirPath)
	}

	// Generar el reporte DOT
	var sbBuilder strings.Builder
	sbBuilder.WriteString("digraph G {\n")
	sbBuilder.WriteString("  node [shape=plaintext]\n")
	sbBuilder.WriteString("  dir [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD COLSPAN=\"9\">Contenido de %s</TD></TR>\n", dirPath))
	sbBuilder.WriteString("    <TR><TD>Permisos</TD><TD>Owner</TD><TD>Grupo</TD><TD>Size (en Bytes)</TD><TD>Fecha Mod.</TD><TD>Hora Mod.</TD><TD>Fecha Creación</TD><TD>Tipo</TD><TD>Name</TD></TR>\n")

	hasContent := false
	for _, blockNum := range dirInode.I_block[:12] {
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
			if name != "" && content.B_inodo != -1 && name != "." && name != ".." {
				// Leer el inodo del archivo/carpeta
				itemInode := &structures.Inode{}
				err = itemInode.Deserialize(diskPath, int64(sb.S_inode_start+content.B_inodo*sb.S_inode_size))
				if err != nil {
					return "", fmt.Errorf("error deserializando inodo %d: %v", content.B_inodo, err)
				}

				// Formatear permisos usando %s en lugar de %c
				perm := fmt.Sprintf("%s%s%s-%s%s%s-%s%s%s",
					ifElse(itemInode.I_type[0] == '0', "d", "-"),
					ifElse(itemInode.I_perm[0]&4 != 0, "r", "-"), ifElse(itemInode.I_perm[0]&2 != 0, "w", "-"), ifElse(itemInode.I_perm[0]&1 != 0, "x", "-"),
					ifElse(itemInode.I_perm[1]&4 != 0, "r", "-"), ifElse(itemInode.I_perm[1]&2 != 0, "w", "-"), ifElse(itemInode.I_perm[1]&1 != 0, "x", "-"),
					ifElse(itemInode.I_perm[2]&4 != 0, "r", "-"), ifElse(itemInode.I_perm[2]&2 != 0, "w", "-"))

				// Obtener propietario y grupo
				owner := fmt.Sprintf("user%d", itemInode.I_uid)
				group := fmt.Sprintf("group%d", itemInode.I_gid)

				// Fechas
				mtime := time.Unix(int64(itemInode.I_mtime), 0)
				ctime := time.Unix(int64(itemInode.I_ctime), 0)

				sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%d</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD><TD>%s</TD></TR>\n",
					perm, owner, group, itemInode.I_size,
					mtime.Format("02/01/2006"), mtime.Format("15:04"), ctime.Format("02/01/2006"),
					ifElse(itemInode.I_type[0] == '1', "Archivo", "Carpeta"), name))
				hasContent = true
			}
		}
	}

	if !hasContent {
		sbBuilder.WriteString("    <TR><TD COLSPAN=\"9\">Directorio vacío</TD></TR>\n")
	}

	sbBuilder.WriteString("  </TABLE>>];\n")
	sbBuilder.WriteString("}\n")
	return sbBuilder.String(), nil
}

func ifElse(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}
