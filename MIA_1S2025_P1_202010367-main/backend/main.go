package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	analyzer "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/analyzer"
	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type CommandRequest struct {
	Command string `json:"command"`
}

type LoginRequest struct {
	User string `json:"user"`
	Pass string `json:"pass"`
	ID   string `json:"id"`
}

type CommandResponse struct {
	Output string `json:"output"`
}

type Disk struct {
	Name              string   `json:"name"`
	Path              string   `json:"path"`
	SizeMB            float64  `json:"sizeMB"`
	Fit               string   `json:"fit"`
	MountedPartitions []string `json:"mountedPartitions"`
}

type Partition struct {
	ID     string  `json:"id"`
	Path   string  `json:"path"`
	Name   string  `json:"name"`
	SizeKB float64 `json:"sizeKB"`
	Fit    string  `json:"fit"`
	Status string  `json:"status"`
}

type FileSystemEntry struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`     // "folder" o "file"
	Size     int32   `json:"size"`     // Tamaño en bytes (0 para carpetas)
	Content  string  `json:"content"`  // Contenido (solo para archivos, "" para carpetas)
	Perm     string  `json:"perm"`     // Permisos (ej. "664")
	UID      int32   `json:"uid"`      // User ID
	GID      int32   `json:"gid"`      // Group ID
	Created  float32 `json:"created"`  // Fecha de creación (I_ctime)
	Modified float32 `json:"modified"` // Fecha de modificación (I_mtime)
}

type FileSystemResponse struct {
	Entries []FileSystemEntry `json:"entries"`
}

type DisksResponse struct {
	Disks []Disk `json:"disks"`
}

type PartitionsResponse struct {
	Partitions []Partition `json:"partitions"`
}

// Comandos que no requieren sesión activa
var noSessionCommands = map[string]bool{
	"mkdisk":  true,
	"rmdisk":  true,
	"fdisk":   true,
	"mount":   true,
	"unmount": true,
	"mounted": true,
	"mkfs":    true,
}

func main() {
	app := fiber.New()

	app.Use(cors.New(cors.Config{}))

	app.Post("/execute", func(c *fiber.Ctx) error {
		var req CommandRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(CommandResponse{
				Output: "Error: Petición inválida",
			})
		}

		commands := strings.Split(req.Command, "\n")
		output := ""

		for _, cmd := range commands {
			if strings.TrimSpace(cmd) == "" {
				continue
			}

			cmdParts := strings.Fields(cmd)
			if len(cmdParts) == 0 {
				output += "Error: Comando vacío\n"
				continue
			}
			commandName := strings.ToLower(cmdParts[0])

			if !noSessionCommands[commandName] && stores.CurrentSession.ID == "" {
				output += "Error: Inicie sesión para ejecutar este comando\n"
				continue
			}

			result, err := analyzer.Analyzer(cmd)
			if err != nil {
				output += fmt.Sprintf("Error: %s\n", err.Error())
			} else {
				output += fmt.Sprintf("%s\n", result)
			}
		}

		if output == "" {
			output = "No se ejecutó ningún comando"
		}

		return c.JSON(CommandResponse{
			Output: output,
		})
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(CommandResponse{
				Output: "Error: Petición inválida",
			})
		}

		if req.User == "" || req.Pass == "" || req.ID == "" {
			return c.Status(400).JSON(CommandResponse{
				Output: "Error: Todos los campos son obligatorios",
			})
		}

		command := fmt.Sprintf("login -user=%s -pass=%s -id=%s", req.User, req.Pass, req.ID)
		result, err := analyzer.Analyzer(command)
		if err != nil {
			return c.Status(400).JSON(CommandResponse{
				Output: fmt.Sprintf("Error: %s", err.Error()),
			})
		}

		return c.JSON(CommandResponse{
			Output: result,
		})
	})

	app.Get("/disks", func(c *fiber.Ctx) error {
		diskDir := "/home/marcelo-juarez/Calificacion_MIA/Discos/"
		var disks []Disk

		err := filepath.Walk(diskDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mia") {
				// Leer el MBR para obtener el tamaño del disco
				var mbr structures.MBR
				if err := mbr.Deserialize(path); err != nil {
					return err
				}

				// Tamaño del disco en MB
				sizeMB := float64(mbr.Mbr_size) / (1024 * 1024)

				// Buscar particiones montadas asociadas a este disco
				var mounted []string
				var fit string
				for id := range stores.MountedPartitions {
					mbr, _, _, err := stores.GetMountedPartitionRep(id)
					if err != nil {
						continue
					}
					// Verificar particiones primarias
					for _, p := range mbr.Mbr_partitions {
						if strings.Trim(string(p.Part_id[:]), "\x00") == id && stores.MountedPartitions[id] == path {
							mounted = append(mounted, id)
							if fit == "" {
								fitChar := string(p.Part_fit[0])
								if fitChar == "B" {
									fit = "BestFit"
								} else if fitChar == "F" {
									fit = "FirstFit"
								} else if fitChar == "W" {
									fit = "WorstFit"
								} else {
									fit = "Unknown"
								}
							}
						}
					}
					// Verificar particiones lógicas
					file, err := os.OpenFile(path, os.O_RDWR, 0644)
					if err != nil {
						continue
					}
					defer file.Close()

					var extPartition *structures.Partition
					for _, p := range mbr.Mbr_partitions {
						if p.Part_type[0] == 'E' && p.Part_status[0] != 'N' {
							extPartition = &p
							break
						}
					}
					if extPartition == nil {
						continue
					}

					var currentEBR structures.EBR
					currentOffset := int64(extPartition.Part_start)
					fileInfo, err := file.Stat()
					if err != nil {
						continue
					}
					fileSize := fileInfo.Size()

					for currentOffset < fileSize {
						if err := currentEBR.Deserialize(file, currentOffset); err != nil {
							break
						}
						if strings.Trim(string(currentEBR.Part_id[:]), "\x00") == id && stores.MountedPartitions[id] == path {
							mounted = append(mounted, id)
							if fit == "" {
								fitChar := string(currentEBR.Part_fit[0])
								if fitChar == "B" {
									fit = "BestFit"
								} else if fitChar == "F" {
									fit = "FirstFit"
								} else if fitChar == "W" {
									fit = "WorstFit"
								} else {
									fit = "Unknown"
								}
							}
						}
						if currentEBR.Part_next == -1 {
							break
						}
						currentOffset = int64(currentEBR.Part_next)
					}
				}
				if fit == "" {
					fit = "Unknown"
				}

				disks = append(disks, Disk{
					Name:              info.Name(),
					Path:              path,
					SizeMB:            sizeMB,
					Fit:               fit,
					MountedPartitions: mounted,
				})
			}
			return nil
		})

		if err != nil {
			return c.Status(500).JSON(CommandResponse{
				Output: fmt.Sprintf("Error al leer discos: %s", err.Error()),
			})
		}

		// Log para depuración
		fmt.Printf("Respuesta de /disks: %+v\n", DisksResponse{Disks: disks})

		return c.JSON(DisksResponse{
			Disks: disks,
		})
	})

	app.Get("/partitions", func(c *fiber.Ctx) error {
		diskPath := c.Query("diskPath")
		if diskPath == "" {
			return c.Status(400).JSON(CommandResponse{
				Output: "Error: diskPath es requerido",
			})
		}

		var partitions []Partition
		for id := range stores.MountedPartitions {
			if stores.MountedPartitions[id] != diskPath {
				continue
			}

			mbr, _, _, err := stores.GetMountedPartitionRep(id)
			if err != nil {
				continue
			}

			// Buscar partición primaria
			for _, p := range mbr.Mbr_partitions {
				if strings.Trim(string(p.Part_id[:]), "\x00") == id {
					fit := string(p.Part_fit[0])
					fitText := "Unknown"
					if fit == "B" {
						fitText = "BestFit"
					} else if fit == "F" {
						fitText = "FirstFit"
					} else if fit == "W" {
						fitText = "WorstFit"
					}

					status := string(p.Part_status[0])
					statusText := "Inactive"
					if status == "1" {
						statusText = "Active"
					}

					partitions = append(partitions, Partition{
						ID:     id,
						Path:   diskPath,
						Name:   strings.Trim(string(p.Part_name[:]), "\x00"),
						SizeKB: float64(p.Part_size) / 1024,
						Fit:    fitText,
						Status: statusText,
					})
					break
				}
			}

			// Buscar partición lógica
			file, err := os.OpenFile(diskPath, os.O_RDWR, 0644)
			if err != nil {
				continue
			}
			defer file.Close()

			var extPartition *structures.Partition
			for _, p := range mbr.Mbr_partitions {
				if p.Part_type[0] == 'E' && p.Part_status[0] != 'N' {
					extPartition = &p
					break
				}
			}
			if extPartition == nil {
				continue
			}

			var currentEBR structures.EBR
			currentOffset := int64(extPartition.Part_start)
			fileInfo, err := file.Stat()
			if err != nil {
				continue
			}
			fileSize := fileInfo.Size()

			for currentOffset < fileSize {
				if err := currentEBR.Deserialize(file, currentOffset); err != nil {
					break
				}
				if strings.Trim(string(currentEBR.Part_id[:]), "\x00") == id {
					fit := string(currentEBR.Part_fit[0])
					fitText := "Unknown"
					if fit == "B" {
						fitText = "BestFit"
					} else if fit == "F" {
						fitText = "FirstFit"
					} else if fit == "W" {
						fitText = "WorstFit"
					}

					status := string(currentEBR.Part_status[0])
					statusText := "Inactive"
					if status == "1" {
						statusText = "Active"
					}

					partitions = append(partitions, Partition{
						ID:     id,
						Path:   diskPath,
						Name:   strings.Trim(string(currentEBR.Part_name[:]), "\x00"),
						SizeKB: float64(currentEBR.Part_size) / 1024,
						Fit:    fitText,
						Status: statusText,
					})
					break
				}
				if currentEBR.Part_next == -1 {
					break
				}
				currentOffset = int64(currentEBR.Part_next)
			}
		}

		return c.JSON(PartitionsResponse{
			Partitions: partitions,
		})
	})

	app.Get("/filesystem", func(c *fiber.Ctx) error {
		partitionID := c.Query("id")
		path := c.Query("path")

		if partitionID == "" {
			return c.Status(400).JSON(CommandResponse{
				Output: "Error: id es requerido",
			})
		}

		// Obtener el superbloque y la partición montada
		sb, _, diskPath, err := stores.GetMountedPartitionSuperblock(partitionID)
		if err != nil {
			return c.Status(400).JSON(CommandResponse{
				Output: fmt.Sprintf("Error al obtener la partición montada: %s", err.Error()),
			})
		}

		// Si no se especifica un path, empezamos desde la raíz (/)
		if path == "" {
			path = "/"
		}

		// Navegar al directorio especificado
		var entries []FileSystemEntry
		currentInode := int32(0) // Comenzar desde el inodo raíz
		if path != "/" {
			// Dividir el path en directorios (ignorar la raíz inicial "/")
			dirs := strings.Split(strings.Trim(path, "/"), "/")
			for _, dir := range dirs {
				inode := &structures.Inode{}
				err := inode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
				if err != nil {
					return c.Status(500).JSON(CommandResponse{
						Output: fmt.Sprintf("Error al leer inodo %d: %s", currentInode, err.Error()),
					})
				}
				if inode.I_type[0] != '0' { // Debe ser una carpeta
					return c.Status(400).JSON(CommandResponse{
						Output: fmt.Sprintf("El path %s no es un directorio", path),
					})
				}

				// Buscar la entrada del directorio actual
				found := false
				for _, blockIndex := range inode.I_block {
					if blockIndex == -1 {
						break
					}
					folderBlock := &structures.FolderBlock{}
					err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
					if err != nil {
						return c.Status(500).JSON(CommandResponse{
							Output: fmt.Sprintf("Error al leer bloque %d: %s", blockIndex, err.Error()),
						})
					}
					for _, content := range folderBlock.B_content {
						name := strings.Trim(string(content.B_name[:]), "\x00")
						if name == dir && content.B_inodo != -1 {
							currentInode = content.B_inodo
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					return c.Status(400).JSON(CommandResponse{
						Output: fmt.Sprintf("Directorio %s no encontrado en el path %s", dir, path),
					})
				}
			}
		}

		// Leer el inodo del directorio actual
		dirInode := &structures.Inode{}
		err = dirInode.Deserialize(diskPath, int64(sb.S_inode_start+currentInode*sb.S_inode_size))
		if err != nil {
			return c.Status(500).JSON(CommandResponse{
				Output: fmt.Sprintf("Error al leer inodo del directorio %d: %s", currentInode, err.Error()),
			})
		}
		if dirInode.I_type[0] != '0' {
			return c.Status(400).JSON(CommandResponse{
				Output: fmt.Sprintf("El path %s no es un directorio", path),
			})
		}

		// Leer los bloques del directorio y listar las entradas
		for _, blockIndex := range dirInode.I_block {
			if blockIndex == -1 {
				break
			}
			folderBlock := &structures.FolderBlock{}
			err = folderBlock.Deserialize(diskPath, int64(sb.S_block_start+blockIndex*sb.S_block_size))
			if err != nil {
				return c.Status(500).JSON(CommandResponse{
					Output: fmt.Sprintf("Error al leer bloque %d: %s", blockIndex, err.Error()),
				})
			}
			for _, content := range folderBlock.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				// Ignorar entradas vacías y las entradas especiales . y ..
				if content.B_inodo == -1 || name == "" || name == "." || name == ".." {
					continue
				}

				// Leer el inodo de la entrada
				entryInode := &structures.Inode{}
				err = entryInode.Deserialize(diskPath, int64(sb.S_inode_start+content.B_inodo*sb.S_inode_size))
				if err != nil {
					continue // Saltar entradas corruptas
				}

				entryType := "folder"
				contentStr := ""
				if entryInode.I_type[0] == '1' { // Archivo
					entryType = "file"
					// Leer el contenido del archivo
					var contentBuilder strings.Builder
					for _, fileBlockIndex := range entryInode.I_block {
						if fileBlockIndex == -1 {
							break
						}
						fileBlock := &structures.FileBlock{}
						err = fileBlock.Deserialize(diskPath, int64(sb.S_block_start+fileBlockIndex*sb.S_block_size))
						if err != nil {
							continue
						}
						contentBuilder.WriteString(strings.Trim(string(fileBlock.B_content[:]), "\x00"))
					}
					contentStr = contentBuilder.String()
				}

				// Construir la entrada
				perm := string(entryInode.I_perm[:])
				entries = append(entries, FileSystemEntry{
					Name:     name,
					Type:     entryType,
					Size:     entryInode.I_size,
					Content:  contentStr,
					Perm:     perm,
					UID:      entryInode.I_uid,
					GID:      entryInode.I_gid,
					Created:  entryInode.I_ctime,
					Modified: entryInode.I_mtime,
				})
			}
		}

		return c.JSON(FileSystemResponse{
			Entries: entries,
		})
	})

	app.Listen(":3001")
}
