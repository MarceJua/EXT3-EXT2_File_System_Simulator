import React from "react";

interface InputTerminalProps {
  value: string;
  onChange: (value: string) => void;
}

const InputTerminal = ({ value, onChange }: InputTerminalProps) => {
    return (
      <div className="rounded-xl overflow-hidden shadow-lg border border-blue-800">
        <div className="bg-blue-900 px-4 py-2 flex justify-between items-center">
          <span className="text-sm font-semibold text-orange-300">Terminal de Entrada</span>
          <div className="flex space-x-2">
            <div className="w-3 h-3 rounded-full bg-orange-500"></div>
            <div className="w-3 h-3 rounded-full bg-blue-500"></div>
            <div className="w-3 h-3 rounded-full bg-gray-500"></div>
          </div>
        </div>
        <textarea
          className="w-full h-56 bg-gray-800 text-orange-100 p-4 font-mono resize-none focus:outline-none focus:ring-2 focus:ring-orange-500"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Escribe tus comandos aquí (ej. mkdisk -size=5 -unit=M -path=/tmp/test)"
          spellCheck="false"
        />
      </div>
    );
  };
  
  export default InputTerminal;