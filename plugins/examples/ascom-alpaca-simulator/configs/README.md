# Seestar Telescope ASCOM Alpaca Simulator Configurations

Complete ASCOM Alpaca simulator configurations for Seestar S30, S30 Pro, and S50 telescopes with three mount types: Alt/Az, Equatorial, and German Equatorial.

## Generated Configurations

✓ **45 configuration files** created (3 models × 3 mount types × 5 devices)

### Models
- **Seestar S30**: 30mm aperture, 150mm focal length, f/5, Sony IMX662 sensor
- **Seestar S30 Pro**: 30mm aperture, 160mm focal length, f/5.3, Sony IMX585 sensor (4K)
- **Seestar S50**: 50mm aperture, 250mm focal length, f/5, Sony IMX462 sensor

### Mount Types
1. **Alt/Az** (AlignMode=0): Altitude-Azimuth mount, can use both Alt/Az and Equatorial coordinates
2. **Equatorial** (AlignMode=1): Polar-aligned equatorial mount, equatorial coordinates only
3. **German Equatorial** (AlignMode=2): German equatorial mount with meridian flip, equatorial only

### Devices Configured
Each configuration includes:
- **Telescope**: Mount with GOTO, tracking, parking
- **Camera**: Color CMOS sensor with gain control
- **Filter Wheel**: 3-position (UV/IR Cut, Duo-Band Ha/OIII, Dark Field)
- **Focuser**: Motorized focuser with 3000 steps
- **Switch**: Dew heater control

## Directory Structure

```
/tmp/seestar-configs/
├── s30/
│   ├── altaz/
│   │   ├── telescope/v1/instance-0.xml
│   │   ├── camera/v1/instance-0.xml
│   │   ├── filterwheel/v1/instance-0.xml
│   │   ├── focuser/v1/instance-0.xml
│   │   └── switch/v1/instance-0.xml
│   ├── equatorial/
│   │   └── [same structure]
│   └── german-equatorial/
│       └── [same structure]
├── s30-pro/
│   └── [same structure as s30]
├── s50/
│   └── [same structure as s30]
├── generate_configs.py
├── SEESTAR_CONFIG_SUMMARY.md
└── README.md (this file)
```

## Usage

### 1. Choose Your Configuration

Select the model and mount type you want to simulate:

```bash
# For Seestar S50 with Alt/Az mount
MODEL=s50
MOUNT=altaz

# For Seestar S30 Pro with German Equatorial mount
MODEL=s30-pro
MOUNT=german-equatorial

# For Seestar S30 with Equatorial mount
MODEL=s30
MOUNT=equatorial
```

### 2. Deploy to ASCOM Simulator

Copy the chosen configuration to your ASCOM simulator:

```bash
# Stop the running simulator
docker stop ascom-alpaca-simulator

# Copy configuration files
cp -r /tmp/seestar-configs/${MODEL}/${MOUNT}/* /tmp/ascom-config/alpaca/ascom-alpaca-simulator/

# Start the simulator
docker start ascom-alpaca-simulator

# Wait for startup
sleep 5

# Check logs
docker logs ascom-alpaca-simulator --tail 50
```

### 3. Verify Configuration

Test the telescope API:

```bash
# Get telescope aperture
curl -s "http://localhost:32323/api/v1/telescope/0/aperture" | python3 -m json.tool

# Get camera resolution
curl -s "http://localhost:32323/api/v1/camera/0/cameraxsize" | python3 -m json.tool
curl -s "http://localhost:32323/api/v1/camera/0/cameraysize" | python3 -m json.tool

# Get filter wheel names
curl -s "http://localhost:32323/api/v1/filterwheel/0/names" | python3 -m json.tool

# Get focuser position
curl -s "http://localhost:32323/api/v1/focuser/0/position" | python3 -m json.tool

# Check dew heater status
curl -s "http://localhost:32323/api/v1/switch/0/getswitch?Id=0" | python3 -m json.tool
```

## Device Specifications

### Seestar S30
- **Optical**: 30mm f/5 APO triplet, 150mm focal length
- **Camera**: Sony IMX662, 1920×1080, 2.9µm pixels
- **Sensor Format**: 1/2.8" (6.46mm diagonal)
- **Field of View**: 2.46° (telephoto)
- **Max Exposure**: 30 seconds
- **Weight**: 1.65 kg

### Seestar S30 Pro
- **Optical**: 30mm f/5.3 APO quadruplet, 160mm focal length
- **Camera**: Sony IMX585, 3840×2160 (4K), 2.9µm pixels
- **Sensor Format**: 1/1.2" (larger sensor)
- **Internal Filters**: UV/IR Cut, Duo-Band (20nm Hα + 30nm OIII)
- **Max Exposure**: 30 seconds
- **Autofocus**: Yes
- **Weight**: 1.65 kg

### Seestar S50
- **Optical**: 50mm f/5 APO triplet, 250mm focal length
- **Camera**: Sony IMX462, 1920×1080, 2.9µm pixels
- **Sensor Format**: 1/2.8"
- **Field of View**: 1.29° × 0.73°
- **Max Exposure**: 30 seconds
- **Internal Filters**: UV/IR Cut, Duo-Band (20nm Hα + 30nm OIII)
- **Weight**: 2.5 kg

## Mount Type Details

### Alt/Az (Altitude-Azimuth)
- **Use Case**: Terrestrial viewing, casual astronomy, quick setup
- **Tracking**: Software-compensated Alt/Az tracking
- **Field Rotation**: Yes (limits long exposures to ~30 seconds)
- **Meridian Flip**: Not required
- **Polar Alignment**: Not required
- **ASCOM Setting**: `AlignMode=0`, `CanAltAz=True`, `CanEquatorial=True`

