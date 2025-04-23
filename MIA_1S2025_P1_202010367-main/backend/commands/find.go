package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

// FIND estructura que representa el comando find
type FIND struct {
	path string // Ruta de la carpeta donde inicia la búsqueda
	name string // Patrón de nombre a buscar
}

// ParseFind parsea los tokens del comando find
func ParseFind(tokens []string) (string, error) {
	cmd := &FIND{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := strings.Trim(parts[1], "\"")

		switch key {
		case "-path":
			if value == "" {
				return "", errors.New("la ruta no puede estar vacía")
			}
			cmd.path = value
		case "-name":
			if value == "" {
				return "", errors.New("el nombre no puede estar vacío")
			}
			cmd.name = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.path == "" || cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -path, -name")
	}

	result, err := commandFind(cmd)
	if err != nil {
		return "", fmt.Errorf("error al ejecutar find: %v", err)
	}

	return result, nil
}

// commandFind implementa la lógica del comando find
func commandFind(find *FIND) (string, error) {
	// Verificar sesión activa
	if stores.CurrentSession.ID == "" {
		return "", errors.New("no hay sesión activa, inicie sesión primero")
	}

	// Obtener superbloque y ruta del disco
	partitionSuperblock, _, partitionPath, err := stores.GetMountedPartitionSuperblock(stores.CurrentSession.ID)
	if err != nil {
		return "", fmt.Errorf("error al obtener la partición montada: %v", err)
	}

	// Abrir archivo del disco
	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0644)
	if err != nil {
		return "", fmt.Errorf("error al abrir disco %s: %v", partitionPath, err)
	}
	defer file.Close()

	// Encontrar el inodo de la ruta inicial
	inodeNum, err := findInodeByPath(partitionPath, partitionSuperblock, find.path)
	if err != nil {
		return "", fmt.Errorf("error al localizar la ruta %s: %v", find.path, err)
	}

	// Verificar que sea una carpeta
	inode := &structures.Inode{}
	err = inode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+inodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return "", fmt.Errorf("error al leer inodo %d: %v", inodeNum, err)
	}
	if inode.I_type[0] != '0' {
		return "", fmt.Errorf("la ruta %s no es una carpeta", find.path)
	}

	// Verificar permisos de lectura
	if !hasReadPermission(inode, stores.CurrentSession.UID, stores.CurrentSession.GID) {
		return "", fmt.Errorf("permiso denegado: no tiene permisos de lectura en %s", find.path)
	}

	// Realizar búsqueda recursiva
	matches := []string{}
	err = searchFiles(partitionPath, partitionSuperblock, inodeNum, find.path, find.name, &matches)
	if err != nil {
		return "", fmt.Errorf("error durante la búsqueda: %v", err)
	}

	// Registrar en el Journal (si es EXT3)
	if partitionSuperblock.S_filesystem_type == 3 {
		err = AddJournalEntry(partitionSuperblock, partitionPath, "find", find.path, find.name)
		if err != nil {
			return "", fmt.Errorf("error al registrar en el Journal: %v", err)
		}
	}

	// Formatear resultado
	if len(matches) == 0 {
		return "FIND: No se encontraron archivos o carpetas", nil
	}
	return fmt.Sprintf("FIND:\n%s", strings.Join(matches, "\n")), nil
}

// findInodeByPath encuentra el inodo correspondiente a una ruta
func findInodeByPath(diskPath string, sb *structures.SuperBlock, searchPath string) (int32, error) {
	// Normalizar ruta
	searchPath = strings.Trim(searchPath, "/")
	if searchPath == "" {
		return 0, nil // Raíz
	}

	// Comenzar desde el inodo raíz
	currentInodeNum := int32(0)
	components := strings.Split(searchPath, "/")

	for _, component := range components {
		if component == "" {
			continue
		}
		inode := &structures.Inode{}
		err := inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInodeNum*sb.S_inode_size))
		if err != nil {
			return -1, fmt.Errorf("error al leer inodo %d: %v", currentInodeNum, err)
		}
		if inode.I_type[0] != '0' {
			return -1, fmt.Errorf("el inodo %d no es una carpeta", currentInodeNum)
		}

		// Buscar la entrada en los bloques de la carpeta
		found := false
		for _, blockNum := range inode.I_block[:12] {
			if blockNum == -1 {
				break
			}
			folderBlock := &structures.FolderBlock{}
			err := folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return -1, fmt.Errorf("error al leer bloque %d: %v", blockNum, err)
			}
			for _, content := range folderBlock.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				if name == component && name != "." && name != ".." && name != "-" && content.B_inodo != -1 {
					currentInodeNum = content.B_inodo
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return -1, fmt.Errorf("componente %s no encontrado en la ruta", component)
		}
	}

	return currentInodeNum, nil
}

