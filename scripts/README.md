# Build Scripts

This directory contains scripts for building the EC2 Instance Selector.

## Windows Build Script (build.cmd)

A Windows batch script to build both the API server and CLI components.

### Prerequisites

- Windows operating system
- Go 1.19 or later installed and in PATH
- Git installed and in PATH (for release builds)

### Usage

#### Development Build
For development builds with version set to "dev":
```cmd
scripts\build.cmd
```

#### Release Build
For release builds that use the Git tag as version:
```cmd
scripts\build.cmd --release
```

### Output

The script creates a `build` directory containing:
- `api-server.exe` - The API server binary
- `ec2-instance-selector.exe` - The CLI binary

### Build Information

The binaries include the following build information:
- Version: Either "dev" or the Git tag for release builds
- Build Date: Current date in YYYYMMDD format

### Error Handling

- The script will exit with a non-zero status code if the build fails
- Error messages will be displayed for build failures
- The build directory is created automatically if it doesn't exist

### Examples

1. Build for development:
```cmd
C:\> cd amazon-ec2-instance-selector
C:\amazon-ec2-instance-selector> scripts\build.cmd
Building ec2-instance-selector version dev...
Build date: 20240315
Building API server...
Building CLI...
Build completed successfully
Binaries are in the build directory:
  build\api-server.exe
  build\ec2-instance-selector.exe
```

2. Build for release:
```cmd
C:\> cd amazon-ec2-instance-selector
C:\amazon-ec2-instance-selector> scripts\build.cmd --release
Building ec2-instance-selector version v2.0.0...
Build date: 20240315
Building API server...
Building CLI...
Build completed successfully
Binaries are in the build directory:
  build\api-server.exe
  build\ec2-instance-selector.exe
``` 