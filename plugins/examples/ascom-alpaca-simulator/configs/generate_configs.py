#!/usr/bin/env python3
"""
Generate ASCOM Alpaca Simulator configurations for Seestar telescopes
with different mount types (Alt/Az, Equatorial, German Equatorial)
"""

import os
import uuid

# Seestar specifications
SEESTAR_SPECS = {
    's30': {
        'name': 'Seestar S30',
        'aperture': 0.03,  # 30mm in meters
        'aperture_area': 0.00070686,  # π * (0.015)^2
        'focal_length': 0.15,  # 150mm in meters
        'sensor': 'Sony IMX662',
        'resolution_x': 1920,
        'resolution_y': 1080,
        'pixel_size': 2.9,  # microns
        'uuid_telescope': 'a1b2c3d4-e5f6-4a5b-8c9d-111111111111',
        'uuid_camera': 's30-cam01-1111-2222-3333-444444444444',
        'uuid_filterwheel': 's30-fw001-1111-2222-3333-444444444444',
        'uuid_focuser': 's30-foc01-1111-2222-3333-444444444444',
        'uuid_switch': 's30-sw001-1111-2222-3333-444444444444',
    },
    's30-pro': {
        'name': 'Seestar S30 Pro',
        'aperture': 0.03,  # 30mm in meters
        'aperture_area': 0.00070686,
        'focal_length': 0.16,  # 160mm in meters
        'sensor': 'Sony IMX585',
        'resolution_x': 3840,
        'resolution_y': 2160,
        'pixel_size': 2.9,  # microns
        'uuid_telescope': 'b2c3d4e5-f6a7-4b5c-9d0e-222222222222',
        'uuid_camera': 's30p-cam01-2222-3333-4444-555555555555',
        'uuid_filterwheel': 's30p-fw001-2222-3333-4444-555555555555',
        'uuid_focuser': 's30p-foc01-2222-3333-4444-555555555555',
        'uuid_switch': 's30p-sw001-2222-3333-4444-555555555555',
    },
    's50': {
        'name': 'Seestar S50',
        'aperture': 0.05,  # 50mm in meters
        'aperture_area': 0.00196349,  # π * (0.025)^2
        'focal_length': 0.25,  # 250mm in meters
        'sensor': 'Sony IMX462',
        'resolution_x': 1920,
        'resolution_y': 1080,
        'pixel_size': 2.9,  # microns
        'uuid_telescope': 'c3d4e5f6-a7b8-4c5d-0e1f-333333333333',
        'uuid_camera': 's50-cam01-3333-4444-5555-666666666666',
        'uuid_filterwheel': 's50-fw001-3333-4444-5555-666666666666',
        'uuid_focuser': 's50-foc01-3333-4444-5555-666666666666',
        'uuid_switch': 's50-sw001-3333-4444-5555-666666666666',
    }
}

# Mount type specifications
MOUNT_TYPES = {
    'altaz': {
        'align_mode': 0,  # Alt-Az
        'can_equatorial': 'True',
        'can_altaz': 'True',
    },
    'equatorial': {
        'align_mode': 1,  # Polar (Equatorial)
        'can_equatorial': 'True',
        'can_altaz': 'False',
    },
    'german-equatorial': {
        'align_mode': 2,  # German Polar
        'can_equatorial': 'True',
        'can_altaz': 'False',
    }
}

# Location (Kansas City area from existing config)
LOCATION = {
    'latitude': 37.82439566666667,
    'longitude': -94.50201666666666,
    'elevation': 306
}


