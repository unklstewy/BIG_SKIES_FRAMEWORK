# Blazor to WPF Control Mapping

## Overview
This document provides a comprehensive mapping of UI elements from the ASCOM Alpaca Simulators Blazor interface to Windows Presentation Foundation (WPF) controls, designed for integration with the BigSkies framework's UI element coordinator.

## Core UI Element Mappings

### Layout Containers

| Blazor Element | WPF Control | BigSkies Element Type | Notes |
|----------------|-------------|----------------------|-------|
| `<fieldset>` | `GroupBox` | `panel` | Group box with header |
| `<legend>` | GroupBox `Header` | N/A | Set via Header property |
| `<div class="grid-container-two">` | `Grid` with 2 columns | `panel` | Define `ColumnDefinitions` |
| `<div class="grid-item-left">` | Grid column 0 | N/A | `Grid.Column="0"` |
| `<div class="grid-item-right">` | Grid column 1 | N/A | `Grid.Column="1"` |
| `<div class="centered">` | `StackPanel` with `HorizontalAlignment="Center"` | `panel` | Center-aligned horizontal layout |
| `<body>` | `Window` or `UserControl` | `panel` | Main container |

### Input Controls

| Blazor Element | WPF Control | BigSkies Element Type | Notes |
|----------------|-------------|----------------------|-------|
| `<button>` | `Button` | `widget` | Handle `Click` event |
| `<input type="checkbox">` | `CheckBox` | `widget` | Bind to `IsChecked` property |
| `<input type="number">` | `TextBox` with validation or custom control | `widget` | Use numeric validation |
| `<input type="text">` | `TextBox` | `widget` | Single-line text input |
| `<select>` | `ComboBox` | `widget` | Dropdown selection |
| `<option>` | `ComboBoxItem` | N/A | Items in ComboBox |

### Display Controls

| Blazor Element | WPF Control | BigSkies Element Type | Notes |
|----------------|-------------|----------------------|-------|
| `<label>` | `Label` or `TextBlock` | `widget` | Static text display |
| `<p>` | `TextBlock` with `TextWrapping` | `widget` | Paragraph text |
| `<h2>`, `<h3>` | `TextBlock` with larger `FontSize` | `widget` | Use styles for sizing |
| `<svg>` (status circle) | `Ellipse` with binding | `widget` | Bind `Fill` property |
| Dynamic text binding | `TextBlock` with binding | N/A | Use `{Binding}` syntax |

## ASCOM Telescope Control UI Mapping

### XAML Definition
```xml
<UserControl x:Class="BigSkies.TelescopeControlView"
             xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
             xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml"
             xmlns:d="http://schemas.microsoft.com/expression/blend/2008"
             xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"
             mc:Ignorable="d">
    <GroupBox Header="Telescope" Margin="10" Padding="10">
        <Grid>
            <Grid.ColumnDefinitions>
                <ColumnDefinition Width="Auto"/>
                <ColumnDefinition Width="*"/>
                <ColumnDefinition Width="Auto"/>
            </Grid.ColumnDefinitions>
            
            <!-- Connection Status Indicator -->
            <Ellipse Grid.Column="0" 
                     Width="30" Height="30"
                     Stroke="Black" StrokeThickness="3"
                     Fill="{Binding ConnectionColor}"/>
            
            <!-- Connect Button -->
            <Button Grid.Column="1" 
                    Content="{Binding ConnectButtonText}"
                    Command="{Binding ToggleConnectionCommand}"
                    Margin="10,0"/>
            
            <!-- Setup Button -->
            <Button Grid.Column="2"
                    Content="Setup"
                    Command="{Binding NavigateToSetupCommand}"/>
        </Grid>
    </GroupBox>
</UserControl>
```

