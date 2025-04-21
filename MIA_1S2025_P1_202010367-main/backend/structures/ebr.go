package structures

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type EBR struct {
	Part_status [1]byte  // Estado: 'N' (no usada), '0' (creada), '1' (montada)
	Part_fit    [1]byte  // Ajuste: 'B', 'F', 'W'
	Part_start  int32    // Byte de inicio
	Part_size   int32    // Tamaño en bytes
	Part_next   int32    // Byte de inicio del siguiente EBR, o -1 si no hay más
	Part_name   [16]byte // Nombre de la partición
	Part_id     [4]byte  // ID de la partición (nuevo campo)
}

// Serialize escribe el EBR en el archivo en la posición especificada
func (ebr *EBR) Serialize(file *os.File, offset int64) error {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return err
	}
	return binary.Write(file, binary.LittleEndian, ebr)
}

// Deserialize lee un EBR desde el archivo en la posición especificada
// Deserialize lee la estructura EBR desde un archivo binario en la posición especificada
func (ebr *EBR) Deserialize(file *os.File, offset int64) error {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return err
	}
	ebrSize := binary.Size(ebr)
	if ebrSize <= 0 {
		return fmt.Errorf("invalid EBR size: %d", ebrSize)
	}
	buffer := make([]byte, ebrSize)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buffer)
	return binary.Read(reader, binary.LittleEndian, ebr)
}

// PrintEBR imprime los valores del EBR para depuración
func (ebr *EBR) Print() {
	fmt.Printf("Part_status: %c\n", ebr.Part_status[0])
	fmt.Printf("Part_fit: %c\n", ebr.Part_fit[0])
	fmt.Printf("Part_start: %d\n", ebr.Part_start)
	fmt.Printf("Part_size: %d\n", ebr.Part_size)
	fmt.Printf("Part_next: %d\n", ebr.Part_next)
	fmt.Printf("Part_name: %s\n", string(ebr.Part_name[:]))
	fmt.Printf("Part_id: %s\n", string(ebr.Part_id[:]))
}
