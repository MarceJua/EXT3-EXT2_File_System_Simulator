package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
	"github.com/MarceJua/MIA_1S2025_P1_202010367/backend/utils"
)

// CHOWN estructura que representa el comando chown
type CHOWN struct {
	path string // Ruta del archivo o carpeta
	user string // Nombre del nuevo usuario propietario
	r    bool   // Bandera para cambio recursivo
}

// ParseChown parsea los tokens del comando chown
func ParseChown(tokens []string) (string, error) {
	cmd := &CHOWN{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		key := strings.ToLower(parts[0])
		var value string
		if len(parts) == 2 {
			value = strings.Trim(parts[1], "\"")
		}

		switch key {
		case "-path":
			if value == "" {
				return "", errors.New("la ruta no puede estar vacía")
			}
			cmd.path = value
		case "-user":
			if value == "" {
				return "", errors.New("el nombre de usuario no puede estar vacío")
			}
			cmd.user = value
		case "-r":
			cmd.r = true
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.path == "" || cmd.user == "" {
		return "", errors.New("faltan parámetros requeridos: -path, -user")
	}

	result, err := commandChown(cmd)
	if err != nil {
		return "", fmt.Errorf("error al ejecutar chown: %v", err)
	}

	return result, nil
}

// commandChown implementa la lógica del comando chown
func commandChown(chown *CHOWN) (string, error) {
	// Verificar sesión activa
	if stores.CurrentSession.ID == "" {
		return "", errors.New("no hay sesión activa, inicie sesión primero")
	}

	// Verificar que el usuario es root
	if stores.CurrentSession.UID != "1" {
		return "", errors.New("permiso denegado: solo root puede ejecutar chown")
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

	// Obtener UID del usuario
	newUID, err := getUserUID(partitionPath, partitionSuperblock, chown.user)
	if err != nil {
		return "", fmt.Errorf("error al obtener UID de %s: %v", chown.user, err)
	}

	// Encontrar el inodo de la ruta
	inodeNum, err := findInodeByPath(partitionPath, partitionSuperblock, chown.path)
	if err != nil {
		return "", fmt.Errorf("error al localizar la ruta %s: %v", chown.path, err)
	}

	// Cambiar propietario
	err = changeOwner(partitionPath, partitionSuperblock, inodeNum, newUID, chown.r)
	if err != nil {
		return "", fmt.Errorf("error al cambiar propietario: %v", err)
	}

	// Registrar en el Journal (si es EXT3)
	if partitionSuperblock.S_filesystem_type == 3 {
		err = AddJournalEntry(partitionSuperblock, partitionPath, "chown", chown.path, chown.user)
		if err != nil {
			return "", fmt.Errorf("error al registrar en el Journal: %v", err)
		}
	}

	return "CHOWN: Propietario cambiado exitosamente", nil
}

// getUserUID obtiene el UID de un usuario desde /users.txt
func getUserUID(diskPath string, sb *structures.SuperBlock, username string) (int32, error) {
	// Suponer que /users.txt está en el inodo 1
	inodeNum := int32(1)
	inode := &structures.Inode{}
	err := inode.Deserialize(diskPath, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return -1, fmt.Errorf("error al leer inodo de users.txt: %v", err)
	}
	if inode.I_type[0] != '1' {
		return -1, errors.New("el inodo de users.txt no es un archivo")
	}

	// Leer contenido de users.txt
	var content bytes.Buffer
	for _, blockNum := range inode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		fileBlock := &structures.FileBlock{}
		err := fileBlock.Deserialize(diskPath, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return -1, fmt.Errorf("error al leer bloque %d: %v", blockNum, err)
		}
		data := strings.Trim(string(fileBlock.B_content[:]), "\x00")
		content.WriteString(data)
	}

	// Parsear users.txt
	lines := strings.Split(content.String(), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		if parts[1] == "U" && parts[3] == username {
			uid, err := utils.StringToInt(parts[0])
			if err != nil {
				return -1, fmt.Errorf("error al convertir UID %s: %v", parts[0], err)
			}
			return int32(uid), nil
		}
	}

	return -1, fmt.Errorf("usuario %s no encontrado en users.txt", username)
}

// changeOwner cambia el propietario de un inodo y, si es recursivo, de sus hijos
func changeOwner(diskPath string, sb *structures.SuperBlock, inodeNum int32, newUID int32, recursive bool) error {
	inode := &structures.Inode{}
	err := inode.Deserialize(diskPath, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer inodo %d: %v", inodeNum, err)
	}

	// Cambiar propietario
	inode.I_uid = newUID
	inode.I_mtime = float32(time.Now().Unix())
	err = inode.Serialize(diskPath, int64(sb.S_inode_start+inodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al serializar inodo %d: %v", inodeNum, err)
	}

	// Si no es carpeta o no es recursivo, terminar
	if inode.I_type[0] != '0' || !recursive {
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
			// Cambiar propietario recursivamente
			err = changeOwner(diskPath, sb, content.B_inodo, newUID, recursive)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
