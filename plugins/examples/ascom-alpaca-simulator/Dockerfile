# Multi-stage build for ASCOM Alpaca Simulators Plugin
# Builds the ASCOM simulators and adds MQTT health reporting wrapper

# Stage 1: Build ASCOM Alpaca Simulators from submodule
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS ascom-builder
WORKDIR /app

# Copy the ASCOM.Alpaca.Simulators submodule
COPY external/ASCOM.Alpaca.Simulators/NuGet.config .
COPY external/ASCOM.Alpaca.Simulators/license.md .
COPY external/ASCOM.Alpaca.Simulators/*.sln .
COPY external/ASCOM.Alpaca.Simulators/ASCOM.Alpaca.Simulators/*.csproj ./ASCOM.Alpaca.Simulators/
COPY external/ASCOM.Alpaca.Simulators/ASCOM.Alpaca.Razor/*.csproj ./ASCOM.Alpaca.Razor/
COPY external/ASCOM.Alpaca.Simulators/ASCOM.COM.LocalServer/*.csproj ./ASCOM.COM.LocalServer/
COPY external/ASCOM.Alpaca.Simulators/Camera.Simulator/*.csproj ./Camera.Simulator/
COPY external/ASCOM.Alpaca.Simulators/CoverCalibratorSimulator/*.csproj ./CoverCalibratorSimulator/
COPY external/ASCOM.Alpaca.Simulators/DomeSimulator/*.csproj ./DomeSimulator/
COPY external/ASCOM.Alpaca.Simulators/FilterWheelSimulator/*.csproj ./FilterWheelSimulator/
COPY external/ASCOM.Alpaca.Simulators/FocuserSimulator/*.csproj ./FocuserSimulator/
COPY external/ASCOM.Alpaca.Simulators/ObservingConditionsSimulator/*.csproj ./ObservingConditionsSimulator/
COPY external/ASCOM.Alpaca.Simulators/OmniSim.BaseDriver/*.csproj ./OmniSim.BaseDriver/
COPY external/ASCOM.Alpaca.Simulators/OmniSim.SettingsAPIGenerator/*.csproj ./OmniSim.SettingsAPIGenerator/
COPY external/ASCOM.Alpaca.Simulators/OmniSim.Tools/*.csproj ./OmniSim.Tools/
COPY external/ASCOM.Alpaca.Simulators/RotatorSimulator/*.csproj ./RotatorSimulator/
COPY external/ASCOM.Alpaca.Simulators/SafetyMonitorSimulator/*.csproj ./SafetyMonitorSimulator/
COPY external/ASCOM.Alpaca.Simulators/SwitchSimulator/*.csproj ./SwitchSimulator/
COPY external/ASCOM.Alpaca.Simulators/TelescopeSimulator/*.csproj ./TelescopeSimulator/
COPY external/ASCOM.Alpaca.Simulators/OmniSimCOMProxy/*.csproj ./OmniSimCOMProxy/
COPY external/ASCOM.Alpaca.Simulators/WindowsBase.Vector/*.sln ./WindowsBase.Vector/
COPY external/ASCOM.Alpaca.Simulators/WindowsBase.Vector/WindowsBase.Vector/*.csproj ./WindowsBase.Vector/WindowsBase.Vector/

RUN dotnet restore

# Copy everything else and build
COPY external/ASCOM.Alpaca.Simulators/ .
WORKDIR /app/ASCOM.Alpaca.Simulators
RUN dotnet publish -c Release -o out

# Stage 2: Build Go health reporter
FROM golang:1.24-alpine AS health-reporter-builder
WORKDIR /app
COPY plugins/examples/ascom-alpaca-simulator/health-reporter/ .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o health-reporter .

# Stage 3: Build Go config service
FROM golang:1.24-alpine AS config-service-builder
WORKDIR /app
COPY plugins/examples/ascom-alpaca-simulator/config-service/ .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o config-service .

# Stage 4: Build Go state publisher
FROM golang:1.24-alpine AS state-publisher-builder
WORKDIR /app
COPY plugins/examples/ascom-alpaca-simulator/state-publisher/ .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o state-publisher .

# Stage 5: Runtime image
FROM mcr.microsoft.com/dotnet/aspnet:8.0 AS runtime
WORKDIR /app

# Install required packages
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy ASCOM Alpaca Simulators
COPY --from=ascom-builder /app/ASCOM.Alpaca.Simulators/out ./ascom/

# Copy health reporter
COPY --from=health-reporter-builder /app/health-reporter /usr/local/bin/

# Copy config service
COPY --from=config-service-builder /app/config-service /usr/local/bin/

# Copy state publisher
COPY --from=state-publisher-builder /app/state-publisher /usr/local/bin/

# Copy configuration files
COPY plugins/examples/ascom-alpaca-simulator/configs/ /app/configs/

# Copy startup script
COPY plugins/examples/ascom-alpaca-simulator/start.sh /app/
RUN chmod +x /app/start.sh

# Expose ports
# Port 80 for ASCOM Alpaca API and web UI
# Port 32227 for ASCOM Discovery
EXPOSE 80 32227/udp

# Environment variables
ENV ASPNETCORE_URLS=http://+:80 \
    MQTT_BROKER=tcp://mqtt-broker:1883 \
    PLUGIN_ID=f7e8d9c6-b5a4-3210-9876-543210fedcba \
    PLUGIN_NAME="ASCOM Alpaca Simulators" \
    LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost/api/v1/management/apiversions || exit 1

# Use startup script that launches both ASCOM and health reporter
ENTRYPOINT ["/app/start.sh"]