def create_telescope_config(model, mount_type, specs):
    """Generate telescope configuration XML"""
    mount = MOUNT_TYPES[mount_type]
    
    return f"""﻿<?xml version="1.0" encoding="utf-8"?>
<ArrayOfSettingsPair xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <SettingsPair>
    <Key>RegVer</Key>
    <Value>1</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>AlwaysOnTop</Key>
    <Value>false</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>StartAzimuth</Key>
    <Value>180</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>StartAltitude</Key>
    <Value>38.92139</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>DateDelta</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>UniqueID</Key>
    <Value>{specs['uuid_telescope']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>AutoTrack</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>NoCoordAtPark</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>DiscPark</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>MaxSlewRate</Key>
    <Value>6</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Refraction</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Latitude</Key>
    <Value>{LOCATION['latitude']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Longitude</Key>
    <Value>{LOCATION['longitude']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Elevation</Key>
    <Value>{LOCATION['elevation']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>AlignMode</Key>
    <Value>{mount['align_mode']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Aperture</Key>
    <Value>{specs['aperture']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ApertureArea</Key>
    <Value>{specs['aperture_area']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FocalLength</Key>
    <Value>{specs['focal_length']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>StartUpMode</Key>
    <Value>Start up at configured Park Position</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanAltAz</Key>
    <Value>{mount['can_altaz']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSlewAltAz</Key>
    <Value>{mount['can_altaz']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSyncAltAz</Key>
    <Value>{mount['can_altaz']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSlewAltAzAsync</Key>
    <Value>{mount['can_altaz']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanEquatorial</Key>
    <Value>{mount['can_equatorial']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSlew</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSync</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSlewAsync</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanFindHome</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanPark</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSetPark</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanUnpark</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>NumMoveAxis</Key>
    <Value>2</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanLatLongElev</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanOptics</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanAlignMode</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanPointingState</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSetPointingState</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanDestinationSideOfPier</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanDoesRefraction</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>EquatorialSystem</Key>
    <Value>2</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanDateTime</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSiderealTime</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSetTracking</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanTrackingRates</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSetGuideRates</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanPulseGuide</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanDualAxisPulseGuide</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSetEquRates</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>NoSyncPastMeridian</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>V1</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>HomeAzimuth</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>HomeAltitude</Key>
    <Value>90</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ParkAzimuth</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ParkAltitude</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>StartAzimuthConfigured</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>StartAltitudeConfigured</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ShutdownAzimuth</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ShutdownAltitude</Key>
    <Value>0</Value>
  </SettingsPair>
</ArrayOfSettingsPair>"""


def create_camera_config(specs):
    """Generate camera configuration XML"""
    return f"""﻿<?xml version="1.0" encoding="utf-8"?>
<ArrayOfSettingsPair xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <SettingsPair>
    <Key>UniqueID</Key>
    <Value>{specs['uuid_camera']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>InterfaceVersion</Key>
    <Value>3</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>PixelSizeX</Key>
    <Value>{specs['pixel_size']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>PixelSizeY</Key>
    <Value>{specs['pixel_size']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FullWellCapacity</Key>
    <Value>28000</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>MaxADU</Key>
    <Value>4095</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ElectronsPerADU</Key>
    <Value>1.0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CameraXSize</Key>
    <Value>{specs['resolution_x']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CameraYSize</Key>
    <Value>{specs['resolution_y']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanAsymmetricBin</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>MaxBinX</Key>
    <Value>1</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>MaxBinY</Key>
    <Value>1</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>HasShutter</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>SensorName</Key>
    <Value>{specs['sensor']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>SensorType</Key>
    <Value>2</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>BayerOffsetX</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>BayerOffsetY</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>OmitOddBins</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>HasCooler</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanSetCCDTemperature</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanGetCoolerPower</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanAbortExposure</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanStopExposure</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>MaxExposure</Key>
    <Value>30</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>MinExposure</Key>
    <Value>0.001</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ExposureResolution</Key>
    <Value>0.001</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ImageFile</Key>
    <Value>/app/m42-800x600.jpg</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ApplyNoise</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanPulseGuide</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>GainMode</Key>
    <Value>2</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Gain</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>GainMin</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>GainMax</Key>
    <Value>100</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Gains</Key>
    <Value>0dB,6dB,12dB,18dB,24dB</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>OffsetMode</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Offset</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>OffsetMin</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>OffsetMax</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>HasSubExposure</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>SubExposureInterval</Key>
    <Value>10</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Offsets</Key>
    <Value>0,0,0,0,0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanFastReadout</Key>
    <Value>True</Value>
  </SettingsPair>
</ArrayOfSettingsPair>"""


def create_filterwheel_config(specs):
    """Generate filter wheel configuration XML"""
    return f"""﻿<?xml version="1.0" encoding="utf-8"?>
<ArrayOfSettingsPair xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <SettingsPair>
    <Key>UniqueID</Key>
    <Value>{specs['uuid_filterwheel']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>RegVer</Key>
    <Value>1</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Position</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Slots</Key>
    <Value>3</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FilterChangeTimeInterval</Key>
    <Value>1000</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FilterNames 0</Key>
    <Value>UV/IR Cut</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FocusOffsets 0</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Filter 0 Color</Key>
    <Value>White</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FilterNames 1</Key>
    <Value>Duo-Band (Ha/OIII)</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FocusOffsets 1</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Filter 1 Color</Key>
    <Value>DarkRed</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FilterNames 2</Key>
    <Value>Dark Field</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>FocusOffsets 2</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Filter 2 Color</Key>
    <Value>Black</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ImplementsNames</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>ImplementsOffsets</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>PreemptMoves</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>InterfaceVersion</Key>
    <Value>3</Value>
  </SettingsPair>
</ArrayOfSettingsPair>"""


