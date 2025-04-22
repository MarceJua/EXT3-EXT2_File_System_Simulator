package structures

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// Information representa los detalles de una operación en el Journal.
type Information struct {
	Operation [10]byte // Operación (e.g., "mkdir", "mkfile")
	Path      [32]byte // Ruta de la operación
	Content   [64]byte // Contenido (si es un archivo)
	Date      float32  // Fecha de la operación (alineado con SuperBlock/Inode)
	// Total: 10 + 32 + 64 + 4 = 110 bytes
}

// Journal representa una entrada en la bitácora de EXT3.
type Journal struct {
	Count   int32       // Contador de la entrada
	Content Information // Información de la operación
	// Total: 4 + 110 = 114 bytes
}

// Serialize escribe la estructura Journal en un archivo binario en la posición especificada.
func (j *Journal) Serialize(path string, offset int64) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Seek(offset, 0); err != nil {
		return err
	}
	return binary.Write(file, binary.LittleEndian, j)
}

// Deserialize lee la estructura Journal desde un archivo binario en la posición especificada.
func (j *Journal) Deserialize(path string, offset int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Seek(offset, 0); err != nil {
		return err
	}

	jSize := binary.Size(j)
	if jSize <= 0 {
		return fmt.Errorf("invalid Journal size: %d", jSize)
	}

	buffer := make([]byte, jSize)
	if _, err := file.Read(buffer); err != nil {
		return err
	}

	reader := bytes.NewReader(buffer)
	return binary.Read(reader, binary.LittleEndian, j)
}