### View Model (C#)
```csharp
using System;
using System.ComponentModel;
using System.Windows.Input;
using System.Windows.Media;
using MQTTnet;
using MQTTnet.Client;
using Newtonsoft.Json;

namespace BigSkies.ViewModels
{
    public class TelescopeControlViewModel : INotifyPropertyChanged
    {
        private readonly IMqttClient _mqttClient;
        private bool _isConnected;

        public TelescopeControlViewModel()
        {
            _mqttClient = new MqttFactory().CreateMqttClient();
            InitializeMqtt();
            
            ToggleConnectionCommand = new RelayCommand(ToggleConnection);
            NavigateToSetupCommand = new RelayCommand(NavigateToSetup);
        }

        #region Properties

        public bool IsConnected
        {
            get => _isConnected;
            set
            {
                if (_isConnected != value)
                {
                    _isConnected = value;
                    OnPropertyChanged(nameof(IsConnected));
                    OnPropertyChanged(nameof(ConnectButtonText));
                    OnPropertyChanged(nameof(ConnectionColor));
                }
            }
        }

        public string ConnectButtonText => IsConnected ? "Disconnect" : "Connect";

        public SolidColorBrush ConnectionColor => 
            IsConnected ? new SolidColorBrush(Colors.Red) : new SolidColorBrush(Colors.Gray);

        #endregion

        #region Commands

        public ICommand ToggleConnectionCommand { get; }
        public ICommand NavigateToSetupCommand { get; }

        private async void ToggleConnection()
        {
            var payload = JsonConvert.SerializeObject(new { action = "toggle" });
            var message = new MqttApplicationMessageBuilder()
                .WithTopic("bigskies/telescope/0/command/connect")
                .WithPayload(payload)
                .Build();

            await _mqttClient.PublishAsync(message);
        }

        private void NavigateToSetup()
        {
            // Navigate to setup view
        }

        #endregion

        #region MQTT

        private async void InitializeMqtt()
        {
            var options = new MqttClientOptionsBuilder()
                .WithTcpServer("localhost", 1883)
                .Build();

            _mqttClient.ApplicationMessageReceivedAsync += OnMqttMessageReceived;

            await _mqttClient.ConnectAsync(options);
            await _mqttClient.SubscribeAsync("bigskies/telescope/0/state/+");
        }

        private async Task OnMqttMessageReceived(MqttApplicationMessageReceivedEventArgs e)
        {
            var topic = e.ApplicationMessage.Topic;
            var payload = System.Text.Encoding.UTF8.GetString(e.ApplicationMessage.Payload);

            if (topic == "bigskies/telescope/0/state/connected")
            {
                var data = JsonConvert.DeserializeObject<dynamic>(payload);
                await System.Windows.Application.Current.Dispatcher.InvokeAsync(() =>
                {
                    IsConnected = (bool)data.connected;
                });
            }
        }

        #endregion

        #region INotifyPropertyChanged

        public event PropertyChangedEventHandler PropertyChanged;

        protected virtual void OnPropertyChanged(string propertyName)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }

        #endregion
    }

    // Helper class for commands
    public class RelayCommand : ICommand
    {
        private readonly Action _execute;
        private readonly Func<bool> _canExecute;

        public RelayCommand(Action execute, Func<bool> canExecute = null)
        {
            _execute = execute ?? throw new ArgumentNullException(nameof(execute));
            _canExecute = canExecute;
        }

        public event EventHandler CanExecuteChanged
        {
            add => CommandManager.RequerySuggested += value;
            remove => CommandManager.RequerySuggested -= value;
        }

        public bool CanExecute(object parameter) => _canExecute?.Invoke() ?? true;
        public void Execute(object parameter) => _execute();
    }
}
```

## Telescope Status Display

### XAML
```xml
<GroupBox Header="Status" Margin="10" Padding="10">
    <Grid>
        <Grid.ColumnDefinitions>
            <ColumnDefinition Width="Auto"/>
            <ColumnDefinition Width="*"/>
        </Grid.ColumnDefinitions>
        <Grid.RowDefinitions>
            <RowDefinition Height="Auto"/>
            <RowDefinition Height="Auto"/>
            <RowDefinition Height="Auto"/>
            <RowDefinition Height="Auto"/>
            <RowDefinition Height="Auto"/>
        </Grid.RowDefinitions>

        <TextBlock Grid.Row="0" Grid.Column="0" Text="LST:" FontWeight="Bold" Margin="0,2"/>
        <TextBlock Grid.Row="0" Grid.Column="1" Text="{Binding LST}" Margin="10,2"/>

        <TextBlock Grid.Row="1" Grid.Column="0" Text="RA:" FontWeight="Bold" Margin="0,2"/>
        <TextBlock Grid.Row="1" Grid.Column="1" Text="{Binding RA}" Margin="10,2"/>

        <TextBlock Grid.Row="2" Grid.Column="0" Text="Dec:" FontWeight="Bold" Margin="0,2"/>
        <TextBlock Grid.Row="2" Grid.Column="1" Text="{Binding Dec}" Margin="10,2"/>

        <TextBlock Grid.Row="3" Grid.Column="0" Text="Az:" FontWeight="Bold" Margin="0,2"/>
        <TextBlock Grid.Row="3" Grid.Column="1" Text="{Binding Az}" Margin="10,2"/>

        <TextBlock Grid.Row="4" Grid.Column="0" Text="Alt:" FontWeight="Bold" Margin="0,2"/>
        <TextBlock Grid.Row="4" Grid.Column="1" Text="{Binding Alt}" Margin="10,2"/>
    </Grid>
</GroupBox>
```

