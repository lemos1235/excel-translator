## ADDED Requirements

### Requirement: Log View Visibility
The log view SHALL be visible and occupy a fixed or flexible height in the window.

#### Scenario: App Launch
- **WHEN** the app is launched
- **THEN** the log area is visible with a height of at least 150 points

## MODIFIED Requirements
### Requirement: Feedback and Logging
The application SHALL provide real-time feedback.

#### Scenario: Log Output
- **WHEN** the translator emits a log message
- **THEN** the message is appended to a scrollable text area in the window
- **AND** the text area automatically scrolls to the bottom
