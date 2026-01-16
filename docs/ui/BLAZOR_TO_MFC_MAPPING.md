# Blazor to MFC Control Mapping

## Overview
This document provides a comprehensive mapping of UI elements from the ASCOM Alpaca Simulators Blazor interface to Microsoft Foundation Classes (MFC) controls, designed for integration with the BigSkies framework's UI element coordinator.

## Architecture Integration

The UI element coordinator in BigSkies framework will:
1. Track UI element definitions from plugins via MQTT
2. Provide UI element metadata including MFC control mappings
3. Enable MFC frontend to dynamically generate UI from backend API definitions
4. Support real-time updates via MQTT data bindings

## Core UI Element Mappings

### Layout Containers

| Blazor Element | MFC Control | BigSkies Element Type | Notes |
|----------------|-------------|----------------------|-------|
| `<fieldset>` | `CStatic` with `SS_GROUPBOX` | `panel` | Group box for visual grouping |
| `<legend>` | Group box text | N/A | Set via `SetWindowText()` |
| `<div class="grid-container-two">` | Manual layout in `OnSize()` | `panel` | Calculate positions in 2-column grid |
| `<div class="grid-item-left">` | Left column position | N/A | Position child controls |
| `<div class="grid-item-right">` | Right column position | N/A | Position child controls |
| `<div class="centered">` | Centered layout | `panel` | Center controls horizontally |
| `<body>` | `CDialog` or `CFormView` | `panel` | Main dialog/form container |

### Input Controls

| Blazor Element | MFC Control | BigSkies Element Type | Notes |
|----------------|-------------|----------------------|-------|
| `<button>` | `CButton` | `widget` | Push button with `BN_CLICKED` message |
| `<input type="checkbox">` | `CButton` with `BS_AUTOCHECKBOX` | `widget` | Checkbox control |
| `<input type="number">` | `CEdit` with `CSpinButtonCtrl` | `widget` | Edit control with spin buddy |
| `<input type="text">` | `CEdit` | `widget` | Single-line text edit |
| `<select>` | `CComboBox` | `widget` | Dropdown combo box |
| `<option>` | ComboBox item | N/A | Add via `AddString()` |

### Display Controls

| Blazor Element | MFC Control | BigSkies Element Type | Notes |
|----------------|-------------|----------------------|-------|
| `<label>` | `CStatic` | `widget` | Static text control |
| `<p>` | `CStatic` | `widget` | Multi-line static text |
| `<h2>`, `<h3>` | `CStatic` with custom font | `widget` | Use larger font via `CFont` |
| `<svg>` (status circle) | `CStatic` with `SS_OWNERDRAW` | `widget` | Owner-draw for custom painting |
| Dynamic text binding | `SetWindowText()` or `SetDlgItemText()` | N/A | Update via message handlers |

### Navigation

| Blazor Element | MFC Control | BigSkies Element Type | Notes |
|----------------|-------------|----------------------|-------|
| `<NavLink>` | `CTreeCtrl` or `CListBox` item | `menu` | Navigation tree/list item |
| `<ul class="nav flex-column">` | `CTreeCtrl` or `CListBox` | `menu` | Vertical navigation list |
| `<li class="nav-item">` | Tree/list item | `menu` | Individual nav item |
| Navbar | `CToolBar` or `CMenuBar` | `panel` | Application toolbar/menu |

## ASCOM Telescope Control UI Mapping

