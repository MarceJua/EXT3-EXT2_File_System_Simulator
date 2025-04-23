package main

import (
	"fmt"
	"strings"

	analyzer "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/analyzer"
	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"

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

			// Obtener el nombre del comando (primera palabra en minúsculas)
			cmdParts := strings.Fields(cmd)
			if len(cmdParts) == 0 {
				output += "Error: Comando vacío\n"
				continue
			}
			commandName := strings.ToLower(cmdParts[0])

			// Verificar si el comando requiere sesión
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

	app.Listen(":3001")
}
