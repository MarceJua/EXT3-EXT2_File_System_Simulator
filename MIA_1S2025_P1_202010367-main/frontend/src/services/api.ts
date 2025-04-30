const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";

export const executeCommands = async (command: string): Promise<string> => {
  console.log(API_URL);

  try {
    const response = await fetch(`${API_URL}/execute`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ command }),
    });

    if (!response.ok) {
      throw new Error("Error en la respuesta del servidor");
    }

    const data = await response.json();
    return data.output;
  } catch (error) {
    console.error("Error:", error);
    throw new Error("Error al ejecutar los comandos");
  }
};

export const login = async (user: string, pass: string, id: string): Promise<string> => {
  try {
    const response = await fetch(`${API_URL}/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ user, pass, id }),
    });

    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.output || "Error al iniciar sesión");
    }

    const data = await response.json();
    return data.output;
  } catch (error) {
    console.error("Error:", error);
    throw error;
  }
};

export const getDisks = async (): Promise<
  { name: string; path: string; sizeMB: number; fit: string; mountedPartitions: string[] | null }[]
> => {
  try {
    const response = await fetch(`${API_URL}/disks`);
    if (!response.ok) {
      throw new Error(`Error al cargar discos: ${response.statusText}`);
    }
    const data = await response.json();
    if (!Array.isArray(data.disks)) {
      throw new Error("Respuesta inválida del servidor: 'disks' no es un array");
    }
    // Asegurarnos que mountedPartitions sea un array (o null)
    return data.disks.map((disk: any) => ({
      ...disk,
      mountedPartitions: Array.isArray(disk.mountedPartitions) ? disk.mountedPartitions : null,
    }));
  } catch (error) {
    console.error("Error al cargar discos:", error);
    throw error;
  }
};

export const getPartitions = async (
  diskPath: string
): Promise<
  { id: string; path: string; name: string; sizeKB: number; fit: string; status: string }[]
> => {
  try {
    const response = await fetch(
      `${API_URL}/partitions?diskPath=${encodeURIComponent(diskPath)}`
    );
    if (!response.ok) {
      throw new Error(`Error al cargar particiones: ${response.statusText}`);
    }
    const data = await response.json();
    if (!Array.isArray(data.partitions)) {
      throw new Error("Respuesta inválida del servidor: 'partitions' no es un array");
    }
    return data.partitions;
  } catch (error) {
    console.error("Error al cargar particiones:", error);
    throw error;
  }
};

export const getFileSystemEntries = async (
  partitionID: string,
  path: string
): Promise<
  {
    name: string;
    type: "folder" | "file";
    size: number;
    content: string;
    perm: string;
    uid: number;
    gid: number;
    created: number;
    modified: number;
  }[]
> => {
  try {
    const response = await fetch(
      `${API_URL}/filesystem?id=${encodeURIComponent(partitionID)}&path=${encodeURIComponent(path)}`
    );
    if (!response.ok) {
      throw new Error(`Error al cargar el directorio: ${response.statusText}`);
    }
    const data = await response.json();
    if (!Array.isArray(data.entries)) {
      throw new Error("Respuesta inválida del servidor: 'entries' no es un array");
    }
    return data.entries;
  } catch (error) {
    console.error("Error al cargar el directorio:", error);
    throw error;
  }
};

export const getJournalEntries = async (
  partitionID: string
): Promise<
  {
    count: number;
    operation: string;
    path: string;
    content: string;
    date: number;
  }[]
> => {
  try {
    const response = await fetch(
      `${API_URL}/journal?id=${encodeURIComponent(partitionID)}`
    );
    if (!response.ok) {
      throw new Error(`Error al cargar el Journal: ${response.statusText}`);
    }
    const data = await response.json();
    if (!Array.isArray(data.entries)) {
      throw new Error("Respuesta inválida del servidor: 'entries' no es un array");
    }
    return data.entries;
  } catch (error) {
    console.error("Error al cargar el Journal:", error);
    throw error;
  }
};