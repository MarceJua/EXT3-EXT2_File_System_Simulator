"use client";

import { useState, useEffect } from "react";
import { useSession } from "@/context/SessionContext";
import { useRouter } from "next/navigation";
import { login } from "@/services/api";

export default function Login() {
  const { login: setSessionLogin } = useSession();
  const router = useRouter();
  const [form, setForm] = useState({ user: "", pass: "", id: "" });
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    console.log("Página /login renderizada");
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    console.log("Formulario enviado:", form);
    if (!form.user || !form.pass || !form.id) {
      setError("Todos los campos son obligatorios");
      return;
    }

    setIsLoading(true);
    setError("");
    try {
      const response = await login(form.user, form.pass, form.id);
      setSessionLogin(form.user);
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error al iniciar sesión");
    } finally {
      setIsLoading(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6 font-sans flex items-center justify-center">
      <div className="max-w-md w-full bg-gray-800 rounded-xl shadow-lg p-8">
        <h2 className="text-2xl font-bold text-orange-400 mb-6 text-center">
          Iniciar Sesión
        </h2>
        {error && (
          <p className="text-red-500 mb-4 text-center">{error}</p>
        )}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="user" className="block text-sm font-medium text-orange-300">
              Usuario
            </label>
            <input
              type="text"
              id="user"
              name="user"
              value={form.user}
              onChange={handleChange}
              className="w-full mt-1 p-2 bg-gray-700 text-white rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
              placeholder="Ej. root"
              disabled={isLoading}
            />
          </div>
          <div>
            <label htmlFor="pass" className="block text-sm font-medium text-orange-300">
              Contraseña
            </label>
            <input
              type="password"
              id="pass"
              name="pass"
              value={form.pass}
              onChange={handleChange}
              className="w-full mt-1 p-2 bg-gray-700 text-white rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
              placeholder="Ej. 123"
              disabled={isLoading}
            />
          </div>
          <div>
            <label htmlFor="id" className="block text-sm font-medium text-orange-300">
              ID de Partición
            </label>
            <input
              type="text"
              id="id"
              name="id"
              value={form.id}
              onChange={handleChange}
              className="w-full mt-1 p-2 bg-gray-700 text-white rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
              placeholder="Ej. 671A"
              disabled={isLoading}
            />
          </div>
          <button
            type="submit"
            disabled={isLoading}
            className={`w-full py-2 bg-orange-500 text-white rounded-md hover:bg-orange-600 transition-all duration-300 ${
              isLoading ? "opacity-70 cursor-not-allowed" : ""
            }`}
          >
            {isLoading ? "Iniciando..." : "Iniciar Sesión"}
          </button>
        </form>
      </div>
    </div>
  );
}