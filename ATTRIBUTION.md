# Attribution and License Information

This document lists all third-party libraries, modules, and dependencies used in the BIG_SKIES_FRAMEWORK project, along with their respective licenses.

## Go Dependencies

### Direct Dependencies

#### Eclipse Paho MQTT Go Client
- **Package**: github.com/eclipse/paho.mqtt.golang
- **Version**: v1.5.1
- **License**: Eclipse Public License 2.0 (EPL-2.0) / Eclipse Distribution License 1.0 (EDL-1.0)
- **Purpose**: MQTT client library for message bus communication
- **Repository**: https://github.com/eclipse/paho.mqtt.golang

#### Gin Web Framework
- **Package**: github.com/gin-gonic/gin
- **Version**: v1.11.0
- **License**: MIT License
- **Purpose**: HTTP web framework for REST APIs
- **Repository**: https://github.com/gin-gonic/gin

#### JWT (JSON Web Tokens)
- **Package**: github.com/golang-jwt/jwt/v5
- **Version**: v5.3.0
- **License**: MIT License
- **Purpose**: JWT token generation and validation for authentication
- **Repository**: https://github.com/golang-jwt/jwt

#### PostgreSQL Driver (pgx)
- **Package**: github.com/jackc/pgx/v5
- **Version**: v5.8.0
- **License**: MIT License
- **Purpose**: PostgreSQL database driver and toolkit
- **Repository**: https://github.com/jackc/pgx

#### Google UUID
- **Package**: github.com/google/uuid
- **Version**: v1.6.0
- **License**: BSD 3-Clause License
- **Purpose**: UUID generation and parsing
- **Repository**: https://github.com/google/uuid

#### Viper Configuration
- **Package**: github.com/spf13/viper
- **Version**: v1.21.0
- **License**: MIT License
- **Purpose**: Configuration management with support for multiple formats
- **Repository**: https://github.com/spf13/viper

#### Zap Logger
- **Package**: go.uber.org/zap
- **Version**: v1.27.1
- **License**: MIT License
- **Purpose**: High-performance structured logging
- **Repository**: https://github.com/uber-go/zap

#### Docker Client
- **Package**: github.com/moby/moby/client
- **Version**: v0.2.1
- **License**: Apache License 2.0
- **Purpose**: Docker API client for container management
- **Repository**: https://github.com/moby/moby

#### Testify
- **Package**: github.com/stretchr/testify
- **Version**: v1.11.1
- **License**: MIT License
- **Purpose**: Testing assertions and mocking
- **Repository**: https://github.com/stretchr/testify

### Indirect Dependencies

#### Networking and Protocol Libraries
- **golang.org/x/crypto** (v0.47.0) - BSD 3-Clause License - Cryptographic libraries
- **golang.org/x/net** (v0.48.0) - BSD 3-Clause License - Network libraries
- **golang.org/x/sys** (v0.40.0) - BSD 3-Clause License - System call interfaces
- **golang.org/x/text** (v0.33.0) - BSD 3-Clause License - Text processing libraries
- **github.com/gorilla/websocket** (v1.5.3) - BSD 2-Clause License - WebSocket implementation

#### Container and Docker Libraries
- **github.com/docker/go-connections** (v0.6.0) - Apache License 2.0
- **github.com/docker/go-units** (v0.5.0) - Apache License 2.0
- **github.com/distribution/reference** (v0.6.0) - Apache License 2.0
- **github.com/opencontainers/image-spec** (v1.1.1) - Apache License 2.0
- **github.com/containerd/errdefs** (v1.0.0) - Apache License 2.0
- **github.com/moby/docker-image-spec** (v1.3.1) - Apache License 2.0

#### JSON and Data Processing
- **github.com/bytedance/sonic** (v1.14.0) - Apache License 2.0 - High-performance JSON library
- **github.com/goccy/go-json** (v0.10.2) - MIT License - JSON encoding/decoding
- **github.com/json-iterator/go** (v1.1.12) - MIT License - Fast JSON library

#### Configuration and File Systems
- **github.com/spf13/afero** (v1.15.0) - Apache License 2.0 - Abstract file system
- **github.com/spf13/cast** (v1.10.0) - MIT License - Type conversion
- **github.com/spf13/pflag** (v1.0.10) - BSD 3-Clause License - POSIX-style command-line flags
- **github.com/fsnotify/fsnotify** (v1.9.0) - BSD 3-Clause License - File system notifications
- **github.com/pelletier/go-toml/v2** (v2.2.4) - MIT License - TOML parser
- **github.com/goccy/go-yaml** (v1.18.0) - MIT License - YAML parser

