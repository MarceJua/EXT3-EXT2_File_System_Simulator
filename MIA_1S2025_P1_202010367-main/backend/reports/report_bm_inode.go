package reports

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

// ReportBMInode genera un reporte del bitmap de inodos en formato texto
func ReportBMInode(sb *structures.SuperBlock, diskPath, outputPath string) error {
	if sb == nil {
		return fmt.Errorf("superbloque no proporcionado")
	}

	// Crear directorios padre si no existen
	err := os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return fmt.Errorf("error creando directorios padre: %v", err)
	}

	file, err := os.Open(diskPath)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer file.Close()

	totalInodes := sb.S_inodes_count // Solo S_inodes_count, no sumamos S_free_inodes_count

	var bitmapContent strings.Builder
	for i := int32(0); i < totalInodes; i++ {
		_, err := file.Seek(int64(sb.S_bm_inode_start)+int64(i), 0)
		if err != nil {
			return fmt.Errorf("error al establecer el puntero en el archivo: %v", err)
		}

		char := make([]byte, 1)
		_, err = file.Read(char)
		if err != nil {
			return fmt.Errorf("error al leer el byte del archivo: %v", err)
		}

		if char[0] != '0' && char[0] != '1' {
			return fmt.Errorf("carácter inválido en bitmap: %c (posición %d)", char[0], i)
		}

		bitmapContent.WriteByte(char[0])
		if (i+1)%20 == 0 && i != totalInodes-1 {
			bitmapContent.WriteString("\n")
		}
	}
	if totalInodes%20 != 0 {
		bitmapContent.WriteString("\n")
	}

	// Crear y escribir el archivo .txt
	txtFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error al crear el archivo TXT: %v", err)
	}
	defer txtFile.Close()

	_, err = txtFile.WriteString(bitmapContent.String())
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo TXT: %v", err)
	}

	fmt.Println("Archivo del bitmap de inodos generado:", outputPath)
	return nil
}
