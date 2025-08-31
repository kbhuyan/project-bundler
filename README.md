# Project Bundler CLI

`project-bundler` is a fast, intelligent, and flexible command-line tool written in Go that consolidates all relevant source code files from a project directory into a single, large Markdown file. This is incredibly useful for providing context to Large Language Models (LLMs), creating project archives, or generating documentation.

The tool is ecosystem-aware, with built-in presets for Go, Rust, iOS, and Android projects, and can automatically detect the project type. It's designed to be robust, safely skipping binary files, respecting ignore lists, and providing clear reporting.

## Features

- **Single Binary**: No dependencies needed, easy to install and run.
- **Ecosystem Presets**: Intelligent default configurations for `Go`, `Rust`, `iOS`, and `Android` projects.
- **Auto-Detection**: Automatically detects the project type based on landmark files (`go.mod`, `Cargo.toml`, etc.).
- **Safe Binary Handling**: Scans file contents to detect and skip binary files, preventing output corruption.
- **Smart Language Detection**: Assigns Markdown language identifiers based on file extension and common filenames (like `Dockerfile`, `Makefile`).
- **Highly Configurable**: Customize the source directory, output file, and lists of ignored directories and file extensions.
- **Diagnostic Reporting**: Optional flag to report exactly which files were skipped and why.
- **Efficient**: Uses buffered I/O to handle large projects with minimal memory consumption.

## Installation

### From Source

To build the `project-bundler` from source, you need to have Go installed (version 1.18 or newer).

1.  **Clone the repository (or save the source code):**
    If you have the source in a file named `main.go`, you can skip this step.

2.  **Build the executable:**
    Open your terminal and run the `go build` command. This will create a `project-bundler` executable in your current directory.
    ```sh
    go build -o project-bundler .
    ```

3.  **Make it available system-wide (Optional):**
    You can move the generated binary to a directory in your system's `PATH` to make it easy to run from anywhere.
    ```sh
    # For macOS / Linux
    sudo mv project-bundler /usr/local/bin/

    # For Windows, move project-bundler.exe to a directory in your PATH
    ```

## Usage

The CLI is designed to be simple to use. You can run it with or without flags.

### Basic Usage (Auto-Detection)

The easiest way to use the tool is to `cd` into your project's root directory and run it. It will automatically detect the project type and generate a `bundle.md` file.

```sh
# Navigate to your project directory
cd /path/to/my-go-project/

# Run the bundler
project-bundler
```
**Output:**
```
Auto-detected project type: go
Starting to bundle project from '.' into 'bundle.md' (type: go)...
  + Bundling file: main.go
  + Bundling file: go.mod
  ...
✅ Successfully created project bundle at 'bundle.md'
```

### Advanced Usage (Flags)

You can customize the tool's behavior using command-line flags.

```sh
project-bundler [flags]
```

**Available Flags:**

| Flag              | Type     | Default                                                                 | Description                                                                                             |
| ----------------- | -------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `-src`            | `string` | `.`                                                                     | Source project directory to read from.                                                                  |
| `-output`         | `string` | `bundle.md`                                                             | Name of the output markdown file.                                                                       |
| `-type`           | `string` | `auto`                                                                  | Project type. Overrides auto-detection. Options: `auto`, `go`, `rust`, `ios`, `android`, `generic`.       |
| `-report-skipped` | `bool`   | `false`                                                                 | If set, prints a detailed report of all files that were skipped and the reasons why.                    |
| `-ignore-dirs`    | `string` | *(Varies by type)*                                                       | Comma-separated list of directories to ignore. **Note:** This overrides the default for the selected type. |
| `-ignore-exts`    | `string` | *(Varies by type)*                                                       | Comma-separated list of file extensions to ignore. **Note:** This overrides the default for the selected type. |

### Examples

**1. Bundle a Rust project in a specific directory:**
```sh
project-bundler -src=/path/to/my-rust-app/ -output=rust_project.md
```

**2. Bundle an iOS project and see a report of skipped files:**
```sh
project-bundler -src=/path/to/my-ios-app/ -report-skipped
```
**Output with report:**
```
...
--- Skipped Files Report ---

Reason: Ignored Directory
  - .git
  - Pods
  - build

Reason: Detected Binary Content
  - Resources/Assets.car

Reason: Ignored Extension/File
  - MyProject.xcodeproj/project.xcworkspace/xcuserdata/user.xcuserdatad/UserInterfaceState.xcuserstate
--------------------------

✅ Successfully created project bundle at 'bundle.md'
```

**3. Manually specify the project type:**
Useful if auto-detection fails or you want to force a specific set of rules.
```sh
project-bundler -type=go
```

**4. Override the default ignore list:**
This example bundles the current directory but also ignores the `testdata` directory.
```sh
# The default ignore list for Go is ".git,vendor,build"
# This command appends "testdata" to that list.
project-bundler -type=go -ignore-dirs=".git,vendor,build,testdata"
```

## How It Works

1.  **Configuration**: The tool first determines the project type (either via auto-detection or the `-type` flag) and loads the corresponding preset, which defines which directories, file extensions, and files to ignore.
2.  **File Traversal**: It walks the entire source directory tree recursively.
3.  **Filtering**: For each item found, it applies the following checks in order:
    - Is it a directory in the `ignore-dirs` list? If so, skip the entire directory.
    - Is it a file with an extension in the `ignore-exts` list? If so, skip it.
    - **Is it a binary file?** It reads the first 1KB of the file. If it contains null bytes (`\x00`), it's considered binary and skipped. This is the key safety feature.
4.  **Bundling**: If a file passes all checks, its content is read. The tool determines the appropriate language for the Markdown code block (e.g., `.go` becomes `go`, `Dockerfile` becomes `dockerfile`).
5.  **Writing**: The file's relative path and its content, wrapped in a formatted Markdown code block, are written to the output file. This is done using a buffer for efficiency.

## How to Contribute

This is a self-contained project, but improvements are always welcome!

1.  **Add a New Project Type**:
    - Add a new `ProjectConfig` entry to the `projectConfigs` map in `main.go`.
    - Add a landmark file to the `detectProjectType` function.
    - Re-compile.

2.  **Improve Language Detection**:
    - Add new file extensions to `baseLangMap`.
    - Add new extension-less filenames to `filenameLangMap`.
