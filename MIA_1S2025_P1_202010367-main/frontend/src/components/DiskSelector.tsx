"use client";

import { useState, useEffect } from "react";
import FileSystemViewer from "./FileSystemViewer"; // Importar el nuevo componente

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

interface DiskSelectorProps {
  onDiskSelect: (disk: Disk | null) => void;
  onPartitionSelect: (partition: Partition | null) => void;
}

const DiskSelector: React.FC<DiskSelectorProps> = ({ onDiskSelect, onPartitionSelect }) => {
  const [disks, setDisks] = useState<Disk[]>([]);
  const [partitions, setPartitions] = useState<Partition[]>([]);
  const [selectedDisk, setSelectedDisk] = useState<Disk | null>(null);
  const [selectedPartition, setSelectedPartition] = useState<Partition | null>(null);
  const [error, setError] = useState("");

  // Cargar discos
  useEffect(() => {
    const fetchDisks = async () => {
      try {
        const response = await fetch("http://localhost:3001/disks");
        if (!response.ok) {
          throw new Error(`Error al cargar discos: ${response.statusText}`);
        }
        const data = await response.json();
        if (!Array.isArray(data.disks)) {
          throw new Error("Respuesta inválida del servidor: 'disks' no es un array");
        }
        setDisks(data.disks);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : "Error desconocido al cargar discos";
        setError(errorMessage);
        console.error(errorMessage);
      }
    };
    fetchDisks();
  }, []);

  // Cargar particiones cuando se selecciona un disco
  useEffect(() => {
    if (!selectedDisk) {
      setPartitions([]);
      setSelectedPartition(null);
      onPartitionSelect(null);
      return;
    }

    const fetchPartitions = async () => {
      try {
        const response = await fetch(
          `http://localhost:3001/partitions?diskPath=${encodeURIComponent(selectedDisk.path)}`
        );
        if (!response.ok) {
          throw new Error(`Error al cargar particiones: ${response.statusText}`);
        }
        const data = await response.json();
        if (!Array.isArray(data.partitions)) {
          throw new Error("Respuesta inválida del servidor: 'partitions' no es un array");
        }
        setPartitions(data.partitions);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : "Error desconocido al cargar particiones";
        setError(errorMessage);
        console.error(errorMessage);
      }
    };
    fetchPartitions();
  }, [selectedDisk]);

  const handleDiskClick = (disk: Disk) => {
    setSelectedDisk(disk);
    setSelectedPartition(null);
    onDiskSelect(disk);
    onPartitionSelect(null);
  };

  const handlePartitionClick = (partition: Partition) => {
    setSelectedPartition(partition);
    onPartitionSelect(partition);
  };

  const handleBackToDisks = () => {
    setSelectedDisk(null);
    setSelectedPartition(null);
    onDiskSelect(null);
    onPartitionSelect(null);
  };

  return (
    <div className="bg-gray-800 rounded-xl shadow-lg p-6 border border-blue-800">
      <h3 className="text-lg font-semibold text-orange-400 mb-4">
        Visualizador del Sistema de Archivos
      </h3>
      {error && <p className="text-red-500 mb-4">{error}</p>}

      {!selectedDisk ? (
        <>
          <p className="text-orange-300 mb-4">Seleccione el disco que desea visualizar</p>
          {disks.length === 0 && !error && (
            <p className="text-orange-300 mb-4">No hay discos disponibles.</p>
          )}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {disks.map((disk) => (
              <button
                key={disk.path}
                onClick={() => handleDiskClick(disk)}
                className="flex flex-col items-center p-4 bg-gray-700 rounded-lg hover:bg-gray-600 transition-all duration-300"
              >
                <svg
                  className="w-12 h-12 text-orange-400 mb-2"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                  />
                </svg>
                <span className="text-white font-medium">{disk.name}</span>
                <span className="text-orange-300 text-sm">
                  Capacidad: {disk.sizeMB.toFixed(2)} MB
                </span>
                <span className="text-orange-300 text-sm">Fit: {disk.fit || "N/A"}</span>
                <span className="text-orange-300 text-sm">
                  Particiones Montadas:{" "}
                  {disk.mountedPartitions && disk.mountedPartitions.length > 0
                    ? disk.mountedPartitions.join(", ")
                    : "Ninguna"}
                </span>
              </button>
            ))}
          </div>
        </>
      ) : !selectedPartition ? (
        <>
          <div className="flex justify-between items-center mb-4">
            <p className="text-orange-300">
              Seleccione la partición que desea visualizar (Disco: {selectedDisk.name})
            </p>
            <button
              onClick={handleBackToDisks}
              className="px-3 py-1 bg-gray-600 text-white rounded-md hover:bg-gray-700 transition-all duration-300"
            >
              Volver a Discos
            </button>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {partitions.map((partition) => (
              <button
                key={partition.id}
                onClick={() => handlePartitionClick(partition)}
                className="flex flex-col items-center p-4 bg-gray-700 rounded-lg hover:bg-gray-600 transition-all duration-300"
              >
                <svg
                  className="w-12 h-12 text-orange-400 mb-2"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
                  />
                </svg>
                <span className="text-white font-medium">{partition.name}</span>
                <span className="text-orange-300 text-sm">ID: {partition.id}</span>
                <span className="text-orange-300 text-sm">
                  Tamaño: {partition.sizeKB ? partition.sizeKB.toFixed(2) : "N/A"} KB
                </span>
                <span className="text-orange-300 text-sm">Fit: {partition.fit || "N/A"}</span>
                <span className="text-orange-300 text-sm">Estado: {partition.status || "N/A"}</span>
              </button>
            ))}
          </div>
        </>
      ) : (
        <div>
          <div className="flex justify-between items-center mb-4">
            <p className="text-orange-300">
              Visualizando: {selectedDisk.name} - {selectedPartition.name} (ID: {selectedPartition.id})
            </p>
            <button
              onClick={handleBackToDisks}
              className="px-3 py-1 bg-gray-600 text-white rounded-md hover:bg-gray-700 transition-all duration-300"
            >
              Volver a Discos
            </button>
          </div>
          {/* Mostrar el navegador de archivos */}
          <FileSystemViewer partitionID={selectedPartition.id} />
        </div>
      )}
    </div>
  );
};

export default DiskSelector;