### Connection Control Section
```cpp
// Blazor Structure:
// <fieldset>
//   <legend>Telescope</legend>
//   <div>
//     <svg circle> + <button>Connect/Disconnect</button>
//     <button>Setup</button>
//   </div>
// </fieldset>

// MFC Header (.h)
class CTelescopeControlDlg : public CDialogEx {
public:
    CStatic m_groupTelescope;
    CStatic m_statusIndicator;  // Owner-draw for circle
    CButton m_btnConnect;
    CButton m_btnSetup;
    BOOL m_bConnected;
    
    enum { IDD = IDD_TELESCOPE_CONTROL };
    
protected:
    virtual void DoDataExchange(CDataExchange* pDX);
    virtual BOOL OnInitDialog();
    afx_msg void OnBnClickedConnect();
    afx_msg void OnBnClickedSetup();
    afx_msg void OnDrawItem(int nIDCtl, LPDRAWITEMSTRUCT lpDrawItemStruct);
    afx_msg void OnTimer(UINT_PTR nIDEvent);
    DECLARE_MESSAGE_MAP()
};

// MFC Implementation (.cpp)
void CTelescopeControlDlg::DoDataExchange(CDataExchange* pDX) {
    CDialogEx::DoDataExchange(pDX);
    DDX_Control(pDX, IDC_GROUP_TELESCOPE, m_groupTelescope);
    DDX_Control(pDX, IDC_STATUS_INDICATOR, m_statusIndicator);
    DDX_Control(pDX, IDC_BTN_CONNECT, m_btnConnect);
    DDX_Control(pDX, IDC_BTN_SETUP, m_btnSetup);
}

BOOL CTelescopeControlDlg::OnInitDialog() {
    CDialogEx::OnInitDialog();
    
    m_groupTelescope.SetWindowText(_T("Telescope"));
    m_btnConnect.SetWindowText(_T("Connect"));
    m_btnSetup.SetWindowText(_T("Setup"));
    m_bConnected = FALSE;
    
    // Start MQTT listener timer
    SetTimer(1, 100, NULL);
    
    return TRUE;
}

void CTelescopeControlDlg::OnBnClickedConnect() {
    // Publish MQTT command
    CMqttClient::GetInstance()->Publish(
        _T("bigskies/telescope/0/command/connect"),
        _T("{\"action\":\"toggle\"}")
    );
}

void CTelescopeControlDlg::OnDrawItem(int nIDCtl, LPDRAWITEMSTRUCT lpDrawItemStruct) {
    if (nIDCtl == IDC_STATUS_INDICATOR) {
        CDC dc;
        dc.Attach(lpDrawItemStruct->hDC);
        
        // Draw connection status circle
        CBrush brush(m_bConnected ? RGB(255, 0, 0) : RGB(128, 128, 128));
        CBrush* pOldBrush = dc.SelectObject(&brush);
        CPen pen(PS_SOLID, 3, RGB(0, 0, 0));
        CPen* pOldPen = dc.SelectObject(&pen);
        
        CRect rect = lpDrawItemStruct->rcItem;
        dc.Ellipse(rect);
        
        dc.SelectObject(pOldBrush);
        dc.SelectObject(pOldPen);
        dc.Detach();
    }
    CDialogEx::OnDrawItem(nIDCtl, lpDrawItemStruct);
}

void CTelescopeControlDlg::OnTimer(UINT_PTR nIDEvent) {
    if (nIDEvent == 1) {
        // Check MQTT messages
        CString strPayload;
        if (CMqttClient::GetInstance()->GetMessage(
            _T("bigskies/telescope/0/state/connected"), strPayload)) {
            // Parse JSON and update state
            m_bConnected = ParseConnectedState(strPayload);
            m_btnConnect.SetWindowText(m_bConnected ? _T("Disconnect") : _T("Connect"));
            m_statusIndicator.Invalidate();
        }
    }
    CDialogEx::OnTimer(nIDEvent);
}
```

### Resource Definition (.rc)
```cpp
IDD_TELESCOPE_CONTROL DIALOGEX 0, 0, 320, 240
STYLE DS_SETFONT | DS_FIXEDSYS | WS_CHILD | WS_SYSMENU
FONT 8, "MS Shell Dlg", 400, 0, 0x1
BEGIN
    GROUPBOX        "Telescope", IDC_GROUP_TELESCOPE, 10, 10, 300, 80
    CONTROL         "", IDC_STATUS_INDICATOR, "Static", SS_OWNERDRAW, 20, 30, 30, 30
    PUSHBUTTON      "Connect", IDC_BTN_CONNECT, 60, 30, 80, 30
    PUSHBUTTON      "Setup", IDC_BTN_SETUP, 200, 30, 80, 30
END
```

