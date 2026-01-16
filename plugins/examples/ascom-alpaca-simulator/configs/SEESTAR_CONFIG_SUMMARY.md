# Seestar Telescope Simulator Configurations

This directory contains ASCOM Alpaca simulator configurations for three Seestar telescope models.

## Models Configured

### Seestar S30
- **Aperture**: 30mm (0.03m)
- **Focal Length**: 150mm (0.15m)
- **Focal Ratio**: f/5
- **Sensor**: Sony IMX662 (STARVIS 2)
- **Resolution**: 1920x1080 (Full HD)
- **Pixel Size**: 2.9µm
- **Sensor Format**: 1/2.8"
- **Max Exposure**: 30 seconds (Alt-Az mode)
- **Filter Wheel**: 3-position (UV/IR Cut, Duo-Band, Dark Field)
- **UniqueID**: a1b2c3d4-e5f6-4a5b-8c9d-111111111111

### Seestar S30 Pro
- **Aperture**: 30mm (0.03m)
- **Focal Length**: 160mm (0.16m)
- **Focal Ratio**: f/5.3
- **Sensor**: Sony IMX585 (telephoto)
- **Resolution**: 3840x2160 (4K)
- **Pixel Size**: 2.9µm
- **Sensor Format**: 1/1.2"
- **Max Exposure**: 30 seconds
- **Filter Wheel**: 3-position (UV/IR Cut, Duo-Band 20nm Hα/30nm OIII, Dark Field)
- **Internal Focuser**: Yes (autofocus)
- **UniqueID**: b2c3d4e5-f6a7-4b5c-9d0e-222222222222

### Seestar S50
- **Aperture**: 50mm (0.05m)
- **Focal Length**: 250mm (0.25m)
- **Focal Ratio**: f/5
- **Sensor**: Sony IMX462
- **Resolution**: 1920x1080 (Full HD)
- **Pixel Size**: 2.9µm
- **Sensor Format**: 1/2.8"
- **Max Exposure**: 30 seconds
- **Filter Wheel**: 3-position (UV/IR Cut, Duo-Band 20nm Hα/30nm OIII, Dark Field)
- **UniqueID**: c3d4e5f6-a7b8-4c5d-0e1f-333333333333

## Device Configurations

### Telescope Configuration
Each telescope includes:
- Alt-Az mount with goto capability
- Tracking enabled
- Can slew, sync, find home, park
- Latitude: 37.824°N
- Longitude: -94.502°W
- Elevation: 306m

### Camera Configuration
- Color sensor (Bayer RGGB)
- No mechanical shutter
- No cooling (ambient temperature)
- Variable gain control
- Fast readout capable
- Live stacking support

###  Filter Wheel Configuration
All Seestar models include internal filter wheels with 3 positions:

**Position 0**: UV/IR Cut Filter
- Standard broadband imaging
- For galaxies, star clusters

**Position 1**: Duo-Band Filter
- Ha (656nm, 20nm bandwidth) + OIII (500nm, 30nm bandwidth)
- For emission nebulae in light-polluted areas

**Position 2**: Dark Field Filter
- Automatically used for dark frame subtraction
- Not user-selectable during imaging

### Focuser Configuration
- Absolute positioning focuser
- Range: 0-3000 steps
- Step size: 2 microns per step
- Initial position: 1500 (mid-range)
- Auto-focus capable (S30 Pro and S50)
- Temperature compensated: No
- Can halt: Yes
- Settle time: 200ms

### Switch Configuration (Dew Heater Control)
**Switch 0**: Dew Heater
- Name: "Dew Heater"
- Description: "Internal Anti-Dew Heater"
- Type: Boolean (On/Off)
- Min: 0 (Off)
- Max: 1 (On)
- Can Write: True
- Initial State: Off

## Directory Structure

```
/tmp/seestar-configs/
├── s30/
│   ├── telescope/v1/instance-0.xml
│   ├── camera/v1/instance-0.xml
│   ├── filterwheel/v1/instance-0.xml
│   ├── focuser/v1/instance-0.xml
│   └── switch/v1/instance-0.xml
├── s30-pro/
│   ├── telescope/v1/instance-0.xml
│   ├── camera/v1/instance-0.xml
│   ├── filterwheel/v1/instance-0.xml
│   ├── focuser/v1/instance-0.xml
│   └── switch/v1/instance-0.xml
└── s50/
    ├── telescope/v1/instance-0.xml
    ├── camera/v1/instance-0.xml
    ├── filterwheel/v1/instance-0.xml
    ├── focuser/v1/instance-0.xml
    └── switch/v1/instance-0.xml
```

## Usage

To use these configurations with the ASCOM Alpaca simulator:

1. Stop the running ASCOM simulator container
2. Copy the appropriate model's configuration directory to the ASCOM config location
3. Restart the ASCOM simulator container

Example for S50:
```bash
docker stop ascom-alpaca-simulator
cp -r /tmp/seestar-configs/s50/* /tmp/ascom-config/alpaca/ascom-alpaca-simulator/
docker start ascom-alpaca-simulator
```

## API Endpoints

Once configured, each device will be available at:

- **Telescope**: `http://localhost:32323/api/v1/telescope/0/`
- **Camera**: `http://localhost:32323/api/v1/camera/0/`
- **Filter Wheel**: `http://localhost:32323/api/v1/filterwheel/0/`
- **Focuser**: `http://localhost:32323/api/v1/focuser/0/`
- **Switch**: `http://localhost:32323/api/v1/switch/0/`

## References

- Seestar S30: https://store.seestar.com/products/seestar-s30-all-in-one-smart-telescope
- Seestar S30 Pro: https://store.seestar.com/products/seestar-s30-pro
- Seestar S50: https://store.seestar.com/products/seestar-s50
- ASCOM Alpaca API: https://ascom-standards.org/AlpacaDeveloper/Index.htm