// hasReadPermission verifica si el usuario tiene permisos de lectura
func hasReadPermission(inode *structures.Inode, uid, gid string) bool {
	permStr := strings.Trim(string(inode.I_perm[:]), "\x00")
	if len(permStr) != 3 {
		return false
	}
	uPerm := int(permStr[0] - '0')
	gPerm := int(permStr[1] - '0')
	oPerm := int(permStr[2] - '0')

	inodeUID := fmt.Sprintf("%d", inode.I_uid)
	inodeGID := fmt.Sprintf("%d", inode.I_gid)

	// Usuario propietario
	if uid == inodeUID {
		return uPerm&4 != 0 // Bit de lectura (4)
	}
	// Mismo grupo
	if gid == inodeGID {
		return gPerm&4 != 0
	}
	// Otros
	return oPerm&4 != 0
}

// searchFiles realiza la búsqueda recursiva
func searchFiles(diskPath string, sb *structures.SuperBlock, inodeNum int32, currentPath, pattern string, matches *[]string) error {
	inode := &structures.Inode{}
	err := inode.Deserialize(diskPath, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo %d: %v", inodeNum, err)
	}

	// Verificar permisos de lectura
	if !hasReadPermission(inode, stores.CurrentSession.UID, stores.CurrentSession.GID) {
		return nil // Ignorar si no hay permisos
	}

	// Verificar si el nombre coincide con el patrón (excluir raíz)
	name := filepath.Base(currentPath)
	if currentPath == "/" {
		name = "" // Evitar que "/" coincida con el patrón
	}
	if name != "" && matchPattern(name, pattern) {
		*matches = append(*matches, currentPath)
	}

	// Si no es carpeta, no continuar
	if inode.I_type[0] != '0' {
		return nil
	}

	// Recorrer bloques de la carpeta
	for _, blockNum := range inode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		folderBlock := &structures.FolderBlock{}
		err := folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer bloque %d: %v", blockNum, err)
		}
		for _, content := range folderBlock.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == "." || name == ".." || name == "" || name == "-" || content.B_inodo == -1 {
				continue
			}
			childInodeNum := content.B_inodo
			childPath := currentPath
			if childPath == "/" {
				childPath = "/" + name
			} else {
				childPath += "/" + name
			}

			// Buscar recursivamente
			err = searchFiles(diskPath, sb, childInodeNum, childPath, pattern, matches)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// matchPattern verifica si un nombre coincide con el patrón usando expresiones regulares
func matchPattern(name, pattern string) bool {
	regexPattern := "^" + strings.ReplaceAll(strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\?", "."), "\\*", ".*") + "$"
	matched, err := regexp.MatchString(regexPattern, name)
	if err != nil {
		return false
	}
	return matched
}

/*
// matchPattern verifica si un nombre coincide con el patrón
func matchPattern(name, pattern string) bool {
	// Convertir patrón a expresión regular
	 regexPattern := "^" + strings.ReplaceAll(strings.ReplaceAll(pattern, "?", "."), "*", ".*") + "$"
	// Simplificación: usar strings.Contains para coincidencia básica
	// Para soporte completo, usar regexp.MatchString (requiere importar "regexp")
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		// Asumir que la implementación completa usará regexp
		// Por ahora, simular coincidencia básica
		if pattern == "*" {
			return true
		}
		if pattern == "?.*" && len(name) >= 2 && strings.Contains(name, ".") {
			return true
		}
		return strings.Contains(name, pattern) // Simplificado
	}
	return name == pattern
}*/