### Status Display Section
```cpp
// Header
class CTelescopeStatusDlg : public CDialogEx {
public:
    CStatic m_lblLST, m_lblRA, m_lblDec, m_lblAz, m_lblAlt;
    CStatic m_txtLST, m_txtRA, m_txtDec, m_txtAz, m_txtAlt;
    
protected:
    void UpdateStatus();
    afx_msg void OnTimer(UINT_PTR nIDEvent);
};

// Implementation
void CTelescopeStatusDlg::UpdateStatus() {
    CString strPayload;
    CMqttClient* pClient = CMqttClient::GetInstance();
    
    if (pClient->GetMessage(_T("bigskies/telescope/0/state/lst"), strPayload)) {
        m_txtLST.SetWindowText(strPayload);
    }
    if (pClient->GetMessage(_T("bigskies/telescope/0/state/ra"), strPayload)) {
        m_txtRA.SetWindowText(strPayload);
    }
    if (pClient->GetMessage(_T("bigskies/telescope/0/state/dec"), strPayload)) {
        m_txtDec.SetWindowText(strPayload);
    }
    if (pClient->GetMessage(_T("bigskies/telescope/0/state/az"), strPayload)) {
        m_txtAz.SetWindowText(strPayload);
    }
    if (pClient->GetMessage(_T("bigskies/telescope/0/state/alt"), strPayload)) {
        m_txtAlt.SetWindowText(strPayload);
    }
}
```

## ASCOM Telescope Setup UI Mapping

### Configuration Sections
```cpp
// Header
class CTelescopeSetupDlg : public CDialogEx {
public:
    CButton m_chkAutoUnpark;
    CEdit m_editSlewRate;
    CSpinButtonCtrl m_spinSlewRate;
    BOOL m_bAutoUnpark;
    int m_nSlewRate;
    BOOL m_bDeviceConnected;
    
protected:
    virtual void DoDataExchange(CDataExchange* pDX);
    virtual BOOL OnInitDialog();
    afx_msg void OnBnClickedAutoUnpark();
    afx_msg void OnEnChangeSlewRate();
    void EnableControls();
};

// Implementation
void CTelescopeSetupDlg::DoDataExchange(CDataExchange* pDX) {
    CDialogEx::DoDataExchange(pDX);
    DDX_Control(pDX, IDC_CHK_AUTO_UNPARK, m_chkAutoUnpark);
    DDX_Control(pDX, IDC_EDIT_SLEW_RATE, m_editSlewRate);
    DDX_Control(pDX, IDC_SPIN_SLEW_RATE, m_spinSlewRate);
    DDX_Check(pDX, IDC_CHK_AUTO_UNPARK, m_bAutoUnpark);
    DDX_Text(pDX, IDC_EDIT_SLEW_RATE, m_nSlewRate);
    DDV_MinMaxInt(pDX, m_nSlewRate, 0, 360);
}

BOOL CTelescopeSetupDlg::OnInitDialog() {
    CDialogEx::OnInitDialog();
    
    m_spinSlewRate.SetBuddy(&m_editSlewRate);
    m_spinSlewRate.SetRange(0, 360);
    m_spinSlewRate.SetPos(10);
    
    EnableControls();
    
    return TRUE;
}

void CTelescopeSetupDlg::EnableControls() {
    m_chkAutoUnpark.EnableWindow(!m_bDeviceConnected);
    m_editSlewRate.EnableWindow(!m_bDeviceConnected);
    m_spinSlewRate.EnableWindow(!m_bDeviceConnected);
}

void CTelescopeSetupDlg::OnBnClickedAutoUnpark() {
    UpdateData(TRUE);
    CString strPayload;
    strPayload.Format(_T("{\"auto_unpark\":%s}"), 
        m_bAutoUnpark ? _T("true") : _T("false"));
    CMqttClient::GetInstance()->Publish(
        _T("bigskies/telescope/0/command/config"), strPayload);
}
```

### Site Information Section
```cpp
// Header
class CSiteInfoDlg : public CDialogEx {
public:
    CComboBox m_cmbLatSign;
    CEdit m_editLatDeg, m_editLatMin;
    CSpinButtonCtrl m_spinLatDeg, m_spinLatMin;
    int m_nLatSign;  // 1=N, -1=S
    int m_nLatDeg, m_nLatMin;
    
protected:
    virtual void DoDataExchange(CDataExchange* pDX);
    virtual BOOL OnInitDialog();
};

// Implementation
BOOL CSiteInfoDlg::OnInitDialog() {
    CDialogEx::OnInitDialog();
    
    m_cmbLatSign.AddString(_T("N"));
    m_cmbLatSign.AddString(_T("S"));
    m_cmbLatSign.SetCurSel(0);
    
    m_spinLatDeg.SetBuddy(&m_editLatDeg);
    m_spinLatDeg.SetRange(0, 90);
    
    m_spinLatMin.SetBuddy(&m_editLatMin);
    m_spinLatMin.SetRange(0, 60);
    
    return TRUE;
}
```

