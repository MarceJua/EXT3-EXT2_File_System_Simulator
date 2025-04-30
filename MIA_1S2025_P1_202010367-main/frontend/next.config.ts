import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  eslint: {
    // Ignora los errores de ESLint durante la construcción
    ignoreDuringBuilds: true,
  },
};

export default nextConfig;