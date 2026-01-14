## ADDED Requirements

### Requirement: Native Main Window
The application SHALL provide a main window using native macOS styling.

#### Scenario: Window Launch
- **WHEN** the application is started
- **THEN** a window appears with the title "Excel Translator (Native)"
- **AND** the window has a fixed or reasonable minimum size

### Requirement: File Selection
The user SHALL be able to select an Excel file for translation.

#### Scenario: Browse Button
- **WHEN** the user clicks the "Browse..." button
- **THEN** a native system file picker (`NSOpenPanel`) appears
- **AND** it filters for `.xlsx` and `.docx` files
- **AND** selecting a file populates the file path field

### Requirement: Configuration Interface
The user SHALL be able to configure translation parameters.

#### Scenario: Settings Fields
- **WHEN** the user views the window
- **THEN** they see input fields for "API Key", "API URL", "Model", and "Prompt"
- **AND** these fields are pre-populated from the configuration file (if available)

#### Scenario: CJK Toggle
- **WHEN** the user toggles "Only Translate CJK"
- **THEN** the preference is visually updated

### Requirement: Translation Execution
The user SHALL be able to start and stop the translation process.

#### Scenario: Start Translation
- **WHEN** the user clicks "Start Translation" with a valid file
- **THEN** the UI disables inputs and the "Start" button
- **AND** enables the "Stop" button
- **AND** a progress indicator shows activity

#### Scenario: Stop Translation
- **WHEN** the user clicks "Stop Translation" during an active job
- **THEN** the process halts
- **AND** the UI resets to a ready state

### Requirement: Feedback and Logging
The application SHALL provide real-time feedback.

#### Scenario: Progress Bar
- **WHEN** translation progresses
- **THEN** the progress bar updates to reflect the percentage complete

#### Scenario: Log Output
- **WHEN** the translator emits a log message
- **THEN** the message is appended to a scrollable text area in the window
