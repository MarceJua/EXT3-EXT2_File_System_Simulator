# EXT3-EXT2 File System Simulator (Phase 2)

## Overview
This project, developed as part of the *Laboratorio Manejo e Implementación de Archivos* course (Section B, 2025) by **Marcelo Andre Juarez Alfaro (202010367)**, implements a simulation of the **EXT2 and EXT3 file systems**. Phase 2 extends the Phase 1 functionality by adding **EXT3 support with Journaling**, new file system commands, a graphical interface for login and file system navigation, and deployment on **AWS** (frontend on S3, backend on a Dockerized EC2 instance). The system uses a client-server architecture with a **Go backend** (Fiber framework) and a **Next.js frontend**, communicating via a RESTful API. Data is stored in `.mia` binary files representing virtual disks.

For a detailed technical breakdown, refer to the [Technical Manual](Fase%202%20Archivos.pdf) in this repository.

## Features
- **Disk and Partition Management**:
  - Create (`MKDISK`), delete (`RMDISK`), and manage partitions (`FDISK`, with `ADD` and `DELETE` options).
  - Mount (`MOUNT`), unmount (`UNMOUNT`), and list (`MOUNTED`) partitions (primary, extended, logical).
- **File System Operations**:
  - Format partitions with EXT2 or EXT3 (`MKFS -fs=2fs|3fs`), creating `users.txt`.
  - Create directories (`MKDIR`), files (`MKFILE`), and view file contents (`CAT`).
  - New commands: Delete files/folders (`REMOVE`), edit files (`EDIT`), rename (`RENAME`), copy (`COPY`), move (`MOVE`), and search (`FIND`).
- **User and Group Management**:
  - Create (`MKUSR`, `MKGRP`), delete (`RMUSR`, `RMGRP`), and modify (`CHGRP`) users/groups.
  - Change ownership (`CHOWN`) and permissions (`CHMOD`), with recursive options.
  - Graphical login/logout interface replacing command-based `LOGIN`/`LOGOUT`.
- **EXT3 Journaling**:
  - Log operations in a Journal for recovery (`RECOVERY`) after simulated failures (`LOSS`).
  - Visualize Journal entries (`JOURNALING`) in the graphical interface.
- **Graphical Interface**:
  - Next.js-based frontend with input terminal, script upload, and output display.
  - New: Visual file system navigator for browsing disks, partitions, folders, and files.
- **Reporting**: Generate Graphviz-based reports (`REP`) for structures like MBR, Superblock, and Journal.
- **AWS Deployment**:
  - Frontend hosted on an AWS S3 bucket as a static website.
  - Backend deployed in a Docker container on an AWS EC2 instance (Ubuntu 22.04 LTS).

## Technologies
- **Frontend**: Next.js with React for client-side rendering and graphical navigation.
- **Backend**: Go with Fiber framework for a high-performance RESTful API.
- **File System**: Simulated EXT2/EXT3 structures (MBR, Superblock, Journal, Inodes, Bitmaps, etc.) stored in `.mia` files.
- **Communication**: HTTP-based RESTful API.
- **Utilities**: Go packages (`utils`, `stores`, `commands`) for disk operations, serialization, and session management.
- **Deployment**: AWS S3 (frontend), AWS EC2 with Docker (backend).

## Setup Instructions
### Local Development
1. **Prerequisites**:
   - Node.js (v16 or higher) for the frontend.
   - Go (v1.18 or higher) for the backend.
   - Docker (for backend containerization).
   - Git for cloning the repository.

2. **Clone the Repository**:
   ```bash
   git clone https://github.com/MarceJua/EXT3-EXT2_File_System_Simulator.git
   cd EXT3-EXT2_File_System_Simulator
   ```

3. **Backend Setup**:
   - Navigate to the backend directory:
     ```bash
     cd backend
     ```
   - Install dependencies:
     ```bash
     go mod tidy
     ```
   - Build and run the Docker container:
     ```bash
     docker build -t go-fiber-app .
     docker run -p 3000:3000 go-fiber-app
     ```
     The backend will start on `http://localhost:3000`.

4. **Frontend Setup**:
   - Navigate to the frontend directory:
     ```bash
     cd frontend
     ```
   - Install dependencies:
     ```bash
     npm install
     ```
   - Run the development server:
     ```bash
     npm run dev
     ```
     The frontend will be available at `http://localhost:3000`.

5. **Usage**:
   - Access the web interface at `http://localhost:3000`.
   - Use the graphical interface to log in, navigate the file system, or input commands (e.g., `mkfs -id=671A -fs=3fs`).
   - Upload scripts via the file upload feature or view Journal entries and reports.

### AWS Deployment
- **Frontend**: Hosted on an AWS S3 bucket as a static website. Access via the S3 URL (e.g., `http://<bucket-name>.s3-website-us-east-1.amazonaws.com`).
- **Backend**: Deployed in a Docker container on an AWS EC2 instance (t2.micro, Ubuntu 22.04 LTS). Access via the EC2 public IP (e.g., `http://<ip-ec2>:3000`).
- **Note**: Ensure the EC2 Security Group allows traffic on ports 22 (SSH) and 3000 (backend). Contact the author for access to the deployed instance.

## Screenshots
Below are screenshots showcasing the graphical interface and key functionalities of the EXT2/EXT3 File System Simulator:

- **Main Interface**: The primary Next.js interface with input/output terminals and file upload capabilities.
  ![Interfaz](https://i.ibb.co/bj0MM7bG/interfaz1.png)

- **Login Interface**: Graphical login screen replacing the command-based `LOGIN`.
  ![Login](https://i.ibb.co/Gvbqh3Mn/login.png)

- **File System Visualization**: Visual navigator for browsing disks, partitions, folders, and files.
  ![Visualización de Carpetas](https://i.ibb.co/4RTf81N1/visualizacion-Carpetas5.png)

- **Command Execution (Part 1)**: Example of executing initial commands (e.g., `MKDISK`, `FDISK`) via the input terminal.
  ![Primeros Comandos](https://i.ibb.co/qMKP5Cn9/primeros-Comandos2.png)

- **Command Execution (Part 2)**: Example of additional commands (e.g., `MKFS`, `MKDIR`) with output display.
  ![Segundos Comandos](https://i.ibb.co/hJyHTQWY/segundoscomandos4.png)

- **Journal Visualization**: Display of Journal entries for EXT3, showing logged operations.
  ![Journal](https://i.ibb.co/qFyb5QNC/journal6.png)

## Documentation
For a comprehensive overview of the architecture, data structures (MBR, Superblock, Journal, etc.), and command implementations, consult the [Technical Manual](Fase%202%20Archivos.pdf) in this repository.

## Contributing
This project is part of an academic assignment and is not currently open to contributions. For feedback or inquiries, contact the author at [mjuarez2017ig@gmail.com].
