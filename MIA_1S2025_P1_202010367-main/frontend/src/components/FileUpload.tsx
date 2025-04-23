import React from "react";

interface FileUploadProps {
  onFileContent: (content: string) => void;
}

const FileUpload: React.FC<FileUploadProps> = ({ onFileContent }) => {
  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    console.log("FileUpload: Archivo seleccionado");
    const file = event.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const content = e.target?.result as string;
        console.log("FileUpload: Contenido leído:", content);
        onFileContent(content);
      };
      reader.readAsText(file);
    }
    event.target.value = ""; // Resetear el input
  };

  return (
    <input
      type="file"
      id="file-upload"
      className="hidden"
      onChange={handleFileChange}
      accept=".smia"
    />
  );
};

export default FileUpload;