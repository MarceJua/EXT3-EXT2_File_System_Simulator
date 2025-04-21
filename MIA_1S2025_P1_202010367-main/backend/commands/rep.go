package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	reports "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/reports"
	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
)

// REP estructura que representa el comando rep con sus parámetros
type REP struct {
	id           string // ID del disco
	path         string // Ruta del archivo del disco
	name         string // Nombre del reporte
	path_file_ls string // Ruta del archivo ls (opcional)
}

// ParserRep parsea el comando rep y devuelve una instancia de REP
func ParseRep(tokens []string) (string, error) {
	cmd := &REP{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := strings.Trim(parts[1], "\"")

		switch key {
		case "-id":
			if value == "" {
				return "", errors.New("el id no puede estar vacío")
			}
			cmd.id = value
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-name":
			validNames := []string{"mbr", "ebr", "disk", "inode", "block", "bm_inode", "bm_block", "tree", "sb", "file", "ls"}
			if !contains(validNames, value) {
				return "", errors.New("nombre inválido, debe ser: mbr, ebr, disk, inode, block, bm_inode, bm_block, tree, sb, file, ls")
			}
			cmd.name = value
		case "-path_file_ls":
			if value == "" {
				return "", errors.New("el path_file_ls no puede estar vacío")
			}
			cmd.path_file_ls = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.id == "" || cmd.path == "" || cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -id, -path, -name")
	}
	if (cmd.name == "ls" || cmd.name == "file") && cmd.path_file_ls == "" {
		return "", errors.New("falta parámetro -path_file_ls para reporte " + cmd.name)
	}

	err := commandRep(cmd)
	if err != nil {
		return "", err
	}

	// Ajustar mensaje de salida según el tipo de reporte
	if cmd.name == "bm_inode" || cmd.name == "bm_block" {
		return fmt.Sprintf("REP: Reporte %s generado en %s", cmd.name, cmd.path), nil
	}
	if cmd.name == "file" {
		return fmt.Sprintf("REP: Reporte %s generado en %s", cmd.name, cmd.path), nil
	}
	return fmt.Sprintf("REP: Reporte %s generado en %s", cmd.name, strings.TrimSuffix(cmd.path, filepath.Ext(cmd.path))+".png"), nil
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func commandRep(rep *REP) error {
	mountedMbr, mountedSb, mountedDiskPath, err := stores.GetMountedPartitionRep(rep.id)
	if err != nil {
		return err
	}

	requiresSuperblock := []string{"inode", "block", "bm_inode", "bm_block", "tree", "sb", "file", "ls"}
	if contains(requiresSuperblock, rep.name) && mountedSb == nil {
		return fmt.Errorf("error interno: superbloque no cargado para la partición %s", rep.id)
	}

	var dotContent string
	switch rep.name {
	case "mbr":
		dotContent, err = reports.ReportMBR(mountedMbr)
	case "ebr":
		dotContent, err = reports.ReportEBR(mountedMbr, mountedDiskPath)
	case "disk":
		dotContent, err = reports.ReportDisk(mountedMbr, mountedDiskPath)
	case "inode":
		dotContent, err = reports.ReportInode(mountedSb, mountedDiskPath)
	case "block":
		dotContent, err = reports.ReportBlock(mountedSb, mountedDiskPath)
	case "bm_inode":
		err = reports.ReportBMInode(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			return fmt.Errorf("error generando reporte bm_inode: %v", err)
		}
		return nil
	case "bm_block":
		err = reports.ReportBMBlock(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			return fmt.Errorf("error generando reporte bm_block: %v", err)
		}
		return nil
	case "tree":
		dotContent, err = reports.ReportTree(mountedSb, mountedDiskPath)
	case "sb":
		dotContent, err = reports.ReportSB(mountedSb)
	case "file":
		dotContent, err = reports.ReportFile(mountedSb, mountedDiskPath, rep.path_file_ls)
		if err != nil {
			return fmt.Errorf("error generando reporte file: %v", err)
		}
		// Escribir directamente el contenido en un .txt
		err = os.WriteFile(rep.path, []byte(dotContent), 0644)
		if err != nil {
			return fmt.Errorf("error escribiendo reporte file en %s: %v", rep.path, err)
		}
		return nil // No necesitamos generar imagen
	case "ls":
		dotContent, err = reports.ReportLS(mountedSb, mountedDiskPath, rep.path_file_ls)
	default:
		return fmt.Errorf("reporte no implementado: %s", rep.name)
	}
	if err != nil {
		return fmt.Errorf("error generando reporte %s: %v", rep.name, err)
	}

	// Para reportes gráficos, generar .dot y .png
	dotFile := rep.path + ".dot"
	err = writeDotFile(dotFile, dotContent)
	if err != nil {
		return fmt.Errorf("error escribiendo archivo DOT: %v", err)
	}

	outputFile := strings.TrimSuffix(rep.path, filepath.Ext(rep.path)) + ".png"
	err = generateImage(dotFile, outputFile)
	if err != nil {
		return fmt.Errorf("error generando imagen: %v", err)
	}

	return nil
}

func writeDotFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

func generateImage(dotFile, outputFile string) error {
	cmd := exec.Command("dot", "-Tpng", dotFile, "-o", outputFile)
	return cmd.Run()
}
