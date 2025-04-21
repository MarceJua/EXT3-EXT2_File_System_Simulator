package reports

import (
	"fmt"
	"os"
	"strings"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

func ReportEBR(mbr *structures.MBR, diskPath string) (string, error) {
	// Buscar la partición extendida en el MBR proporcionado
	var extendedPartition *structures.Partition
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_type[0] == 'E' && mbr.Mbr_partitions[i].Part_status[0] != 'N' {
			extendedPartition = &mbr.Mbr_partitions[i]
			break
		}
	}
	if extendedPartition == nil {
		return "", fmt.Errorf("no se encontró una partición extendida en %s", diskPath)
	}

	// Abrir el archivo para leer los EBRs
	file, err := os.Open(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Iniciar el grafo DOT
	var sbBuilder strings.Builder
	sbBuilder.WriteString("digraph G {\n")
	sbBuilder.WriteString("  node [shape=plaintext]\n")
	sbBuilder.WriteString("  rankdir=LR;\n") // Ordenar de izquierda a derecha

	// Recorrer la cadena de EBRs
	currentOffset := int64(extendedPartition.Part_start)
	ebrCount := 0
	for currentOffset != -1 {
		ebr := &structures.EBR{}
		err = ebr.Deserialize(file, currentOffset)
		if err != nil {
			return "", fmt.Errorf("error deserializando EBR en offset %d: %v", currentOffset, err)
		}

		// Si el EBR está en uso (status '0' o '1'), incluirlo en el reporte
		if ebr.Part_status[0] != 'N' {
			sbBuilder.WriteString(fmt.Sprintf("  ebr%d [label=<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n", ebrCount))
			sbBuilder.WriteString(fmt.Sprintf("    <TR><TD COLSPAN=\"2\">EBR %d</TD></TR>\n", ebrCount))
			sbBuilder.WriteString("    <TR><TD>part_status</TD><TD>")
			sbBuilder.WriteString(string(ebr.Part_status[:]))
			sbBuilder.WriteString("</TD></TR>\n")
			sbBuilder.WriteString("    <TR><TD>part_fit</TD><TD>")
			sbBuilder.WriteString(string(ebr.Part_fit[:]))
			sbBuilder.WriteString("</TD></TR>\n")
			sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>part_start</TD><TD>%d</TD></TR>\n", ebr.Part_start))
			sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>part_size</TD><TD>%d</TD></TR>\n", ebr.Part_size))
			sbBuilder.WriteString(fmt.Sprintf("    <TR><TD>part_next</TD><TD>%d</TD></TR>\n", ebr.Part_next))
			sbBuilder.WriteString("    <TR><TD>part_name</TD><TD>")
			sbBuilder.WriteString(strings.TrimRight(string(ebr.Part_name[:]), "\x00"))
			sbBuilder.WriteString("</TD></TR>\n")
			sbBuilder.WriteString("    <TR><TD>part_id</TD><TD>")
			sbBuilder.WriteString(strings.TrimRight(string(ebr.Part_id[:]), "\x00"))
			sbBuilder.WriteString("</TD></TR>\n")
			sbBuilder.WriteString("  </TABLE>>];\n")

			// Conectar con el siguiente EBR si existe
			if ebr.Part_next != -1 {
				sbBuilder.WriteString(fmt.Sprintf("  ebr%d -> ebr%d;\n", ebrCount, ebrCount+1))
			}
			ebrCount++
		}

		// Avanzar al siguiente EBR
		if ebr.Part_next == -1 {
			break
		}
		currentOffset = int64(ebr.Part_next)
	}

	if ebrCount == 0 {
		sbBuilder.WriteString("  node0 [label=\"No hay particiones lógicas\"];\n")
	}

	sbBuilder.WriteString("}\n")
	return sbBuilder.String(), nil
}
