"use client";

import { useState, useEffect } from "react";

interface FileSystemEntry {
  name: string;
  type: "folder" | "file";
  size: number;
  content: string;
  perm: string;
  uid: number;
  gid: number;
  created: number;
  modified: number;
}

interface FileSystemViewerProps {
  partitionID: string; // ID de la partición (ej. "671A")
}

const FileSystemViewer: React.FC<FileSystemViewerProps> = ({ partitionID }) => {
  const [currentPath, setCurrentPath] = useState("/");
  const [entries, setEntries] = useState<FileSystemEntry[]>([]);
  const [error, setError] = useState("");
  const [pathHistory, setPathHistory] = useState<string[]>(["/"]); // Para navegar hacia atrás

  // Cargar el contenido del directorio actual
  useEffect(() => {
    const fetchEntries = async () => {
      try {
        const response = await fetch(
          `http://localhost:3001/filesystem?id=${encodeURIComponent(partitionID)}&path=${encodeURIComponent(
            currentPath
          )}`
        );
        if (!response.ok) {
          throw new Error(`Error al cargar el directorio: ${response.statusText}`);
        }
        const data = await response.json();
        if (!Array.isArray(data.entries)) {
          throw new Error("Respuesta inválida del servidor: 'entries' no es un array");
        }
        setEntries(data.entries);
        setError("");
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : "Error desconocido al cargar el directorio";
        setError(errorMessage);
        console.error(errorMessage);
      }
    };
    fetchEntries();
  }, [currentPath, partitionID]);

  const handleFolderClick = (folderName: string) => {
    // Navegar a un subdirectorio
    const newPath = currentPath === "/" ? `/${folderName}` : `${currentPath}/${folderName}`;
    setCurrentPath(newPath);
    setPathHistory([...pathHistory, newPath]);
  };

  const handleBackClick = () => {
    // Volver al directorio padre
    if (pathHistory.length <= 1) {
      setCurrentPath("/");
      setPathHistory(["/"]);
      return;
    }
    const newHistory = pathHistory.slice(0, -1);
    setPathHistory(newHistory);
    setCurrentPath(newHistory[newHistory.length - 1]);
  };

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleString();
  };

  return (
    <div className="bg-gray-800 rounded-xl shadow-lg p-6 border border-blue-800 mt-4">
      <h3 className="text-lg font-semibold text-orange-400 mb-4">
        Navegador de Archivos (Ruta: {currentPath})
      </h3>
      {error && <p className="text-red-500 mb-4">{error}</p>}

      <div className="flex justify-between items-center mb-4">
        <button
          onClick={handleBackClick}
          className="px-3 py-1 bg-gray-600 text-white rounded-md hover:bg-gray-700 transition-all duration-300"
          disabled={pathHistory.length <= 1}
        >
          Volver al Directorio Padre
        </button>
      </div>

      {entries.length === 0 && !error && (
        <p className="text-orange-300 mb-4">El directorio está vacío.</p>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {entries.map((entry) => (
          <div
            key={entry.name}
            className={`p-4 rounded-lg transition-all duration-300 ${
              entry.type === "folder"
                ? "bg-gray-700 hover:bg-gray-600 cursor-pointer"
                : "bg-gray-900"
            }`}
            onClick={entry.type === "folder" ? () => handleFolderClick(entry.name) : undefined}
          >
            <div className="flex items-center space-x-3">
              {entry.type === "folder" ? (
                <svg
                  className="w-8 h-8 text-orange-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
                  />
                </svg>
              ) : (
                <svg
                  className="w-8 h-8 text-orange-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                  />
                </svg>
              )}
              <div>
                <span className="text-white font-medium">{entry.name}</span>
                <p className="text-orange-300 text-sm">
                  Tipo: {entry.type === "folder" ? "Carpeta" : "Archivo"}
                </p>
                {entry.type === "file" && (
                  <>
                    <p className="text-orange-300 text-sm">Tamaño: {entry.size} bytes</p>
                    <p className="text-orange-300 text-sm">
                      Contenido: {entry.content || "Vacío"}
                    </p>
                  </>
                )}
                <p className="text-orange-300 text-sm">Permisos: {entry.perm}</p>
                <p className="text-orange-300 text-sm">UID: {entry.uid}, GID: {entry.gid}</p>
                <p className="text-orange-300 text-sm">
                  Creado: {formatDate(entry.created)}
                </p>
                <p className="text-orange-300 text-sm">
                  Modificado: {formatDate(entry.modified)}
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default FileSystemViewer;