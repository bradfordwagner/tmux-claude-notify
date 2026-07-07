## ADDED Requirements

### Requirement: `/` key is handled in both views to enter search mode
The TUI key handler SHALL intercept `/` in Notifications view and Sessions view (both levels) and delegate to the dashboard-search capability.

#### Scenario: / handled in Notifications view
- **WHEN** `searchMode` is `false` and the user presses `/` in Notifications view
- **THEN** `searchMode` and `searchFocus` are set to `true` and the filter input is initialized and focused

#### Scenario: / handled in Sessions view
- **WHEN** `searchMode` is `false` and the user presses `/` in Sessions view (L1 or L2)
- **THEN** `searchMode` and `searchFocus` are set to `true` and the filter input is initialized and focused

### Requirement: Alphanumeric keys route to the filter input only when input-focused
When `searchMode` is `true` and `searchFocus` is `true`, any key not otherwise intercepted SHALL be forwarded to `textinput.Update` and `searchQuery` updated. When `searchFocus` is `false`, all normal action keys are active.

#### Scenario: Printable keys routed to filter input when input-focused
- **WHEN** `searchMode` is `true`, `searchFocus` is `true`, and the user presses any printable key that is not an action key
- **THEN** the keystroke is forwarded to `textinput.Update` and `searchQuery` is updated

#### Scenario: Action keys active when table-focused
- **WHEN** `searchMode` is `true`, `searchFocus` is `false`, and the user presses an action key (j/k/p/w/h/v/enter/q/s/f)
- **THEN** the action fires normally on the filtered list

## MODIFIED Requirements

### Requirement: `esc` key closes the dashboard or navigates back
The `esc` key SHALL first exit search mode if active (`searchMode=true`). Only when `searchMode` is already `false` does `esc` perform its normal action (back-navigate in Sessions L2, or quit).

#### Scenario: esc quits from Notifications view when not in search mode
- **WHEN** `searchMode` is `false` and `activeView` is `viewNotifications` and the user presses `esc`
- **THEN** the dashboard closes

#### Scenario: esc back-navigates from Sessions L2 when not in search mode
- **WHEN** `searchMode` is `false` and `drillProject` is non-empty and the user presses `esc`
- **THEN** `drillProject` is cleared and the view returns to L1

#### Scenario: esc exits search mode
- **WHEN** `searchMode` is `true` (either `searchFocus` state) and the user presses `esc`
- **THEN** `searchMode` and `searchFocus` become `false` and `searchQuery` is cleared
- **AND** the dashboard does not quit and navigation does not change

### Requirement: `Tab` toggles between input focus and table focus when searching; switches views when not searching
When `searchMode` is `false`, `Tab` switches between Notifications and Sessions views (existing behavior). When `searchMode` is `true`, `Tab` toggles `searchFocus` between `true` (input has keyboard) and `false` (table has keyboard) without switching views or clearing the filter.

#### Scenario: Tab switches view when not in search mode
- **WHEN** `searchMode` is `false` and the user presses `Tab`
- **THEN** the active view switches between Notifications and Sessions

#### Scenario: Tab from input-focused moves focus to table
- **WHEN** `searchMode` is `true`, `searchFocus` is `true`, and the user presses `Tab`
- **THEN** `searchFocus` becomes `false` and the filter stays active

#### Scenario: Tab from table-focused returns focus to input
- **WHEN** `searchMode` is `true`, `searchFocus` is `false`, and the user presses `Tab`
- **THEN** `searchFocus` becomes `true` and the textinput is focused
