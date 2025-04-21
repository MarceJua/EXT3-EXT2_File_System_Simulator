package reports

import (
	"fmt"
	"os"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

func ReportBMBlock(sb *structures.SuperBlock, diskPath string, outputPath string) error {
	file, err := os.Open(diskPath)
	if err != nil {
		return fmt.Errorf("error abriendo disco: %v", err)
	}
	defer file.Close()

	_, err = file.Seek(int64(sb.S_bm_block_start), 0)
	if err != nil {
		return fmt.Errorf("error buscando bitmap de bloques: %v", err)
	}

	buffer := make([]byte, sb.S_blocks_count)
	_, err = file.Read(buffer)
	if err != nil {
		return fmt.Errorf("error leyendo bitmap de bloques: %v", err)
	}

	// Escribir directamente en el archivo de salida
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creando archivo de salida: %v", err)
	}
	defer outFile.Close()

	for i, bit := range buffer {
		_, err = outFile.Write([]byte{bit})
		if err != nil {
			return fmt.Errorf("error escribiendo en archivo: %v", err)
		}
		if (i+1)%20 == 0 && i != len(buffer)-1 {
			_, err = outFile.WriteString("\n")
			if err != nil {
				return fmt.Errorf("error escribiendo salto de línea: %v", err)
			}
		}
	}
	if len(buffer)%20 != 0 {
		_, err = outFile.WriteString("\n")
		if err != nil {
			return fmt.Errorf("error escribiendo salto de línea final: %v", err)
		}
	}

	return nil
}
