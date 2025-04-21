package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
)

// RMDISK estructura que representa el comando rmdisk con sus parámetros
type RMDISK struct {
	path string
}

/*
   rmdisk -path="/home/marcelo-juarez/Desktop/MIA_1S2025_P1_202010367/disks/DiscoLab.mia"
*/

func ParseRmdisk(tokens []string) (string, error) {
	cmd := &RMDISK{}

	for _, token := range tokens {
		if strings.HasPrefix(token, "-path=") {
			parts := strings.SplitN(token, "=", 2)
			if len(parts) != 2 || parts[1] == "" {
				return "", errors.New("formato inválido para -path, debe ser -path=PATH")
			}
			cmd.path = strings.Trim(parts[1], "\"")
		} else {
			return "", fmt.Errorf("parámetro inválido: %s", token)
		}
	}

	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path es obligatorio")
	}

	err := commandRmdisk(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("RMDISK: Disco en %s eliminado correctamente", cmd.path), nil
}

func commandRmdisk(rmdisk *RMDISK) error {
	// Verificar si el disco existe
	if _, err := os.Stat(rmdisk.path); os.IsNotExist(err) {
		return fmt.Errorf("el disco en %s no existe", rmdisk.path)
	}

	// Verificar si alguna partición del disco está montada
	for id, mountedPath := range stores.MountedPartitions {
		if mountedPath == rmdisk.path {
			return fmt.Errorf("el disco en %s tiene una partición montada (ID: %s), desmonte primero", rmdisk.path, id)
		}
	}

	// Eliminar el archivo del disco
	err := os.Remove(rmdisk.path)
	if err != nil {
		return fmt.Errorf("error al eliminar el disco: %w", err)
	}

	return nil
}
