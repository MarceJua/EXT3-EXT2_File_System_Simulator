const API_URL = "http://localhost:3001";

export const executeCommands = async (command: string): Promise<string> => {
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