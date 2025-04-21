package commands

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	utils "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/utils"
)

type MKDISK struct {
	size int
	unit string
	fit  string
	path string
}

func ParseMkdisk(tokens []string) (string, error) {
	cmd := &MKDISK{}

	// Unir tokens en una sola cadena
	args := strings.Join(tokens, " ")

	// Expresión regular para encontrar cualquier parámetro
	re := regexp.MustCompile(`-\w+=(?:"[^"]+"|[^\s]+)`)
	matches := re.FindAllString(args, -1)

	// Conjunto de parámetros válidos
	validParams := map[string]bool{
		"-size": true,
		"-unit": true,
		"-fit":  true,
		"-path": true,
	}

	// Procesar cada parámetro encontrado
	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}
		key, value := strings.ToLower(kv[0]), kv[1]

		// Verificar si el parámetro es válido
		if !validParams[key] {
			return "", fmt.Errorf("error: parámetro no reconocido: %s", key)
		}

		// Quitar comillas del valor si están presentes
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		switch key {
		case "-size":
			size, err := strconv.Atoi(value)
			if err != nil || size <= 0 {
				return "", errors.New("el tamaño debe ser un número entero positivo")
			}
			cmd.size = size
		case "-unit":
			value = strings.ToUpper(value)
			if value != "K" && value != "M" {
				return "", errors.New("la unidad debe ser K o M")
			}
			cmd.unit = value
		case "-fit":
			value = strings.ToUpper(value)
			if value != "BF" && value != "FF" && value != "WF" {
				return "", errors.New("el ajuste debe ser BF, FF o WF")
			}
			cmd.fit = value
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		}
	}

	// Verificar parámetros obligatorios
	if cmd.size == 0 {
		return "", errors.New("faltan parámetros requeridos: -size")
	}
	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	// Establecer valores por defecto
	if cmd.unit == "" {
		cmd.unit = "M"
	}
	if cmd.fit == "" {
		cmd.fit = "FF"
	}

	// Ejecutar el comando solo si todas las validaciones pasan
	err := commandMkdisk(cmd)
	if err != nil {
		return "", fmt.Errorf("error: %v", err)
	}

	return fmt.Sprintf("MKDISK: Disco creado exitosamente en %s", cmd.path), nil
}

func commandMkdisk(mkdisk *MKDISK) error {
	// Convertir el tamaño a bytes
	sizeBytes, err := utils.ConvertToBytes(mkdisk.size, mkdisk.unit)
	if err != nil {
		return err
	}

	// Crear el disco
	err = createDisk(mkdisk, sizeBytes)
	if err != nil {
		return err
	}

	// Crear el MBR
	err = createMBR(mkdisk, sizeBytes)
	if err != nil {
		return err
	}

	return nil
}

func createDisk(mkdisk *MKDISK, sizeBytes int) error {
	// Crear las carpetas necesarias
	err := os.MkdirAll(filepath.Dir(mkdisk.path), os.ModePerm)
	if err != nil {
		return err
	}

	// Crear el archivo binario
	file, err := os.Create(mkdisk.path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Escribir ceros usando un buffer de 1 MB
	buffer := make([]byte, 1024*1024)
	for sizeBytes > 0 {
		writeSize := len(buffer)
		if sizeBytes < writeSize {
			writeSize = sizeBytes
		}
		if _, err := file.Write(buffer[:writeSize]); err != nil {
			return err
		}
		sizeBytes -= writeSize
	}
	return nil
}

func createMBR(mkdisk *MKDISK, sizeBytes int) error {
	var fitByte byte
	switch mkdisk.fit {
	case "FF":
		fitByte = 'F'
	case "BF":
		fitByte = 'B'
	case "WF":
		fitByte = 'W'
	default:
		return nil
	}

	mbr := &structures.MBR{
		Mbr_size:           int32(sizeBytes),
		Mbr_creation_date:  float32(time.Now().Unix()),
		Mbr_disk_signature: rand.Int31(),
		Mbr_disk_fit:       [1]byte{fitByte},
		Mbr_partitions: [4]structures.Partition{
			{Part_status: [1]byte{'N'}, Part_type: [1]byte{'N'}, Part_fit: [1]byte{'N'}, Part_start: -1, Part_size: -1, Part_name: [16]byte{'N'}, Part_correlative: -1, Part_id: [4]byte{'N'}},
			{Part_status: [1]byte{'N'}, Part_type: [1]byte{'N'}, Part_fit: [1]byte{'N'}, Part_start: -1, Part_size: -1, Part_name: [16]byte{'N'}, Part_correlative: -1, Part_id: [4]byte{'N'}},
			{Part_status: [1]byte{'N'}, Part_type: [1]byte{'N'}, Part_fit: [1]byte{'N'}, Part_start: -1, Part_size: -1, Part_name: [16]byte{'N'}, Part_correlative: -1, Part_id: [4]byte{'N'}},
			{Part_status: [1]byte{'N'}, Part_type: [1]byte{'N'}, Part_fit: [1]byte{'N'}, Part_start: -1, Part_size: -1, Part_name: [16]byte{'N'}, Part_correlative: -1, Part_id: [4]byte{'N'}},
		},
	}

	err := mbr.Serialize(mkdisk.path)
	if err != nil {
		return err
	}

	return nil
}
