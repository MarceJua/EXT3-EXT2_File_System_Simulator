package reports

import (
	"fmt"
	"strings"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

func ReportDisk(mbr *structures.MBR, diskPath string) (string, error) {
	var sb strings.Builder
	sb.WriteString("digraph G {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=record];\n")
	sb.WriteString("  disk [label=\"{MBR|")

	totalSize := float64(mbr.Mbr_size)
	start := int32(0)
	for _, part := range mbr.Mbr_partitions {
		if part.Part_size <= 0 || part.Part_status[0] == 'N' {
			continue
		}
		percent := (float64(part.Part_size) / totalSize) * 100
		if part.Part_start > start {
			freePercent := (float64(part.Part_start-start) / totalSize) * 100
			sb.WriteString(fmt.Sprintf("Libre %.1f%%|", freePercent))
		}
		partType := "Primaria"
		if part.Part_type[0] == 'E' {
			partType = "Extendida"
		}
		sb.WriteString(fmt.Sprintf("%s %s %.1f%%|", strings.Trim(string(part.Part_name[:]), "\x00"), partType, percent))
		start = part.Part_start + part.Part_size
	}
	if start < mbr.Mbr_size {
		freePercent := (float64(mbr.Mbr_size-start) / totalSize) * 100
		sb.WriteString(fmt.Sprintf("Libre %.1f%%|", freePercent))
	}
	sb.WriteString("}\"];\n")
	sb.WriteString("}\n")
	return sb.String(), nil
}
