"use client";

import { useState, useEffect } from "react";

interface JournalEntry {
  count: number;
  operation: string;
  path: string;
  content: string;
  date: number;
}

interface JournalViewerProps {
  defaultPartitionID?: string; // ID de la partición seleccionada (opcional)
}

const JournalViewer: React.FC<JournalViewerProps> = ({ defaultPartitionID = "" }) => {
  const [partitionID, setPartitionID] = useState(defaultPartitionID);
  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [error, setError] = useState("");
  const [showTable, setShowTable] = useState(false);

  // Actualizar partitionID cuando defaultPartitionID cambie
  useEffect(() => {
    setPartitionID(defaultPartitionID);
  }, [defaultPartitionID]);

  const fetchJournal = async () => {
    if (!partitionID) {
      setError("Por favor, ingrese el ID de la partición.");
      return;
    }

    try {
      const response = await fetch(
        `http://localhost:3001/journal?id=${encodeURIComponent(partitionID)}`
      );
      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.output || `Error al cargar el Journal: ${response.statusText}`);
      }
      const data = await response.json();
      if (!Array.isArray(data.entries)) {
        throw new Error("Respuesta inválida del servidor: 'entries' no es un array");
      }
      setEntries(data.entries);
      setError("");
      setShowTable(true);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Error desconocido al cargar el Journal";
      setError(errorMessage);
      setShowTable(false);
      console.error(errorMessage);
    }
  };

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleString();
  };

  return (
    <div className="bg-gray-800 rounded-xl shadow-lg p-6 border border-blue-800 mt-4">
      <h3 className="text-lg font-semibold text-orange-400 mb-4">
        Visualizador del Journal
      </h3>

      <div className="mb-4">
        <label className="text-orange-300 mr-2">ID de la Partición:</label>
        <input
          type="text"
          value={partitionID}
          onChange={(e) => setPartitionID(e.target.value)}
          placeholder="Ej. 671A"
          className="px-3 py-1 bg-gray-700 text-white rounded-md border border-gray-600 focus:outline-none focus:border-orange-400"
        />
        <button
          onClick={fetchJournal}
          className="ml-4 px-3 py-1 bg-orange-500 text-white rounded-md hover:bg-orange-600 transition-all duration-300"
        >
          Ver Journal
        </button>
      </div>

      {error && <p className="text-red-500 mb-4">{error}</p>}

      {showTable && (
        <div className="overflow-x-auto">
          {entries.length === 0 ? (
            <p className="text-orange-300">No se encontraron entradas en el Journal.</p>
          ) : (
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-gray-700">
                  <th className="p-3 text-orange-400 border-b border-gray-600">Entrada #</th>
                  <th className="p-3 text-orange-400 border-b border-gray-600">Operación</th>
                  <th className="p-3 text-orange-400 border-b border-gray-600">Ruta</th>
                  <th className="p-3 text-orange-400 border-b border-gray-600">Contenido</th>
                  <th className="p-3 text-orange-400 border-b border-gray-600">Fecha</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr key={entry.count} className="bg-gray-900 hover:bg-gray-800">
                    <td className="p-3 text-white border-b border-gray-600">{entry.count}</td>
                    <td className="p-3 text-white border-b border-gray-600">{entry.operation}</td>
                    <td className="p-3 text-white border-b border-gray-600">{entry.path}</td>
                    <td className="p-3 text-white border-b border-gray-600">{entry.content}</td>
                    <td className="p-3 text-white border-b border-gray-600">{formatDate(entry.date)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
};

export default JournalViewer;