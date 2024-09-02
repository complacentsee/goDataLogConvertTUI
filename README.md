# goDataLogConvertTUI
[![Build Status](https://github.com/complacentsee/goDataLogConvertTUI/actions/workflows/buildvalidate.yml/badge.svg)](https://github.com/complacentsee/goDataLogConvertTUI/actions/workflows/buildvalidate.yml)

`goDataLogConvertTUI` is a Go-based graphical tool for importing large amounts of raw DAT files into a FactoryTalk Historian server. This tool leverages a low-level C API (`piapi.dll`) to push data efficiently into the historian, capable of processing up to 250,000 points per second.

![Screenshot 1](https://github.com/complacentsee/goDataLogConvertTUI/blob/main/images/goDataLogConvertTUI-Loading.png?raw=true)
![Screenshot 2](https://github.com/complacentsee/goDataLogConvertTUI/blob/main/images/goDataLogConvertTUI-Processing.png?raw=true)

## Features

- Graphical user interface for easier configuration and monitoring.
- Imports raw `.DAT` files directly into a FactoryTalk Historian server.
- Supports mapping of Datalog tags to Historian tags using a CSV file.
- Allows configurable logging levels for better debugging and monitoring.
- Concurrent processing of multiple DAT files for efficient data import.

## Requirements

To run `goDataLogConvertTUI`, ensure the following dependencies are installed:

- **piapi.dll**: This is installed by default on all servers with the pi-sdk installed/ all PINS servers. 
  
  When running the import from a remote node:
  - Ensure that the remote IP address has write permissions in the Historian server settings (SMT > Security > Mappings & Trusts).
  - You may need to configure access based on the process name (`dat2fth`) in SMT > Security.

## Latest Release
[Latest Release](https://github.com/complacentsee/goDataLogConvertTUI/releases/latest)

## Previous Version
Looking for the command-line version? Check out [goDatalogConvert](link_to_previous_version).

## Building

1. Clone the repository:
    ```bash
    git clone https://github.com/complacentsee/goDataLogConvertTUI.git
    cd goDataLogConvertTUI
    ```

2. Build the executable (if cross-compiling):
    ```bash
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
    CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
    go build -v -o goDataLogConvertTUI.exe
    ```

3. Run the executable with the appropriate flags:
    ```bash
    ./goDataLogConvertTUI.exe -path /path/to/dat/files -host historian_server -processName dat2fth -tagMapCSV /path/to/tagmap.csv
    ```

## Usage

The `goDataLogConvertTUI` tool reads all `.DAT` files in the specified directory and pushes the values onto a FactoryTalk Historian server.

### Command-line Arguments

- `-path` (default: `.`): Path to the directory containing DAT files.
- `-host` (default: `localhost`): The hostname of the FactoryTalk Historian server.
- `-processName` (default: `dat2fth`): The process name used for the historian connection.
- `-tagMapCSV`: Path to a CSV file containing the tag map for translating Datalog tags to Historian tags.
- `-debug`: Enable debug-level logging for detailed output.

### Example

```bash
./goDataLogConvertTUI.exe -path /data/datfiles -host historian-server -processName dat2fth -tagMapCSV tagmap.csv -debug