## Telescope Setup UI Mapping

### XAML
```xml
<UserControl x:Class="BigSkies.TelescopeSetupView"
             xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
             xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml">
    <StackPanel>
        <!-- Telescope Settings -->
        <GroupBox Header="Telescope Settings" Margin="10" Padding="10"
                  IsEnabled="{Binding IsDeviceDisconnected}">
            <StackPanel>
                <CheckBox Content="Auto Unpark / Track on Start"
                          IsChecked="{Binding AutoUnpark}"
                          Margin="0,5"/>

                <StackPanel Orientation="Horizontal" Margin="0,10">
                    <TextBlock Text="Slew Rate (deg/sec):" 
                               VerticalAlignment="Center" Margin="0,0,10,0"/>
                    <Slider Minimum="0" Maximum="360" 
                            Value="{Binding SlewRate}"
                            Width="200" TickFrequency="10"
                            IsSnapToTickEnabled="True"/>
                    <TextBlock Text="{Binding SlewRate, StringFormat={}{0:F0}}" 
                               VerticalAlignment="Center" Margin="10,0"/>
                </StackPanel>
            </StackPanel>
        </GroupBox>

        <!-- Site Information -->
        <GroupBox Header="Site Information" Margin="10" Padding="10">
            <StackPanel Orientation="Horizontal">
                <TextBlock Text="Latitude:" VerticalAlignment="Center" Margin="0,0,10,0"/>
                <ComboBox SelectedIndex="{Binding LatitudeSignIndex}" Width="50">
                    <ComboBoxItem Content="N"/>
                    <ComboBoxItem Content="S"/>
                </ComboBox>
                <TextBox Text="{Binding LatitudeDegrees}" Width="60" Margin="10,0"/>
                <TextBlock Text="°" VerticalAlignment="Center" Margin="5,0"/>
                <TextBox Text="{Binding LatitudeMinutes}" Width="60"/>
                <TextBlock Text="'" VerticalAlignment="Center" Margin="5,0"/>
            </StackPanel>
        </GroupBox>
    </StackPanel>
</UserControl>
```

### View Model
```csharp
public class TelescopeSetupViewModel : INotifyPropertyChanged
{
    private bool _autoUnpark;
    private double _slewRate = 10;
    private bool _deviceConnected;
    private int _latitudeSignIndex;
    private int _latitudeDegrees;
    private int _latitudeMinutes;

    public bool AutoUnpark
    {
        get => _autoUnpark;
        set
        {
            if (_autoUnpark != value)
            {
                _autoUnpark = value;
                OnPropertyChanged(nameof(AutoUnpark));
                PublishConfiguration();
            }
        }
    }

    public double SlewRate
    {
        get => _slewRate;
        set
        {
            if (Math.Abs(_slewRate - value) > 0.01)
            {
                _slewRate = value;
                OnPropertyChanged(nameof(SlewRate));
                PublishConfiguration();
            }
        }
    }

    public bool IsDeviceDisconnected => !_deviceConnected;

    private async void PublishConfiguration()
    {
        var config = new
        {
            auto_unpark = AutoUnpark,
            slew_rate = SlewRate
        };

        var payload = JsonConvert.SerializeObject(config);
        var message = new MqttApplicationMessageBuilder()
            .WithTopic("bigskies/telescope/0/command/config")
            .WithPayload(payload)
            .Build();

        await _mqttClient.PublishAsync(message);
    }

    // INotifyPropertyChanged implementation...
}
```

## BigSkies Framework Integration