#### Validation and Utilities
- **github.com/go-playground/validator/v10** (v10.27.0) - MIT License - Struct validation
- **github.com/gabriel-vasile/mimetype** (v1.4.8) - MIT License - MIME type detection
- **github.com/go-viper/mapstructure/v2** (v2.4.0) - MIT License - Struct decoding

#### Database and Connection Pooling
- **github.com/jackc/pgpassfile** (v1.0.0) - MIT License
- **github.com/jackc/pgservicefile** (v0.0.0-20240606120523-5a60cdf6a761) - MIT License
- **github.com/jackc/puddle/v2** (v2.2.2) - MIT License - Connection pooling

#### Concurrency and Error Handling
- **go.uber.org/multierr** (v1.10.0) - MIT License - Multiple error handling
- **github.com/sourcegraph/conc** (v0.3.1-0.20240121214520-5f936abd7ae8) - MIT License - Concurrency utilities
- **golang.org/x/sync** (v0.19.0) - BSD 3-Clause License

#### Observability (OpenTelemetry)
- **go.opentelemetry.io/otel** (v1.35.0) - Apache License 2.0
- **go.opentelemetry.io/otel/trace** (v1.35.0) - Apache License 2.0
- **go.opentelemetry.io/otel/metric** (v1.35.0) - Apache License 2.0
- **go.opentelemetry.io/otel/sdk** (v1.35.0) - Apache License 2.0
- **go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp** (v0.60.0) - Apache License 2.0
- **github.com/felixge/httpsnoop** (v1.0.4) - MIT License

#### Gin Framework Dependencies
- **github.com/gin-contrib/sse** (v1.1.0) - MIT License - Server-sent events
- **github.com/go-playground/locales** (v0.14.1) - MIT License
- **github.com/go-playground/universal-translator** (v0.18.1) - MIT License
- **github.com/leodido/go-urn** (v1.4.0) - MIT License
- **github.com/mattn/go-isatty** (v0.0.20) - MIT License
- **github.com/ugorji/go/codec** (v1.3.0) - MIT License
- **github.com/cloudwego/base64x** (v0.1.6) - Apache License 2.0
- **github.com/klauspost/cpuid/v2** (v2.3.0) - MIT License
- **github.com/twitchyliquid64/golang-asm** (v0.15.1) - BSD 2-Clause License
- **golang.org/x/arch** (v0.20.0) - BSD 3-Clause License

#### QUIC Protocol Support
- **github.com/quic-go/quic-go** (v0.54.0) - MIT License
- **github.com/quic-go/qpack** (v0.5.1) - MIT License

#### Testing and Development Tools
- **github.com/davecgh/go-spew** (v1.1.1) - ISC License - Pretty-printing for debugging
- **github.com/pmezard/go-difflib** (v1.0.0) - BSD 3-Clause License - Diff algorithms
- **go.uber.org/mock** (v0.5.0) - Apache License 2.0 - Mock generation
- **go.uber.org/goleak** (v1.3.0) - MIT License - Goroutine leak detection
- **github.com/kr/pretty** (v0.3.1) - MIT License - Pretty printing
- **github.com/kr/text** (v0.2.0) - MIT License - Text utilities
- **github.com/google/go-cmp** (v0.7.0) - BSD 3-Clause License - Comparison utilities
- **github.com/frankban/quicktest** (v1.14.6) - MIT License
- **github.com/rogpeppe/go-internal** (v1.13.1) - BSD 3-Clause License

#### Protobuf and gRPC
- **google.golang.org/protobuf** (v1.36.9) - BSD 3-Clause License
- **github.com/golang/protobuf** (v1.5.0) - BSD 3-Clause License
- **google.golang.org/grpc** (v1.67.0) - Apache License 2.0
- **github.com/gogo/protobuf** (v1.3.2) - BSD 3-Clause License

#### Other Utilities
- **github.com/Microsoft/go-winio** (v0.6.2) - MIT License - Windows I/O
- **github.com/creack/pty** (v1.1.24) - MIT License - PTY interface
- **github.com/moby/term** (v0.5.2) - Apache License 2.0 - Terminal utilities
- **github.com/go-logr/logr** (v1.4.2) - Apache License 2.0
- **github.com/go-logr/stdr** (v1.2.2) - Apache License 2.0
- **golang.org/x/mod** (v0.31.0) - BSD 3-Clause License - Module utilities
- **golang.org/x/tools** (v0.40.0) - BSD 3-Clause License - Go tools

