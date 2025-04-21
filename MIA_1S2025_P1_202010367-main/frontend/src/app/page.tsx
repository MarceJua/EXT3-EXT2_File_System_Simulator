"use client";

import { useState } from "react";
import InputTerminal from "@/components/InputTerminal";
import OutputTerminal from "@/components/OutputTerminal";
import FileUpload from "@/components/FileUpload";
import { executeCommands } from "@/services/api";

export default function Home() {
  const [input, setInput] = useState("");
  const [output, setOutput] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleExecute = async () => {
    if (!input.trim()) {
      setOutput("Error: Ingrese al menos un comando.");
      return;
    }

    setIsLoading(true);
    try {
      const response = await fetch("http://localhost:3001/execute", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ command: input }),
      });
      const data = await response.json();
      setOutput(data.output);
    } catch (error) {
      setOutput(`Error: ${error instanceof Error ? error.message : "Desconocido"}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleClear = () => {
    setInput("");
    setOutput("");
  };

  const handleFileContent = (content: string) => {
    setInput(content);
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6 font-sans">
      <div className="max-w-4xl mx-auto space-y-8">
        {/* Encabezado */}
        <header className="flex justify-between items-center">
          <h1 className="text-3xl font-extrabold text-orange-400">
            Sistema de Archivos EXT2
          </h1>
          <div className="flex gap-3">
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
        <FileUpload onFileContent={handleFileContent} />
        <InputTerminal value={input} onChange={setInput} />
        <OutputTerminal output={output} />
      </div>
    </div>
  );
}