### MQTT Client Service
```csharp
using MQTTnet;
using MQTTnet.Client;
using System.Threading.Tasks;

namespace BigSkies.Services
{
    public interface IMqttService
    {
        Task ConnectAsync(string server, int port = 1883);
        Task SubscribeAsync(string topic);
        Task PublishAsync(string topic, string payload);
        event EventHandler<MessageReceivedEventArgs> MessageReceived;
    }

    public class MqttService : IMqttService
    {
        private IMqttClient _client;

        public event EventHandler<MessageReceivedEventArgs> MessageReceived;

        public MqttService()
        {
            _client = new MqttFactory().CreateMqttClient();
            _client.ApplicationMessageReceivedAsync += OnMessageReceived;
        }

        public async Task ConnectAsync(string server, int port = 1883)
        {
            var options = new MqttClientOptionsBuilder()
                .WithTcpServer(server, port)
                .Build();

            await _client.ConnectAsync(options);
        }

        public async Task SubscribeAsync(string topic)
        {
            await _client.SubscribeAsync(topic);
        }

        public async Task PublishAsync(string topic, string payload)
        {
            var message = new MqttApplicationMessageBuilder()
                .WithTopic(topic)
                .WithPayload(payload)
                .Build();

            await _client.PublishAsync(message);
        }

        private Task OnMessageReceived(MqttApplicationMessageReceivedEventArgs e)
        {
            var topic = e.ApplicationMessage.Topic;
            var payload = System.Text.Encoding.UTF8.GetString(e.ApplicationMessage.Payload);

            MessageReceived?.Invoke(this, new MessageReceivedEventArgs(topic, payload));
            return Task.CompletedTask;
        }
    }

    public class MessageReceivedEventArgs : EventArgs
    {
        public string Topic { get; }
        public string Payload { get; }

        public MessageReceivedEventArgs(string topic, string payload)
        {
            Topic = topic;
            Payload = payload;
        }
    }
}
```

### Query UI Elements
```csharp
public async Task QueryUIElementsAsync()
{
    var request = new
    {
        action = "list_elements",
        framework = "wpf",
        type = "panel"
    };

    var payload = JsonConvert.SerializeObject(request);
    await _mqttService.PublishAsync(
        "bigskies/uielement-coordinator/command/query",
        payload
    );

    await _mqttService.SubscribeAsync(
        "bigskies/uielement-coordinator/response/query/+"
    );
}
```

## Control Property Mappings

| Blazor Property | WPF Property/Binding | Notes |
|----------------|---------------------|-------|
| `@bind` | `{Binding Path, Mode=TwoWay}` | Data binding |
| `disabled` | `IsEnabled="False"` | Disables control |
| `@onclick` | `Command="{Binding CommandName}"` | MVVM command binding |
| `style="color:red"` | `Foreground="Red"` or Style | Control styling |
| `id` | `x:Name` | Element name in XAML |
| `class` | `Style="{StaticResource StyleName}"` | Style reference |
| `min`, `max`, `step` | Slider properties | `Minimum`, `Maximum`, `TickFrequency` |

## Best Practices

1. **MVVM Pattern**: Use Model-View-ViewModel for separation of concerns
2. **Data Binding**: Leverage WPF's powerful binding system
3. **Commands**: Use `ICommand` for user actions
4. **Styles & Templates**: Define reusable styles in ResourceDictionaries
5. **Thread Safety**: Use `Dispatcher.Invoke()` for UI thread updates
6. **MQTT Integration**: Use async/await for MQTT operations
7. **Validation**: Implement `IDataErrorInfo` or `INotifyDataErrorInfo`
8. **Dependency Injection**: Use DI container for services

## Device-Specific Mappings

- **Camera**: `Image` control with `BitmapSource`
- **Dome**: Custom control with `OnRender()` for azimuth dial
- **Focuser**: `Slider` with numeric display
- **Switch**: `ListView` or `ItemsControl` with `CheckBox` items

## References

- WPF Documentation: https://docs.microsoft.com/dotnet/desktop/wpf/
- MQTTnet Library: https://github.com/dotnet/MQTTnet
- MVVM Pattern: https://docs.microsoft.com/archive/msdn-magazine/2009/february/patterns-wpf-apps-with-the-model-view-viewmodel-design-pattern
- ASCOM Alpaca API: https://ascom-standards.org/api/
- BigSkies UI Element Coordinator: `internal/coordinators/uielement_coordinator.go`
