package commands

import (
	"errors"
	"fmt"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	utils "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/utils"
)

// MKDIR estructura que representa el comando mkdir con sus parámetros
type MKDIR struct {
	path string // Path del directorio
	p    bool   // Opción -p (crea directorios padres si no existen)
}

/*
   mkdir -p -path=/home/user/docs/usac
   mkdir -path="/home/mis documentos/archivos clases"
*/

func ParseMkdir(tokens []string) (string, error) {
	cmd := &MKDIR{}

	// Procesar cada token
	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		key := strings.ToLower(parts[0])

		switch key {
		case "-path":
			if len(parts) != 2 {
				return "", fmt.Errorf("formato inválido para -path: %s", token)
			}
			value := parts[1]
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-p":
			if len(parts) != 1 {
				return "", fmt.Errorf("formato inválido para -p: %s", token)
			}
			cmd.p = true
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Validar parámetro requerido
	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	// Ejecutar el comando
	err := commandMkdir(cmd)
	if err != nil {
		return "", fmt.Errorf("error al crear el directorio: %v", err)
	}

	return fmt.Sprintf("MKDIR: Directorio %s creado correctamente", cmd.path), nil
}

// commandMkdir implementa la lógica para crear el directorio
func commandMkdir(mkdir *MKDIR) error {
	// Verificar si hay una sesión activa
	if stores.CurrentSession.ID == "" {
		return errors.New("debe iniciar sesión primero")
	}

	// Obtener la partición montada usando el ID de la sesión
	partitionSuperblock, mountedPartition, partitionPath, err := stores.GetMountedPartitionSuperblock(stores.CurrentSession.ID)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	// Crear el directorio
	err = createDirectory(mkdir.path, partitionSuperblock, partitionPath, mountedPartition)
	if err != nil {
		return err
	}

	return nil
}

// createDirectory crea el directorio en la partición
func createDirectory(dirPath string, sb *structures.SuperBlock, partitionPath string, mountedPartition *structures.Partition) error {
	parentDirs, destDir := utils.GetParentDirectories(dirPath)

	// Crear el directorio según el path proporcionado
	err := sb.CreateFolder(partitionPath, parentDirs, destDir)
	if err != nil {
		return fmt.Errorf("error al crear el directorio: %w", err)
	}

	// Serializar el superbloque
	err = sb.Serialize(partitionPath, int64(mountedPartition.Part_start))
	if err != nil {
		return fmt.Errorf("error al serializar el superbloque: %w", err)
	}

	return nil
}