## BigSkies Framework Integration

### MQTT Client Wrapper
```cpp
// MqttClient.h
class CMqttClient {
public:
    static CMqttClient* GetInstance();
    
    BOOL Connect(LPCTSTR lpszServer, int nPort = 1883);
    void Disconnect();
    BOOL Publish(LPCTSTR lpszTopic, LPCTSTR lpszPayload);
    BOOL Subscribe(LPCTSTR lpszTopic);
    BOOL GetMessage(LPCTSTR lpszTopic, CString& strPayload);
    
private:
    CMqttClient();
    static CMqttClient* m_pInstance;
    mqtt::async_client* m_pClient;
    std::map<CString, CString> m_messages;
    CCriticalSection m_cs;
};

// Usage example
CMqttClient::GetInstance()->Connect(_T("localhost"), 1883);
CMqttClient::GetInstance()->Subscribe(_T("bigskies/telescope/0/state/+"));
CMqttClient::GetInstance()->Publish(
    _T("bigskies/telescope/0/command/connect"),
    _T("{\"action\":\"toggle\"}")
);
```

### UI Element Query
```cpp
void CMainFrame::QueryUIElements() {
    nlohmann::json request = {
        {"action", "list_elements"},
        {"framework", "mfc"},
        {"type", "panel"}
    };
    
    CString strRequest = CA2T(request.dump().c_str());
    CMqttClient::GetInstance()->Publish(
        _T("bigskies/uielement-coordinator/command/query"),
        strRequest
    );
    
    // Subscribe to response
    CMqttClient::GetInstance()->Subscribe(
        _T("bigskies/uielement-coordinator/response/query/+")
    );
}
```

## Control Property Mappings

### Common Properties

| Blazor Property | MFC Property/Method | Notes |
|----------------|---------------------|-------|
| `@bind` | `UpdateData()` / `DoDataExchange()` | Dialog data exchange |
| `disabled` | `EnableWindow(FALSE)` | Disables control |
| `@onclick` | `ON_BN_CLICKED` message handler | Button click |
| `style="color:red"` | `SetTextColor()` or owner-draw | Custom colors |
| `id` | Control ID constant | Define in resource.h |
| `class` | Control style flags | Set in dialog template |
| `min`, `max`, `step` | `CSpinButtonCtrl::SetRange()` | For numeric inputs |

## Best Practices

1. **Dialog Data Exchange**: Use DDX/DDV for data binding
2. **Resource Management**: Define all controls in .rc files
3. **Message Maps**: Use `BEGIN_MESSAGE_MAP` for event handling
4. **MQTT Integration**: Use background thread for MQTT processing
5. **Unicode Support**: Use `_T()` macro and `TCHAR` types
6. **Error Handling**: Use `AfxMessageBox()` for user notifications
7. **Memory Management**: Follow COM/MFC ownership rules
8. **Thread Safety**: Use `CCriticalSection` for shared data

## Device-Specific Mappings

### Camera Control
- Image display: `CStatic` with owner-draw or `CBitmap`
- Exposure controls: `CSliderCtrl` for duration
- Binning: `CComboBox` with predefined values

### Dome Control
- Azimuth control: Custom control derived from `CWnd`
- Shutter: `CButton` with `BS_AUTOCHECKBOX` styled as switch

### Focuser Control
- Position: `CSliderCtrl` with `CEdit` buddy
- Absolute/Relative: `CButton` group with `BS_AUTORADIOBUTTON`

### Switch Control
- Multiple switches: `CListCtrl` with checkboxes enabled

## References

- MFC Documentation: https://docs.microsoft.com/cpp/mfc/
- MQTT C++ Client: https://github.com/eclipse/paho.mqtt.cpp
- ASCOM Alpaca API: https://ascom-standards.org/api/
- BigSkies UI Element Coordinator: `internal/coordinators/uielement_coordinator.go`