### Equatorial (Polar-Aligned)
- **Use Case**: Deep-sky astrophotography, longer exposures
- **Tracking**: True sidereal tracking
- **Field Rotation**: None
- **Meridian Flip**: Optional (depends on mount design)
- **Polar Alignment**: Required
- **ASCOM Setting**: `AlignMode=1`, `CanAltAz=False`, `CanEquatorial=True`

### German Equatorial (GEM)
- **Use Case**: Serious astrophotography, observatory setups
- **Tracking**: Precision sidereal tracking
- **Field Rotation**: None
- **Meridian Flip**: Required (automatic or manual)
- **Polar Alignment**: Critical for best performance
- **ASCOM Setting**: `AlignMode=2`, `CanAltAz=False`, `CanEquatorial=True`

## Filter Wheel Details

All configurations include a 3-position filter wheel:

| Position | Filter Name | Description | Use Case |
|----------|-------------|-------------|----------|
| 0 | UV/IR Cut | Standard broadband filter | Galaxies, star clusters, planets |
| 1 | Duo-Band (Hα/OIII) | Narrowband 20nm Hα + 30nm OIII | Emission nebulae, light pollution rejection |
| 2 | Dark Field | Opaque filter for dark frames | Calibration, noise reduction |

## API Examples

### Control Telescope
```bash
# Slew to coordinates (M42 Orion Nebula)
curl -X PUT "http://localhost:32323/api/v1/telescope/0/slewtocoordinatesasync" \
  -d "RightAscension=5.583&Declination=-5.391&ClientID=1&ClientTransactionID=1"

# Park telescope
curl -X PUT "http://localhost:32323/api/v1/telescope/0/park" \
  -d "ClientID=1&ClientTransactionID=1"

# Start tracking
curl -X PUT "http://localhost:32323/api/v1/telescope/0/tracking" \
  -d "Tracking=true&ClientID=1&ClientTransactionID=1"
```

### Control Camera
```bash
# Start 10-second exposure
curl -X PUT "http://localhost:32323/api/v1/camera/0/startexposure" \
  -d "Duration=10&Light=true&ClientID=1&ClientTransactionID=1"

# Check exposure status
curl -s "http://localhost:32323/api/v1/camera/0/camerastateready" | python3 -m json.tool

# Set gain
curl -X PUT "http://localhost:32323/api/v1/camera/0/gain" \
  -d "Gain=50&ClientID=1&ClientTransactionID=1"
```

### Control Filter Wheel
```bash
# Move to Duo-Band filter
curl -X PUT "http://localhost:32323/api/v1/filterwheel/0/position" \
  -d "Position=1&ClientID=1&ClientTransactionID=1"

# Get current filter
curl -s "http://localhost:32323/api/v1/filterwheel/0/position" | python3 -m json.tool
```

### Control Focuser
```bash
# Move to position 2000
curl -X PUT "http://localhost:32323/api/v1/focuser/0/move" \
  -d "Position=2000&ClientID=1&ClientTransactionID=1"

# Get current position
curl -s "http://localhost:32323/api/v1/focuser/0/position" | python3 -m json.tool
```

### Control Dew Heater (Switch)
```bash
# Turn on dew heater
curl -X PUT "http://localhost:32323/api/v1/switch/0/setswitch" \
  -d "Id=0&State=true&ClientID=1&ClientTransactionID=1"

# Check dew heater status
curl -s "http://localhost:32323/api/v1/switch/0/getswitch?Id=0" | python3 -m json.tool
```

## Location Settings

All configurations use the same default location (Kansas City area):
- **Latitude**: 37.824°N
- **Longitude**: 94.502°W
- **Elevation**: 306m

To change location, edit the telescope configuration XML:
```xml
<SettingsPair>
  <Key>Latitude</Key>
  <Value>YOUR_LATITUDE</Value>
</SettingsPair>
<SettingsPair>
  <Key>Longitude</Key>
  <Value>YOUR_LONGITUDE</Value>
</SettingsPair>
<SettingsPair>
  <Key>Elevation</Key>
  <Value>YOUR_ELEVATION_IN_METERS</Value>
</SettingsPair>
```

## Regenerating Configurations

To regenerate all configurations (e.g., after modifying specifications):

```bash
python3 /tmp/seestar-configs/generate_configs.py
```

## Troubleshooting

### Configuration not loading
1. Verify files are in correct location: `/tmp/ascom-config/alpaca/ascom-alpaca-simulator/`
2. Check file permissions: `chmod 644 /tmp/ascom-config/alpaca/ascom-alpaca-simulator/*/v1/instance-0.xml`
3. Restart container: `docker restart ascom-alpaca-simulator`
4. Check logs: `docker logs ascom-alpaca-simulator`

### Wrong mount type showing
- Verify you copied the correct mount directory (altaz/equatorial/german-equatorial)
- Check AlignMode in telescope/v1/instance-0.xml

### Devices not appearing
- Confirm all 5 device directories exist: telescope, camera, filterwheel, focuser, switch
- Each must have v1/instance-0.xml

## References

- **ASCOM Alpaca API**: https://ascom-standards.org/AlpacaDeveloper/
- **Seestar Official**: https://store.seestar.com/
- **ASCOM Standards**: https://ascom-standards.org/

## Next Steps

After deploying a configuration:
1. Test basic telescope commands (slew, park, track)
2. Verify camera exposures work
3. Test filter wheel changes
4. Check focuser movement
5. Toggle dew heater on/off
6. Integrate with your astrophotography software (N.I.N.A., SGP, etc.)