def create_focuser_config(specs):
    """Generate focuser configuration XML"""
    return f"""﻿<?xml version="1.0" encoding="utf-8"?>
<ArrayOfSettingsPair xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <SettingsPair>
    <Key>UniqueID</Key>
    <Value>{specs['uuid_focuser']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Absolute</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>MaxIncrement</Key>
    <Value>1000</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>MaxStep</Key>
    <Value>3000</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Position</Key>
    <Value>1500</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>StepSize</Key>
    <Value>2</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>TempComp</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>TempCompAvailable</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Temperature</Key>
    <Value>20</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanHalt</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanStepSize</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>TempMax</Key>
    <Value>50</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>TempMin</Key>
    <Value>-50</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>TempPeriod</Key>
    <Value>3</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>TempProbe</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>TempSteps</Key>
    <Value>5</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>SettleTime</Key>
    <Value>200</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>InterfaceVersion</Key>
    <Value>4</Value>
  </SettingsPair>
</ArrayOfSettingsPair>"""


def create_switch_config(specs):
    """Generate switch configuration XML (dew heater)"""
    return f"""﻿<?xml version="1.0" encoding="utf-8"?>
<ArrayOfSettingsPair xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <SettingsPair>
    <Key>UniqueID</Key>
    <Value>{specs['uuid_switch']}</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>NumSwitches</Key>
    <Value>1</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Name Switch0</Key>
    <Value>Dew Heater</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Description Switch0</Key>
    <Value>Internal Anti-Dew Heater</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Minimum Switch0</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Maximum Switch0</Key>
    <Value>1</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>StepSize Switch0</Key>
    <Value>1</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanWrite Switch0</Key>
    <Value>True</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Value Switch0</Key>
    <Value>0</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>CanAsync Switch0</Key>
    <Value>False</Value>
  </SettingsPair>
  <SettingsPair>
    <Key>Duration Switch0</Key>
    <Value>0</Value>
  </SettingsPair>
</ArrayOfSettingsPair>"""


def main():
    """Generate all configuration files"""
    base_path = "/tmp/seestar-configs"
    
    for model, specs in SEESTAR_SPECS.items():
        print(f"Generating configurations for {specs['name']}...")
        
        for mount_type in MOUNT_TYPES.keys():
            print(f"  - {mount_type} mount")
            mount_path = os.path.join(base_path, model, mount_type)
            
            # Telescope config
            telescope_file = os.path.join(mount_path, "telescope/v1/instance-0.xml")
            with open(telescope_file, 'w') as f:
                f.write(create_telescope_config(model, mount_type, specs))
            
            # Camera config
            camera_file = os.path.join(mount_path, "camera/v1/instance-0.xml")
            with open(camera_file, 'w') as f:
                f.write(create_camera_config(specs))
            
            # Filter wheel config
            filterwheel_file = os.path.join(mount_path, "filterwheel/v1/instance-0.xml")
            with open(filterwheel_file, 'w') as f:
                f.write(create_filterwheel_config(specs))
            
            # Focuser config
            focuser_file = os.path.join(mount_path, "focuser/v1/instance-0.xml")
            with open(focuser_file, 'w') as f:
                f.write(create_focuser_config(specs))
            
            # Switch config
            switch_file = os.path.join(mount_path, "switch/v1/instance-0.xml")
            with open(switch_file, 'w') as f:
                f.write(create_switch_config(specs))
    
    print("\n✓ All configurations generated successfully!")
    print(f"\nConfigurations saved to: {base_path}")
    print("\nStructure:")
    print("  s30/")
    print("  ├── altaz/")
    print("  ├── equatorial/")
    print("  └── german-equatorial/")
    print("  s30-pro/")
    print("  ├── altaz/")
    print("  ├── equatorial/")
    print("  └── german-equatorial/")
    print("  s50/")
    print("  ├── altaz/")
    print("  ├── equatorial/")
    print("  └── german-equatorial/")


if __name__ == "__main__":
    main()
