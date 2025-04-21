import React from "react";

interface OutputTerminalProps {
  output: string;
}

const OutputTerminal = ({ output }: OutputTerminalProps) => {
    return (
      <div className="rounded-xl overflow-hidden shadow-lg border border-blue-800">
        <div className="bg-blue-900 px-4 py-2 flex justify-between items-center">
          <span className="text-sm font-semibold text-orange-300">Terminal de Salida</span>
          <div className="flex space-x-2">
            <div className="w-3 h-3 rounded-full bg-orange-500"></div>
            <div className="w-3 h-3 rounded-full bg-blue-500"></div>
            <div className="w-3 h-3 rounded-full bg-gray-500"></div>
          </div>
        </div>
        <div className="w-full h-56 bg-gray-800 text-orange-100 p-4 font-mono overflow-auto">
          <pre className="whitespace-pre-wrap">{output || "Resultados aparecerán aquí..."}</pre>
        </div>
      </div>
    );
  };
  
  export default OutputTerminal;