package reports

import (
	"fmt"
	"strings"
	"time"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

func ReportSB(sb *structures.SuperBlock) (string, error) {
	var sbBuilder strings.Builder

	// Formatear tiempos correctamente
	mtime := "No establecido"
	if sb.S_mtime != 0 {
		mtime = time.Unix(int64(sb.S_mtime), 0).Format(time.RFC3339)
	}

	umtime := "No establecido"
	if sb.S_umtime != 0 {
		umtime = time.Unix(int64(sb.S_umtime), 0).Format(time.RFC3339)
	}

	sbBuilder.WriteString("digraph G {\n")
	sbBuilder.WriteString("  node [shape=plaintext]\n")
	sbBuilder.WriteString("  tbl [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
	sbBuilder.WriteString("    <TR><TD COLSPAN=\"2\">REPORTE SUPERBLOQUE</TD></TR>\n")
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_filesystem_type</TD><TD>%d</TD></TR>\n", sb.S_filesystem_type))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_inodes_count</TD><TD>%d</TD></TR>\n", sb.S_inodes_count))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_blocks_count</TD><TD>%d</TD></TR>\n", sb.S_blocks_count))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_free_inodes_count</TD><TD>%d</TD></TR>\n", sb.S_free_inodes_count))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_free_blocks_count</TD><TD>%d</TD></TR>\n", sb.S_free_blocks_count))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_mtime</TD><TD>%s</TD></TR>\n", mtime))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_umtime</TD><TD>%s</TD></TR>\n", umtime))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_mnt_count</TD><TD>%d</TD></TR>\n", sb.S_mnt_count))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_magic</TD><TD>%d</TD></TR>\n", sb.S_magic))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_inode_size</TD><TD>%d</TD></TR>\n", sb.S_inode_size))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_block_size</TD><TD>%d</TD></TR>\n", sb.S_block_size))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_first_ino</TD><TD>%d</TD></TR>\n", sb.S_first_ino))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_first_blo</TD><TD>%d</TD></TR>\n", sb.S_first_blo))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_bm_inode_start</TD><TD>%d</TD></TR>\n", sb.S_bm_inode_start))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_bm_block_start</TD><TD>%d</TD></TR>\n", sb.S_bm_block_start))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_inode_start</TD><TD>%d</TD></TR>\n", sb.S_inode_start))
	sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>S_block_start</TD><TD>%d</TD></TR>\n", sb.S_block_start))
	sbBuilder.WriteString("  </TABLE>>];\n")
	sbBuilder.WriteString("}\n")

	return sbBuilder.String(), nil
}
