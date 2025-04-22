package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

// AddJournalEntry añade una entrada al Journal en la partición
// AddJournalEntry añade una entrada al Journal en la partición
func AddJournalEntry(sb *structures.SuperBlock, diskPath, operation, path, content string) error {
	if sb.S_filesystem_type != 3 {
		return nil // Solo EXT3 soporta Journaling
	}

	// Abrir el disco
	file, err := os.OpenFile(diskPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Contar entradas válidas
	currentCount := int32(0)
	for i := int32(0); i < sb.S_journal_count; i++ {
		journalEntry := &structures.Journal{}
		offset := int64(sb.S_journal_start) + int64(i*int32(binary.Size(journalEntry)))
		err := journalEntry.Deserialize(diskPath, offset)
		if err != nil {
			return fmt.Errorf("error al deserializar entrada %d: %v", i, err)
		}
		// Considerar entrada válida si Count > 0 y Operation no está vacía
		if journalEntry.Count == 0 || strings.Trim(string(journalEntry.Content.Operation[:]), "\x00") == "" {
			break
		}
		currentCount = journalEntry.Count
	}

	// Verificar si hay espacio en el Journal
	if currentCount >= sb.S_journal_count {
		return fmt.Errorf("el Journal está lleno, no se pueden añadir más entradas")
	}

	// Truncar operation, path y content si son demasiado largos
	op := operation
	if len(op) > 10 {
		op = op[:10]
	}
	p := path
	if len(p) > 32 {
		p = p[:32]
	}
	c := content
	if len(c) > 64 {
		c = c[:64]
	}

	// Crear entrada del Journal
	journalEntry := &structures.Journal{
		Count: currentCount + 1,
		Content: structures.Information{
			Date: float32(time.Now().Unix()),
		},
	}
	copy(journalEntry.Content.Operation[:], op)
	copy(journalEntry.Content.Path[:], p)
	copy(journalEntry.Content.Content[:], c)

	// Calcular el offset para la nueva entrada
	offset := int64(sb.S_journal_start) + int64(currentCount*int32(binary.Size(journalEntry)))

	// Serializar la entrada
	err = journalEntry.Serialize(diskPath, offset)
	if err != nil {
		return fmt.Errorf("error al serializar entrada del Journal: %v", err)
	}

	return nil
}