## Python Dependencies

### Direct Dependencies

#### Paho MQTT Python Client
- **Package**: paho-mqtt
- **Version**: >= 1.6.1
- **License**: Eclipse Public License 2.0 (EPL-2.0) / Eclipse Distribution License 1.0 (EDL-1.0)
- **Purpose**: MQTT client library for Python GTK UI
- **Repository**: https://github.com/eclipse/paho.mqtt.python

#### Python dotenv
- **Package**: python-dotenv
- **Version**: >= 0.19.0
- **License**: BSD 3-Clause License
- **Purpose**: Environment variable management
- **Repository**: https://github.com/theskumar/python-dotenv

### System Dependencies (Python GTK UI)

#### PyGObject (Python GObject Introspection)
- **Package**: PyGObject / python3-gi
- **License**: GNU LGPL v2.1+
- **Purpose**: Python bindings for GObject introspection
- **Website**: https://pygobject.readthedocs.io/

#### GTK+ 3
- **Package**: GTK 3 (gir1.2-gtk-3.0)
- **License**: GNU LGPL v2.1+
- **Purpose**: GUI toolkit for the Python GTK application
- **Website**: https://www.gtk.org/

#### Cairo
- **Package**: python3-gi-cairo
- **License**: GNU LGPL v2.1+ / MPL 1.1
- **Purpose**: 2D graphics library for telescope preview rendering
- **Website**: https://www.cairographics.org/

## Plugin Dependencies

### ASCOM Alpaca Simulator Plugin

The ASCOM Alpaca Simulator plugin uses separate Go modules with the following dependencies:

#### Config Service
- **github.com/eclipse/paho.mqtt.golang** (v1.4.3) - EPL-2.0 / EDL-1.0
- **github.com/gorilla/websocket** (v1.5.0) - BSD 2-Clause License
- **golang.org/x/net** (v0.8.0) - BSD 3-Clause License
- **golang.org/x/sync** (v0.1.0) - BSD 3-Clause License

#### State Publisher
- **github.com/eclipse/paho.mqtt.golang** (v1.5.0) - EPL-2.0 / EDL-1.0
- **github.com/gorilla/websocket** (v1.5.3) - BSD 2-Clause License
- **golang.org/x/net** (v0.27.0) - BSD 3-Clause License
- **golang.org/x/sync** (v0.7.0) - BSD 3-Clause License

## External Integration

### ASCOM Alpaca API
- **Project**: ASCOM Initiative
- **License**: Creative Commons Attribution-ShareAlike 4.0 International (for specifications)
- **Purpose**: Telescope control and astronomy device interface
- **Website**: https://ascom-standards.org/
- **Repository**: https://github.com/ASCOMInitiative

## License Summary

The BIG_SKIES_FRAMEWORK uses dependencies under the following license families:

### Permissive Licenses
- **MIT License**: Majority of Go dependencies
- **BSD 2-Clause & 3-Clause Licenses**: Networking, system libraries, and utilities
- **Apache License 2.0**: Docker, container libraries, and observability tools
- **ISC License**: Limited use (go-spew)

### Copyleft Licenses
- **GNU LGPL v2.1+**: GTK and PyGObject (Python UI only)
- **Eclipse Public License 2.0 / Eclipse Distribution License 1.0**: MQTT client libraries

### Dual Licenses
- **MPL 1.1 / LGPL v2.1+**: Cairo graphics library

## Compliance Notes

1. All dependencies are compatible with the project's license requirements
2. LGPL dependencies (GTK, PyGObject) are dynamically linked, maintaining license compliance
3. EPL/EDL licensed MQTT libraries are used as-is without modification
4. Apache 2.0, MIT, and BSD licensed components permit commercial and open-source use
5. All third-party licenses are preserved in their respective package repositories

## Attribution

This project gratefully acknowledges the contributions of all open-source maintainers and communities whose work makes this framework possible.

For specific license texts, please refer to:
- Go dependencies: See individual package repositories or `go.mod` entries
- Python dependencies: https://pypi.org/ (package pages)
- System libraries: Distribution package documentation

Last Updated: 2026-01-16

Co-Authored-By: Warp <agent@warp.dev>
