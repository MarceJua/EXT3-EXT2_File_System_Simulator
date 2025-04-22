package commands

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

// JOURNALREPORT representa el comando temporal para inspeccionar el Journal
type JOURNALREPORT struct {
	id string // ID de la partición montada
}

// ParseJournalReport parsea los tokens del comando journal_report
func ParseJournalReport(tokens []string) (string, error) {
	cmd := &JOURNALREPORT{}

	// Procesar cada token
	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		key := strings.ToLower(parts[0])

		switch key {
		case "-id":
			if len(parts) != 2 {
				return "", fmt.Errorf("formato inválido para -id: %s", token)
			}
			cmd.id = parts[1]
			if cmd.id == "" {
				return "", fmt.Errorf("el ID no puede estar vacío")
			}
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Validar parámetro requerido
	if cmd.id == "" {
		return "", fmt.Errorf("faltan parámetros requeridos: -id")
	}

	// Ejecutar el comando
	output, err := commandJournalReport(cmd)
	if err != nil {
		return "", fmt.Errorf("error al generar reporte del Journal: %v", err)
	}

	return output, nil
}

// commandJournalReport implementa la lógica del comando journal_report
func commandJournalReport(cmd *JOURNALREPORT) (string, error) {
	// Obtener la partición montada
	sb, _, diskPath, err := stores.GetMountedPartitionSuperblock(cmd.id)
	if err != nil {
		return "", fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	// Verificar si es EXT3
	if sb.S_filesystem_type != 3 {
		return "", fmt.Errorf("la partición %s no soporta Journaling (no es EXT3)", cmd.id)
	}

	// Leer y mostrar las entradas del Journal
	output := "Reporte del Journal:\n"
	found := false
	for i := int32(0); i < sb.S_journal_count; i++ {
		journalEntry := &structures.Journal{}
		offset := int64(sb.S_journal_start) + int64(i*int32(binary.Size(journalEntry)))
		err := journalEntry.Deserialize(diskPath, offset)
		if err != nil {
			return output, fmt.Errorf("error al deserializar entrada %d: %v", i, err)
		}
		// Detenerse si la entrada es inválida (Count == 0 o Operation vacía)
		if journalEntry.Count == 0 || strings.Trim(string(journalEntry.Content.Operation[:]), "\x00") == "" {
			break
		}
		found = true
		output += fmt.Sprintf("Entrada %d:\n", journalEntry.Count)
		output += fmt.Sprintf("  Operación: %s\n", strings.Trim(string(journalEntry.Content.Operation[:]), "\x00"))
		output += fmt.Sprintf("  Ruta: %s\n", strings.Trim(string(journalEntry.Content.Path[:]), "\x00"))
		output += fmt.Sprintf("  Contenido: %s\n", strings.Trim(string(journalEntry.Content.Content[:]), "\x00"))
		output += fmt.Sprintf("  Fecha: %v\n", time.Unix(int64(journalEntry.Content.Date), 0))
	}

	if !found {
		output += "No se encontraron entradas válidas en el Journal.\n"
	}

	return output, nil
}
