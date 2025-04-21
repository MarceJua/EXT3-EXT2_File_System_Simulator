package reports

import (
	"fmt"
	"strings"
	"time"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	//utils "github.com/MarceJua/MIA_1S2025_P1_202010367/utils"
)

// ReportMBR genera un reporte del MBR y lo guarda en la ruta especificada
func ReportMBR(mbr *structures.MBR) (string, error) {
	var sb strings.Builder
	sb.WriteString("digraph G {\n")
	sb.WriteString("  node [shape=plaintext]\n")
	sb.WriteString("  tbl [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
	sb.WriteString("    <TR><TD COLSPAN=\"2\">REPORTE MBR</TD></TR>\n")
	sb.WriteString(fmt.Sprintf("    <TR><TD>mbr_tamano</TD><TD>%d</TD></TR>\n", mbr.Mbr_size))
	sb.WriteString(fmt.Sprintf("    <TR><TD>mrb_fecha_creacion</TD><TD>%s</TD></TR>\n", time.Unix(int64(mbr.Mbr_creation_date), 0).Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("    <TR><TD>mbr_disk_signature</TD><TD>%d</TD></TR>\n", mbr.Mbr_disk_signature))

	for i, part := range mbr.Mbr_partitions {
		if part.Part_size <= 0 || part.Part_status[0] == 'N' {
			continue // Omitir particiones no asignadas
		}
		partName := strings.TrimRight(string(part.Part_name[:]), "\x00")
		sb.WriteString(fmt.Sprintf("    <TR><TD COLSPAN=\"2\">PARTICIÓN %d</TD></TR>\n", i+1))
		sb.WriteString(fmt.Sprintf("    <TR><TD>part_status</TD><TD>%c</TD></TR>\n", part.Part_status[0]))
		sb.WriteString(fmt.Sprintf("    <TR><TD>part_type</TD><TD>%c</TD></TR>\n", part.Part_type[0]))
		sb.WriteString(fmt.Sprintf("    <TR><TD>part_fit</TD><TD>%c</TD></TR>\n", part.Part_fit[0]))
		sb.WriteString(fmt.Sprintf("    <TR><TD>part_start</TD><TD>%d</TD></TR>\n", part.Part_start))
		sb.WriteString(fmt.Sprintf("    <TR><TD>part_size</TD><TD>%d</TD></TR>\n", part.Part_size))
		sb.WriteString(fmt.Sprintf("    <TR><TD>part_name</TD><TD>%s</TD></TR>\n", partName))
	}
	sb.WriteString("  </TABLE>>];\n")
	sb.WriteString("}\n")
	return sb.String(), nil
}
