"use client";

import { useState } from "react";
import { useSession } from "@/context/SessionContext";
import InputTerminal from "@/components/InputTerminal";
import OutputTerminal from "@/components/OutputTerminal";
import FileUpload from "@/components/FileUpload";
import DiskSelector from "@/components/DiskSelector";
import { executeCommands } from "@/services/api";
import Link from "next/link";

interface Disk {
  name: string;
  path: string;
  sizeMB: number;
  fit: string;
  mountedPartitions: string[] | null | undefined;
}

interface Partition {
  id: string;
  path: string;
  name: string;
  sizeKB: number;
  fit: string;
  status: string;
}

export default function Home() {
  const { session, logout } = useSession();
  const [input, setInput] = useState("");
  const [output, setOutput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [selectedDisk, setSelectedDisk] = useState<Disk | null>(null);
  const [selectedPartition, setSelectedPartition] = useState<Partition | null>(null);

  // Comandos permitidos sin sesión
  const noSessionCommands = [
    "mkdisk",
    "rmdisk",
    "fdisk",
    "mount",
    "unmount",
    "mkfs",
  ];

  const handleExecute = async () => {
    if (!input.trim()) {
      setOutput("Error: Ingrese al menos un comando.");
      return;
    }

    const commands = input.split("\n").map((cmd) => cmd.trim()).filter((cmd) => cmd);
    for (const cmd of commands) {
      const commandName = cmd.split(/\s+/)[0].toLowerCase();
      if (!noSessionCommands.includes(commandName) && !session.isAuthenticated) {
        setOutput(
          `Error: Inicie sesión para ejecutar '${commandName}'. Comandos permitidos sin sesión: ${noSessionCommands.join(", ")}.`
        );
        return;
      }
    }

    setIsLoading(true);
    try {
      const response = await executeCommands(input);
      setOutput(response);
    } catch (error) {
      setOutput(`Error: ${error instanceof Error ? error.message : "Desconocido"}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleClear = () => {
    console.log("handleClear ejecutado");
    setInput("");
    setOutput("");
  };

  const handleFileContent = (content: string) => {
    console.log("Archivo cargado:", content);
    setInput(content);
  };

  const handleLogout = async () => {
    console.log("handleLogout ejecutado");
    try {
      const response = await executeCommands("logout");
      setOutput(response);
      logout();
      setSelectedDisk(null);
      setSelectedPartition(null);
    } catch (error) {
      setOutput(`Error al cerrar sesión: ${error instanceof Error ? error.message : "Desconocido"}`);
    }
  };

  const handleLoginClick = () => {
    console.log("Botón Iniciar Sesión clicado");
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6 font-sans">
      <div className="max-w-4xl mx-auto space-y-8">
        {/* Encabezado */}
        <header className="flex justify-between items-center">
          <h1 className="text-3xl font-extrabold text-orange-400">
            Sistema de Archivos EXT3
          </h1>
          <div className="flex gap-3 items-center">
            {session.isAuthenticated ? (
              <>
                <span className="text-orange-300 font-medium">
                  Usuario: {session.username}
                </span>
                <button
                  onClick={handleLogout}
                  className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-all duration-300 shadow-md flex items-center gap-2"
                >
                  <svg
                    className="w-5 h-5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M17 16l4-4m0 0l-4-4m4 4H7m5 4v-7a3 3 0 00-3-3H5"
                    />
                  </svg>
                  Cerrar Sesión
                </button>
              </>
            ) : (
              <Link href="/login">
                <button
                  onClick={handleLoginClick}
                  className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 transition-all duration-300 shadow-md flex items-center gap-2"
                >
                  <svg
                    className="w-5 h-5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1"
                    />
                  </svg>
                  Iniciar Sesión
                </button>
              </Link>
            )}
            <label
              htmlFor="file-upload"
              className="px-4 py-2 bg-blue-700 text-white rounded-md hover:bg-blue-800 transition-all duration-300 shadow-md cursor-pointer flex items-center gap-2"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M7 16V4m0 0L3 8m4-4l4 4m10 4v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6m6-4h6"
                />
              </svg>
              Cargar Script
            </label>
            <button
              onClick={handleClear}
              className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 transition-all duration-300 shadow-md flex items-center gap-2"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
              Limpiar
            </button>
            <button
              onClick={handleExecute}
              disabled={isLoading}
              className={`px-4 py-2 bg-orange-500 text-white rounded-md hover:bg-orange-600 transition-all duration-300 shadow-md flex items-center gap-2 ${
                isLoading ? "opacity-70 cursor-not-allowed" : ""
              }`}
            >
              {isLoading ? (
                <svg className="animate-spin w-5 h-5" viewBox="0 0 24 24">
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                    fill="none"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                  />
                </svg>
              ) : (
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              )}
              {isLoading ? "Procesando..." : "Ejecutar"}
            </button>
          </div>
        </header>

        {/* Componentes */}
        {session.isAuthenticated && (
          <DiskSelector
            onDiskSelect={setSelectedDisk}
            onPartitionSelect={setSelectedPartition}
          />
        )}
        <FileUpload onFileContent={handleFileContent} />
        <InputTerminal value={input} onChange={setInput} />
        <OutputTerminal output={output} />
      </div>
    </div>
  );
}